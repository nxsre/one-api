package requestaudit

import (
	"strings"
	"testing"
)

func TestExtractUserInputSummary_messages(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-4",
		"messages": [
			{"role": "system", "content": "You are helpful."},
			{"role": "user", "content": "Hello world"}
		]
	}`)
	got := ExtractUserInputSummary(raw, 4096)
	if !strings.Contains(got, "[user]") || !strings.Contains(got, "Hello world") {
		t.Fatalf("expected user message, got: %q", got)
	}
}

func TestExtractUserInputSummary_anthropicBlocks(t *testing.T) {
	raw := []byte(`{
		"system": "Be concise",
		"messages": [
			{"role": "user", "content": [{"type": "text", "text": "Hi there"}]}
		]
	}`)
	got := ExtractUserInputSummary(raw, 4096)
	if !strings.Contains(got, "[system]") || !strings.Contains(got, "Hi there") {
		t.Fatalf("expected system and user text, got: %q", got)
	}
}

func TestExtractUserInputSummary_prompt(t *testing.T) {
	raw := []byte(`{"model":"dall-e","prompt":"a red cat"}`)
	got := ExtractUserInputSummary(raw, 4096)
	if !strings.Contains(got, "a red cat") {
		t.Fatalf("expected prompt, got: %q", got)
	}
}
