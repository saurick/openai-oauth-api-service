package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("codex cli did not stop with request context, elapsed=%s", elapsed)
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
	if req["instructions"] != "Be concise." {
		t.Fatalf("instructions = %q", req["instructions"])
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
	if req["instructions"] != defaultCodexBackendPrompt {
		t.Fatalf("instructions = %q, want default prompt", req["instructions"])
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
	t.Setenv("CODEX_AUTH_FILE", filepath.Join(tmp, "missing-auth.json"))
	t.Setenv("CODEX_CLI_BIN", bin)
	t.Setenv("CODEX_CLI_TIMEOUT_SECONDS", "30")

	result, err := (&openAIGatewayHandler{}).runCodexUpstream(
		context.Background(),
		codexUpstreamModeBackend,
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
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"O\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"K\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":9,\"input_tokens_details\":{\"cached_tokens\":3},\"output_tokens\":5,\"output_tokens_details\":{\"reasoning_tokens\":1},\"total_tokens\":14}}}\n\n"))
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_BACKEND_TIMEOUT_SECONDS", "30")

	client := &codexBackendClient{httpClient: upstream.Client()}
	content, metrics, err := client.run(
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
	if metrics.InputTokens != 9 || metrics.CachedTokens != 3 || metrics.OutputTokens != 5 || metrics.ReasoningTokens != 1 || metrics.TotalTokens != 14 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
	if seen["stream"] != true || seen["model"] != "gpt-5.5" {
		t.Fatalf("unexpected backend request: %#v", seen)
	}
}

func TestCodexBackendRefreshesExpiredAccessToken(t *testing.T) {
	expiredToken := testJWT(time.Now().Add(-time.Hour).Unix(), "acct_123")
	freshToken := testJWT(time.Now().Add(time.Hour).Unix(), "acct_123")
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"`+expiredToken+`","refresh_token":"old-refresh","account_id":"acct_123"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	refresh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Fatalf("refresh path = %s", r.URL.Path)
		}
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
	}))
	defer refresh.Close()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+freshToken {
			t.Fatalf("authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"OK\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"))
	}))
	defer upstream.Close()

	t.Setenv("CODEX_AUTH_FILE", authPath)
	t.Setenv("CODEX_BACKEND_BASE_URL", upstream.URL)
	t.Setenv("CODEX_REFRESH_TOKEN_URL_OVERRIDE", refresh.URL+"/oauth/token")

	client := &codexBackendClient{httpClient: upstream.Client()}
	content, _, err := client.run(
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
