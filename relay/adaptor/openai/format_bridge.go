package openai

import (
	"encoding/json"
	"strings"
)

// TryNormalizeResponsesRequest 若请求体为 Chat 风格（messages）且无有效 input，则生成 Responses API 的 input 数组。
func TryNormalizeResponsesRequest(body []byte) ([]byte, bool) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, false
	}
	hasRealInput := false
	if raw := m["input"]; len(raw) > 0 {
		var probe interface{}
		if json.Unmarshal(raw, &probe) == nil && probe != nil {
			hasRealInput = true
		}
	}
	if hasRealInput {
		return nil, false
	}
	msgsRaw := m["messages"]
	if len(msgsRaw) == 0 {
		return nil, false
	}
	var msgs []map[string]interface{}
	if err := json.Unmarshal(msgsRaw, &msgs); err != nil || len(msgs) == 0 {
		return nil, false
	}
	input := make([]interface{}, 0, len(msgs))
	for _, msg := range msgs {
		role, _ := msg["role"].(string)
		if role == "" {
			role = "user"
		}
		parts := normalizeContentParts(msg["content"])
		input = append(input, map[string]interface{}{
			"role":    role,
			"content": parts,
		})
	}
	rawInp, err := json.Marshal(input)
	if err != nil {
		return nil, false
	}
	m["input"] = rawInp
	delete(m, "messages")
	out, err := json.Marshal(m)
	if err != nil {
		return nil, false
	}
	return out, true
}

func normalizeContentParts(content interface{}) []interface{} {
	switch v := content.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			v = " "
		}
		return []interface{}{map[string]interface{}{"type": "input_text", "text": v}}
	case []interface{}:
		out := make([]interface{}, 0, len(v))
		for _, part := range v {
			pm, ok := part.(map[string]interface{})
			if !ok {
				continue
			}
			t, _ := pm["type"].(string)
			if t == "text" {
				tx, _ := pm["text"].(string)
				out = append(out, map[string]interface{}{"type": "input_text", "text": tx})
				continue
			}
			if t == "image_url" {
				if iu, ok := pm["image_url"].(map[string]interface{}); ok {
					out = append(out, map[string]interface{}{
						"type": "input_image",
						"image_url": map[string]interface{}{
							"url": iu["url"],
						},
					})
					continue
				}
			}
			out = append(out, pm)
		}
		if len(out) == 0 {
			return []interface{}{map[string]interface{}{"type": "input_text", "text": ""}}
		}
		return out
	default:
		b, _ := json.Marshal(content)
		return []interface{}{map[string]interface{}{"type": "input_text", "text": string(b)}}
	}
}

// TryNormalizeRealtimeSessionRequest 为 Realtime session 创建补全常用默认字段（modalities / voice）。
func TryNormalizeRealtimeSessionRequest(body []byte) ([]byte, bool) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, false
	}
	changed := false
	if _, ok := m["modalities"]; !ok {
		m["modalities"], _ = json.Marshal([]string{"text", "audio"})
		changed = true
	}
	if _, ok := m["voice"]; !ok {
		m["voice"], _ = json.Marshal("alloy")
		changed = true
	}
	if !changed {
		return nil, false
	}
	out, err := json.Marshal(m)
	if err != nil {
		return nil, false
	}
	return out, true
}
