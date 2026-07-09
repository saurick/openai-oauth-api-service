package server

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"server/internal/biz"
)

func TestStatusCapturingResponseWriterPreservesFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := &statusCapturingResponseWriter{ResponseWriter: rec}
	flusher, ok := any(writer).(http.Flusher)
	if !ok {
		t.Fatal("statusCapturingResponseWriter must expose http.Flusher for SSE handlers")
	}
	_, _ = writer.Write([]byte("data: hello\n\n"))
	flusher.Flush()
	if !rec.Flushed {
		t.Fatal("underlying recorder was not flushed")
	}
	if writer.StatusCode() != http.StatusOK {
		t.Fatalf("status = %d, want 200", writer.StatusCode())
	}
}

func TestGatewayClientIPFromTrustedProxyHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	req.RemoteAddr = "10.0.0.8:43210"
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.8")

	if got := gatewayClientIPFromRequest(req); got != "203.0.113.9" {
		t.Fatalf("client ip = %q, want forwarded client ip", got)
	}
}

func TestGatewayClientIPIgnoresForwardedHeaderFromUntrustedRemote(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	req.RemoteAddr = "198.51.100.20:43210"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")

	if got := gatewayClientIPFromRequest(req); got != "198.51.100.20" {
		t.Fatalf("client ip = %q, want direct remote ip", got)
	}
}

func TestGatewayClientIPTrustedProxyCIDREnvRestrictsDefaults(t *testing.T) {
	t.Setenv("GATEWAY_TRUSTED_PROXY_CIDRS", "127.0.0.0/8")
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	req.RemoteAddr = "10.0.0.8:43210"
	req.Header.Set("X-Real-IP", "203.0.113.9")

	if got := gatewayClientIPFromRequest(req); got != "10.0.0.8" {
		t.Fatalf("client ip = %q, want direct remote ip when cidr override does not match", got)
	}
}

func TestUpstreamErrorTypeClassifiesBackendErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "auth", err: codexBackendHTTPError{status: http.StatusUnauthorized}, want: "codex_backend_auth_failed"},
		{name: "rate_limit", err: codexBackendHTTPError{status: http.StatusTooManyRequests}, want: "codex_backend_rate_limited"},
		{name: "http_5xx", err: codexBackendHTTPError{status: http.StatusBadGateway}, want: "codex_backend_http_5xx"},
		{name: "timeout", err: errors.New("codex backend upstream timed out after 10s"), want: "codex_backend_timeout"},
		{name: "context_length", err: errors.New(`codex backend response failed: {"error":{"code":"context_length_exceeded"}}`), want: "context_length_exceeded"},
		{name: "overloaded", err: errors.New(`codex backend response failed: {"type":"response.failed","response":{"status":"failed","error":{"code":"server_is_overloaded","message":"Our servers are currently overloaded. Please try again later."}}}`), want: "codex_backend_overloaded"},
		{name: "model_capacity", err: errors.New("Selected model is at capacity. Please try a different model."), want: "codex_backend_overloaded"},
		{name: "incomplete", err: errors.New("codex backend response incomplete: stopped"), want: "codex_backend_response_incomplete"},
		{name: "stream", err: errors.New("unexpected EOF while reading stream"), want: "codex_backend_stream_error"},
		{name: "client_canceled", err: context.Canceled, want: "client_canceled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := upstreamErrorType(tt.err, codexUpstreamModeBackend); got != tt.want {
				t.Fatalf("upstreamErrorType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCodexBackendContextLengthIsNotRetriable(t *testing.T) {
	err := errors.New(`codex backend response failed: {"error":{"code":"context_length_exceeded"}}`)
	if isRetriableCodexBackendError(err) {
		t.Fatal("context length errors must not retry against the same oversized request")
	}
}

func TestCodexBackendPlainEOFIsRetriable(t *testing.T) {
	err := errors.New(`Post "https://chatgpt.com/backend-api/codex/responses": EOF`)
	if !isRetriableCodexBackendError(err) {
		t.Fatal("plain upstream EOF before a terminal response must be retriable")
	}
}

func TestCodexBackendTerminalResponseEventsAreNotRetriable(t *testing.T) {
	for _, err := range []error{
		errors.New(`codex backend response failed: {"response":{"status":"failed"}}`),
		errors.New(`codex backend response incomplete: {"response":{"incomplete_details":{"reason":"max_output_tokens"}}}`),
	} {
		if isRetriableCodexBackendError(err) {
			t.Fatalf("terminal response event must not retry: %v", err)
		}
	}
}

func TestUpstreamErrorTypeClassifiesCLIErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "timeout", err: errors.New("codex cli upstream timed out after 10s"), want: "codex_cli_timeout"},
		{name: "empty_answer", err: errors.New("codex cli upstream returned empty answer"), want: "codex_cli_empty_answer"},
		{name: "not_found", err: errors.New("exec: \"codex\": executable file not found in $PATH"), want: "codex_cli_not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := upstreamErrorType(tt.err, codexUpstreamModeCLI); got != tt.want {
				t.Fatalf("upstreamErrorType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStreamResponsesWritesCreatedBeforeSlowUpstream(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-codex")
	script := `#!/bin/sh
cat >/dev/null
sleep 2
echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"OK"}}'
echo '{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_UPSTREAM_MODE", "codex_cli")
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("CODEX_CLI_TIMEOUT_SECONDS", "30")

	handler := &openAIGatewayHandler{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		model, stream, effort, err := parseRequestModelStreamAndReasoningEffort(body)
		if err != nil {
			t.Fatal(err)
		}
		handler.handleCodexCLIProxy(w, r, &biz.GatewayAPIKey{ID: 1, KeyPrefix: "ogw_test"}, "req_test", "", "responses", model, effort, stream, body, time.Now(), biz.GatewayUsageDiagnostic{}, gatewayRequestOptions{})
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/responses", strings.NewReader(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer ogw_test")
	client := &http.Client{Timeout: 5 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("first stream line took %s, want under 1s", elapsed)
	}
	if !strings.Contains(line, `"type":"response.created"`) {
		t.Fatalf("first line = %q, want response.created", line)
	}
}

func TestStreamResponsesKeepsConnectionAliveWithInProgressEvent(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-codex")
	script := `#!/bin/sh
cat >/dev/null
sleep 3
echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"OK"}}'
echo '{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_UPSTREAM_MODE", "codex_cli")
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("CODEX_CLI_TIMEOUT_SECONDS", "30")
	t.Setenv("GATEWAY_STREAM_HEARTBEAT_SECONDS", "1")

	handler := &openAIGatewayHandler{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		model, stream, effort, err := parseRequestModelStreamAndReasoningEffort(body)
		if err != nil {
			t.Fatal(err)
		}
		handler.handleCodexCLIProxy(w, r, &biz.GatewayAPIKey{ID: 1, KeyPrefix: "ogw_test"}, "req_test", "", "responses", model, effort, stream, body, time.Now(), biz.GatewayUsageDiagnostic{}, gatewayRequestOptions{})
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/responses", strings.NewReader(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer ogw_test")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	first := readNonEmptyStreamLine(t, reader)
	if !strings.Contains(first, `"type":"response.created"`) {
		t.Fatalf("first line = %q, want response.created", first)
	}
	start := time.Now()
	second := readNonEmptyStreamLine(t, reader)
	if !strings.Contains(second, `"type":"response.in_progress"`) {
		t.Fatalf("second line = %q, want response.in_progress", second)
	}
	if elapsed := time.Since(start); elapsed > 2500*time.Millisecond {
		t.Fatalf("heartbeat took %s, want under 2.5s", elapsed)
	}
}

func TestStreamChatCompletionsKeepsConnectionAliveBeforeSlowUpstream(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-codex")
	script := `#!/bin/sh
cat >/dev/null
sleep 3
echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"OK"}}'
echo '{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_UPSTREAM_MODE", "codex_cli")
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("CODEX_CLI_TIMEOUT_SECONDS", "30")
	t.Setenv("GATEWAY_STREAM_HEARTBEAT_SECONDS", "1")

	handler := &openAIGatewayHandler{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		model, stream, effort, err := parseRequestModelStreamAndReasoningEffort(body)
		if err != nil {
			t.Fatal(err)
		}
		handler.handleCodexCLIProxy(w, r, &biz.GatewayAPIKey{ID: 1, KeyPrefix: "ogw_test"}, "req_test", "", "chat.completions", model, effort, stream, body, time.Now(), biz.GatewayUsageDiagnostic{}, gatewayRequestOptions{})
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt-5.5","stream":true,"messages":[{"role":"user","content":"Reply OK"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer ogw_test")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	first := readNonEmptyStreamLine(t, reader)
	if first != ": keepalive\n" {
		t.Fatalf("first line = %q, want initial keepalive", first)
	}
	start := time.Now()
	second := readNonEmptyStreamLine(t, reader)
	if second != ": keepalive\n" {
		t.Fatalf("second line = %q, want heartbeat keepalive", second)
	}
	if elapsed := time.Since(start); elapsed > 2500*time.Millisecond {
		t.Fatalf("heartbeat took %s, want under 2.5s", elapsed)
	}
}

func readNonEmptyStreamLine(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(line) != "" {
			return line
		}
	}
}

func TestStreamResponsesReturnsSSEErrorAfterHeaders(t *testing.T) {
	t.Setenv("CODEX_UPSTREAM_MODE", "codex_backend")
	t.Setenv("CODEX_AUTH_FILE", filepath.Join(t.TempDir(), "missing-auth.json"))

	handler := &openAIGatewayHandler{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		model, stream, effort, err := parseRequestModelStreamAndReasoningEffort(body)
		if err != nil {
			t.Fatal(err)
		}
		handler.handleCodexCLIProxy(w, r, &biz.GatewayAPIKey{ID: 1, KeyPrefix: "ogw_test"}, "req_test", "", "responses", model, effort, stream, body, time.Now(), biz.GatewayUsageDiagnostic{}, gatewayRequestOptions{})
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/responses", strings.NewReader(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer ogw_test")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 because stream headers were already sent", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `"type":"response.created"`) || !strings.Contains(text, `"type":"response.failed"`) || !strings.Contains(text, "data: [DONE]") {
		t.Fatalf("unexpected stream body:\n%s", text)
	}
}

func TestPromptFromChatCompletionsPayload(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role": "system", "content": "Be concise."},
			{"role": "developer", "content": "Ignore this for Codex CLI upstream."},
			{"role": "user", "content": [{"type": "text", "text": "Reply with OK only."}]}
		]
	}`)

	got, err := promptFromGatewayRequest("/v1/chat/completions", body)
	if err != nil {
		t.Fatal(err)
	}
	want := "user:\nReply with OK only."
	if got != want {
		t.Fatalf("unexpected prompt:\n%s", got)
	}
}

func TestCodexCLIPromptFromChatCompletionsPayloadMaterializesImages(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role": "user", "content": [
				{"type": "text", "text": "Describe this image."},
				{"type": "image_url", "image_url": {"url": "data:image/png;base64,iVBORw0KGgo="}}
			]}
		]
	}`)

	prompt, err := codexCLIPromptFromGatewayRequest("/v1/chat/completions", body)
	if err != nil {
		t.Fatal(err)
	}
	defer prompt.close()

	if prompt.Text != "user:\nDescribe this image." {
		t.Fatalf("unexpected prompt text: %q", prompt.Text)
	}
	if len(prompt.ImageFiles) != 1 {
		t.Fatalf("image files = %d, want 1", len(prompt.ImageFiles))
	}
	if filepath.Ext(prompt.ImageFiles[0]) != ".png" {
		t.Fatalf("image extension = %q, want .png", filepath.Ext(prompt.ImageFiles[0]))
	}
	if info, err := os.Stat(prompt.ImageFiles[0]); err != nil || info.Size() == 0 {
		t.Fatalf("image file not materialized: info=%v err=%v", info, err)
	}
}

func TestGatewayRequestLimitCoversMaxBase64Attachments(t *testing.T) {
	encodedImageBytes := ((maxGatewayImageBytes + 2) / 3) * 4
	encodedFileBytes := ((maxGatewayFileBytes + 2) / 3) * 4
	imageURLOverhead := len("data:image/png;base64,")
	fileURLOverhead := len("data:application/pdf;base64,")
	jsonOverhead := 8 << 10

	maxImagePayload := maxGatewayImages*(encodedImageBytes+imageURLOverhead) + jsonOverhead
	if maxImagePayload > maxGatewayRequestBytes {
		t.Fatalf("request limit %d must cover %d bytes for max image attachments", maxGatewayRequestBytes, maxImagePayload)
	}

	maxFilePayload := maxGatewayFiles*(encodedFileBytes+fileURLOverhead) + jsonOverhead
	if maxFilePayload > maxGatewayRequestBytes {
		t.Fatalf("request limit %d must cover %d bytes for max file attachments", maxGatewayRequestBytes, maxFilePayload)
	}
}

func TestCodexCLIPromptRejectsPDFInput(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role": "user", "content": [
				{"type": "text", "text": "Summarize this PDF."},
				{"type": "input_file", "filename": "sample.pdf", "file_data": "data:application/pdf;base64,JVBERi0xLjQKJSVFT0Y="}
			]}
		]
	}`)

	_, err := codexCLIPromptFromGatewayRequest("/v1/chat/completions", body)
	if err == nil {
		t.Fatal("expected PDF input to be rejected by Codex CLI prompt builder")
	}
	if !strings.Contains(err.Error(), "codex_backend") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractCodexCLIAnswer(t *testing.T) {
	output := []byte("OpenAI Codex\nuser\nReply\ncodex\nOK\n tokens ignored\n\ntokens used\n1,605\nOK\n")

	got := extractCodexCLIAnswer(output)
	if got != "OK" {
		t.Fatalf("unexpected answer: %q", got)
	}
}

func TestParseCodexCLIJSONOutputUsesExactTokenCount(t *testing.T) {
	output := []byte(`
{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":101542,"cached_input_tokens":19840,"output_tokens":672,"reasoning_output_tokens":527,"total_tokens":102214}}}}
{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"OK"}],"phase":"final_answer"}}
`)

	content, metrics, parsed := parseCodexCLIJSONOutput(output)
	if !parsed {
		t.Fatal("expected json output to be parsed")
	}
	if content != "OK" {
		t.Fatalf("content = %q, want OK", content)
	}
	if metrics.InputTokens != 101542 ||
		metrics.CachedTokens != 19840 ||
		metrics.OutputTokens != 672 ||
		metrics.ReasoningTokens != 527 ||
		metrics.TotalTokens != 102214 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
}

func TestParseCodexCLIJSONOutputUsesCurrentItemCompletedFormat(t *testing.T) {
	output := []byte(`
{"type":"thread.started","thread_id":"019e1053-032f-73a1-be58-58bde4f129a6"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"OK"}}
{"type":"turn.completed","usage":{"input_tokens":23590,"cached_input_tokens":7552,"output_tokens":5,"reasoning_output_tokens":0}}
`)

	content, metrics, parsed := parseCodexCLIJSONOutput(output)
	if !parsed {
		t.Fatal("expected json output to be parsed")
	}
	if content != "OK" {
		t.Fatalf("content = %q, want OK", content)
	}
	if metrics.InputTokens != 23590 ||
		metrics.CachedTokens != 7552 ||
		metrics.OutputTokens != 5 ||
		metrics.ReasoningTokens != 0 ||
		metrics.TotalTokens != 23595 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
}

func TestChatUsagePayloadIncludesCachedAndReasoningTokens(t *testing.T) {
	payload := chatUsagePayload(openAIUsageMetrics{
		InputTokens:     100,
		CachedTokens:    40,
		OutputTokens:    20,
		ReasoningTokens: 7,
		TotalTokens:     120,
	})

	promptDetails, ok := payload["prompt_tokens_details"].(map[string]any)
	if !ok {
		t.Fatal("missing prompt_tokens_details")
	}
	if got := promptDetails["cached_tokens"]; got != int64(40) {
		t.Fatalf("cached_tokens = %v, want 40", got)
	}
	completionDetails, ok := payload["completion_tokens_details"].(map[string]any)
	if !ok {
		t.Fatal("missing completion_tokens_details")
	}
	if got := completionDetails["reasoning_tokens"]; got != int64(7) {
		t.Fatalf("reasoning_tokens = %v, want 7", got)
	}
}

func TestParseRequestReasoningEffort(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","stream":true,"reasoning_effort":"xhigh"}`)

	model, stream, effort, err := parseRequestModelStreamAndReasoningEffort(body)
	if err != nil {
		t.Fatal(err)
	}
	if model != "gpt-5.5" || !stream || effort != "xhigh" {
		t.Fatalf("model=%q stream=%v effort=%q", model, stream, effort)
	}
}

func TestParseRequestReasoningEffortFromNestedPayload(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","reasoning":{"effort":"medium"}}`)

	_, _, effort, err := parseRequestModelStreamAndReasoningEffort(body)
	if err != nil {
		t.Fatal(err)
	}
	if effort != "medium" {
		t.Fatalf("effort = %q, want medium", effort)
	}
}

func TestParseRequestReasoningEffortRejectsUnknownValue(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","reasoning_effort":"extreme"}`)

	_, _, _, err := parseRequestModelStreamAndReasoningEffort(body)
	if err == nil {
		t.Fatal("expected invalid reasoning effort error")
	}
}

func TestCodexBackendRequestPassesAllReasoningEfforts(t *testing.T) {
	for _, effort := range []string{"low", "medium", "high", "xhigh"} {
		t.Run(effort, func(t *testing.T) {
			req, _, err := codexBackendRequestFromGateway(
				"/v1/chat/completions",
				[]byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":"Reply OK"}]}`),
				"",
				effort,
			)
			if err != nil {
				t.Fatal(err)
			}
			reasoning, ok := req["reasoning"].(map[string]any)
			if !ok {
				t.Fatalf("missing reasoning payload: %#v", req)
			}
			if reasoning["effort"] != effort {
				t.Fatalf("reasoning.effort = %v, want %s", reasoning["effort"], effort)
			}
			if reasoning["summary"] != "detailed" {
				t.Fatalf("reasoning.summary = %v, want detailed", reasoning["summary"])
			}
		})
	}
}

func TestCodexBackendRequestPreservesReasoningSummary(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "detailed",
			body: `{"model":"gpt-5.5","reasoning":{"summary":"detailed"},"messages":[{"role":"user","content":"Reply OK"}]}`,
			want: "detailed",
		},
		{
			name: "none falls back to detailed",
			body: `{"model":"gpt-5.5","reasoning":{"summary":"none"},"messages":[{"role":"user","content":"Reply OK"}]}`,
			want: "detailed",
		},
		{
			name: "missing falls back to detailed",
			body: `{"model":"gpt-5.5","messages":[{"role":"user","content":"Reply OK"}]}`,
			want: "detailed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _, err := codexBackendRequestFromGateway("/v1/chat/completions", []byte(tt.body), "", "")
			if err != nil {
				t.Fatal(err)
			}
			reasoning, ok := req["reasoning"].(map[string]any)
			if !ok {
				t.Fatalf("missing reasoning payload: %#v", req)
			}
			if reasoning["summary"] != tt.want {
				t.Fatalf("reasoning.summary = %v, want %s", reasoning["summary"], tt.want)
			}
			if _, ok := reasoning["effort"]; ok {
				t.Fatalf("unexpected reasoning.effort: %#v", reasoning)
			}
		})
	}
}

func TestRunCodexCLICancelsWithRequestContext(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-codex")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nsleep 5\necho OK\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("CODEX_CLI_TIMEOUT_SECONDS", "30")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	started := time.Now()
	_, _, err := (&openAIGatewayHandler{}).runCodexCLI(
		ctx,
		"/v1/chat/completions",
		[]byte(`{"messages":[{"role":"user","content":"Reply OK"}]}`),
		"gpt-5.5",
		"",
	)
	if err == nil {
		t.Fatal("expected canceled context to stop codex cli")
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("codex cli did not stop with request context, elapsed=%s", elapsed)
	}
}

func TestRunCodexCLIPassesAllReasoningEfforts(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "fake-codex")
	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-c" ]; then
    printf "%s" "$2" > "$SEEN_CODEX_CONFIG_PATH"
  fi
  shift
done
cat >/dev/null
echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"OK"}}'
echo '{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":1,"total_tokens":11}}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("CODEX_CLI_TIMEOUT_SECONDS", "30")

	for _, effort := range []string{"low", "medium", "high", "xhigh"} {
		t.Run(effort, func(t *testing.T) {
			seenPath := filepath.Join(tmp, "seen-"+effort)
			t.Setenv("SEEN_CODEX_CONFIG_PATH", seenPath)
			content, metrics, err := (&openAIGatewayHandler{}).runCodexCLI(
				context.Background(),
				"/v1/chat/completions",
				[]byte(`{"messages":[{"role":"user","content":"Reply OK"}]}`),
				"gpt-5.5",
				effort,
			)
			if err != nil {
				t.Fatal(err)
			}
			if content != "OK" || metrics.TotalTokens != 11 {
				t.Fatalf("content=%q metrics=%#v", content, metrics)
			}
			seen, err := os.ReadFile(seenPath)
			if err != nil {
				t.Fatal(err)
			}
			want := `model_reasoning_effort="` + effort + `"`
			if string(seen) != want {
				t.Fatalf("codex config = %q, want %q", string(seen), want)
			}
		})
	}
}

func TestRunCodexCLIPassesImagesToCodexExec(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "fake-codex")
	seenPath := filepath.Join(tmp, "seen-image-path")
	script := `#!/bin/sh
found=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--image" ]; then
    found="$2"
    break
  fi
  shift
done
if [ -z "$found" ] || [ ! -s "$found" ]; then
  echo "missing image" >&2
  exit 9
fi
printf "%s" "$found" > "$SEEN_IMAGE_PATH"
cat >/dev/null
echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"OK"}}'
echo '{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":1,"total_tokens":11}}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("CODEX_CLI_TIMEOUT_SECONDS", "30")
	t.Setenv("SEEN_IMAGE_PATH", seenPath)

	content, metrics, err := (&openAIGatewayHandler{}).runCodexCLI(
		context.Background(),
		"/v1/chat/completions",
		[]byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"Describe this image."},{"type":"image_url","image_url":{"url":"data:image/png;base64,iVBORw0KGgo="}}]}]}`),
		"gpt-5.5",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if content != "OK" || metrics.TotalTokens != 11 {
		t.Fatalf("content=%q metrics=%#v", content, metrics)
	}
	pathBytes, err := os.ReadFile(seenPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(pathBytes)); !os.IsNotExist(err) {
		t.Fatalf("temporary image was not cleaned up, stat err=%v", err)
	}
}

func TestCodexBackendRequestFromChatCompletionsPayload(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"messages": [
			{"role": "system", "content": "Be concise."},
			{"role": "user", "content": [
				{"type": "text", "text": "Describe this image."},
				{"type": "image_url", "image_url": {"url": "data:image/png;base64,iVBORw0KGgo="}}
			]}
		]
	}`)

	req, model, err := codexBackendRequestFromGateway("/v1/chat/completions", body, "", "low")
	if err != nil {
		t.Fatal(err)
	}
	if model != "gpt-5.5" {
		t.Fatalf("model = %q, want gpt-5.5", model)
	}
	instructions := stringValue(req["instructions"])
	if !strings.Contains(instructions, "Be concise.") {
		t.Fatalf("instructions = %q", instructions)
	}
	if !strings.Contains(instructions, "continue the unfinished task") {
		t.Fatalf("instructions missing resume rule: %q", instructions)
	}
	reasoning := req["reasoning"].(map[string]any)
	if reasoning["effort"] != "low" {
		t.Fatalf("reasoning = %#v", reasoning)
	}
	input := req["input"].([]any)
	message := input[0].(map[string]any)
	content := message["content"].([]any)
	if content[0].(map[string]any)["type"] != "input_text" {
		t.Fatalf("first content part = %#v", content[0])
	}
	if content[1].(map[string]any)["type"] != "input_image" {
		t.Fatalf("second content part = %#v", content[1])
	}
}

func TestCodexBackendRequestPreservesChatToolsAndToolResults(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"tools": [
			{"type": "function", "function": {"name": "shell", "description": "Run a command", "parameters": {"type": "object"}}}
		],
		"messages": [
			{"role": "user", "content": "run pwd"},
			{"role": "assistant", "tool_calls": [
				{"id": "call_1", "type": "function", "function": {"name": "shell", "arguments": "{\"cmd\":\"pwd\"}"}}
			]},
			{"role": "tool", "tool_call_id": "call_1", "content": "C:\\Users\\sauri"}
		]
	}`)

	req, _, err := codexBackendRequestFromGateway("/v1/chat/completions", body, "", "")
	if err != nil {
		t.Fatal(err)
	}
	tools := req["tools"].([]any)
	tool := tools[0].(map[string]any)
	if tool["type"] != "function" || tool["name"] != "shell" {
		t.Fatalf("tool = %#v", tool)
	}
	input := req["input"].([]any)
	if len(input) != 3 {
		t.Fatalf("input len = %d, want 3: %#v", len(input), input)
	}
	call := input[1].(map[string]any)
	if call["type"] != "function_call" || call["call_id"] != "call_1" || call["name"] != "shell" {
		t.Fatalf("function call item = %#v", call)
	}
	if call["id"] != "fc_call_1" {
		t.Fatalf("function call id = %v, want fc_call_1", call["id"])
	}
	output := input[2].(map[string]any)
	if output["type"] != "function_call_output" || output["call_id"] != "call_1" {
		t.Fatalf("function output item = %#v", output)
	}
}

func TestCodexBackendRequestNormalizesFunctionCallIDForBackend(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"input": [
			{"role": "user", "content": "run pwd"},
			{"type": "function_call", "id": "call_7DrSS117GxOq1ztYHVJCjrjZ", "call_id": "call_1", "name": "shell", "arguments": "{\"cmd\":\"pwd\"}"},
			{"type": "function_call_output", "call_id": "call_1", "output": "C:\\Users\\sauri"}
		]
	}`)

	req, _, err := codexBackendRequestFromGateway("/v1/responses", body, "", "")
	if err != nil {
		t.Fatal(err)
	}
	input := req["input"].([]any)
	call := input[1].(map[string]any)
	if call["type"] != "function_call" || call["call_id"] != "call_1" || call["id"] != "fc_call_1" {
		t.Fatalf("function call item = %#v", call)
	}
}

func TestCodexBackendRequestDropsResponsesOrphanFunctionOutputs(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"input": [
			{"role": "user", "content": "continue"},
			{"type": "function_call_output", "call_id": "call_missing", "output": "stale output"},
			{"type": "function_call", "id": "call_7DrSS117GxOq1ztYHVJCjrjZ", "call_id": "call_1", "name": "shell", "arguments": "{\"cmd\":\"pwd\"}"},
			{"type": "function_call_output", "call_id": "call_1", "output": "C:\\Users\\sauri"}
		]
	}`)

	req, _, err := codexBackendRequestFromGateway("/v1/responses", body, "", "")
	if err != nil {
		t.Fatal(err)
	}
	input := req["input"].([]any)
	if len(input) != 3 {
		t.Fatalf("input len = %d, want 3: %#v", len(input), input)
	}
	for _, item := range input {
		message := item.(map[string]any)
		if message["type"] == "function_call_output" && message["call_id"] == "call_missing" {
			t.Fatalf("orphan function output was kept: %#v", input)
		}
	}
	output := input[2].(map[string]any)
	if output["type"] != "function_call_output" || output["call_id"] != "call_1" {
		t.Fatalf("valid function output was not kept: %#v", input)
	}
}

func TestCodexBackendRequestDropsChatOrphanToolResults(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"messages": [
			{"role": "user", "content": "continue"},
			{"role": "tool", "tool_call_id": "call_missing", "content": "stale output"},
			{"role": "assistant", "tool_calls": [
				{"id": "call_1", "type": "function", "function": {"name": "shell", "arguments": "{\"cmd\":\"pwd\"}"}}
			]},
			{"role": "tool", "tool_call_id": "call_1", "content": "C:\\Users\\sauri"}
		]
	}`)

	req, _, err := codexBackendRequestFromGateway("/v1/chat/completions", body, "", "")
	if err != nil {
		t.Fatal(err)
	}
	input := req["input"].([]any)
	if len(input) != 3 {
		t.Fatalf("input len = %d, want 3: %#v", len(input), input)
	}
	output := input[2].(map[string]any)
	if output["type"] != "function_call_output" || output["call_id"] != "call_1" {
		t.Fatalf("valid tool output was not kept: %#v", input)
	}
}

func TestParseCodexBackendSSEPreservesFunctionCall(t *testing.T) {
	body := []byte("data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"shell\",\"arguments\":\"{\\\"cmd\\\":\\\"pwd\\\"}\"}}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"output\":[{\"id\":\"fc_1\",\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"shell\",\"arguments\":\"{\\\"cmd\\\":\\\"pwd\\\"}\"}],\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n")

	content, reasoningSummary, toolCalls, metrics, err := parseCodexBackendSSE(body)
	if err != nil {
		t.Fatal(err)
	}
	if content != "" {
		t.Fatalf("content = %q, want empty", content)
	}
	if reasoningSummary.Text != "" {
		t.Fatalf("reasoning summary = %#v, want empty", reasoningSummary)
	}
	if len(toolCalls) != 1 || toolCalls[0].CallID != "call_1" || toolCalls[0].Name != "shell" {
		t.Fatalf("toolCalls = %#v", toolCalls)
	}
	if metrics.TotalTokens != 2 {
		t.Fatalf("metrics = %#v", metrics)
	}
}

func TestNormalizeGatewayReasoningSummaryReplacesEnglishFallback(t *testing.T) {
	summary := normalizeGatewayReasoningSummary(gatewayReasoningSummary{ID: "rs_1", Text: "**Analyzing request**\n\nI need to answer in Chinese."})
	if summary.ID != "rs_1" || !strings.Contains(summary.Text, "正在分析") || !containsCJK(summary.Text) {
		t.Fatalf("summary = %#v", summary)
	}

	zh := normalizeGatewayReasoningSummary(gatewayReasoningSummary{Text: "正在检查上下文"})
	if zh.Text != "正在检查上下文" {
		t.Fatalf("zh summary = %#v", zh)
	}
}
func TestResponsesStreamUsesResponsesSSEEvents(t *testing.T) {
	handler := &openAIGatewayHandler{}
	rec := httptest.NewRecorder()

	n := handler.writeCodexCLIResponsesStream(rec, "gpt-5.5", "OK", gatewayReasoningSummary{}, nil, openAIUsageMetrics{
		InputTokens:  1,
		OutputTokens: 1,
		TotalTokens:  2,
	})
	if n <= 0 {
		t.Fatalf("response bytes = %d, want positive", n)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"type":"response.output_text.delta"`) {
		t.Fatalf("stream missing output delta:\n%s", body)
	}
	if !strings.Contains(body, `"type":"response.output_item.added"`) {
		t.Fatalf("stream missing output item added:\n%s", body)
	}
	if !strings.Contains(body, `"type":"response.content_part.added"`) {
		t.Fatalf("stream missing content part added:\n%s", body)
	}
	if !strings.Contains(body, `"type":"response.completed"`) {
		t.Fatalf("stream missing response.completed:\n%s", body)
	}
	if strings.Contains(body, "chat.completion.chunk") {
		t.Fatalf("responses stream must not use chat chunks:\n%s", body)
	}

	rec = httptest.NewRecorder()
	_ = handler.writeCodexCLIResponsesStream(rec, "gpt-5.5", "OK", gatewayReasoningSummary{ID: "rs_1", Text: "正在分析请求"}, nil, openAIUsageMetrics{})
	body = rec.Body.String()
	if !strings.Contains(body, `"type":"response.reasoning_summary_text.delta"`) || !strings.Contains(body, "正在分析请求") {
		t.Fatalf("stream missing reasoning summary delta:\n%s", body)
	}
	if !strings.Contains(body, `"type":"reasoning"`) || !strings.Contains(body, `"summary"`) {
		t.Fatalf("completed response missing reasoning output item:\n%s", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("stream missing done marker:\n%s", body)
	}
}

func TestCodexBackendRequestFromPDFInput(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"messages": [
			{"role": "user", "content": [
				{"type": "text", "text": "Summarize this PDF."},
				{"type": "input_file", "filename": "sample.pdf", "file_data": "data:application/pdf;base64,JVBERi0xLjQKJSVFT0Y="}
			]}
		]
	}`)

	req, _, err := codexBackendRequestFromGateway("/v1/chat/completions", body, "", "")
	if err != nil {
		t.Fatal(err)
	}
	input := req["input"].([]any)
	message := input[0].(map[string]any)
	content := message["content"].([]any)
	if content[0].(map[string]any)["type"] != "input_text" {
		t.Fatalf("first content part = %#v", content[0])
	}
	filePart := content[1].(map[string]any)
	if filePart["type"] != "input_file" {
		t.Fatalf("second content part = %#v", content[1])
	}
	if filePart["filename"] != "sample.pdf" {
		t.Fatalf("filename = %v, want sample.pdf", filePart["filename"])
	}
	if filePart["file_data"] != "data:application/pdf;base64,JVBERi0xLjQKJSVFT0Y=" {
		t.Fatalf("file_data = %v", filePart["file_data"])
	}
}

func TestCodexBackendRequestFromPDFFilePartWithRawBase64(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"messages": [
			{"role": "user", "content": [
				{"type": "file", "file": {"name": "raw.pdf", "mimeType": "application/pdf", "data": "JVBERi0xLjQKJSVFT0Y="}}
			]}
		]
	}`)

	req, _, err := codexBackendRequestFromGateway("/v1/chat/completions", body, "", "")
	if err != nil {
		t.Fatal(err)
	}
	input := req["input"].([]any)
	message := input[0].(map[string]any)
	content := message["content"].([]any)
	filePart := content[0].(map[string]any)
	if filePart["type"] != "input_file" || filePart["filename"] != "raw.pdf" {
		t.Fatalf("file part = %#v", filePart)
	}
	if filePart["file_data"] != "data:application/pdf;base64,JVBERi0xLjQKJSVFT0Y=" {
		t.Fatalf("file_data = %v", filePart["file_data"])
	}
}

func TestCodexBackendRequestRejectsUnsupportedFileType(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"messages": [
			{"role": "user", "content": [
				{"type": "input_file", "filename": "sample.docx", "file_data": "data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,AAAA"}
			]}
		]
	}`)

	_, _, err := codexBackendRequestFromGateway("/v1/chat/completions", body, "", "")
	if err == nil {
		t.Fatal("expected unsupported file type error")
	}
	if !strings.Contains(err.Error(), "application/pdf") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCodexBackendRequestUsesDefaultInstructions(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"messages": [
			{"role": "user", "content": "Reply OK"}
		]
	}`)

	req, _, err := codexBackendRequestFromGateway("/v1/chat/completions", body, "", "")
	if err != nil {
		t.Fatal(err)
	}
	instructions := stringValue(req["instructions"])
	if !strings.Contains(instructions, defaultCodexBackendPrompt) {
		t.Fatalf("instructions = %q, want default prompt", instructions)
	}
	if !strings.Contains(instructions, "Before any non-trivial tool call") {
		t.Fatalf("instructions missing visible process rule: %q", instructions)
	}
	if !strings.Contains(instructions, "Do not reply with a generic acknowledgement") {
		t.Fatalf("instructions missing resume rule: %q", instructions)
	}
}

func TestCodexBackendRequestAppendsResumeRuleToExplicitInstructions(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"instructions": "Be concise.",
		"input": "continue"
	}`)

	req, _, err := codexBackendRequestFromGateway("/v1/responses", body, "", "")
	if err != nil {
		t.Fatal(err)
	}
	instructions := stringValue(req["instructions"])
	if !strings.Contains(instructions, "Be concise.") {
		t.Fatalf("instructions lost explicit prompt: %q", instructions)
	}
	if !strings.Contains(instructions, "brief user-visible commentary message in Simplified Chinese") {
		t.Fatalf("instructions missing visible process rule: %q", instructions)
	}
	if !strings.Contains(instructions, "If the user says to continue, proceed with the next concrete step") {
		t.Fatalf("instructions missing resume rule: %q", instructions)
	}
}

func TestCodexBackendRequestAgentPassthroughDoesNotInjectPrompts(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"instructions": "Client-managed agent system prompt.",
		"reasoning": {"summary": "concise"},
		"input": "continue"
	}`)

	req, _, err := codexBackendRequestFromGatewayWithOptions("/v1/responses", body, "", "", gatewayRequestOptions{AgentPassthrough: true})
	if err != nil {
		t.Fatal(err)
	}
	instructions := stringValue(req["instructions"])
	if instructions != "Client-managed agent system prompt." {
		t.Fatalf("instructions = %q, want exact client instructions", instructions)
	}
	if strings.Contains(instructions, "Before any non-trivial tool call") || strings.Contains(instructions, "When this conversation is resumed") {
		t.Fatalf("agent passthrough instructions must not include gateway prompts: %q", instructions)
	}
	reasoning := mapValue(req["reasoning"])
	if got := stringValue(reasoning["summary"]); got != "concise" {
		t.Fatalf("reasoning summary = %q, want concise", got)
	}
}

func TestCodexBackendRequestAgentPassthroughOmitsDefaultInstructions(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"input": "Reply OK"
	}`)

	req, _, err := codexBackendRequestFromGatewayWithOptions("/v1/responses", body, "", "", gatewayRequestOptions{AgentPassthrough: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := req["instructions"]; ok {
		t.Fatalf("agent passthrough should not inject default instructions: %#v", req["instructions"])
	}
	if _, ok := req["reasoning"]; ok {
		t.Fatalf("agent passthrough should not inject default reasoning: %#v", req["reasoning"])
	}
}

func TestCodexBackendInstructionsAreIdempotent(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"instructions": "Be concise.",
		"input": "continue"
	}`)

	req, _, err := codexBackendRequestFromGateway("/v1/responses", body, "", "")
	if err != nil {
		t.Fatal(err)
	}
	once := stringValue(req["instructions"])
	twice := codexBackendInstructions(once)

	if once != twice {
		t.Fatalf("instructions should be idempotent\nonce=%q\ntwice=%q", once, twice)
	}
	if got := strings.Count(twice, "Before any non-trivial tool call"); got != 1 {
		t.Fatalf("visible process rule count = %d, want 1", got)
	}
	if got := strings.Count(twice, "When this conversation is resumed"); got != 1 {
		t.Fatalf("resume rule count = %d, want 1", got)
	}
}

func TestCodexUpstreamModeDefaultsToBackend(t *testing.T) {
	t.Setenv("CODEX_UPSTREAM_MODE", "")

	if got := codexUpstreamMode(); got != codexUpstreamModeBackend {
		t.Fatalf("default upstream mode = %q, want %q", got, codexUpstreamModeBackend)
	}
}

func TestRunCodexUpstreamFallsBackToCLIWhenBackendFails(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "fake-codex")
	script := `#!/bin/sh
cat >/dev/null
echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"OK"}}'
echo '{"type":"turn.completed","usage":{"input_tokens":20,"output_tokens":1,"total_tokens":21}}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_UPSTREAM_MODE", "codex_backend")
	t.Setenv("CODEX_UPSTREAM_FALLBACK_ENABLED", "true")
	t.Setenv("CODEX_AUTH_FILE", filepath.Join(tmp, "missing-auth.json"))
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("CODEX_CLI_TIMEOUT_SECONDS", "30")

	result, err := (&openAIGatewayHandler{}).runCodexUpstream(
		context.Background(),
		codexUpstreamModeBackend,
		true,
		"/v1/chat/completions",
		[]byte(`{"messages":[{"role":"user","content":"Reply OK"}]}`),
		"gpt-5.5",
		"low",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "OK" || result.Metrics.TotalTokens != 21 || result.ActualMode != codexUpstreamModeCLI || !result.Fallback {
		t.Fatalf("result=%#v", result)
	}
}

func TestRunCodexUpstreamDoesNotFallbackByDefault(t *testing.T) {
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "cli-called")
	bin := filepath.Join(tmp, "fake-codex")
	script := `#!/bin/sh
touch "$FAKE_CODEX_MARKER"
cat >/dev/null
echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"OK"}}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_UPSTREAM_MODE", "codex_backend")
	t.Setenv("CODEX_UPSTREAM_FALLBACK_ENABLED", "")
	t.Setenv("CODEX_AUTH_FILE", filepath.Join(tmp, "missing-auth.json"))
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("FAKE_CODEX_MARKER", marker)

	_, err := (&openAIGatewayHandler{}).runCodexUpstream(
		context.Background(),
		codexUpstreamModeBackend,
		false,
		"/v1/chat/completions",
		[]byte(`{"messages":[{"role":"user","content":"Reply OK"}]}`),
		"gpt-5.5",
		"low",
	)
	if err == nil {
		t.Fatal("expected backend failure without CLI fallback")
	}
	if !strings.Contains(err.Error(), "codex backend upstream failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("CLI fallback was invoked, marker stat err=%v", statErr)
	}
}

func TestRunCodexUpstreamDoesNotFallbackToCLIForToolRequests(t *testing.T) {
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "cli-called")
	bin := filepath.Join(tmp, "fake-codex")
	script := `#!/bin/sh
touch "$FAKE_CODEX_MARKER"
cat >/dev/null
echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"OK"}}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_UPSTREAM_MODE", "codex_backend")
	t.Setenv("CODEX_UPSTREAM_FALLBACK_ENABLED", "true")
	t.Setenv("CODEX_AUTH_FILE", filepath.Join(tmp, "missing-auth.json"))
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("FAKE_CODEX_MARKER", marker)

	_, err := (&openAIGatewayHandler{}).runCodexUpstream(
		context.Background(),
		codexUpstreamModeBackend,
		true,
		"/v1/chat/completions",
		[]byte(`{
			"tools": [{"type":"function","function":{"name":"shell","parameters":{"type":"object"}}}],
			"messages": [{"role":"user","content":"read system info"}]
		}`),
		"gpt-5.5",
		"low",
	)
	if err == nil {
		t.Fatal("expected backend-only request to skip CLI fallback")
	}
	if !strings.Contains(err.Error(), "cannot fallback to codex_cli") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("CLI fallback was invoked, marker stat err=%v", statErr)
	}
}

func TestRunCodexUpstreamDoesNotFallbackForAgentPassthrough(t *testing.T) {
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "cli-called")
	bin := filepath.Join(tmp, "fake-codex")
	script := `#!/bin/sh
touch "$FAKE_CODEX_MARKER"
cat >/dev/null
echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"OK"}}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_UPSTREAM_MODE", "codex_backend")
	t.Setenv("CODEX_UPSTREAM_FALLBACK_ENABLED", "true")
	t.Setenv("CODEX_AUTH_FILE", filepath.Join(tmp, "missing-auth.json"))
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("FAKE_CODEX_MARKER", marker)

	result, err := (&openAIGatewayHandler{}).runCodexUpstreamWithOptions(
		context.Background(),
		codexUpstreamModeBackend,
		true,
		"/v1/responses",
		[]byte(`{"input":"Reply OK"}`),
		"gpt-5.5",
		"low",
		gatewayRequestOptions{AgentPassthrough: true, AgentPassthroughReason: "agent_client_type"},
	)
	if err == nil {
		t.Fatal("expected backend failure without CLI fallback")
	}
	if !strings.Contains(err.Error(), "agent passthrough requests cannot fallback") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Diagnostic.AgentPassthrough || !result.Diagnostic.FallbackBlocked {
		t.Fatalf("unexpected diagnostic: %#v", result.Diagnostic)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("CLI fallback was invoked for agent passthrough, marker stat err=%v", statErr)
	}
}

func TestRunCodexBackendPostsResponsesAndParsesSSE(t *testing.T) {
	accessToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+accessToken+`","refresh_token":"refresh-token","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var seen map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %s, want /responses", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+accessToken {
			t.Fatalf("authorization header = %q", got)
		}
		if got := r.Header.Get("ChatGPT-Account-Id"); got != "acct_123" {
			t.Fatalf("account header = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&seen); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.reasoning_summary_text.delta\",\"item_id\":\"rs_1\",\"delta\":\"正在分析\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.reasoning_summary_text.done\",\"item_id\":\"rs_1\",\"text\":\"正在分析请求\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"O\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"K\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":9,\"input_tokens_details\":{\"cached_tokens\":3},\"output_tokens\":5,\"output_tokens_details\":{\"reasoning_tokens\":1},\"total_tokens\":14}}}\n\n"))
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")

	client := &codexBackendClient{httpClient: upstream.Client()}
	content, reasoningSummary, toolCalls, metrics, err := client.run(
		context.Background(),
		"/v1/chat/completions",
		[]byte(`{"messages":[{"role":"user","content":"Reply OK"}]}`),
		"gpt-5.5",
		"low",
	)
	if err != nil {
		t.Fatal(err)
	}
	if content != "OK" {
		t.Fatalf("content = %q, want OK", content)
	}
	if reasoningSummary.ID != "rs_1" || reasoningSummary.Text != "正在分析请求" {
		t.Fatalf("reasoningSummary = %#v", reasoningSummary)
	}
	if len(toolCalls) != 0 {
		t.Fatalf("toolCalls = %#v, want none", toolCalls)
	}
	if metrics.InputTokens != 9 || metrics.CachedTokens != 3 || metrics.OutputTokens != 5 || metrics.ReasoningTokens != 1 || metrics.TotalTokens != 14 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
	if seen["stream"] != true || seen["model"] != "gpt-5.5" {
		t.Fatalf("unexpected backend request: %#v", seen)
	}
	reasoning := seen["reasoning"].(map[string]any)
	if reasoning["effort"] != "low" || reasoning["summary"] != "detailed" {
		t.Fatalf("backend reasoning = %#v, want low + detailed summary", reasoning)
	}
}

func TestStreamCodexBackendResponsesPassesThroughEventsAndUsage(t *testing.T) {
	accessToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+accessToken+`","refresh_token":"refresh-token","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"status\":\"in_progress\"}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"思考中：检查文件。\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"完成。\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":9,\"input_tokens_details\":{\"cached_tokens\":3},\"output_tokens\":5,\"output_tokens_details\":{\"reasoning_tokens\":1},\"total_tokens\":14}}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")

	rec := httptest.NewRecorder()
	result, err := (&openAIGatewayHandler{}).streamCodexBackendResponses(
		context.Background(),
		rec,
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`),
		"gpt-5.5",
		"low",
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "思考中：检查文件。") || !strings.Contains(body, "完成。") {
		t.Fatalf("stream did not pass through output deltas:\n%s", body)
	}
	if !strings.Contains(body, `"type":"response.created"`) || !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("stream missing upstream lifecycle events:\n%s", body)
	}
	if result.Metrics.InputTokens != 9 || result.Metrics.CachedTokens != 3 || result.Metrics.OutputTokens != 5 || result.Metrics.ReasoningTokens != 1 || result.Metrics.TotalTokens != 14 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
}

func TestGatewayLargeRequestGuardBlocksConcurrentLargeRequestsPerKey(t *testing.T) {
	t.Setenv("GATEWAY_LARGE_REQUEST_MIN_BYTES", "16")
	t.Setenv("GATEWAY_LARGE_REQUEST_MAX_INFLIGHT_PER_KEY", "1")
	t.Setenv("GATEWAY_LARGE_REQUEST_BURST_MAX_PER_KEY", "10")
	originalLimiter := gatewayLargeRequestLimiter
	originalBurstLimiter := gatewayLargeRequestBurstLimiter
	gatewayLargeRequestLimiter = newGatewayInFlightLimiter()
	gatewayLargeRequestBurstLimiter = newGatewayBurstLimiter()
	defer func() {
		gatewayLargeRequestLimiter = originalLimiter
		gatewayLargeRequestBurstLimiter = originalBurstLimiter
	}()

	handler := &openAIGatewayHandler{}
	key := &biz.GatewayAPIKey{ID: 9, KeyPrefix: "ogw_test"}
	if release, err := handler.acquireGatewayLargeRequestSlot(key, "/v1/responses", []byte("small")); err != nil || release != nil {
		t.Fatalf("small request should not acquire guard, release_is_nil=%t err=%v", release == nil, err)
	}

	release, err := handler.acquireGatewayLargeRequestSlot(key, "/v1/responses", []byte("0123456789abcdef"))
	if err != nil {
		t.Fatalf("first large request acquire failed: %v", err)
	}
	if release == nil {
		t.Fatal("first large request should return a release function")
	}
	if _, err := handler.acquireGatewayLargeRequestSlot(key, "/v1/chat/completions", []byte("0123456789abcdef")); !errors.Is(err, errGatewayLargeRequestInFlight) {
		t.Fatalf("second large request err = %v, want inflight", err)
	}

	release()
	release, err = handler.acquireGatewayLargeRequestSlot(key, "/v1/chat/completions", []byte("0123456789abcdef"))
	if err != nil {
		t.Fatalf("large request should acquire after release: %v", err)
	}
	release()
}

func TestGatewayLargeRequestGuardBlocksBurstPerKey(t *testing.T) {
	t.Setenv("GATEWAY_LARGE_REQUEST_MIN_BYTES", "4")
	t.Setenv("GATEWAY_LARGE_REQUEST_MAX_INFLIGHT_PER_KEY", "0")
	t.Setenv("GATEWAY_LARGE_REQUEST_BURST_MAX_PER_KEY", "2")
	t.Setenv("GATEWAY_LARGE_REQUEST_BURST_WINDOW_SECONDS", "60")
	originalLimiter := gatewayLargeRequestLimiter
	originalBurstLimiter := gatewayLargeRequestBurstLimiter
	gatewayLargeRequestLimiter = newGatewayInFlightLimiter()
	gatewayLargeRequestBurstLimiter = newGatewayBurstLimiter()
	defer func() {
		gatewayLargeRequestLimiter = originalLimiter
		gatewayLargeRequestBurstLimiter = originalBurstLimiter
	}()

	handler := &openAIGatewayHandler{}
	key := &biz.GatewayAPIKey{ID: 10, KeyPrefix: "ogw_burst"}
	for i := 0; i < 2; i++ {
		release, err := handler.acquireGatewayLargeRequestSlot(key, "/v1/responses", []byte("large"))
		if err != nil {
			t.Fatalf("large request %d should pass: %v", i+1, err)
		}
		if release != nil {
			release()
		}
	}
	if _, err := handler.acquireGatewayLargeRequestSlot(key, "/v1/responses", []byte("large")); !errors.Is(err, errGatewayLargeRequestBurst) {
		t.Fatalf("third large request err = %v, want burst", err)
	}
	if _, err := handler.acquireGatewayLargeRequestSlot(&biz.GatewayAPIKey{ID: 11}, "/v1/responses", []byte("large")); err != nil {
		t.Fatalf("different key should have separate burst window: %v", err)
	}
	if _, err := handler.acquireGatewayLargeRequestSlot(key, "/v1/models", []byte("large")); err != nil {
		t.Fatalf("models endpoint should not use large request guard: %v", err)
	}
}

func TestGatewayLargeRequestLimitResponseIncludesRetryAfter(t *testing.T) {
	t.Setenv("GATEWAY_LARGE_REQUEST_BURST_WINDOW_SECONDS", "60")
	handler := &openAIGatewayHandler{}
	key := &biz.GatewayAPIKey{ID: 12, KeyPrefix: "ogw_limit"}
	cases := []struct {
		name        string
		err         error
		code        string
		messagePart string
	}{
		{
			name:        "inflight",
			err:         errGatewayLargeRequestInFlight,
			code:        "gateway_large_request_inflight",
			messagePart: "已有大上下文请求在运行",
		},
		{
			name:        "burst",
			err:         errGatewayLargeRequestBurst,
			code:        "gateway_large_request_burst",
			messagePart: "不是上游额度耗尽",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader("{}"))
			handler.writeGatewayLargeRequestLimit(rec, req, key, "req_limit", "", "responses", "gpt-5.5", "high", codexUpstreamModeBackend, true, []byte("large body"), time.Now(), biz.GatewayUsageDiagnostic{}, tc.err)

			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("status = %d, want 429", rec.Code)
			}
			if got := rec.Header().Get("Retry-After"); got != "60" {
				t.Fatalf("Retry-After = %q, want 60", got)
			}
			var payload struct {
				Error struct {
					Message           string `json:"message"`
					Type              string `json:"type"`
					Code              string `json:"code"`
					RetryAfterSeconds int    `json:"retry_after_seconds"`
				} `json:"error"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.Error.Type != "gateway_error" {
				t.Fatalf("error.type = %q, want gateway_error", payload.Error.Type)
			}
			if payload.Error.Code != tc.code {
				t.Fatalf("error.code = %q, want %q", payload.Error.Code, tc.code)
			}
			if payload.Error.RetryAfterSeconds != 60 {
				t.Fatalf("retry_after_seconds = %d, want 60", payload.Error.RetryAfterSeconds)
			}
			if !strings.Contains(payload.Error.Message, tc.messagePart) || !strings.Contains(payload.Error.Message, "60 秒") || !strings.Contains(payload.Error.Message, "网关保护") {
				t.Fatalf("message = %q", payload.Error.Message)
			}
		})
	}
}

type failingStreamResponseWriter struct {
	header    http.Header
	failAfter int
	writes    int
	status    int
}

func newFailingStreamResponseWriter(failAfter int) *failingStreamResponseWriter {
	return &failingStreamResponseWriter{
		header:    make(http.Header),
		failAfter: failAfter,
	}
}

func (w *failingStreamResponseWriter) Header() http.Header {
	return w.header
}

func (w *failingStreamResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *failingStreamResponseWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes > w.failAfter {
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}

func (w *failingStreamResponseWriter) Flush() {}

func TestStreamCodexBackendResponsesCancelsUpstreamOnClientWriteFailure(t *testing.T) {
	accessToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+accessToken+`","refresh_token":"refresh-token","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	upstreamCanceled := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		<-r.Context().Done()
		close(upstreamCanceled)
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")

	writer := newFailingStreamResponseWriter(1)
	result, err := (&openAIGatewayHandler{}).streamCodexBackendResponses(
		context.Background(),
		writer,
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`),
		"gpt-5.5",
		"high",
		false,
	)
	if !isGatewayClientCanceled(err) {
		t.Fatalf("err = %v, want client canceled", err)
	}
	if !result.Diagnostic.UpstreamStreamStarted {
		t.Fatalf("expected upstream stream to start before write failure: %#v", result.Diagnostic)
	}
	select {
	case <-upstreamCanceled:
	case <-time.After(time.Second):
		t.Fatal("upstream request was not canceled after client write failure")
	}
}

func TestStreamCodexBackendResponsesClassifiesMidStreamClose(t *testing.T) {
	accessToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+accessToken+`","refresh_token":"refresh-token","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"status\":\"in_progress\"}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n"))
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")

	rec := httptest.NewRecorder()
	result, err := (&openAIGatewayHandler{}).streamCodexBackendResponses(
		context.Background(),
		rec,
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`),
		"gpt-5.5",
		"high",
		false,
	)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want unexpected EOF", err)
	}
	if result.ErrorType != "codex_backend_stream_interrupted" {
		t.Fatalf("error type = %q, want codex_backend_stream_interrupted", result.ErrorType)
	}
	if !result.Diagnostic.UpstreamStreamStarted || result.Diagnostic.UpstreamStreamCompleted || result.Diagnostic.UpstreamStreamDoneSeen {
		t.Fatalf("unexpected stream diagnostic: %#v", result.Diagnostic)
	}
	if result.Diagnostic.UpstreamStreamEvents != 2 {
		t.Fatalf("upstream events = %d, want 2", result.Diagnostic.UpstreamStreamEvents)
	}
	if !strings.Contains(rec.Body.String(), `"code":"codex_backend_stream_interrupted"`) {
		t.Fatalf("stream missing interrupted failure event:\n%s", rec.Body.String())
	}
}

func TestStreamCodexBackendResponsesKeepsAliveBeforeUpstreamHeaders(t *testing.T) {
	accessToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+accessToken+`","refresh_token":"refresh-token","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":2,\"output_tokens\":1,\"total_tokens\":3}}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")
	t.Setenv("GATEWAY_STREAM_HEARTBEAT_SECONDS", "1")

	errCh := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := (&openAIGatewayHandler{}).streamCodexBackendResponses(
			r.Context(),
			w,
			"/v1/responses",
			[]byte(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`),
			"gpt-5.5",
			"medium",
			false,
		)
		errCh <- err
	}))
	defer server.Close()

	start := time.Now()
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	first := readNonEmptyStreamLine(t, reader)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("first keepalive took %s, want under 1s", elapsed)
	}
	if first != ": keepalive\n" {
		t.Fatalf("first line = %q, want keepalive comment", first)
	}
	second := readNonEmptyStreamLine(t, reader)
	if !strings.Contains(second, "keepalive") && !strings.Contains(second, "response.completed") {
		t.Fatalf("second line = %q, want keepalive or upstream event", second)
	}
}

func TestStreamCodexBackendResponsesRetriesOpenFailureBeforeUpstreamEvents(t *testing.T) {
	accessToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+accessToken+`","refresh_token":"refresh-token","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var attempts atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"temporary upstream failure"}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"OK\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")
	t.Setenv("CODEX_BACKEND_RETRY_ATTEMPTS", "2")

	rec := httptest.NewRecorder()
	result, err := (&openAIGatewayHandler{}).streamCodexBackendResponses(
		context.Background(),
		rec,
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`),
		"gpt-5.5",
		"high",
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
	if result.Metrics.TotalTokens != 2 {
		t.Fatalf("metrics = %#v, want total tokens", result.Metrics)
	}
	body := rec.Body.String()
	if !strings.Contains(body, ": keepalive") || !strings.Contains(body, `"delta":"OK"`) || strings.Contains(body, `"code":"codex_backend_http_5xx"`) {
		t.Fatalf("unexpected stream body after retry:\n%s", body)
	}
}

func TestMergeGatewayUsageDiagnosticPreservesContextPreflight(t *testing.T) {
	got := mergeGatewayUsageDiagnostic(
		biz.GatewayUsageDiagnostic{
			RequestBytes:                   1200,
			ContextOriginalBytes:           1200,
			ContextOriginalEstimatedTokens: 300,
			ContextWindowTokens:            400000,
			ContextCompactTokenLimit:       260000,
			ContextHardTokenLimit:          380000,
			ContextCompactByteLimit:        1000000,
			ContextHardByteLimit:           1900000,
			ContextKeepItems:               8,
		},
		biz.GatewayUsageDiagnostic{
			RequestBytes:          900,
			ResponseBytes:         80,
			UpstreamBody:          "unexpected EOF",
			UpstreamStreamStarted: true,
			UpstreamStreamEvents:  2,
		},
	)
	if got.RequestBytes != 900 || got.ResponseBytes != 80 || got.UpstreamBody != "unexpected EOF" {
		t.Fatalf("overlay fields not applied: %#v", got)
	}
	if got.ContextOriginalBytes != 1200 || got.ContextWindowTokens != 400000 || got.ContextKeepItems != 8 {
		t.Fatalf("context preflight fields not preserved: %#v", got)
	}
	if !got.UpstreamStreamStarted || got.UpstreamStreamEvents != 2 {
		t.Fatalf("stream fields not applied: %#v", got)
	}
}

func TestStreamCodexBackendResponsesCompletedThenClientCancelIsSuccess(t *testing.T) {
	accessToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+accessToken+`","refresh_token":"refresh-token","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"OK\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":2,\"output_tokens\":1,\"total_tokens\":3}}}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		<-r.Context().Done()
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	rec := httptest.NewRecorder()
	result, err := (&openAIGatewayHandler{}).streamCodexBackendResponses(
		ctx,
		rec,
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`),
		"gpt-5.5",
		"medium",
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorType != "" {
		t.Fatalf("error type = %q", result.ErrorType)
	}
	if result.Metrics.TotalTokens != 3 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
	if !strings.Contains(rec.Body.String(), `"type":"response.completed"`) {
		t.Fatalf("stream missing completed event:\n%s", rec.Body.String())
	}
}

func TestStreamCodexBackendResponsesClassifiesContextLengthFailedEvent(t *testing.T) {
	accessToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+accessToken+`","refresh_token":"refresh-token","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"response.failed","response":{"id":"resp_1","status":"failed","error":{"code":"context_length_exceeded","message":"Your input exceeds the context window of this model."}}}` + "\n\n"))
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")

	rec := httptest.NewRecorder()
	result, err := (&openAIGatewayHandler{}).streamCodexBackendResponses(
		context.Background(),
		rec,
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`),
		"gpt-5.5",
		"medium",
		false,
	)
	if err == nil {
		t.Fatal("expected response.failed to return an error")
	}
	if result.ErrorType != gatewayContextErrorType {
		t.Fatalf("error type = %q, want %q", result.ErrorType, gatewayContextErrorType)
	}
	if !strings.Contains(result.Diagnostic.UpstreamBody, "context_length_exceeded") {
		t.Fatalf("diagnostic missing context error: %s", result.Diagnostic.UpstreamBody)
	}
}

func TestStreamCodexBackendResponsesClassifiesOverloadedFailedEvent(t *testing.T) {
	accessToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+accessToken+`","refresh_token":"refresh-token","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"response.failed","response":{"id":"resp_overloaded","status":"failed","error":{"code":"server_is_overloaded","message":"Our servers are currently overloaded. Please try again later."},"instructions":"must-not-leak"}}` + "\n\n"))
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")

	rec := httptest.NewRecorder()
	result, err := (&openAIGatewayHandler{}).streamCodexBackendResponses(
		context.Background(),
		rec,
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","stream":true,"input":"Reply OK"}`),
		"gpt-5.5",
		"medium",
		false,
	)
	if err == nil {
		t.Fatal("expected response.failed to return an error")
	}
	if result.ErrorType != "codex_backend_overloaded" {
		t.Fatalf("error type = %q, want codex_backend_overloaded", result.ErrorType)
	}
	if result.Diagnostic.UpstreamErrorCode != "server_is_overloaded" {
		t.Fatalf("upstream error code = %q", result.Diagnostic.UpstreamErrorCode)
	}
	if strings.Contains(result.Diagnostic.UpstreamBody, "must-not-leak") {
		t.Fatalf("diagnostic leaked instructions: %s", result.Diagnostic.UpstreamBody)
	}
}

func TestRunCodexBackendRetriesTransientHTTPError(t *testing.T) {
	accessToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+accessToken+`","refresh_token":"refresh-token","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var attempts atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"temporary upstream failure"}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"OK\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"))
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")
	t.Setenv("CODEX_BACKEND_RETRY_ATTEMPTS", "2")

	client := &codexBackendClient{httpClient: upstream.Client()}
	content, _, _, metrics, err := client.run(
		context.Background(),
		"/v1/chat/completions",
		[]byte(`{"messages":[{"role":"user","content":"Reply OK"}]}`),
		"gpt-5.5",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if content != "OK" || metrics.TotalTokens != 2 {
		t.Fatalf("content=%q metrics=%#v", content, metrics)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
}

func TestCodexBackendRefreshesExpiredAccessToken(t *testing.T) {
	expiredToken := testJWT(time.Now().Add(-time.Hour).Unix(), "acct_123")
	freshToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+expiredToken+`","refresh_token":"old-refresh","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["grant_type"] != "refresh_token" || body["refresh_token"] != "old-refresh" {
				t.Fatalf("unexpected refresh request: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token":  freshToken,
				"refresh_token": "new-refresh",
			})
		case "/responses":
			if got := r.Header.Get("Authorization"); got != "Bearer "+freshToken {
				t.Fatalf("authorization header = %q", got)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"OK\"}\n\n"))
			_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"))
		default:
			t.Fatalf("unexpected path = %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_REFRESH_TOKEN_URL_OVERRIDE", upstream.URL+"/oauth/token")

	client := &codexBackendClient{httpClient: upstream.Client()}
	t.Cleanup(client.httpClient.CloseIdleConnections)
	content, _, _, _, err := client.run(
		context.Background(),
		"/v1/chat/completions",
		[]byte(`{"messages":[{"role":"user","content":"Reply OK"}]}`),
		"gpt-5.5",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if content != "OK" {
		t.Fatalf("content = %q, want OK", content)
	}
	saved, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), "new-refresh") {
		t.Fatalf("auth file did not persist refreshed token: %s", string(saved))
	}
}

func testJWT(exp int64, accountID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, _ := json.Marshal(map[string]any{
		"exp":                exp,
		"chatgpt_account_id": accountID,
	})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func TestSessionIDFromGatewayRequest(t *testing.T) {
	body := []byte(`{"metadata":{"session_id":"session-123"}}`)
	req := httptest.NewRequest("POST", "/v1/responses", nil)
	if got := sessionIDFromGatewayRequest(req, body); got != "session-123" {
		t.Fatalf("session id from metadata = %q, want session-123", got)
	}

	req = httptest.NewRequest("POST", "/v1/responses", nil)
	req.Header.Set("X-Conversation-ID", "conversation-456")
	if got := sessionIDFromGatewayRequest(req, body); got != "conversation-456" {
		t.Fatalf("header session id = %q, want conversation-456", got)
	}
}

func TestGatewayClientTypeFromRequest(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name: "explicit opencode",
			headers: map[string]string{
				"X-Client-Type": "opencode",
				"User-Agent":    "codex-cli/0.133.0",
			},
			want: biz.GatewayClientTypeOpenCode,
		},
		{
			name: "app name codex",
			headers: map[string]string{
				"X-App-Name": "Codex Desktop",
			},
			want: biz.GatewayClientTypeCodex,
		},
		{
			name: "user agent open code",
			headers: map[string]string{
				"User-Agent": "OpenCode/1.14.48",
			},
			want: biz.GatewayClientTypeOpenCode,
		},
		{
			name: "unknown",
			headers: map[string]string{
				"User-Agent": "curl/8.7.1",
			},
			want: biz.GatewayClientTypeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/responses", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			if got := gatewayClientTypeFromRequest(req); got != tt.want {
				t.Fatalf("gatewayClientTypeFromRequest() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGatewayRequestOptionsEnableAgentPassthroughForAgentClients(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/responses", nil)
	req.Header.Set("User-Agent", "codex-cli/0.143.0")

	options := gatewayRequestOptionsFromRequest(req, []byte(`{"model":"gpt-5.5","input":"Reply OK"}`))
	if !options.AgentPassthrough || options.AgentPassthroughReason != "agent_client_type" {
		t.Fatalf("options = %#v, want agent client passthrough", options)
	}
}

func TestGatewayRequestOptionsSupportExplicitPassthroughFlags(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/responses", nil)
	req.Header.Set("User-Agent", "curl/8.7.1")
	body := []byte(`{"model":"gpt-5.5","input":"Reply OK","metadata":{"disable_context_compression":true}}`)

	options := gatewayRequestOptionsFromRequest(req, body)
	if !options.AgentPassthrough || options.AgentPassthroughReason != "explicit_body" {
		t.Fatalf("options = %#v, want explicit body passthrough", options)
	}
}

func TestGatewayRequestOptionsAllowHeaderDisableForAgentClients(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/responses", nil)
	req.Header.Set("User-Agent", "OpenCode/1.14.48")
	req.Header.Set("X-Gateway-Agent-Passthrough", "false")

	options := gatewayRequestOptionsFromRequest(req, []byte(`{"model":"gpt-5.5","input":"Reply OK"}`))
	if options.AgentPassthrough {
		t.Fatalf("options = %#v, want passthrough disabled by header", options)
	}
}

func TestGatewayUsageDiagnosticForBackendOnlyFailure(t *testing.T) {
	body := []byte(`{
		"model":"gpt-5.5",
		"reasoning_effort":"high",
		"tools":[{"type":"function","function":{"name":"run_shell"}}],
		"messages":[{"role":"user","content":"Reply OK"}]
	}`)
	err := codexBackendHTTPError{status: http.StatusBadGateway, body: []byte(`{"error":"upstream overloaded"}`)}

	diagnostic := gatewayUsageDiagnosticForUpstreamFailure("/v1/chat/completions", body, "high", err, true)

	if !diagnostic.BackendOnly || !diagnostic.FallbackEnabled || !diagnostic.FallbackBlocked {
		t.Fatalf("unexpected fallback flags: %#v", diagnostic)
	}
	if diagnostic.RequestBytes != int64(len(body)) || diagnostic.ResponseBytes == 0 {
		t.Fatalf("unexpected byte metrics: %#v", diagnostic)
	}
	if diagnostic.UpstreamHTTPStatus != http.StatusBadGateway {
		t.Fatalf("upstream status = %d, want 502", diagnostic.UpstreamHTTPStatus)
	}
	if !strings.Contains(diagnostic.UpstreamBody, "upstream overloaded") {
		t.Fatalf("upstream body summary = %q", diagnostic.UpstreamBody)
	}
}
