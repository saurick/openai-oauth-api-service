package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
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

	h.handleCodexCLIProxy(w, r, key, requestID, endpoint, requestModel, stream, body, start)
}

func (h *openAIGatewayHandler) gatewayRateLimitEnabled() bool {
	if h.dataCfg == nil || h.dataCfg.Api == nil {
		return true
	}
	return h.dataCfg.Api.RateLimitEnabled
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
		model = biz.DefaultCodexModelID
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

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}
