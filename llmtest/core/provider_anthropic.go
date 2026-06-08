package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type anthropicProvider struct{ client *Client }

func (p *anthropicProvider) Protocol() Protocol { return ProtocolAnthropic }

// Anthropic 在本工具范围内仅覆盖 Messages（对话）。
func (p *anthropicProvider) Supports(k EndpointKind) bool { return k == KindChat }

func (p *anthropicProvider) headers(t Target) map[string]string {
	h := map[string]string{
		"x-api-key":         t.APIKey,
		"anthropic-version": "2023-06-01",
	}
	for k, v := range t.Headers {
		h[k] = v
	}
	return h
}

func (p *anthropicProvider) buildMessages(req ChatRequest) []map[string]any {
	var msgs []map[string]any
	for _, m := range req.Messages {
		role := m.Role
		if role == "system" {
			continue // system 走顶层字段
		}
		var blocks []any
		if m.Text != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": m.Text})
		}
		if m.ImageB64 != "" {
			blocks = append(blocks, map[string]any{
				"type": "image",
				"source": map[string]any{
					"type": "base64", "media_type": m.ImageTyp, "data": m.ImageB64,
				},
			})
		}
		msgs = append(msgs, map[string]any{"role": role, "content": blocks})
	}
	return msgs
}

func (p *anthropicProvider) Chat(ctx context.Context, t Target, req ChatRequest) (*ChatResponse, error) {
	url := trimBase(t.BaseURL) + "/v1/messages"
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	payload := map[string]any{
		"model":       req.Model,
		"max_tokens":  maxTokens,
		"temperature": req.Temperature,
		"messages":    p.buildMessages(req),
	}
	if req.System != "" {
		payload["system"] = req.System
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, tl := range req.Tools {
			tools = append(tools, map[string]any{
				"name": tl.Name, "description": tl.Description, "input_schema": tl.Parameters,
			})
		}
		payload["tools"] = tools
	}
	if req.Stream {
		payload["stream"] = true
		return p.chatStream(ctx, t, url, payload)
	}
	return p.chatNonStream(ctx, t, url, payload)
}

func (p *anthropicProvider) chatNonStream(ctx context.Context, t Target, url string, payload map[string]any) (*ChatResponse, error) {
	raw, err := p.client.postJSON(ctx, url, p.headers(t), payload)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w (原始: %s)", err, truncate(string(raw), 200))
	}
	resp := &ChatResponse{Raw: raw, FinishReason: parsed.StopReason, Usage: Usage{
		PromptTokens: parsed.Usage.InputTokens, CompletionTokens: parsed.Usage.OutputTokens,
		TotalTokens: parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
	}}
	var textB strings.Builder
	for _, c := range parsed.Content {
		switch c.Type {
		case "text":
			textB.WriteString(c.Text)
		case "tool_use":
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{Name: c.Name, Args: c.Input})
		}
	}
	resp.Text = textB.String()
	return resp, nil
}

func (p *anthropicProvider) chatStream(ctx context.Context, t Target, url string, payload map[string]any) (*ChatResponse, error) {
	var textB strings.Builder
	resp := &ChatResponse{}
	// 按 content block index 累积 tool_use 名称与 partial_json。
	toolName := map[int]string{}
	toolArgs := map[int]*strings.Builder{}
	var order []int

	rawLog, err := p.client.postSSE(ctx, url, p.headers(t), payload, func(ev sseEvent) error {
		var e struct {
			Type    string `json:"type"`
			Index   int    `json:"index"`
			Message struct {
				Usage struct {
					InputTokens int `json:"input_tokens"`
				} `json:"usage"`
			} `json:"message"`
			ContentBlock struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"content_block"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
				StopReason  string `json:"stop_reason"`
			} `json:"delta"`
			Usage struct {
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(ev.Data), &e); err != nil {
			return nil
		}
		switch e.Type {
		case "message_start":
			resp.Usage.PromptTokens = e.Message.Usage.InputTokens
		case "content_block_start":
			if e.ContentBlock.Type == "tool_use" {
				toolName[e.Index] = e.ContentBlock.Name
				toolArgs[e.Index] = &strings.Builder{}
				order = append(order, e.Index)
			}
		case "content_block_delta":
			resp.Chunks++
			if e.Delta.Type == "text_delta" {
				textB.WriteString(e.Delta.Text)
			} else if e.Delta.Type == "input_json_delta" {
				if b := toolArgs[e.Index]; b != nil {
					b.WriteString(e.Delta.PartialJSON)
				}
			}
		case "message_delta":
			if e.Delta.StopReason != "" {
				resp.FinishReason = e.Delta.StopReason
			}
			if e.Usage.OutputTokens > 0 {
				resp.Usage.CompletionTokens = e.Usage.OutputTokens
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp.Text = textB.String()
	resp.Raw = rawLog
	resp.Usage.TotalTokens = resp.Usage.PromptTokens + resp.Usage.CompletionTokens
	for _, idx := range order {
		args := "{}"
		if b := toolArgs[idx]; b != nil && b.Len() > 0 {
			args = b.String()
		}
		resp.ToolCalls = append(resp.ToolCalls, ToolCall{Name: toolName[idx], Args: json.RawMessage(args)})
	}
	return resp, nil
}

func (p *anthropicProvider) Embedding(ctx context.Context, t Target, req EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("anthropic 协议不支持 embedding")
}
