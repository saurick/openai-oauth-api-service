package server

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"server/internal/biz"
)

func TestCompactGatewayContextChatPreservesRecentMessages(t *testing.T) {
	t.Setenv("GATEWAY_CONTEXT_KEEP_ITEMS", "2")
	oldText := strings.Repeat("old failure log /tmp/app/main.go context_length_exceeded\n", 200)
	body := []byte(`{"model":"gpt-5.5","stream":true,"messages":[` +
		`{"role":"system","content":"follow project rules"},` +
		`{"role":"user","content":` + mustJSONQuote(oldText) + `},` +
		`{"role":"assistant","content":"old answer"},` +
		`{"role":"user","content":"latest question"},` +
		`{"role":"assistant","content":"latest answer"}` +
		`]}`)

	compacted, err := compactGatewayContextRequest("/v1/chat/completions", body, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	if compacted.CompactedBytes >= compacted.OriginalBytes {
		t.Fatalf("compacted bytes = %d, original = %d", compacted.CompactedBytes, compacted.OriginalBytes)
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	messages := payload["messages"].([]any)
	if got := stringValue(messages[len(messages)-2].(map[string]any)["content"]); got != "latest question" {
		t.Fatalf("latest user message = %q", got)
	}
	if !strings.Contains(compacted.Summary, "/tmp/app/main.go") {
		t.Fatalf("summary missing path: %s", compacted.Summary)
	}
}

func TestCompactGatewayContextResponsesString(t *testing.T) {
	oldText := strings.Repeat("tool output line with 502 and stream heartbeat\n", 300)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(oldText) + `}`)

	compacted, err := compactGatewayContextRequest("/v1/responses", body, "previous summary", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	input := stringValue(payload["input"])
	if !strings.Contains(input, "previous summary") {
		t.Fatalf("compacted input missing previous summary: %s", input)
	}
	if !strings.Contains(input, "tool output line") {
		t.Fatalf("compacted input missing tail context")
	}
}

func TestCompactGatewayContextCarriesPreviousSummaryAcrossCompactions(t *testing.T) {
	firstHistory := strings.Repeat("phase one failed at /workspace/service/auth.go with oauth callback timeout\n", 240)
	firstBody := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(firstHistory+"next step is patch auth callback handling") + `}`)

	first, err := compactGatewayContextRequest("/v1/responses", firstBody, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed {
		t.Fatal("expected first compaction")
	}
	if !strings.Contains(first.Summary, "/workspace/service/auth.go") {
		t.Fatalf("first summary missing auth path: %s", first.Summary)
	}

	secondHistory := strings.Repeat("phase two terminal output says regression passed but deploy smoke still pending\n", 240)
	secondBody := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(secondHistory+"继续，从上次打断的部署验证开始") + `}`)

	second, err := compactGatewayContextRequest("/v1/responses", secondBody, first.Summary, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Changed {
		t.Fatal("expected second compaction")
	}

	var payload map[string]any
	if err := json.Unmarshal(second.Body, &payload); err != nil {
		t.Fatal(err)
	}
	input := stringValue(payload["input"])
	if !strings.Contains(input, "/workspace/service/auth.go") {
		t.Fatalf("second compacted input missing previous summary: %s", input)
	}
	if !strings.Contains(input, "部署验证开始") {
		t.Fatalf("second compacted input missing latest resume instruction: %s", input)
	}
	if !strings.Contains(second.Summary, "/workspace/service/auth.go") {
		t.Fatalf("second summary did not carry first summary forward: %s", second.Summary)
	}
}

func TestCompactGatewayContextSingleLargeChatMessageIncludesSummary(t *testing.T) {
	oldText := strings.Repeat("single huge prompt with /workspace/app/main.go and context_length_exceeded\n", 200)
	body := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":` + mustJSONQuote(oldText+"reply FINAL_OK") + `}]}`)

	compacted, err := compactGatewayContextRequest("/v1/chat/completions", body, "", 8)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	messages := payload["messages"].([]any)
	content := stringValue(messages[0].(map[string]any)["content"])
	if !strings.Contains(content, "自动压缩摘要") {
		t.Fatalf("compacted content missing summary: %s", content)
	}
	if !strings.Contains(content, "FINAL_OK") {
		t.Fatalf("compacted content missing tail instruction: %s", content)
	}
}

func TestPrepareGatewayContextCompactsBeforeUpstream(t *testing.T) {
	t.Setenv("GATEWAY_CONTEXT_COMPACT_BYTES", "1000")
	t.Setenv("GATEWAY_CONTEXT_HARD_BYTES", "1000000")
	t.Setenv("GATEWAY_CONTEXT_KEEP_ITEMS", "2")
	oldText := strings.Repeat("old failing context /srv/app/server.go context_length_exceeded\n", 200)
	body := []byte(`{"model":"gpt-5.5","messages":[` +
		`{"role":"system","content":"follow project rules"},` +
		`{"role":"user","content":` + mustJSONQuote(oldText) + `},` +
		`{"role":"assistant","content":"old answer"},` +
		`{"role":"user","content":"latest request"}` +
		`]}`)

	prepared, err := (&openAIGatewayHandler{}).prepareGatewayContext(
		context.Background(),
		&biz.GatewayAPIKey{ID: 12, KeyPrefix: "ogw_test"},
		"req_test",
		"session_test",
		"/v1/chat/completions",
		body,
		"unknown-test-model",
		"xhigh",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.Diagnostic.ContextCompacted {
		t.Fatal("expected preflight compaction diagnostic")
	}
	if prepared.Diagnostic.ContextCompactionCount != 1 {
		t.Fatalf("compaction count = %d, want 1", prepared.Diagnostic.ContextCompactionCount)
	}
	if len(prepared.Body) >= len(body) {
		t.Fatalf("prepared body length = %d, original = %d", len(prepared.Body), len(body))
	}
	if !strings.Contains(prepared.Diagnostic.ContextCompactionSummary, "/srv/app/server.go") {
		t.Fatalf("summary missing path: %s", prepared.Diagnostic.ContextCompactionSummary)
	}
}

func TestPrepareGatewayContextBlocksUncompactableOversizedRequest(t *testing.T) {
	t.Setenv("GATEWAY_CONTEXT_COMPACT_BYTES", "1000")
	body := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":"tiny"}],"metadata":{"blob":` + mustJSONQuote(strings.Repeat("x", 2000)) + `}}`)

	prepared, err := (&openAIGatewayHandler{}).prepareGatewayContext(
		context.Background(),
		&biz.GatewayAPIKey{ID: 12, KeyPrefix: "ogw_test"},
		"req_test",
		"session_test",
		"/v1/chat/completions",
		body,
		"unknown-test-model",
		"xhigh",
	)
	if !errors.Is(err, errGatewayContextLengthExceeded) {
		t.Fatalf("err = %v, want errGatewayContextLengthExceeded", err)
	}
	if prepared.Diagnostic.ContextCompactionReason != "not_compactable" {
		t.Fatalf("reason = %q, want not_compactable", prepared.Diagnostic.ContextCompactionReason)
	}
	if prepared.Diagnostic.ContextCompacted {
		t.Fatal("uncompactable request must not be marked compacted")
	}
}

func TestEstimateGatewayRequestTokensIncludesOlderToolContext(t *testing.T) {
	toolOutput := strings.Repeat("legacy tool output context_length_exceeded /tmp/app/main.go\n", 2000)
	body := []byte(`{"model":"gpt-5.5","stream":true,"messages":[` +
		`{"role":"system","content":"follow project rules"},` +
		`{"role":"tool","content":` + mustJSONQuote(toolOutput) + `},` +
		`{"role":"assistant","content":"old answer"},` +
		`{"role":"user","content":"latest short question"}` +
		`]}`)

	got := estimateGatewayRequestTokens("/v1/chat/completions", body)
	wantAtLeast := estimateTokenCount(toolOutput)
	if got < wantAtLeast {
		t.Fatalf("estimated tokens = %d, want at least old tool output tokens %d", got, wantAtLeast)
	}
}

func TestPrepareGatewayContextCompactsLargeRequestBelowPreviousByteLimit(t *testing.T) {
	t.Setenv("GATEWAY_CONTEXT_COMPACT_TOKENS", "999999")
	t.Setenv("GATEWAY_CONTEXT_HARD_TOKENS", "1000000")
	oldText := strings.Repeat("old function output /srv/app/server.go context_length_exceeded stream heartbeat retry\n", 11800)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":[` +
		`{"type":"function_call_output","call_id":"call_old","output":` + mustJSONQuote(oldText) + `},` +
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"latest request"}]}` +
		`]}`)
	if len(body) < 1_000_000 || len(body) >= 1_040_000 {
		t.Fatalf("test body length = %d, want [1000000,1040000)", len(body))
	}

	prepared, err := (&openAIGatewayHandler{}).prepareGatewayContext(
		context.Background(),
		&biz.GatewayAPIKey{ID: 12, KeyPrefix: "ogw_test"},
		"req_test",
		"",
		"/v1/responses",
		body,
		"gpt-5.5",
		"high",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.Diagnostic.ContextCompacted {
		t.Fatal("expected large request to be compacted before upstream")
	}
	if prepared.Diagnostic.ContextCompactByteLimit != 1_000_000 {
		t.Fatalf("compact byte limit = %d, want 1000000", prepared.Diagnostic.ContextCompactByteLimit)
	}
	if len(prepared.Body) >= len(body) {
		t.Fatalf("prepared body length = %d, original = %d", len(prepared.Body), len(body))
	}
}

func TestEffectiveGatewayContextPolicyUsesOfficialModelRecommendation(t *testing.T) {
	t.Setenv("GATEWAY_CONTEXT_COMPACT_TOKENS", "")
	t.Setenv("GATEWAY_CONTEXT_HARD_TOKENS", "")
	t.Setenv("GATEWAY_CONTEXT_COMPACT_BYTES", "")
	t.Setenv("GATEWAY_CONTEXT_HARD_BYTES", "")
	policy := (&openAIGatewayHandler{}).effectiveGatewayContextPolicy(context.Background(), "gpt-5.3-codex")
	if policy.ContextWindowTokens != 400_000 {
		t.Fatalf("window = %d, want 400000", policy.ContextWindowTokens)
	}
	if policy.ContextCompactTokens != 260_000 || policy.ContextHardTokens != 380_000 {
		t.Fatalf("token thresholds = %d/%d, want 260000/380000", policy.ContextCompactTokens, policy.ContextHardTokens)
	}
	if policy.ContextCompactBytes != 1_000_000 || policy.ContextHardBytes != 1_900_000 {
		t.Fatalf("byte thresholds = %d/%d, want 1000000/1900000", policy.ContextCompactBytes, policy.ContextHardBytes)
	}
}

func mustJSONQuote(text string) string {
	raw, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}
	return string(raw)
}
