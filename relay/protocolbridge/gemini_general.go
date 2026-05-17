package protocolbridge

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/songquanpeng/one-api/relay/adaptor/gemini"
	"github.com/songquanpeng/one-api/relay/constant/role"
	"github.com/songquanpeng/one-api/relay/model"
)

// GeneralFromGeminiChatRequest 将 Gemini generateContent 请求体转为 OpenAI 通用结构。
func GeneralFromGeminiChatRequest(chat *gemini.ChatRequest, modelName string) (*model.GeneralOpenAIRequest, error) {
	if chat == nil {
		return nil, fmt.Errorf("nil chat request")
	}
	g := &model.GeneralOpenAIRequest{
		Model:    modelName,
		Messages: make([]model.Message, 0),
	}
	if chat.GenerationConfig.MaxOutputTokens > 0 {
		g.MaxTokens = chat.GenerationConfig.MaxOutputTokens
	}
	if chat.GenerationConfig.Temperature != nil {
		g.Temperature = chat.GenerationConfig.Temperature
	}
	if chat.GenerationConfig.TopP != nil {
		g.TopP = chat.GenerationConfig.TopP
	}
	if m := strings.TrimSpace(chat.GenerationConfig.ResponseMimeType); m != "" {
		g.ResponseFormat = &model.ResponseFormat{}
		if strings.Contains(strings.ToLower(m), "json") {
			g.ResponseFormat.Type = "json_object"
			if chat.GenerationConfig.ResponseSchema != nil {
				if sm, ok := chat.GenerationConfig.ResponseSchema.(map[string]interface{}); ok {
					g.ResponseFormat.JsonSchema = &model.JSONSchema{Name: "response", Schema: sm}
				}
			}
		}
	}

	if chat.SystemInstruction != nil && len(chat.SystemInstruction.Parts) > 0 && gemini.IsModelSupportSystemInstruction(modelName) {
		var sb strings.Builder
		for _, p := range chat.SystemInstruction.Parts {
			sb.WriteString(p.Text)
		}
		if s := sb.String(); strings.TrimSpace(s) != "" {
			g.Messages = append(g.Messages, model.Message{Role: role.System, Content: s})
		}
	} else if chat.SystemInstruction != nil && len(chat.SystemInstruction.Parts) > 0 {
		var sb strings.Builder
		for _, p := range chat.SystemInstruction.Parts {
			sb.WriteString(p.Text)
		}
		if s := sb.String(); strings.TrimSpace(s) != "" {
			g.Messages = append(g.Messages, model.Message{Role: "user", Content: s})
			g.Messages = append(g.Messages, model.Message{Role: role.Assistant, Content: "Okay"})
		}
	}

	for _, c := range chat.Contents {
		r := strings.TrimSpace(c.Role)
		if r == "model" {
			r = role.Assistant
		}
		var parts []model.MessageContent
		for _, p := range c.Parts {
			if strings.TrimSpace(p.Text) != "" {
				parts = append(parts, model.MessageContent{Type: model.ContentTypeText, Text: p.Text})
			}
			if p.InlineData != nil && p.InlineData.Data != "" {
				mime := p.InlineData.MimeType
				if mime == "" {
					mime = "image/png"
				}
				url := fmt.Sprintf("data:%s;base64,%s", mime, p.InlineData.Data)
				parts = append(parts, model.MessageContent{Type: model.ContentTypeImageURL, ImageURL: &model.ImageURL{Url: url}})
			}
		}
		if len(parts) == 0 {
			continue
		}
		if len(parts) == 1 && parts[0].Type == model.ContentTypeText {
			g.Messages = append(g.Messages, model.Message{Role: r, Content: parts[0].Text})
			continue
		}
		wrap := make([]map[string]any, 0, len(parts))
		for _, part := range parts {
			if part.Type == model.ContentTypeText {
				wrap = append(wrap, map[string]any{"type": "text", "text": part.Text})
			} else if part.Type == model.ContentTypeImageURL && part.ImageURL != nil {
				wrap = append(wrap, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": part.ImageURL.Url,
					},
				})
			}
		}
		raw, _ := json.Marshal(wrap)
		g.Messages = append(g.Messages, model.Message{Role: r, Content: json.RawMessage(raw)})
	}

	if len(chat.Tools) > 0 {
		raw, err := json.Marshal(chat.Tools[0].FunctionDeclarations)
		if err == nil {
			var funcs []model.Function
			if json.Unmarshal(raw, &funcs) == nil {
				for _, fn := range funcs {
					g.Tools = append(g.Tools, model.Tool{Type: "function", Function: fn})
				}
			}
		}
	}

	if len(g.Messages) == 0 {
		return nil, fmt.Errorf("no messages in gemini request")
	}
	return g, nil
}
