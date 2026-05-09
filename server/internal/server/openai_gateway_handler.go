package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"server/internal/biz"
	"server/internal/conf"
	"server/internal/errcode"

	"github.com/go-kratos/kratos/v2/log"
	httpx "github.com/go-kratos/kratos/v2/transport/http"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	maxGatewayRequestBytes   = 32 << 20
	maxCapturedStreamBytes   = 32 << 20
	defaultCodexCLITimeout   = 600 * time.Second
	gatewayUsageWriteTimeout = 5 * time.Second
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

var codexCLIUpstreamMu sync.Mutex

type openAIUsageMetrics struct {
	Model           string
	InputTokens     int64
	OutputTokens    int64
	TotalTokens     int64
	CachedTokens    int64
	ReasoningTokens int64
}

func registerOpenAIGatewayRoutes(
	srv *httpx.Server,
	logger log.Logger,
	tp *sdktrace.TracerProvider,
	gatewayUC *biz.GatewayUsecase,
	dataCfg *conf.Data,
) {
	helper := log.NewHelper(log.With(logger, "logger.name", "server.http.gateway"))
	handler := newOpenAIGatewayHandler(helper, gatewayUC, dataCfg)

	srv.HandlePrefix("/v1/", newObservedHTTPHandler(logger, tp, "server.http.gateway", func(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
		handler.ServeHTTP(w, r.WithContext(ctx))
	}))
}

type openAIGatewayHandler struct {
	log       *log.Helper
	gatewayUC *biz.GatewayUsecase
	dataCfg   *conf.Data
}

func newOpenAIGatewayHandler(log *log.Helper, gatewayUC *biz.GatewayUsecase, dataCfg *conf.Data) *openAIGatewayHandler {
	return &openAIGatewayHandler{
		log:       log,
		gatewayUC: gatewayUC,
		dataCfg:   dataCfg,
	}
}

func (h *openAIGatewayHandler) ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	start := time.Now()
	requestID := requestIDFromRequest(r)
	if requestID == "" {
		requestID = traceIDFromContext(r.Context())
	}

	key, authErr := h.authenticate(r)
	if authErr != nil {
		h.writeGatewayError(w, stdhttp.StatusUnauthorized, authErr, "")
		return
	}

	switch {
	case r.Method == stdhttp.MethodGet && r.URL.Path == "/v1/models":
		h.handleModels(w, r, key, requestID, start)
	case r.Method == stdhttp.MethodPost && (r.URL.Path == "/v1/chat/completions" || r.URL.Path == "/v1/responses"):
		h.handleProxy(w, r, key, requestID, start)
	default:
		h.writeGatewayError(w, stdhttp.StatusNotFound, fmt.Errorf("unsupported gateway path: %s %s", r.Method, r.URL.Path), "unsupported_path")
	}
}

func (h *openAIGatewayHandler) authenticate(r *stdhttp.Request) (*biz.GatewayAPIKey, error) {
	token := biz.NormalizeGatewayBearer(r.Header.Get("Authorization"))
	if token == "" {
		return nil, biz.ErrGatewayAPIKeyNotFound
	}
	return h.gatewayUC.AuthenticateAPIKey(r.Context(), token)
}

func (h *openAIGatewayHandler) handleModels(w stdhttp.ResponseWriter, r *stdhttp.Request, key *biz.GatewayAPIKey, requestID string, start time.Time) {
	models, _, err := h.gatewayUC.ListModels(r.Context(), 200, 0, true, "")
	status := stdhttp.StatusOK
	responseBytes := int64(0)
	errorType := ""
	if err != nil {
		status = stdhttp.StatusInternalServerError
		errorType = "model_list_failed"
		h.writeGatewayError(w, status, err, errorType)
	} else {
		items := make([]map[string]any, 0, len(models))
		for _, item := range models {
			created := item.CreatedUnix
			if created == 0 {
				created = item.CreatedAt.Unix()
			}
			items = append(items, map[string]any{
				"id":       item.ModelID,
				"object":   "model",
				"created":  created,
				"owned_by": item.OwnedBy,
			})
		}
		payload := map[string]any{
			"object": "list",
			"data":   items,
		}
		body, _ := json.Marshal(payload)
		responseBytes = int64(len(body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}

	h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
		APIKeyID:      key.ID,
		APIKeyPrefix:  key.KeyPrefix,
		RequestID:     requestID,
		Method:        r.Method,
		Path:          r.URL.Path,
		Endpoint:      "models",
		StatusCode:    status,
		Success:       status >= 200 && status < 300,
		ResponseBytes: responseBytes,
		DurationMS:    time.Since(start).Milliseconds(),
		ErrorType:     errorType,
		CreatedAt:     time.Now(),
	})
}

func (h *openAIGatewayHandler) handleProxy(w stdhttp.ResponseWriter, r *stdhttp.Request, key *biz.GatewayAPIKey, requestID string, start time.Time) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxGatewayRequestBytes+1))
	if err != nil {
		h.writeGatewayError(w, stdhttp.StatusBadRequest, err, "read_request_failed")
		return
	}
	if len(body) > maxGatewayRequestBytes {
		h.writeGatewayError(w, stdhttp.StatusRequestEntityTooLarge, fmt.Errorf("request body too large"), "request_too_large")
		return
	}

	requestModel, stream := parseRequestModelAndStream(body)
	endpoint := gatewayEndpointFromPath(r.URL.Path)
	if err := h.gatewayUC.ValidateModelAccess(r.Context(), key, requestModel); err != nil {
		status := stdhttp.StatusForbidden
		h.writeGatewayError(w, status, err, gatewayErrorType(err))
		h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
			APIKeyID:     key.ID,
			APIKeyPrefix: key.KeyPrefix,
			RequestID:    requestID,
			Method:       r.Method,
			Path:         r.URL.Path,
			Endpoint:     endpoint,
			Model:        requestModel,
			StatusCode:   status,
			Success:      false,
			Stream:       stream,
			RequestBytes: int64(len(body)),
			DurationMS:   time.Since(start).Milliseconds(),
			ErrorType:    gatewayErrorType(err),
			CreatedAt:    time.Now(),
		})
		return
	}

	if h.gatewayRateLimitEnabled() {
		if err := h.gatewayUC.CheckAPIKeyTokenQuota(r.Context(), key, time.Now()); err != nil {
			status := stdhttp.StatusTooManyRequests
			h.writeGatewayError(w, status, err, gatewayErrorType(err))
			h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
				APIKeyID:     key.ID,
				APIKeyPrefix: key.KeyPrefix,
				RequestID:    requestID,
				Method:       r.Method,
				Path:         r.URL.Path,
				Endpoint:     endpoint,
				Model:        requestModel,
				StatusCode:   status,
				Success:      false,
				Stream:       stream,
				RequestBytes: int64(len(body)),
				DurationMS:   time.Since(start).Milliseconds(),
				ErrorType:    gatewayErrorType(err),
				CreatedAt:    time.Now(),
			})
			h.gatewayUC.CreateAuditLog(r.Context(), biz.GatewayAuditLog{
				ActorID:    key.ID,
				ActorName:  key.KeyPrefix,
				ActorRole:  "api_key",
				Action:     "api.request_blocked",
				TargetType: "api_key",
				TargetID:   fmt.Sprint(key.ID),
				Metadata: map[string]any{
					"model":      requestModel,
					"endpoint":   endpoint,
					"error_type": gatewayErrorType(err),
				},
			})
			return
		}
		if err := h.gatewayUC.CheckPolicy(r.Context(), key, requestModel, time.Now()); err != nil {
			status := stdhttp.StatusTooManyRequests
			h.writeGatewayError(w, status, err, gatewayErrorType(err))
			h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
				APIKeyID:     key.ID,
				APIKeyPrefix: key.KeyPrefix,
				RequestID:    requestID,
				Method:       r.Method,
				Path:         r.URL.Path,
				Endpoint:     endpoint,
				Model:        requestModel,
				StatusCode:   status,
				Success:      false,
				Stream:       stream,
				RequestBytes: int64(len(body)),
				DurationMS:   time.Since(start).Milliseconds(),
				ErrorType:    gatewayErrorType(err),
				CreatedAt:    time.Now(),
			})
			h.gatewayUC.CreateAuditLog(r.Context(), biz.GatewayAuditLog{
				ActorID:    key.ID,
				ActorName:  key.KeyPrefix,
				ActorRole:  "api_key",
				Action:     "api.request_blocked",
				TargetType: "api_key",
				TargetID:   fmt.Sprint(key.ID),
				Metadata: map[string]any{
					"model":      requestModel,
					"endpoint":   endpoint,
					"error_type": gatewayErrorType(err),
				},
			})
			return
		}
	}

	if h.useCodexCLIUpstream() {
		h.handleCodexCLIProxy(w, r, key, requestID, endpoint, requestModel, stream, body, start)
		return
	}

	upstreamReq, err := h.buildUpstreamRequest(r, body)
	if err != nil {
		status := stdhttp.StatusInternalServerError
		h.writeGatewayError(w, status, err, "upstream_config_error")
		h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
			APIKeyID:     key.ID,
			APIKeyPrefix: key.KeyPrefix,
			RequestID:    requestID,
			Method:       r.Method,
			Path:         r.URL.Path,
			Endpoint:     endpoint,
			Model:        requestModel,
			StatusCode:   status,
			Success:      false,
			Stream:       stream,
			RequestBytes: int64(len(body)),
			DurationMS:   time.Since(start).Milliseconds(),
			ErrorType:    "upstream_config_error",
			CreatedAt:    time.Now(),
		})
		return
	}

	client, err := h.buildHTTPClient()
	if err != nil {
		status := stdhttp.StatusInternalServerError
		h.writeGatewayError(w, status, err, "upstream_proxy_config_error")
		return
	}

	upstreamResp, err := client.Do(upstreamReq)
	if err != nil {
		status := stdhttp.StatusBadGateway
		h.writeGatewayError(w, status, err, "upstream_request_failed")
		h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
			APIKeyID:     key.ID,
			APIKeyPrefix: key.KeyPrefix,
			RequestID:    requestID,
			Method:       r.Method,
			Path:         r.URL.Path,
			Endpoint:     endpoint,
			Model:        requestModel,
			StatusCode:   status,
			Success:      false,
			Stream:       stream,
			RequestBytes: int64(len(body)),
			DurationMS:   time.Since(start).Milliseconds(),
			ErrorType:    "upstream_request_failed",
			CreatedAt:    time.Now(),
		})
		return
	}
	defer func() {
		_ = upstreamResp.Body.Close()
	}()

	if stream || strings.Contains(upstreamResp.Header.Get("Content-Type"), "text/event-stream") {
		h.proxyStream(w, r, upstreamResp, key, requestID, endpoint, requestModel, int64(len(body)), start)
		return
	}
	h.proxyJSON(w, r, upstreamResp, key, requestID, endpoint, requestModel, int64(len(body)), start)
}

func (h *openAIGatewayHandler) gatewayRateLimitEnabled() bool {
	if h.dataCfg == nil || h.dataCfg.Api == nil {
		return true
	}
	return h.dataCfg.Api.RateLimitEnabled
}

func (h *openAIGatewayHandler) useCodexCLIUpstream() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("OAUTH_API_UPSTREAM_PROVIDER")), "codex_cli")
}

func (h *openAIGatewayHandler) handleCodexCLIProxy(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	key *biz.GatewayAPIKey,
	requestID string,
	endpoint string,
	requestModel string,
	stream bool,
	body []byte,
	start time.Time,
) {
	content, metrics, err := h.runCodexCLI(r.Context(), r.URL.Path, body, requestModel)
	if err != nil {
		status := stdhttp.StatusBadGateway
		h.writeGatewayError(w, status, err, "codex_cli_upstream_failed")
		h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
			APIKeyID:     key.ID,
			APIKeyPrefix: key.KeyPrefix,
			RequestID:    requestID,
			Method:       r.Method,
			Path:         r.URL.Path,
			Endpoint:     endpoint,
			Model:        requestModel,
			StatusCode:   status,
			Success:      false,
			Stream:       stream,
			RequestBytes: int64(len(body)),
			DurationMS:   time.Since(start).Milliseconds(),
			ErrorType:    "codex_cli_upstream_failed",
			CreatedAt:    time.Now(),
		})
		return
	}
	if metrics.Model == "" {
		metrics.Model = requestModel
	}

	if stream {
		responseBytes := h.writeCodexCLIChatStream(w, metrics.Model, content, metrics)
		h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
			APIKeyID:      key.ID,
			APIKeyPrefix:  key.KeyPrefix,
			RequestID:     requestID,
			Method:        r.Method,
			Path:          r.URL.Path,
			Endpoint:      endpoint,
			Model:         metrics.Model,
			StatusCode:    stdhttp.StatusOK,
			Success:       true,
			Stream:        true,
			InputTokens:   metrics.InputTokens,
			OutputTokens:  metrics.OutputTokens,
			TotalTokens:   metrics.TotalTokens,
			RequestBytes:  int64(len(body)),
			ResponseBytes: responseBytes,
			DurationMS:    time.Since(start).Milliseconds(),
			CreatedAt:     time.Now(),
		})
		return
	}

	var responseBody []byte
	var marshalErr error
	if r.URL.Path == "/v1/responses" {
		responseBody, marshalErr = json.Marshal(buildCodexCLIResponsesPayload(metrics.Model, content, metrics))
	} else {
		responseBody, marshalErr = json.Marshal(buildCodexCLIChatPayload(metrics.Model, content, metrics))
	}
	if marshalErr != nil {
		h.writeGatewayError(w, stdhttp.StatusInternalServerError, marshalErr, "codex_cli_response_encode_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(stdhttp.StatusOK)
	_, _ = w.Write(responseBody)
	h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
		APIKeyID:      key.ID,
		APIKeyPrefix:  key.KeyPrefix,
		RequestID:     requestID,
		Method:        r.Method,
		Path:          r.URL.Path,
		Endpoint:      endpoint,
		Model:         metrics.Model,
		StatusCode:    stdhttp.StatusOK,
		Success:       true,
		Stream:        false,
		InputTokens:   metrics.InputTokens,
		OutputTokens:  metrics.OutputTokens,
		TotalTokens:   metrics.TotalTokens,
		RequestBytes:  int64(len(body)),
		ResponseBytes: int64(len(responseBody)),
		DurationMS:    time.Since(start).Milliseconds(),
		CreatedAt:     time.Now(),
	})
}

func (h *openAIGatewayHandler) runCodexCLI(ctx context.Context, path string, body []byte, requestModel string) (string, openAIUsageMetrics, error) {
	prompt, err := promptFromGatewayRequest(path, body)
	if err != nil {
		return "", openAIUsageMetrics{}, err
	}
	if strings.TrimSpace(prompt) == "" {
		return "", openAIUsageMetrics{}, fmt.Errorf("codex cli upstream prompt is empty")
	}

	codexCLIUpstreamMu.Lock()
	defer codexCLIUpstreamMu.Unlock()

	model := strings.TrimSpace(requestModel)
	if model == "" {
		model = strings.TrimSpace(os.Getenv("CODEX_CLI_MODEL"))
	}
	if model == "" {
		model = "gpt-5.5"
	}

	timeout := codexCLITimeout()
	cmdCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	bin := strings.TrimSpace(os.Getenv("CODEX_CLI_BIN"))
	if bin == "" {
		bin = "codex"
	}
	args := []string{
		"exec",
		"--skip-git-repo-check",
		"--ephemeral",
		"--ignore-user-config",
		"--ignore-rules",
		"-s", "read-only",
		"-m", model,
		"-",
	}
	cmd := exec.CommandContext(cmdCtx, bin, args...)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Env = os.Environ()
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		cmd.Env = append(cmd.Env, "CODEX_HOME="+home)
	}
	if pathEnv := strings.TrimSpace(os.Getenv("CODEX_CLI_PATH")); pathEnv != "" {
		cmd.Env = append(cmd.Env, "PATH="+pathEnv+":"+os.Getenv("PATH"))
	}
	output, err := cmd.CombinedOutput()
	if cmdCtx.Err() == context.DeadlineExceeded {
		return "", openAIUsageMetrics{}, fmt.Errorf("codex cli upstream timed out after %s", timeout)
	}
	if err != nil {
		return "", openAIUsageMetrics{}, fmt.Errorf("codex cli upstream failed: %w: %s", err, summarizeCommandOutput(output))
	}

	content := extractCodexCLIAnswer(output)
	if strings.TrimSpace(content) == "" {
		return "", openAIUsageMetrics{}, fmt.Errorf("codex cli upstream returned empty answer")
	}
	metrics := estimateCodexCLIUsage(model, prompt, content)
	return content, metrics, nil
}

func promptFromGatewayRequest(path string, body []byte) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if path == "/v1/responses" {
		return promptFromResponsesPayload(payload), nil
	}
	return promptFromChatCompletionsPayload(payload), nil
}

func promptFromChatCompletionsPayload(payload map[string]any) string {
	messages, _ := payload["messages"].([]any)
	parts := make([]string, 0, len(messages))
	for _, item := range messages {
		message, _ := item.(map[string]any)
		if message == nil {
			continue
		}
		role := strings.TrimSpace(stringValue(message["role"]))
		if role != "user" && role != "assistant" {
			continue
		}
		content := contentTextValue(message["content"])
		if strings.TrimSpace(content) == "" {
			continue
		}
		parts = append(parts, role+":\n"+content)
	}
	parts = lastStringItems(parts, 8)
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func promptFromResponsesPayload(payload map[string]any) string {
	input := payload["input"]
	switch v := input.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			message, _ := item.(map[string]any)
			if message == nil {
				continue
			}
			role := strings.TrimSpace(stringValue(message["role"]))
			if role != "user" && role != "assistant" {
				continue
			}
			content := contentTextValue(message["content"])
			if strings.TrimSpace(content) == "" {
				continue
			}
			parts = append(parts, role+":\n"+content)
		}
		parts = lastStringItems(parts, 8)
		return strings.TrimSpace(strings.Join(parts, "\n\n"))
	default:
		return strings.TrimSpace(contentTextValue(input))
	}
}

func lastStringItems(items []string, limit int) []string {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

func contentTextValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			switch typed := item.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					parts = append(parts, typed)
				}
			case map[string]any:
				if text := stringValue(typed["text"]); strings.TrimSpace(text) != "" {
					parts = append(parts, text)
					continue
				}
				if text := stringValue(typed["input_text"]); strings.TrimSpace(text) != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		return stringValue(v["text"])
	default:
		return ""
	}
}

func codexCLITimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("CODEX_CLI_TIMEOUT_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultCodexCLITimeout
}

func extractCodexCLIAnswer(output []byte) string {
	text := ansiEscapePattern.ReplaceAllString(string(output), "")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	marker := "\ntokens used\n"
	if idx := strings.LastIndex(text, marker); idx >= 0 {
		after := strings.TrimSpace(text[idx+len(marker):])
		lines := strings.Split(after, "\n")
		if len(lines) >= 2 {
			return strings.TrimSpace(strings.Join(lines[1:], "\n"))
		}
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}

func summarizeCommandOutput(output []byte) string {
	text := ansiEscapePattern.ReplaceAllString(string(output), "")
	text = strings.TrimSpace(text)
	const maxLen = 800
	if len(text) > maxLen {
		text = text[len(text)-maxLen:]
	}
	return text
}

func estimateCodexCLIUsage(model string, prompt string, content string) openAIUsageMetrics {
	inputTokens := estimateTokenCount(prompt)
	outputTokens := estimateTokenCount(content)
	return openAIUsageMetrics{
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
	}
}

func estimateTokenCount(text string) int64 {
	runes := int64(len([]rune(text)))
	if runes == 0 {
		return 0
	}
	tokens := (runes + 3) / 4
	if tokens < 1 {
		return 1
	}
	return tokens
}

func buildCodexCLIChatPayload(model string, content string, metrics openAIUsageMetrics) map[string]any {
	now := time.Now().Unix()
	return map[string]any{
		"id":      fmt.Sprintf("chatcmpl-codex-%d", now),
		"object":  "chat.completion",
		"created": now,
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": chatUsagePayload(metrics),
	}
}

func buildCodexCLIResponsesPayload(model string, content string, metrics openAIUsageMetrics) map[string]any {
	now := time.Now().Unix()
	return map[string]any{
		"id":                  fmt.Sprintf("resp_codex_%d", now),
		"object":              "response",
		"created_at":          now,
		"model":               model,
		"status":              "completed",
		"output_text":         content,
		"parallel_tool_calls": false,
		"output": []map[string]any{
			{
				"id":     fmt.Sprintf("msg_codex_%d", now),
				"type":   "message",
				"role":   "assistant",
				"status": "completed",
				"content": []map[string]any{
					{
						"type": "output_text",
						"text": content,
					},
				},
			},
		},
		"usage": responsesUsagePayload(metrics),
	}
}

func (h *openAIGatewayHandler) writeCodexCLIChatStream(w stdhttp.ResponseWriter, model string, content string, metrics openAIUsageMetrics) int64 {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(stdhttp.StatusOK)
	flusher, _ := w.(stdhttp.Flusher)

	id := fmt.Sprintf("chatcmpl-codex-%d", time.Now().Unix())
	created := time.Now().Unix()
	chunks := []map[string]any{
		{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []map[string]any{
				{
					"index": 0,
					"delta": map[string]any{
						"role":    "assistant",
						"content": content,
					},
					"finish_reason": nil,
				},
			},
		},
		{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []map[string]any{
				{
					"index":         0,
					"delta":         map[string]any{},
					"finish_reason": "stop",
				},
			},
			"usage": chatUsagePayload(metrics),
		},
	}

	responseBytes := int64(0)
	for _, chunk := range chunks {
		body, _ := json.Marshal(chunk)
		line := append([]byte("data: "), body...)
		line = append(line, '\n', '\n')
		n, _ := w.Write(line)
		responseBytes += int64(n)
		if flusher != nil {
			flusher.Flush()
		}
	}
	n, _ := w.Write([]byte("data: [DONE]\n\n"))
	responseBytes += int64(n)
	if flusher != nil {
		flusher.Flush()
	}
	return responseBytes
}

func chatUsagePayload(metrics openAIUsageMetrics) map[string]any {
	return map[string]any{
		"prompt_tokens":     metrics.InputTokens,
		"completion_tokens": metrics.OutputTokens,
		"total_tokens":      metrics.TotalTokens,
	}
}

func responsesUsagePayload(metrics openAIUsageMetrics) map[string]any {
	return map[string]any{
		"input_tokens":  metrics.InputTokens,
		"output_tokens": metrics.OutputTokens,
		"total_tokens":  metrics.TotalTokens,
	}
}

func (h *openAIGatewayHandler) proxyJSON(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	upstreamResp *stdhttp.Response,
	key *biz.GatewayAPIKey,
	requestID string,
	endpoint string,
	requestModel string,
	requestBytes int64,
	start time.Time,
) {
	body, readErr := io.ReadAll(upstreamResp.Body)
	if readErr != nil {
		h.writeGatewayError(w, stdhttp.StatusBadGateway, readErr, "upstream_read_failed")
		return
	}

	copyGatewayResponseHeaders(w.Header(), upstreamResp.Header)
	w.WriteHeader(upstreamResp.StatusCode)
	_, _ = w.Write(body)

	metrics := extractUsageFromJSON(body)
	if metrics.Model == "" {
		metrics.Model = requestModel
	}
	errorType := ""
	if upstreamResp.StatusCode < 200 || upstreamResp.StatusCode >= 300 {
		errorType = fmt.Sprintf("upstream_http_%d", upstreamResp.StatusCode)
	}
	h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
		APIKeyID:        key.ID,
		APIKeyPrefix:    key.KeyPrefix,
		RequestID:       requestID,
		Method:          r.Method,
		Path:            r.URL.Path,
		Endpoint:        endpoint,
		Model:           metrics.Model,
		StatusCode:      upstreamResp.StatusCode,
		Success:         upstreamResp.StatusCode >= 200 && upstreamResp.StatusCode < 300,
		Stream:          false,
		InputTokens:     metrics.InputTokens,
		OutputTokens:    metrics.OutputTokens,
		TotalTokens:     metrics.TotalTokens,
		CachedTokens:    metrics.CachedTokens,
		ReasoningTokens: metrics.ReasoningTokens,
		RequestBytes:    requestBytes,
		ResponseBytes:   int64(len(body)),
		DurationMS:      time.Since(start).Milliseconds(),
		ErrorType:       errorType,
		CreatedAt:       time.Now(),
	})
}

func (h *openAIGatewayHandler) proxyStream(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	upstreamResp *stdhttp.Response,
	key *biz.GatewayAPIKey,
	requestID string,
	endpoint string,
	requestModel string,
	requestBytes int64,
	start time.Time,
) {
	copyGatewayResponseHeaders(w.Header(), upstreamResp.Header)
	w.WriteHeader(upstreamResp.StatusCode)

	flusher, _ := w.(stdhttp.Flusher)
	buf := make([]byte, 32*1024)
	var captured bytes.Buffer
	responseBytes := int64(0)
	var readErr error
	for {
		n, err := upstreamResp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			responseBytes += int64(n)
			if captured.Len() < maxCapturedStreamBytes {
				remaining := maxCapturedStreamBytes - captured.Len()
				if n > remaining {
					_, _ = captured.Write(chunk[:remaining])
				} else {
					_, _ = captured.Write(chunk)
				}
			}
			_, _ = w.Write(chunk)
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				readErr = err
			}
			break
		}
	}

	metrics := extractUsageFromSSE(captured.Bytes())
	if metrics.Model == "" {
		metrics.Model = requestModel
	}
	errorType := ""
	if upstreamResp.StatusCode < 200 || upstreamResp.StatusCode >= 300 {
		errorType = fmt.Sprintf("upstream_http_%d", upstreamResp.StatusCode)
	}
	if readErr != nil {
		errorType = "upstream_stream_read_failed"
	}

	h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
		APIKeyID:        key.ID,
		APIKeyPrefix:    key.KeyPrefix,
		RequestID:       requestID,
		Method:          r.Method,
		Path:            r.URL.Path,
		Endpoint:        endpoint,
		Model:           metrics.Model,
		StatusCode:      upstreamResp.StatusCode,
		Success:         readErr == nil && upstreamResp.StatusCode >= 200 && upstreamResp.StatusCode < 300,
		Stream:          true,
		InputTokens:     metrics.InputTokens,
		OutputTokens:    metrics.OutputTokens,
		TotalTokens:     metrics.TotalTokens,
		CachedTokens:    metrics.CachedTokens,
		ReasoningTokens: metrics.ReasoningTokens,
		RequestBytes:    requestBytes,
		ResponseBytes:   responseBytes,
		DurationMS:      time.Since(start).Milliseconds(),
		ErrorType:       errorType,
		CreatedAt:       time.Now(),
	})
}

func (h *openAIGatewayHandler) buildUpstreamRequest(r *stdhttp.Request, body []byte) (*stdhttp.Request, error) {
	if h.dataCfg == nil || h.dataCfg.Openai == nil || strings.TrimSpace(h.dataCfg.Openai.ApiKey) == "" {
		return nil, errors.New(errcode.APIUpstreamNotConfigured.Message)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(h.dataCfg.Openai.BaseUrl), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	path := strings.TrimPrefix(r.URL.Path, "/v1")
	target := baseURL + path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	req, err := stdhttp.NewRequestWithContext(r.Context(), r.Method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	copyGatewayRequestHeaders(req.Header, r.Header)
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(h.dataCfg.Openai.ApiKey))
	req.Header.Set("Content-Type", "application/json")
	if requestID := requestIDFromRequest(r); requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
	return req, nil
}

func (h *openAIGatewayHandler) buildHTTPClient() (*stdhttp.Client, error) {
	timeout := 600 * time.Second
	if h.dataCfg != nil && h.dataCfg.Openai != nil && h.dataCfg.Openai.RequestTimeoutSeconds > 0 {
		timeout = time.Duration(h.dataCfg.Openai.RequestTimeoutSeconds) * time.Second
	}

	transport := stdhttp.DefaultTransport.(*stdhttp.Transport).Clone()
	if h.dataCfg != nil && h.dataCfg.Openai != nil && strings.TrimSpace(h.dataCfg.Openai.UpstreamProxyUrl) != "" {
		proxyURL, err := url.Parse(strings.TrimSpace(h.dataCfg.Openai.UpstreamProxyUrl))
		if err != nil {
			return nil, err
		}
		transport.Proxy = stdhttp.ProxyURL(proxyURL)
	}
	return &stdhttp.Client{Transport: transport, Timeout: timeout}, nil
}

func (h *openAIGatewayHandler) recordUsage(ctx context.Context, key *biz.GatewayAPIKey, item *biz.GatewayUsageLog) {
	if item == nil {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.Background(), gatewayUsageWriteTimeout)
	defer cancel()
	if err := h.gatewayUC.CreateUsageLog(writeCtx, item); err != nil {
		h.log.WithContext(ctx).Errorf("record gateway usage failed: %v", err)
	}
	if key != nil && key.ID > 0 {
		if err := h.gatewayUC.TouchAPIKeyUsed(writeCtx, key.ID, time.Now()); err != nil {
			h.log.WithContext(ctx).Warnf("touch gateway key failed: %v", err)
		}
	}
}

func (h *openAIGatewayHandler) writeGatewayError(w stdhttp.ResponseWriter, status int, err error, errorType string) {
	message := stdhttp.StatusText(status)
	if err != nil {
		message = mapGatewayHTTPErrorMessage(err)
	}
	code := errorType
	if code == "" {
		code = "gateway_error"
	}
	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    "gateway_error",
			"code":    code,
		},
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func mapGatewayHTTPErrorMessage(err error) string {
	switch {
	case errors.Is(err, biz.ErrGatewayAPIKeyNotFound):
		return errcode.APIKeyInvalid.Message
	case errors.Is(err, biz.ErrGatewayAPIKeyDisabled):
		return errcode.APIKeyDisabled.Message
	case errors.Is(err, biz.ErrGatewayModelDisabled):
		return errcode.APIModelDisabled.Message
	case errors.Is(err, biz.ErrGatewayModelNotAllowed):
		return errcode.APIModelNotAllowed.Message
	case errors.Is(err, biz.ErrGatewayRateLimited):
		return "API key 限流超限"
	case errors.Is(err, biz.ErrGatewayQuotaExceeded):
		return "API key 配额超限"
	default:
		return err.Error()
	}
}

func gatewayErrorType(err error) string {
	switch {
	case errors.Is(err, biz.ErrGatewayAPIKeyNotFound):
		return "gateway_api_key_invalid"
	case errors.Is(err, biz.ErrGatewayAPIKeyDisabled):
		return "gateway_api_key_disabled"
	case errors.Is(err, biz.ErrGatewayModelDisabled):
		return "gateway_model_disabled"
	case errors.Is(err, biz.ErrGatewayModelNotAllowed):
		return "gateway_model_not_allowed"
	case errors.Is(err, biz.ErrGatewayRateLimited):
		return "gateway_rate_limited"
	case errors.Is(err, biz.ErrGatewayQuotaExceeded):
		return "gateway_quota_exceeded"
	default:
		return "gateway_error"
	}
}

func parseRequestModelAndStream(body []byte) (string, bool) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}
	model, _ := payload["model"].(string)
	stream, _ := payload["stream"].(bool)
	return strings.TrimSpace(model), stream
}

func gatewayEndpointFromPath(path string) string {
	switch path {
	case "/v1/chat/completions":
		return "chat.completions"
	case "/v1/responses":
		return "responses"
	case "/v1/models":
		return "models"
	default:
		return strings.Trim(strings.TrimPrefix(path, "/v1"), "/")
	}
}

func extractUsageFromJSON(body []byte) openAIUsageMetrics {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return openAIUsageMetrics{}
	}
	return usageFromPayload(payload)
}

func extractUsageFromSSE(body []byte) openAIUsageMetrics {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 1024), maxCapturedStreamBytes)
	var out openAIUsageMetrics
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if line == "" || line == "[DONE]" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			continue
		}
		metrics := usageFromPayload(payload)
		if metrics.Model != "" {
			out.Model = metrics.Model
		}
		if metrics.TotalTokens > 0 || metrics.InputTokens > 0 || metrics.OutputTokens > 0 {
			out.InputTokens = metrics.InputTokens
			out.OutputTokens = metrics.OutputTokens
			out.TotalTokens = metrics.TotalTokens
			out.CachedTokens = metrics.CachedTokens
			out.ReasoningTokens = metrics.ReasoningTokens
		}
	}
	return out
}

func usageFromPayload(payload map[string]any) openAIUsageMetrics {
	if payload == nil {
		return openAIUsageMetrics{}
	}
	model := stringValue(payload["model"])
	if response, ok := payload["response"].(map[string]any); ok {
		nested := usageFromPayload(response)
		if nested.Model == "" {
			nested.Model = model
		}
		return nested
	}
	usage, _ := payload["usage"].(map[string]any)
	if usage == nil {
		return openAIUsageMetrics{Model: model}
	}

	inputTokens := int64Value(usage["input_tokens"])
	if inputTokens == 0 {
		inputTokens = int64Value(usage["prompt_tokens"])
	}
	outputTokens := int64Value(usage["output_tokens"])
	if outputTokens == 0 {
		outputTokens = int64Value(usage["completion_tokens"])
	}
	totalTokens := int64Value(usage["total_tokens"])
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}

	cachedTokens := int64(0)
	if details, ok := usage["input_tokens_details"].(map[string]any); ok {
		cachedTokens = int64Value(details["cached_tokens"])
	}
	if cachedTokens == 0 {
		if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
			cachedTokens = int64Value(details["cached_tokens"])
		}
	}

	reasoningTokens := int64(0)
	if details, ok := usage["output_tokens_details"].(map[string]any); ok {
		reasoningTokens = int64Value(details["reasoning_tokens"])
	}
	if reasoningTokens == 0 {
		if details, ok := usage["completion_tokens_details"].(map[string]any); ok {
			reasoningTokens = int64Value(details["reasoning_tokens"])
		}
	}

	return openAIUsageMetrics{
		Model:           model,
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		TotalTokens:     totalTokens,
		CachedTokens:    cachedTokens,
		ReasoningTokens: reasoningTokens,
	}
}

func int64Value(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func copyGatewayRequestHeaders(dst, src stdhttp.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) || strings.EqualFold(key, "Authorization") || strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func copyGatewayResponseHeaders(dst, src stdhttp.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) || strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func isHopByHopHeader(key string) bool {
	switch strings.ToLower(key) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}
