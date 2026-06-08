package anthropic

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MessageContents 兼容 Anthropic API：content 可为字符串或内容块数组。
type MessageContents []Content

func (m *MessageContents) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*m = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*m = MessageContents{{Type: "text", Text: s}}
		return nil
	}
	var blocks []Content
	if err := json.Unmarshal(data, &blocks); err == nil {
		*m = MessageContents(blocks)
		return nil
	} else if data[0] == '[' {
		// 是数组但其中某个内容块解析失败：直接回传底层错误，定位是哪种 block 非法。
		return fmt.Errorf("message content array has an invalid block: %w", err)
	}
	// 既不是 string 也不是数组。附带实际类型与截断片段，便于定位是哪条消息的 content 非法。
	snippet := data
	if len(snippet) > 256 {
		snippet = snippet[:256]
	}
	return fmt.Errorf("message content must be a string or array of content blocks (got %s: %s)", jsonKind(data), snippet)
}

// jsonKind 粗略判断一段 JSON 的类型，仅用于错误提示。
func jsonKind(data []byte) string {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return "object"
		case '[':
			return "array"
		case '"':
			return "string"
		case 't', 'f':
			return "bool"
		case 'n':
			return "null"
		default:
			return "number"
		}
	}
	return "empty"
}

func (m MessageContents) MarshalJSON() ([]byte, error) {
	return json.Marshal([]Content(m))
}

// ToolResultContent 兼容 Anthropic tool_result.content：官方允许为「字符串」或「内容块数组」
// （text/image 等）。解析时保留原始 JSON 以便原样转发给上游，同时抽取纯文本供计费/协议桥接使用。
type ToolResultContent struct {
	raw  json.RawMessage // 原始 JSON（字符串或数组），用于原样回传，不丢失结构（如数组里的 image 块）
	text string          // 扁平化后的纯文本
}

// NewToolResultText 用纯字符串构造（OpenAI tool 消息 -> Anthropic tool_result 时使用）。
func NewToolResultText(s string) *ToolResultContent {
	raw, _ := json.Marshal(s)
	return &ToolResultContent{raw: raw, text: s}
}

// String 返回扁平化文本；nil 安全。
func (t *ToolResultContent) String() string {
	if t == nil {
		return ""
	}
	return t.text
}

func (t *ToolResultContent) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	t.raw = append(t.raw[:0], data...)
	// 字符串形式
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t.text = s
		return nil
	}
	// 内容块数组形式：抽取其中的 text 拼接为纯文本（用于计费/桥接），原始结构保留在 raw 中。
	var blocks []Content
	if err := json.Unmarshal(data, &blocks); err == nil {
		var b strings.Builder
		for _, blk := range blocks {
			b.WriteString(blk.Text)
		}
		t.text = b.String()
		return nil
	}
	// 其它类型（对象等）：保留 raw，不报错，避免因 tool_result 形态多样而整条请求 400。
	return nil
}

func (t ToolResultContent) MarshalJSON() ([]byte, error) {
	if len(t.raw) == 0 {
		return []byte("null"), nil
	}
	return t.raw, nil
}
