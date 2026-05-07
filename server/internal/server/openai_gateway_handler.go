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
	"strings"
	"time"

	"server/internal/biz"
	"server/internal/conf"
	"server/internal/errcode"

	"github.com/go-kratos/kratos/v2/log"
	httpx "github.com/go-kratos/kratos/v2/transport/http"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	maxGatewayRequestBytes = 32 << 20
	maxCapturedStreamBytes = 32 << 20
)

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
	if err := h.gatewayUC.CreateUsageLog(ctx, item); err != nil {
		h.log.WithContext(ctx).Errorf("record gateway usage failed: %v", err)
	}
	if key != nil && key.ID > 0 {
		if err := h.gatewayUC.TouchAPIKeyUsed(ctx, key.ID, time.Now()); err != nil {
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
