package controller

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestBuildAnthropicNativeOutBody_passthroughUnchanged(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-8","thinking":{"type":"adaptive"},"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"plan","signature":"sig"}]}],"extra_field":true}`)
	out, err := buildAnthropicNativeOutBody(body, "claude-opus-4-8", "claude-opus-4-8", "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, body) {
		t.Fatalf("expected zero-copy passthrough, got %s", out)
	}
}

func TestBuildAnthropicNativeOutBody_patchModelOnly(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-8[1m]","thinking":{"type":"adaptive"},"messages":[{"role":"user","content":"hi"}]}`)
	out, err := buildAnthropicNativeOutBody(body, "claude-opus-4-8[1m]", "claude-opus-4-8", "")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["model"]) != `"claude-opus-4-8"` {
		t.Fatalf("model not patched: %s", raw["model"])
	}
	if string(raw["thinking"]) != `{"type":"adaptive"}` {
		t.Fatalf("thinking lost: %s", raw["thinking"])
	}
}

func TestBuildAnthropicNativeOutBody_forcedSystemUsesStruct(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-8","messages":[{"role":"user","content":"hi"}]}`)
	out, err := buildAnthropicNativeOutBody(body, "claude-opus-4-8", "claude-opus-4-8", "forced")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["system"] != "forced" {
		t.Fatalf("system=%v", raw["system"])
	}
}
