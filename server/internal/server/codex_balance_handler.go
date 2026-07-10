package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	httpx "github.com/go-kratos/kratos/v2/transport/http"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const defaultCodexBalanceTimeout = 15 * time.Second
const defaultCodexBalanceCacheTTL = 30 * time.Second
const defaultCodexRateLimitResetCreditsURL = "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits"
const codexRateLimitsRetryDelay = 250 * time.Millisecond

type codexBalanceHTTPHandler struct {
	log *log.Helper

	mu             sync.Mutex
	cache          map[string]any
	cacheExpiresAt time.Time
}

type codexAppServerResponse struct {
	ID     *int            `json:"id,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type codexRateLimitsReadResponse struct {
	RateLimits          codexRateLimitSnapshot            `json:"rateLimits"`
	RateLimitsByLimitID map[string]codexRateLimitSnapshot `json:"rateLimitsByLimitId"`
}

type codexRateLimitSnapshot struct {
	LimitID              *string               `json:"limitId"`
	LimitName            *string               `json:"limitName"`
	Primary              *codexRateLimitWindow `json:"primary"`
	Secondary            *codexRateLimitWindow `json:"secondary"`
	Credits              *codexCreditsSnapshot `json:"credits"`
	PlanType             *string               `json:"planType"`
	RateLimitReachedType *string               `json:"rateLimitReachedType"`
}

type codexRateLimitWindow struct {
	UsedPercent        float64 `json:"usedPercent"`
	WindowDurationMins *int    `json:"windowDurationMins"`
	ResetsAt           *int64  `json:"resetsAt"`
}

type codexCreditsSnapshot struct {
	HasCredits bool    `json:"hasCredits"`
	Unlimited  bool    `json:"unlimited"`
	Balance    *string `json:"balance"`
}

type publicCodexBalanceResponse struct {
	Status                string                                  `json:"status"`
	FetchedAt             string                                  `json:"fetched_at"`
	Stale                 bool                                    `json:"stale,omitempty"`
	Credits               *codexCreditsSnapshot                   `json:"credits"`
	RateLimits            publicCodexRateLimitSnapshot            `json:"rate_limits"`
	RateLimitsByLimitID   map[string]publicCodexRateLimitSnapshot `json:"rate_limits_by_limit_id,omitempty"`
	RateLimitResetCredits publicCodexRateLimitResetCredits        `json:"rate_limit_reset_credits"`
}

type publicCodexRateLimitSnapshot struct {
	LimitID              *string                     `json:"limit_id"`
	LimitName            *string                     `json:"limit_name"`
	Primary              *publicCodexRateLimitWindow `json:"primary"`
	Secondary            *publicCodexRateLimitWindow `json:"secondary"`
	Credits              *codexCreditsSnapshot       `json:"credits"`
	PlanType             *string                     `json:"plan_type"`
	RateLimitReachedType *string                     `json:"rate_limit_reached_type"`
}

type publicCodexRateLimitWindow struct {
	UsedPercent        float64 `json:"used_percent"`
	RemainingPercent   float64 `json:"remaining_percent"`
	WindowDurationMins *int    `json:"window_duration_mins"`
	ResetsAt           *int64  `json:"resets_at"`
	ResetsAtTime       *string `json:"resets_at_time,omitempty"`
}

type codexRateLimitResetCreditsResponse struct {
	Credits          []codexRateLimitResetCredit `json:"credits"`
	AvailableCount   int                         `json:"available_count"`
	TotalEarnedCount int                         `json:"total_earned_count"`
}

type codexRateLimitResetCredit struct {
	ResetType *string `json:"reset_type"`
	Status    *string `json:"status"`
	GrantedAt *string `json:"granted_at"`
	ExpiresAt *string `json:"expires_at"`
	Title     *string `json:"title"`
}

type publicCodexRateLimitResetCredits struct {
	Status           string                            `json:"status"`
	AvailableCount   int                               `json:"available_count"`
	TotalEarnedCount int                               `json:"total_earned_count"`
	Credits          []publicCodexRateLimitResetCredit `json:"credits"`
}

type publicCodexRateLimitResetCredit struct {
	ResetType *string `json:"reset_type"`
	Status    *string `json:"status"`
	GrantedAt *string `json:"granted_at"`
	ExpiresAt *string `json:"expires_at"`
	Title     *string `json:"title"`
}

var codexAppServerCommandContext = exec.CommandContext

func registerCodexBalanceRoutes(srv *httpx.Server, logger log.Logger, tp *sdktrace.TracerProvider) {
	handler := &codexBalanceHTTPHandler{
		log: log.NewHelper(log.With(logger, "logger.name", "server.http.codex_balance")),
	}

	srv.Handle("/public/codex/balance", newObservedHTTPHandler(logger, tp, "server.http.codex_balance", func(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
		handler.ServeHTTP(ctx, w, r)
	}))
}

func (h *codexBalanceHTTPHandler) ServeHTTP(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodGet {
		w.Header().Set("Allow", stdhttp.MethodGet)
		writeJSON(w, stdhttp.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}

	if cached := h.cachedBalance(time.Now()); cached != nil {
		writeJSON(w, stdhttp.StatusOK, cached)
		return
	}

	now := time.Now()
	rateLimits, attempts, err := readCodexRateLimits(ctx)
	if err != nil {
		h.log.WithContext(ctx).Warnw(
			"msg", "codex balance query failed",
			"request_id", requestIDFromRequest(r),
			"trace_id", traceIDFromContext(ctx),
			"attempts", attempts,
			"error", err.Error(),
		)
		if stale := h.staleBalance(now); stale != nil {
			writeJSON(w, stdhttp.StatusOK, stale)
			return
		}
		writeJSON(w, stdhttp.StatusBadGateway, map[string]any{
			"error":   "codex_balance_query_failed",
			"message": "查询 Codex 余额失败，请检查服务器 Codex 登录态或 Codex app-server 是否可用",
		})
		return
	}

	resetCredits, resetCreditsErr := readCodexRateLimitResetCredits(ctx)
	if resetCreditsErr != nil {
		h.log.WithContext(ctx).Warnw(
			"msg", "codex rate limit reset credits query failed",
			"request_id", requestIDFromRequest(r),
			"trace_id", traceIDFromContext(ctx),
			"error", resetCreditsErr.Error(),
		)
	}

	payload := mapPublicCodexBalance(rateLimits, resetCredits, resetCreditsErr == nil, now)
	h.storeBalanceCache(payload, now)
	writeJSON(w, stdhttp.StatusOK, payload)
}

func readCodexRateLimits(ctx context.Context) (*codexRateLimitsReadResponse, int, error) {
	timeout := codexBalanceTimeout()
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := readCodexRateLimitsOnce(reqCtx)
	if err == nil {
		return result, 1, nil
	}
	if !shouldRetryCodexRateLimits(err) || reqCtx.Err() != nil {
		return nil, 1, err
	}

	timer := time.NewTimer(codexRateLimitsRetryDelay)
	defer timer.Stop()
	select {
	case <-reqCtx.Done():
		return nil, 1, reqCtx.Err()
	case <-timer.C:
	}

	result, err = readCodexRateLimitsOnce(reqCtx)
	if err != nil {
		return nil, 2, err
	}
	return result, 2, nil
}

func shouldRetryCodexRateLimits(err error) bool {
	message := strings.ToLower(err.Error())
	for _, marker := range []string{
		"error sending request",
		"connection reset",
		"connection refused",
		"unexpected eof",
		"tls handshake timeout",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func readCodexRateLimitsOnce(ctx context.Context) (*codexRateLimitsReadResponse, error) {
	bin := codexAppServerBin()
	cmd := codexAppServerCommandContext(ctx, bin, "app-server", "--listen", "stdio://")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	go func() {
		_, _ = io.Copy(io.Discard, stderr)
	}()

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	if err := writeCodexAppServerRequest(stdin, 1, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    "openai-oauth-api-service",
			"title":   "OpenAI OAuth API Service",
			"version": "1.0.0",
		},
		"capabilities": nil,
	}); err != nil {
		return nil, err
	}

	initialized := false
	requestedRateLimits := false
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg codexAppServerResponse
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.ID == nil {
			continue
		}
		switch *msg.ID {
		case 1:
			if msg.Error != nil {
				return nil, fmt.Errorf("codex app-server initialize failed: %s", msg.Error.Message)
			}
			if !initialized {
				initialized = true
				if _, err := io.WriteString(stdin, `{"jsonrpc":"2.0","method":"initialized"}`+"\n"); err != nil {
					return nil, err
				}
				if err := writeCodexAppServerRequest(stdin, 2, "account/rateLimits/read", nil); err != nil {
					return nil, err
				}
				requestedRateLimits = true
			}
		case 2:
			if msg.Error != nil {
				return nil, fmt.Errorf("codex app-server rate limits failed: %s", msg.Error.Message)
			}
			var result codexRateLimitsReadResponse
			if err := json.Unmarshal(msg.Result, &result); err != nil {
				return nil, err
			}
			return &result, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if !requestedRateLimits {
		return nil, errors.New("codex app-server exited before rate limit request")
	}
	return nil, errors.New("codex app-server exited without rate limit response")
}

func readCodexRateLimitResetCredits(ctx context.Context) (*codexRateLimitResetCreditsResponse, error) {
	timeout := codexBalanceTimeout()
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return defaultCodexBackendClient.readRateLimitResetCredits(reqCtx, false)
}

func (c *codexBackendClient) readRateLimitResetCredits(ctx context.Context, forceRefresh bool) (*codexRateLimitResetCreditsResponse, error) {
	accessToken, accountID, err := c.codexAccessToken(ctx, forceRefresh)
	if err != nil {
		return nil, err
	}
	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodGet, codexRateLimitResetCreditsURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", codexBackendUserAgent())
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode == stdhttp.StatusUnauthorized && !forceRefresh {
		return c.readRateLimitResetCredits(ctx, true)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("codex reset credits query failed: status=%d body=%s", resp.StatusCode, summarizeCodexBackendBody(body))
	}

	var result codexRateLimitResetCreditsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func writeCodexAppServerRequest(w io.Writer, id int, method string, params any) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body = append(body, '\n')
	_, err = w.Write(body)
	return err
}

func mapPublicCodexBalance(rateLimits *codexRateLimitsReadResponse, resetCredits *codexRateLimitResetCreditsResponse, resetCreditsOK bool, now time.Time) map[string]any {
	payload := publicCodexBalanceResponse{
		Status:                "ok",
		FetchedAt:             now.UTC().Format(time.RFC3339),
		Credits:               rateLimits.RateLimits.Credits,
		RateLimits:            mapPublicCodexRateLimit(rateLimits.RateLimits),
		RateLimitResetCredits: mapPublicCodexRateLimitResetCredits(resetCredits, resetCreditsOK),
	}
	if len(rateLimits.RateLimitsByLimitID) > 0 {
		payload.RateLimitsByLimitID = make(map[string]publicCodexRateLimitSnapshot, len(rateLimits.RateLimitsByLimitID))
		for key, value := range rateLimits.RateLimitsByLimitID {
			payload.RateLimitsByLimitID[key] = mapPublicCodexRateLimit(value)
		}
	}
	body, _ := json.Marshal(payload)
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	return out
}

func mapPublicCodexRateLimitResetCredits(resetCredits *codexRateLimitResetCreditsResponse, ok bool) publicCodexRateLimitResetCredits {
	if !ok || resetCredits == nil {
		return publicCodexRateLimitResetCredits{
			Status:  "unavailable",
			Credits: []publicCodexRateLimitResetCredit{},
		}
	}
	credits := make([]publicCodexRateLimitResetCredit, 0, len(resetCredits.Credits))
	for _, credit := range resetCredits.Credits {
		credits = append(credits, publicCodexRateLimitResetCredit(credit))
	}
	return publicCodexRateLimitResetCredits{
		Status:           "ok",
		AvailableCount:   resetCredits.AvailableCount,
		TotalEarnedCount: resetCredits.TotalEarnedCount,
		Credits:          credits,
	}
}

func mapPublicCodexRateLimit(snapshot codexRateLimitSnapshot) publicCodexRateLimitSnapshot {
	return publicCodexRateLimitSnapshot{
		LimitID:              snapshot.LimitID,
		LimitName:            snapshot.LimitName,
		Primary:              mapPublicCodexRateLimitWindow(snapshot.Primary),
		Secondary:            mapPublicCodexRateLimitWindow(snapshot.Secondary),
		Credits:              snapshot.Credits,
		PlanType:             snapshot.PlanType,
		RateLimitReachedType: snapshot.RateLimitReachedType,
	}
}

func mapPublicCodexRateLimitWindow(window *codexRateLimitWindow) *publicCodexRateLimitWindow {
	if window == nil {
		return nil
	}
	remaining := 100 - window.UsedPercent
	if remaining < 0 {
		remaining = 0
	}
	if remaining > 100 {
		remaining = 100
	}
	var resetsAtTime *string
	if window.ResetsAt != nil && *window.ResetsAt > 0 {
		value := time.Unix(*window.ResetsAt, 0).UTC().Format(time.RFC3339)
		resetsAtTime = &value
	}
	return &publicCodexRateLimitWindow{
		UsedPercent:        window.UsedPercent,
		RemainingPercent:   remaining,
		WindowDurationMins: window.WindowDurationMins,
		ResetsAt:           window.ResetsAt,
		ResetsAtTime:       resetsAtTime,
	}
}

func codexAppServerBin() string {
	if raw := strings.TrimSpace(os.Getenv("CODEX_APP_SERVER_BIN")); raw != "" {
		return raw
	}
	if raw := strings.TrimSpace(os.Getenv("CODEX_CLI_BIN")); raw != "" {
		return raw
	}
	return "codex"
}

func codexBalanceTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("CODEX_BALANCE_TIMEOUT_SECONDS")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultCodexBalanceTimeout
}

func (h *codexBalanceHTTPHandler) cachedBalance(now time.Time) map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cache == nil || !now.Before(h.cacheExpiresAt) {
		return nil
	}
	return cloneBalancePayload(h.cache)
}

func (h *codexBalanceHTTPHandler) staleBalance(now time.Time) map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cache == nil {
		return nil
	}
	payload := cloneBalancePayload(h.cache)
	payload["stale"] = true
	payload["stale_reason"] = "codex_balance_query_failed"
	payload["last_error_at"] = now.UTC().Format(time.RFC3339)
	return payload
}

func (h *codexBalanceHTTPHandler) storeBalanceCache(payload map[string]any, now time.Time) {
	ttl := codexBalanceCacheTTL()
	if ttl <= 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cache = payload
	h.cacheExpiresAt = now.Add(ttl)
}

func cloneBalancePayload(payload map[string]any) map[string]any {
	out := make(map[string]any, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	return out
}

func codexBalanceCacheTTL() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("CODEX_BALANCE_CACHE_SECONDS")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err == nil && seconds >= 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultCodexBalanceCacheTTL
}

func codexRateLimitResetCreditsURL() string {
	if raw := strings.TrimSpace(os.Getenv("CODEX_RATE_LIMIT_RESET_CREDITS_URL")); raw != "" {
		return raw
	}
	return defaultCodexRateLimitResetCreditsURL
}
