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

// tool_result.content 为字符串：解析正常，String() 取到文本，重序列化保持字符串。
func TestToolResultContent_string(t *testing.T) {
	var req Request
	body := `{"model":"m","messages":[{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"done"}]}]}`
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	c := req.Messages[0].Content[0]
	if c.Content.String() != "done" {
		t.Fatalf("text: %q", c.Content.String())
	}
	out, _ := json.Marshal(c)
	if !json.Valid(out) || !contains(string(out), `"content":"done"`) {
		t.Fatalf("remarshal: %s", out)
	}
}

// tool_result.content 为内容块数组（客户端真实形态）：不再 400；抽取文本；原样回传数组结构。
func TestToolResultContent_array(t *testing.T) {
	var req Request
	body := `{"model":"m","messages":[{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":[{"type":"text","text":"ok"},{"type":"text","text":"!"}]}]}]}`
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("数组形态不应报错: %v", err)
	}
	c := req.Messages[0].Content[0]
	if c.Content.String() != "ok!" {
		t.Fatalf("扁平化文本: %q", c.Content.String())
	}
	out, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	// 数组结构应原样保留，便于转发给上游（不丢失结构）。
	if !contains(string(out), `[{"type":"text","text":"ok"}`) {
		t.Fatalf("应保留数组结构, 实际: %s", out)
	}
}

func TestThinkingContentBlock_roundTrip(t *testing.T) {
	body := `{"model":"claude-opus-4-8","max_tokens":1024,"thinking":{"type":"adaptive"},"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"plan steps","signature":"sig123"}]}]}`
	var req Request
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	blk := req.Messages[0].Content[0]
	if blk.Thinking != "plan steps" || blk.Signature != "sig123" {
		t.Fatalf("thinking block: %+v", blk)
	}
	if len(req.Thinking) == 0 {
		t.Fatal("top-level thinking config lost")
	}
	out, err := json.Marshal(&req)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !contains(s, `"thinking":"plan steps"`) || !contains(s, `"signature":"sig123"`) {
		t.Fatalf("remarshal lost thinking fields: %s", s)
	}
	if !contains(s, `"type":"adaptive"`) {
		t.Fatalf("top-level thinking config lost: %s", s)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
