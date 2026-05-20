package anthropic

import (
	"encoding/json"
	"testing"
)

func TestSystemPrompt_string(t *testing.T) {
	var req Request
	if err := json.Unmarshal([]byte(`{"model":"claude-3-5-sonnet-20241022","max_tokens":1,"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}],"system":"you are helpful"}`), &req); err != nil {
		t.Fatal(err)
	}
	if got := req.System.String(); got != "you are helpful" {
		t.Fatalf("system = %q", got)
	}
}

func TestSystemPrompt_blocks(t *testing.T) {
	var req Request
	body := `{"model":"claude-3-5-sonnet-20241022","max_tokens":1,"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}],"system":[{"type":"text","text":"line1"},{"type":"text","text":"line2"}]}`
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if got := req.System.String(); got != "line1\nline2" {
		t.Fatalf("system = %q", got)
	}
	out, err := json.Marshal(&req)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out) {
		t.Fatal("invalid json")
	}
	var probe struct {
		System string `json:"system"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		t.Fatal(err)
	}
	if probe.System != "line1\nline2" {
		t.Fatalf("marshaled system = %q", probe.System)
	}
}

func TestSystemPrompt_invalid(t *testing.T) {
	var req Request
	err := json.Unmarshal([]byte(`{"system":123}`), &req)
	if err == nil {
		t.Fatal("expected error")
	}
}
