package server

import "testing"

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
