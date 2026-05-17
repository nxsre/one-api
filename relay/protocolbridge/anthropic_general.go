package protocolbridge

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/constant/role"
	"github.com/songquanpeng/one-api/relay/model"
)

// GeneralFromAnthropicRequest 将 Anthropic Messages 请求转为内部 OpenAI 通用结构（供任意 APIType 渠道转发）。
func GeneralFromAnthropicRequest(req *anthropic.Request) (*model.GeneralOpenAIRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	g := &model.GeneralOpenAIRequest{
		Model:    req.Model,
		Stream:   req.Stream,
		Messages: make([]model.Message, 0),
	}
	if req.MaxTokens > 0 {
		g.MaxTokens = req.MaxTokens
	}
	if req.Temperature != nil {
		g.Temperature = req.Temperature
	}
	if req.TopP != nil {
		g.TopP = req.TopP
	}
	if req.TopK > 0 {
		g.TopK = req.TopK
	}
	if strings.TrimSpace(req.System) != "" {
		g.Messages = append(g.Messages, model.Message{
			Role:    role.System,
			Content: req.System,
		})
	}
	for _, t := range req.Tools {
		params := map[string]any{
			"type":       t.InputSchema.Type,
			"properties": t.InputSchema.Properties,
			"required":   t.InputSchema.Required,
		}
		g.Tools = append(g.Tools, model.Tool{
			Type: "function",
			Function: model.Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	if req.ToolChoice != nil {
		g.ToolChoice = req.ToolChoice
	}
	for _, m := range req.Messages {
		oms, err := anthropicUserOrAssistantToOpenAIMessages(m)
		if err != nil {
			return nil, err
		}
		g.Messages = append(g.Messages, oms...)
	}
	return g, nil
}

func anthropicUserOrAssistantToOpenAIMessages(m anthropic.Message) ([]model.Message, error) {
	switch m.Role {
	case "system":
		var sb strings.Builder
		for _, part := range m.Content {
			if part.Type == "text" {
				sb.WriteString(part.Text)
			}
		}
		if sb.Len() == 0 {
			return nil, nil
		}
		return []model.Message{{Role: role.System, Content: sb.String()}}, nil
	case "user":
		var toolMsgs []model.Message
		var parts []model.MessageContent
		for _, part := range m.Content {
			switch part.Type {
			case "text":
				parts = append(parts, model.MessageContent{Type: model.ContentTypeText, Text: part.Text})
			case "image":
				if part.Source == nil || part.Source.Data == "" {
					continue
				}
				url := fmt.Sprintf("data:%s;base64,%s", part.Source.MediaType, part.Source.Data)
				parts = append(parts, model.MessageContent{Type: model.ContentTypeImageURL, ImageURL: &model.ImageURL{Url: url}})
			case "tool_result":
				toolMsgs = append(toolMsgs, model.Message{
					Role:       "tool",
					Content:    part.Content,
					ToolCallId: part.ToolUseId,
				})
			}
		}
		var out []model.Message
		if len(parts) > 0 {
			if len(parts) == 1 && parts[0].Type == model.ContentTypeText {
				out = append(out, model.Message{Role: "user", Content: parts[0].Text})
			} else {
				wrap := make([]map[string]any, 0, len(parts))
				for _, p := range parts {
					if p.Type == model.ContentTypeText {
						wrap = append(wrap, map[string]any{"type": "text", "text": p.Text})
					} else if p.Type == model.ContentTypeImageURL && p.ImageURL != nil {
						wrap = append(wrap, map[string]any{
							"type": "image_url",
							"image_url": map[string]any{
								"url": p.ImageURL.Url,
							},
						})
					}
				}
				raw, _ := json.Marshal(wrap)
				out = append(out, model.Message{Role: "user", Content: json.RawMessage(raw)})
			}
		}
		out = append(out, toolMsgs...)
		return out, nil
	case "assistant":
		var textB strings.Builder
		var tools []model.Tool
		for _, part := range m.Content {
			switch part.Type {
			case "text":
				textB.WriteString(part.Text)
			case "tool_use":
				args, _ := json.Marshal(part.Input)
				tools = append(tools, model.Tool{
					Id:   part.Id,
					Type: "function",
					Function: model.Function{
						Name:      part.Name,
						Arguments: string(args),
					},
				})
			}
		}
		msg := model.Message{Role: role.Assistant, Content: textB.String(), ToolCalls: tools}
		return []model.Message{msg}, nil
	default:
		return nil, fmt.Errorf("unsupported anthropic message role: %s", m.Role)
	}
}
