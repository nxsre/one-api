package controller

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
)

// buildAnthropicNativeOutBody 构造发往上游的 Anthropic Messages 请求体。
// 默认原样透传原始 JSON，避免 struct 往返丢失 Claude Code 的 thinking / context_management 等扩展字段。
func buildAnthropicNativeOutBody(body []byte, originModel, mappedModel, forcedSystem string) ([]byte, error) {
	forcedSystem = strings.TrimSpace(forcedSystem)
	originModel = strings.TrimSpace(originModel)
	mappedModel = strings.TrimSpace(mappedModel)

	if forcedSystem == "" && originModel == mappedModel {
		return body, nil
	}

	if forcedSystem != "" {
		var req anthropic.Request
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		req.Model = mappedModel
		if req.System == "" {
			req.System = anthropic.SystemPrompt(forcedSystem)
		} else {
			req.System = anthropic.SystemPrompt(forcedSystem + "\n\n" + req.System.String())
		}
		return json.Marshal(&req)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	modelJSON, err := json.Marshal(mappedModel)
	if err != nil {
		return nil, err
	}
	raw["model"] = modelJSON
	return json.Marshal(raw)
}

func probeAnthropicNativeRequest(body []byte) (modelName string, err error) {
	var probe struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return "", fmt.Errorf("invalid_json: %w", err)
	}
	modelName = strings.TrimSpace(probe.Model)
	if modelName == "" {
		return "", fmt.Errorf("model is required")
	}
	return modelName, nil
}

func probeAnthropicNativeMessages(body []byte) error {
	var probe struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return fmt.Errorf("invalid_json: %w", err)
	}
	if len(probe.Messages) == 0 {
		return fmt.Errorf("messages is required")
	}
	return nil
}
