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

	compacted, err := compactGatewayContextRequest("/v1/chat/completions", body, "")
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

	compacted, err := compactGatewayContextRequest("/v1/responses", body, "previous summary")
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

func TestCompactGatewayContextSingleLargeChatMessageIncludesSummary(t *testing.T) {
	oldText := strings.Repeat("single huge prompt with /workspace/app/main.go and context_length_exceeded\n", 200)
	body := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":` + mustJSONQuote(oldText+"reply FINAL_OK") + `}]}`)

	compacted, err := compactGatewayContextRequest("/v1/chat/completions", body, "")
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

func mustJSONQuote(text string) string {
	raw, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}
	return string(raw)
}
