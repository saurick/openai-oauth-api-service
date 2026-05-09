package server

import "testing"

func TestExtractUsageFromJSONResponses(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.4",
		"usage": {
			"input_tokens": 36,
			"output_tokens": 87,
			"total_tokens": 123,
			"input_tokens_details": {"cached_tokens": 4},
			"output_tokens_details": {"reasoning_tokens": 9}
		}
	}`)

	got := extractUsageFromJSON(body)
	if got.Model != "gpt-5.4" || got.InputTokens != 36 || got.OutputTokens != 87 || got.TotalTokens != 123 {
		t.Fatalf("unexpected usage: %+v", got)
	}
	if got.CachedTokens != 4 || got.ReasoningTokens != 9 {
		t.Fatalf("unexpected detailed usage: %+v", got)
	}
}

func TestExtractUsageFromJSONChatCompletions(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"usage": {
			"prompt_tokens": 19,
			"completion_tokens": 10,
			"total_tokens": 29,
			"prompt_tokens_details": {"cached_tokens": 3},
			"completion_tokens_details": {"reasoning_tokens": 2}
		}
	}`)

	got := extractUsageFromJSON(body)
	if got.Model != "gpt-5.5" || got.InputTokens != 19 || got.OutputTokens != 10 || got.TotalTokens != 29 {
		t.Fatalf("unexpected usage: %+v", got)
	}
	if got.CachedTokens != 3 || got.ReasoningTokens != 2 {
		t.Fatalf("unexpected detailed usage: %+v", got)
	}
}

func TestExtractUsageFromSSEResponsesCompleted(t *testing.T) {
	body := []byte("event: response.created\n" +
		"data: {\"type\":\"response.created\",\"response\":{\"model\":\"gpt-5.4\",\"usage\":null}}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"model\":\"gpt-5.4\",\"usage\":{\"input_tokens\":37,\"output_tokens\":11,\"total_tokens\":48,\"output_tokens_details\":{\"reasoning_tokens\":1}}}}\n\n")

	got := extractUsageFromSSE(body)
	if got.Model != "gpt-5.4" || got.InputTokens != 37 || got.OutputTokens != 11 || got.TotalTokens != 48 {
		t.Fatalf("unexpected usage: %+v", got)
	}
	if got.ReasoningTokens != 1 {
		t.Fatalf("unexpected reasoning tokens: %+v", got)
	}
}

func TestExtractUsageFromSSEChatChunk(t *testing.T) {
	body := []byte("data: {\"model\":\"gpt-5.5\",\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n" +
		"data: {\"model\":\"gpt-5.5\",\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":7,\"total_tokens\":12}}\n\n" +
		"data: [DONE]\n\n")

	got := extractUsageFromSSE(body)
	if got.Model != "gpt-5.5" || got.InputTokens != 5 || got.OutputTokens != 7 || got.TotalTokens != 12 {
		t.Fatalf("unexpected usage: %+v", got)
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

func TestExtractCodexCLIAnswer(t *testing.T) {
	output := []byte("OpenAI Codex\nuser\nReply\ncodex\nOK\n tokens ignored\n\ntokens used\n1,605\nOK\n")

	got := extractCodexCLIAnswer(output)
	if got != "OK" {
		t.Fatalf("unexpected answer: %q", got)
	}
}
