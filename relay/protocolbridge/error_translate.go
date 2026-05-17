package protocolbridge

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/relay/adaptor/gemini"
	relaymodel "github.com/songquanpeng/one-api/relay/model"

	"github.com/songquanpeng/one-api/common/conv"
)

// TranslateOpenAIErrorForNativeClient 将 OpenAI 形 {"error":{...}} 转为 Anthropic / Gemini 原生错误 JSON；无法识别则 ok=false。
func TranslateOpenAIErrorForNativeClient(openAIBody []byte, httpStatus int, clientKind string) (out []byte, contentType string, ok bool) {
	var envelope struct {
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    any    `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(openAIBody, &envelope) != nil || envelope.Error == nil {
		return nil, "", false
	}
	msg := strings.TrimSpace(envelope.Error.Message)
	if msg == "" {
		return nil, "", false
	}
	errType := strings.TrimSpace(envelope.Error.Type)
	if errType == "" {
		errType = "api_error"
	}

	switch strings.ToLower(clientKind) {
	case "anthropic":
		wrap := map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    errType,
				"message": msg,
			},
		}
		b, err := json.Marshal(wrap)
		if err != nil {
			return nil, "", false
		}
		return b, "application/json", true
	case "gemini":
		code := httpStatus
		if code == 0 {
			code = http.StatusBadRequest
		}
		statusStr := "INVALID_ARGUMENT"
		if code >= 500 {
			statusStr = "INTERNAL"
		} else if code == http.StatusUnauthorized || code == http.StatusForbidden {
			statusStr = "PERMISSION_DENIED"
		}
		ge := gemini.Error{
			Code:    code,
			Message: msg,
			Status:  statusStr,
		}
		wrap := map[string]any{"error": ge}
		b, err := json.Marshal(wrap)
		if err != nil {
			return nil, "", false
		}
		return b, "application/json", true
	default:
		return nil, "", false
	}
}

// mergeOpenAIStreamToolCalls 将流式 chunk 中的 tool_calls 增量合并进 acc（与 OpenAI chat.completion.chunk 语义一致）。
func mergeOpenAIStreamToolCalls(acc *[]relaymodel.Tool, delta []relaymodel.Tool) {
	if acc == nil || len(delta) == 0 {
		return
	}
	for _, d := range delta {
		idx := d.Index
		if idx < 0 {
			idx = len(*acc)
		}
		for len(*acc) <= idx {
			*acc = append(*acc, relaymodel.Tool{Type: "function", Function: relaymodel.Function{}})
		}
		t := &(*acc)[idx]
		if d.Id != "" {
			t.Id = d.Id
		}
		if d.Type != "" {
			t.Type = d.Type
		}
		if d.Function.Name != "" {
			t.Function.Name = d.Function.Name
		}
		if d.Function.Arguments != nil {
			piece := conv.AsString(d.Function.Arguments)
			if piece != "" {
				prev := conv.AsString(t.Function.Arguments)
				t.Function.Arguments = prev + piece
			}
		}
	}
}
