package controller

import (
	"strings"
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
)

func TestRewriteAnthropicResponseModelBody(t *testing.T) {
	body := []byte(`{"model":"pa/claude-opus-4-6","id":"abc","type":"message","role":"assistant","content":[],"usage":{"input_tokens":1,"output_tokens":2}}`)
	got := rewriteAnthropicResponseModelBody(body, "claude-opus-4-6")
	if !strings.Contains(string(got), `"model":"claude-opus-4-6"`) || !strings.Contains(string(got), `"id":"abc"`) {
		t.Fatalf("unexpected rewrite: %s", got)
	}
	unchanged := rewriteAnthropicResponseModelBody(got, "claude-opus-4-6")
	if string(unchanged) != string(got) {
		t.Fatalf("expected no second rewrite, got %s", unchanged)
	}
}

func TestRewriteAnthropicStreamLine(t *testing.T) {
	line := `data: {"type":"message_start","message":{"model":"pa/claude-opus-4-6","usage":{"input_tokens":1,"output_tokens":0}}}`
	got := rewriteAnthropicStreamLine(line, "claude-opus-4-6")
	if !strings.Contains(got, `"model":"claude-opus-4-6"`) || !strings.Contains(got, `"type":"message_start"`) {
		t.Fatalf("unexpected stream rewrite: %s", got)
	}
	if rewriteAnthropicStreamLine(`event: ping`, "claude-opus-4-6") != `event: ping` {
		t.Fatal("non-data line should be unchanged")
	}
}

func TestChannelUsesAnthropicNativePassthrough(t *testing.T) {
	m := &meta.Meta{ChannelType: channeltype.Gemini, BaseURL: "https://api.novita.ai/anthropic"}
	if !channelUsesAnthropicNativePassthrough(m) {
		t.Fatal("expected anthropic URL to enable native passthrough")
	}
	m2 := &meta.Meta{ChannelType: channeltype.Anthropic, BaseURL: ""}
	if !channelUsesAnthropicNativePassthrough(m2) {
		t.Fatal("expected Anthropic type")
	}
	m3 := &meta.Meta{ChannelType: channeltype.OpenAI, BaseURL: "https://api.openai.com/v1"}
	if channelUsesAnthropicNativePassthrough(m3) {
		t.Fatal("openai should not passthrough")
	}
}
