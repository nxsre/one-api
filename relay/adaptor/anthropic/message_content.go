package anthropic

import (
	"encoding/json"
	"fmt"
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
	}
	return fmt.Errorf("message content must be a string or array of content blocks")
}

func (m MessageContents) MarshalJSON() ([]byte, error) {
	return json.Marshal([]Content(m))
}
