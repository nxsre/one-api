package anthropic

import (
	"encoding/json"
	"testing"
)

func TestMessageContents_string(t *testing.T) {
	var req Request
	err := json.Unmarshal([]byte(`{"model":"claude-sonnet-4-6","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`), &req)
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Messages) != 1 || len(req.Messages[0].Content) != 1 {
		t.Fatalf("messages: %+v", req.Messages)
	}
	if req.Messages[0].Content[0].Text != "hi" {
		t.Fatalf("text: %q", req.Messages[0].Content[0].Text)
	}
}

func TestMessageContents_blocks(t *testing.T) {
	var req Request
	body := `{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if req.Messages[0].Content[0].Text != "hello" {
		t.Fatalf("text: %q", req.Messages[0].Content[0].Text)
	}
}
