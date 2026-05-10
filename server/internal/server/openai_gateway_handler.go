package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	maxGatewayImages         = 4
	maxGatewayImageBytes     = 16 << 20
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

type codexUpstreamCallResult struct {
	Content           string
	Metrics           openAIUsageMetrics
	ActualMode        string
	Fallback          bool
	UpstreamErrorType string
}

type codexCLIPrompt struct {
	Text       string
	ImageFiles []string
	cleanup    func()
}

type gatewayImageSource struct {
	Raw       string
	MediaType string
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
		SessionID:     sessionIDFromHeaders(r),
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

	requestModel, stream, reasoningEffort, parseErr := parseRequestModelStreamAndReasoningEffort(body)
	sessionID := sessionIDFromGatewayRequest(r, body)
	endpoint := gatewayEndpointFromPath(r.URL.Path)
	if parseErr != nil {
		status := stdhttp.StatusBadRequest
		h.writeGatewayError(w, status, parseErr, "gateway_reasoning_effort_invalid")
		h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
			APIKeyID:     key.ID,
			APIKeyPrefix: key.KeyPrefix,
			SessionID:    sessionID,
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
			ErrorType:    "gateway_reasoning_effort_invalid",
			CreatedAt:    time.Now(),
		})
		return
	}
	if err := h.gatewayUC.ValidateModelAccess(r.Context(), key, requestModel); err != nil {
		status := stdhttp.StatusForbidden
		h.writeGatewayError(w, status, err, gatewayErrorType(err))
		h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
			APIKeyID:     key.ID,
			APIKeyPrefix: key.KeyPrefix,
			SessionID:    sessionID,
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
				SessionID:    sessionID,
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
				SessionID:    sessionID,
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

	h.handleCodexCLIProxy(w, r, key, requestID, sessionID, endpoint, requestModel, reasoningEffort, stream, body, start)
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
	sessionID string,
	endpoint string,
	requestModel string,
	reasoningEffort string,
	stream bool,
	body []byte,
	start time.Time,
) {
	upstreamMode := h.configuredCodexUpstreamMode(r.Context())
	result, err := h.runCodexUpstream(r.Context(), upstreamMode, r.URL.Path, body, requestModel, reasoningEffort)
	if err != nil {
		status := stdhttp.StatusBadGateway
		errorType := "codex_cli_upstream_failed"
		if upstreamMode == codexUpstreamModeBackend {
			errorType = "codex_backend_upstream_failed"
		}
		h.writeGatewayError(w, status, err, errorType)
		h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
			APIKeyID:               key.ID,
			APIKeyPrefix:           key.KeyPrefix,
			SessionID:              sessionID,
			RequestID:              requestID,
			Method:                 r.Method,
			Path:                   r.URL.Path,
			Endpoint:               endpoint,
			Model:                  requestModel,
			StatusCode:             status,
			Success:                false,
			Stream:                 stream,
			RequestBytes:           int64(len(body)),
			DurationMS:             time.Since(start).Milliseconds(),
			UpstreamConfiguredMode: upstreamMode,
			UpstreamMode:           upstreamMode,
			UpstreamErrorType:      errorType,
			ErrorType:              errorType,
			CreatedAt:              time.Now(),
		})
		return
	}
	content := result.Content
	metrics := result.Metrics
	if metrics.Model == "" {
		metrics.Model = requestModel
	}

	if stream {
		responseBytes := h.writeCodexCLIChatStream(w, metrics.Model, content, metrics)
		h.recordUsage(r.Context(), key, &biz.GatewayUsageLog{
			APIKeyID:               key.ID,
			APIKeyPrefix:           key.KeyPrefix,
			SessionID:              sessionID,
			RequestID:              requestID,
			Method:                 r.Method,
			Path:                   r.URL.Path,
			Endpoint:               endpoint,
			Model:                  metrics.Model,
			StatusCode:             stdhttp.StatusOK,
			Success:                true,
			Stream:                 true,
			InputTokens:            metrics.InputTokens,
			OutputTokens:           metrics.OutputTokens,
			TotalTokens:            metrics.TotalTokens,
			RequestBytes:           int64(len(body)),
			ResponseBytes:          responseBytes,
			DurationMS:             time.Since(start).Milliseconds(),
			UpstreamConfiguredMode: upstreamMode,
			UpstreamMode:           result.ActualMode,
			UpstreamFallback:       result.Fallback,
			UpstreamErrorType:      result.UpstreamErrorType,
			CreatedAt:              time.Now(),
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
		APIKeyID:               key.ID,
		APIKeyPrefix:           key.KeyPrefix,
		SessionID:              sessionID,
		RequestID:              requestID,
		Method:                 r.Method,
		Path:                   r.URL.Path,
		Endpoint:               endpoint,
		Model:                  metrics.Model,
		StatusCode:             stdhttp.StatusOK,
		Success:                true,
		Stream:                 false,
		InputTokens:            metrics.InputTokens,
		OutputTokens:           metrics.OutputTokens,
		TotalTokens:            metrics.TotalTokens,
		RequestBytes:           int64(len(body)),
		ResponseBytes:          int64(len(responseBody)),
		DurationMS:             time.Since(start).Milliseconds(),
		UpstreamConfiguredMode: upstreamMode,
		UpstreamMode:           result.ActualMode,
		UpstreamFallback:       result.Fallback,
		UpstreamErrorType:      result.UpstreamErrorType,
		CreatedAt:              time.Now(),
	})
}

func (h *openAIGatewayHandler) configuredCodexUpstreamMode(ctx context.Context) string {
	if h.gatewayUC == nil {
		return codexUpstreamMode()
	}
	mode, err := h.gatewayUC.GetCodexUpstreamMode(ctx)
	if err != nil {
		h.log.WithContext(ctx).Warnf("get codex upstream mode failed: %v", err)
		return codexUpstreamMode()
	}
	if mode == "" {
		return codexUpstreamMode()
	}
	return mode
}

func (h *openAIGatewayHandler) runCodexUpstream(ctx context.Context, upstreamMode string, path string, body []byte, requestModel string, reasoningEffort string) (codexUpstreamCallResult, error) {
	upstreamMode = biz.NormalizeGatewayUpstreamMode(upstreamMode)
	if upstreamMode == "" {
		upstreamMode = codexUpstreamMode()
	}
	switch upstreamMode {
	case codexUpstreamModeBackend:
		content, metrics, err := h.runCodexBackend(ctx, path, body, requestModel, reasoningEffort)
		if err == nil {
			return codexUpstreamCallResult{
				Content:    content,
				Metrics:    metrics,
				ActualMode: codexUpstreamModeBackend,
			}, nil
		}
		fallbackContent, fallbackMetrics, fallbackErr := h.runCodexCLI(ctx, path, body, requestModel, reasoningEffort)
		if fallbackErr == nil {
			return codexUpstreamCallResult{
				Content:           fallbackContent,
				Metrics:           fallbackMetrics,
				ActualMode:        codexUpstreamModeCLI,
				Fallback:          true,
				UpstreamErrorType: "codex_backend_upstream_failed",
			}, nil
		}
		return codexUpstreamCallResult{}, fmt.Errorf("codex backend upstream failed: %v; codex cli fallback failed: %w", err, fallbackErr)
	default:
		content, metrics, err := h.runCodexCLI(ctx, path, body, requestModel, reasoningEffort)
		if err != nil {
			return codexUpstreamCallResult{}, err
		}
		return codexUpstreamCallResult{
			Content:    content,
			Metrics:    metrics,
			ActualMode: codexUpstreamModeCLI,
		}, nil
	}
}

func (h *openAIGatewayHandler) runCodexCLI(ctx context.Context, path string, body []byte, requestModel string, reasoningEffort string) (string, openAIUsageMetrics, error) {
	prompt, err := codexCLIPromptFromGatewayRequest(path, body)
	if err != nil {
		return "", openAIUsageMetrics{}, err
	}
	defer prompt.close()

	if strings.TrimSpace(prompt.Text) == "" {
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
	if ctx == nil {
		ctx = context.Background()
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
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
		"--json",
		"-s", "read-only",
		"-m", model,
	}
	for _, file := range prompt.ImageFiles {
		args = append(args, "--image", file)
	}
	if reasoningEffort != "" {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", reasoningEffort))
	}
	args = append(args, "-")
	cmd := exec.CommandContext(cmdCtx, bin, args...)
	cmd.Stdin = strings.NewReader(prompt.Text)
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

	content, metrics, parsedJSON := parseCodexCLIJSONOutput(output)
	if !parsedJSON {
		content = extractCodexCLIAnswer(output)
	}
	if strings.TrimSpace(content) == "" {
		content = extractCodexCLIAnswer(output)
	}
	if strings.TrimSpace(content) == "" {
		return "", openAIUsageMetrics{}, fmt.Errorf("codex cli upstream returned empty answer")
	}
	if metrics.TotalTokens <= 0 {
		metrics = estimateCodexCLIUsage(model, prompt.Text, content)
	}
	if metrics.Model == "" {
		metrics.Model = model
	}
	return content, metrics, nil
}

func promptFromGatewayRequest(path string, body []byte) (string, error) {
	prompt, err := codexCLIPromptFromGatewayRequest(path, body)
	if err != nil {
		return "", err
	}
	defer prompt.close()
	return prompt.Text, nil
}

func codexCLIPromptFromGatewayRequest(path string, body []byte) (*codexCLIPrompt, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	var text string
	var images []gatewayImageSource
	if path == "/v1/responses" {
		text, images = promptFromResponsesPayload(payload)
	} else {
		text, images = promptFromChatCompletionsPayload(payload)
	}
	imageFiles, cleanup, err := materializeGatewayImages(images)
	if err != nil {
		return nil, err
	}
	return &codexCLIPrompt{
		Text:       text,
		ImageFiles: imageFiles,
		cleanup:    cleanup,
	}, nil
}

func (p *codexCLIPrompt) close() {
	if p != nil && p.cleanup != nil {
		p.cleanup()
	}
}

func promptFromChatCompletionsPayload(payload map[string]any) (string, []gatewayImageSource) {
	messages, _ := payload["messages"].([]any)
	parts := make([]gatewayMessageContent, 0, len(messages))
	for _, item := range messages {
		message, _ := item.(map[string]any)
		if message == nil {
			continue
		}
		role := strings.TrimSpace(stringValue(message["role"]))
		if role != "user" && role != "assistant" {
			continue
		}
		content := contentValue(message["content"])
		if strings.TrimSpace(content.Text) == "" && len(content.Images) == 0 {
			continue
		}
		content.Role = role
		parts = append(parts, content)
	}
	parts = lastGatewayMessageItems(parts, 8)
	return gatewayMessagePromptAndImages(parts)
}

func promptFromResponsesPayload(payload map[string]any) (string, []gatewayImageSource) {
	input := payload["input"]
	switch v := input.(type) {
	case string:
		return strings.TrimSpace(v), nil
	case []any:
		parts := make([]gatewayMessageContent, 0, len(v))
		for _, item := range v {
			message, _ := item.(map[string]any)
			if message == nil {
				continue
			}
			role := strings.TrimSpace(stringValue(message["role"]))
			if role != "user" && role != "assistant" {
				continue
			}
			content := contentValue(message["content"])
			if strings.TrimSpace(content.Text) == "" && len(content.Images) == 0 {
				continue
			}
			content.Role = role
			parts = append(parts, content)
		}
		parts = lastGatewayMessageItems(parts, 8)
		return gatewayMessagePromptAndImages(parts)
	default:
		content := contentValue(input)
		return strings.TrimSpace(content.Text), content.Images
	}
}

type gatewayMessageContent struct {
	Role   string
	Text   string
	Images []gatewayImageSource
}

func lastGatewayMessageItems(items []gatewayMessageContent, limit int) []gatewayMessageContent {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

func gatewayMessagePromptAndImages(items []gatewayMessageContent) (string, []gatewayImageSource) {
	textParts := make([]string, 0, len(items))
	images := make([]gatewayImageSource, 0)
	for _, item := range items {
		text := strings.TrimSpace(item.Text)
		if text == "" && len(item.Images) > 0 {
			text = "[image attached]"
		}
		if text != "" {
			textParts = append(textParts, item.Role+":\n"+text)
		}
		images = append(images, item.Images...)
	}
	return strings.TrimSpace(strings.Join(textParts, "\n\n")), images
}

func contentTextValue(value any) string {
	return contentValue(value).Text
}

func contentValue(value any) gatewayMessageContent {
	switch v := value.(type) {
	case string:
		return gatewayMessageContent{Text: v}
	case []any:
		parts := make([]string, 0, len(v))
		images := make([]gatewayImageSource, 0)
		for _, item := range v {
			switch typed := item.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					parts = append(parts, typed)
				}
			case map[string]any:
				if text := contentPartText(typed); strings.TrimSpace(text) != "" {
					parts = append(parts, text)
				}
				if source, ok := imageSourceFromContentPart(typed); ok {
					images = append(images, source)
				}
			}
		}
		return gatewayMessageContent{Text: strings.Join(parts, "\n"), Images: images}
	case map[string]any:
		content := gatewayMessageContent{Text: contentPartText(v)}
		if source, ok := imageSourceFromContentPart(v); ok {
			content.Images = append(content.Images, source)
		}
		return content
	default:
		return gatewayMessageContent{}
	}
}

func contentPartText(part map[string]any) string {
	if part == nil {
		return ""
	}
	if text := stringValue(part["text"]); text != "" {
		return text
	}
	return stringValue(part["input_text"])
}

func imageSourceFromContentPart(part map[string]any) (gatewayImageSource, bool) {
	if part == nil {
		return gatewayImageSource{}, false
	}
	typ := strings.TrimSpace(stringValue(part["type"]))
	switch typ {
	case "image_url":
		if raw := imageURLValue(part["image_url"]); raw != "" {
			return gatewayImageSource{Raw: raw}, true
		}
	case "input_image":
		if raw := imageURLValue(part["image_url"]); raw != "" {
			return gatewayImageSource{Raw: raw}, true
		}
		if raw := imageURLValue(part["image"]); raw != "" {
			return gatewayImageSource{Raw: raw, MediaType: stringValue(part["media_type"])}, true
		}
	case "image":
		if raw := imageURLValue(part["image"]); raw != "" {
			return gatewayImageSource{Raw: raw, MediaType: stringValue(part["media_type"])}, true
		}
	}
	if raw := imageURLValue(part["image_url"]); raw != "" {
		return gatewayImageSource{Raw: raw}, true
	}
	return gatewayImageSource{}, false
}

func imageURLValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		return stringValue(v["url"])
	default:
		return ""
	}
}

func materializeGatewayImages(sources []gatewayImageSource) ([]string, func(), error) {
	if len(sources) == 0 {
		return nil, nil, nil
	}
	if len(sources) > maxGatewayImages {
		return nil, nil, fmt.Errorf("too many image inputs: max %d", maxGatewayImages)
	}

	dir, err := os.MkdirTemp("", "oauth-api-codex-images-*")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}

	files := make([]string, 0, len(sources))
	for i, source := range sources {
		data, ext, err := decodeGatewayImageSource(source)
		if err != nil {
			cleanup()
			return nil, nil, err
		}
		if len(data) == 0 {
			cleanup()
			return nil, nil, fmt.Errorf("image input is empty")
		}
		if len(data) > maxGatewayImageBytes {
			cleanup()
			return nil, nil, fmt.Errorf("image input too large: max %d bytes", maxGatewayImageBytes)
		}
		if ext == "" {
			ext = ".png"
		}
		file := filepath.Join(dir, fmt.Sprintf("image-%d%s", i+1, ext))
		if err := os.WriteFile(file, data, 0o600); err != nil {
			cleanup()
			return nil, nil, err
		}
		files = append(files, file)
	}
	return files, cleanup, nil
}

func decodeGatewayImageSource(source gatewayImageSource) ([]byte, string, error) {
	raw := strings.TrimSpace(source.Raw)
	if raw == "" {
		return nil, "", fmt.Errorf("image input is empty")
	}
	if !strings.HasPrefix(raw, "data:") {
		return nil, "", fmt.Errorf("unsupported image input: only data URLs are supported by Codex CLI upstream")
	}
	header, payload, ok := strings.Cut(raw, ",")
	if !ok {
		return nil, "", fmt.Errorf("invalid image data URL")
	}
	mediaType := source.MediaType
	if strings.HasPrefix(header, "data:") {
		mediaType = strings.TrimPrefix(strings.Split(header, ";")[0], "data:")
	}
	if !strings.Contains(header, ";base64") {
		return nil, "", fmt.Errorf("invalid image data URL: base64 payload is required")
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", fmt.Errorf("invalid image data URL: %w", err)
	}
	return data, imageExtensionForMediaType(mediaType), nil
}

func imageExtensionForMediaType(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
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

func parseCodexCLIJSONOutput(output []byte) (string, openAIUsageMetrics, bool) {
	content := ""
	metrics := openAIUsageMetrics{}
	parsed := false

	for _, rawLine := range bytes.Split(output, []byte("\n")) {
		rawLine = bytes.TrimSpace(rawLine)
		if len(rawLine) == 0 || !json.Valid(rawLine) {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal(rawLine, &record); err != nil {
			continue
		}
		parsed = true

		switch record["type"] {
		case "turn.completed":
			if usageMetrics := codexUsageMetricsFromMap(mapValue(record["usage"])); usageMetrics.TotalTokens > 0 {
				metrics = usageMetrics
			}
			continue
		case "item.completed":
			item := mapValue(record["item"])
			if item == nil {
				continue
			}
			switch item["type"] {
			case "agent_message":
				if message := stringValue(item["text"]); message != "" {
					content = message
				}
			case "message":
				if item["role"] == "assistant" {
					if text := outputTextFromCodexContent(item["content"]); text != "" {
						content = text
					}
				}
			}
			continue
		}

		payload := mapValue(record["payload"])
		if payload == nil {
			continue
		}

		if record["type"] == "event_msg" {
			switch payload["type"] {
			case "token_count":
				info := mapValue(payload["info"])
				lastUsage := mapValue(info["last_token_usage"])
				if usageMetrics := codexUsageMetricsFromMap(lastUsage); usageMetrics.TotalTokens > 0 {
					metrics = usageMetrics
				}
			case "agent_message":
				if message := stringValue(payload["message"]); message != "" {
					if stringValue(payload["phase"]) == "final_answer" || content == "" {
						content = message
					}
				}
			}
			continue
		}

		if record["type"] == "response_item" &&
			payload["type"] == "message" &&
			payload["role"] == "assistant" {
			if text := outputTextFromCodexContent(payload["content"]); text != "" {
				if stringValue(payload["phase"]) == "final_answer" || content == "" {
					content = text
				}
			}
		}
	}

	return strings.TrimSpace(content), metrics, parsed
}

func outputTextFromCodexContent(value any) string {
	items, ok := value.([]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		m := mapValue(item)
		if m == nil || m["type"] != "output_text" {
			continue
		}
		if text := stringValue(m["text"]); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func codexUsageMetricsFromMap(usage map[string]any) openAIUsageMetrics {
	if usage == nil {
		return openAIUsageMetrics{}
	}
	metrics := openAIUsageMetrics{
		InputTokens:     int64Value(usage["input_tokens"]),
		OutputTokens:    int64Value(usage["output_tokens"]),
		TotalTokens:     int64Value(usage["total_tokens"]),
		CachedTokens:    int64Value(usage["cached_input_tokens"]),
		ReasoningTokens: int64Value(usage["reasoning_output_tokens"]),
	}
	if metrics.CachedTokens <= 0 {
		metrics.CachedTokens = int64Value(mapValue(usage["input_tokens_details"])["cached_tokens"])
	}
	if metrics.ReasoningTokens <= 0 {
		metrics.ReasoningTokens = int64Value(mapValue(usage["output_tokens_details"])["reasoning_tokens"])
	}
	if metrics.TotalTokens <= 0 && (metrics.InputTokens > 0 || metrics.OutputTokens > 0) {
		metrics.TotalTokens = metrics.InputTokens + metrics.OutputTokens
	}
	return metrics
}

func mapValue(value any) map[string]any {
	item, _ := value.(map[string]any)
	return item
}

func int64Value(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	default:
		return 0
	}
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
		"prompt_tokens_details": map[string]any{
			"cached_tokens": metrics.CachedTokens,
		},
		"completion_tokens_details": map[string]any{
			"reasoning_tokens": metrics.ReasoningTokens,
		},
	}
}

func responsesUsagePayload(metrics openAIUsageMetrics) map[string]any {
	return map[string]any{
		"input_tokens":  metrics.InputTokens,
		"output_tokens": metrics.OutputTokens,
		"total_tokens":  metrics.TotalTokens,
		"input_tokens_details": map[string]any{
			"cached_tokens": metrics.CachedTokens,
		},
		"output_tokens_details": map[string]any{
			"reasoning_tokens": metrics.ReasoningTokens,
		},
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

func parseRequestModelStreamAndReasoningEffort(body []byte) (string, bool, string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false, "", nil
	}
	model, _ := payload["model"].(string)
	stream, _ := payload["stream"].(bool)
	reasoningEffort, err := reasoningEffortFromPayload(payload)
	return strings.TrimSpace(model), stream, reasoningEffort, err
}

func reasoningEffortFromPayload(payload map[string]any) (string, error) {
	raw := stringValue(payload["reasoning_effort"])
	if raw == "" {
		raw = stringValue(payload["reasoningEffort"])
	}
	if raw == "" {
		raw = stringValue(mapValue(payload["reasoning"])["effort"])
	}
	if raw == "" {
		return "", nil
	}

	effort := strings.ToLower(strings.TrimSpace(raw))
	switch effort {
	case "low", "medium", "high", "xhigh":
		return effort, nil
	default:
		return "", fmt.Errorf("unsupported reasoning_effort: %s", raw)
	}
}

func sessionIDFromGatewayRequest(r *stdhttp.Request, body []byte) string {
	if sessionID := sessionIDFromHeaders(r); sessionID != "" {
		return sessionID
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	for _, key := range []string{"session_id", "conversation_id", "thread_id"} {
		if sessionID := normalizeGatewaySessionID(stringValue(payload[key])); sessionID != "" {
			return sessionID
		}
	}
	metadata := mapValue(payload["metadata"])
	for _, key := range []string{"session_id", "conversation_id", "thread_id"} {
		if sessionID := normalizeGatewaySessionID(stringValue(metadata[key])); sessionID != "" {
			return sessionID
		}
	}
	return ""
}

func sessionIDFromHeaders(r *stdhttp.Request) string {
	if r == nil {
		return ""
	}
	for _, key := range []string{"X-Session-ID", "X-Conversation-ID", "X-Thread-ID"} {
		if sessionID := normalizeGatewaySessionID(r.Header.Get(key)); sessionID != "" {
			return sessionID
		}
	}
	return ""
}

func normalizeGatewaySessionID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) > 128 {
		return value[:128]
	}
	return value
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
