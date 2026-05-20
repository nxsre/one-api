package requestaudit

import (
	"encoding/json"
	"strings"
)

// ExtractUserInputSummary 从 Relay 请求体 JSON 提取用户可见输入（messages / system / prompt / input 等）。
func ExtractUserInputSummary(raw []byte, maxBytes int) string {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return ""
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil || len(root) == 0 {
		return truncate(string(raw), maxBytes)
	}

	var parts []string
	if s := formatSystemField(root["system"]); s != "" {
		parts = append(parts, "[system]\n"+s)
	}
	if msgs := formatMessagesField(root["messages"]); msgs != "" {
		parts = append(parts, msgs)
	}
	if p := formatScalarOrJSON(root["prompt"]); p != "" {
		parts = append(parts, "[prompt]\n"+p)
	}
	if in := formatScalarOrJSON(root["input"]); in != "" {
		parts = append(parts, "[input]\n"+in)
	}
	if inst := formatScalarOrJSON(root["instruction"]); inst != "" {
		parts = append(parts, "[instruction]\n"+inst)
	}
	if len(parts) == 0 {
		return truncate(string(raw), maxBytes)
	}
	return truncate(strings.Join(parts, "\n\n"), maxBytes)
}

func formatSystemField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return collapseWhitespace(s)
	}
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &blocks); err == nil {
		return formatContentBlocks(blocks)
	}
	return formatScalarOrJSON(raw)
}

func formatMessagesField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var msgs []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &msgs); err != nil {
		return ""
	}
	var lines []string
	for _, msg := range msgs {
		role := strings.TrimSpace(stringFromJSON(msg["role"]))
		if role == "" {
			role = "message"
		}
		body := formatMessageContent(msg["content"])
		if body == "" {
			body = formatScalarOrJSON(msg["text"])
		}
		if body == "" {
			continue
		}
		lines = append(lines, "["+role+"]\n"+body)
	}
	return strings.Join(lines, "\n\n")
}

func formatMessageContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return collapseWhitespace(s)
	}
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &blocks); err == nil {
		return formatContentBlocks(blocks)
	}
	return formatScalarOrJSON(raw)
}

func formatContentBlocks(blocks []map[string]json.RawMessage) string {
	var parts []string
	for _, b := range blocks {
		typ := strings.ToLower(strings.TrimSpace(stringFromJSON(b["type"])))
		switch typ {
		case "text", "":
			if t := collapseWhitespace(stringFromJSON(b["text"])); t != "" {
				parts = append(parts, t)
			}
		case "image", "image_url":
			parts = append(parts, "[image]")
		case "tool_use", "tool_result":
			name := stringFromJSON(b["name"])
			if name == "" {
				name = stringFromJSON(b["tool_use_id"])
			}
			if name != "" {
				parts = append(parts, "[tool:"+name+"]")
			} else {
				parts = append(parts, "[tool]")
			}
		default:
			if t := collapseWhitespace(stringFromJSON(b["text"])); t != "" {
				parts = append(parts, t)
			} else if typ != "" {
				parts = append(parts, "["+typ+"]")
			}
		}
	}
	return strings.Join(parts, "\n")
}

func formatScalarOrJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return collapseWhitespace(s)
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return collapseWhitespace(strings.Join(arr, "\n"))
	}
	compact := strings.TrimSpace(string(raw))
	if len(compact) > 2 && compact[0] == '"' && compact[len(compact)-1] == '"' {
		var unquoted string
		if err := json.Unmarshal(raw, &unquoted); err == nil {
			return collapseWhitespace(unquoted)
		}
	}
	return collapseWhitespace(compact)
}

func stringFromJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(string(raw))
}

func collapseWhitespace(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "data:image/") && len(s) > 256 {
		return "[image data omitted]"
	}
	return s
}
