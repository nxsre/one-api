package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type openAIProvider struct{ client *Client }

func (p *openAIProvider) Protocol() Protocol { return ProtocolOpenAI }

func (p *openAIProvider) Supports(k EndpointKind) bool {
	return k == KindChat || k == KindCompletion || k == KindEmbedding
}

func (p *openAIProvider) headers(t Target) map[string]string {
	h := map[string]string{"Authorization": "Bearer " + t.APIKey}
	for k, v := range t.Headers {
		h[k] = v
	}
	return h
}

// buildMessages 构造 OpenAI messages（支持 system 与多模态图片）。
func (p *openAIProvider) buildMessages(req ChatRequest) []map[string]any {
	var msgs []map[string]any
	if req.System != "" {
		msgs = append(msgs, map[string]any{"role": "system", "content": req.System})
	}
	for _, m := range req.Messages {
		if m.ImageB64 != "" {
			parts := []any{
				map[string]any{"type": "text", "text": m.Text},
				map[string]any{"type": "image_url", "image_url": map[string]any{
					"url": fmt.Sprintf("data:%s;base64,%s", m.ImageTyp, m.ImageB64),
				}},
			}
			msgs = append(msgs, map[string]any{"role": m.Role, "content": parts})
		} else {
			msgs = append(msgs, map[string]any{"role": m.Role, "content": m.Text})
		}
	}
	return msgs
}

func (p *openAIProvider) buildTools(req ChatRequest) []map[string]any {
	if len(req.Tools) == 0 {
		return nil
	}
	tools := make([]map[string]any, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": t.Name, "description": t.Description, "parameters": t.Parameters,
			},
		})
	}
	return tools
}

func (p *openAIProvider) Chat(ctx context.Context, t Target, req ChatRequest) (*ChatResponse, error) {
	if req.Kind() == KindCompletion {
		return p.completion(ctx, t, req)
	}
	url := trimBase(t.BaseURL) + "/v1/chat/completions"
	payload := map[string]any{
		"model":    req.Model,
		"messages": p.buildMessages(req),
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	payload["temperature"] = req.Temperature
	if tools := p.buildTools(req); tools != nil {
		payload["tools"] = tools
	}
	if req.JSONMode {
		payload["response_format"] = map[string]any{"type": "json_object"}
	}

	if req.Stream {
		payload["stream"] = true
		payload["stream_options"] = map[string]any{"include_usage": true}
		return p.chatStream(ctx, t, url, payload)
	}
	return p.chatNonStream(ctx, t, url, payload)
}

func (p *openAIProvider) chatNonStream(ctx context.Context, t Target, url string, payload map[string]any) (*ChatResponse, error) {
	raw, err := p.client.postJSON(ctx, url, p.headers(t), payload)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w (原始: %s)", err, truncate(string(raw), 200))
	}
	resp := &ChatResponse{Raw: raw, Usage: Usage(parsed.Usage)}
	if len(parsed.Choices) > 0 {
		ch := parsed.Choices[0]
		resp.Text = ch.Message.Content
		resp.FinishReason = ch.FinishReason
		for _, tc := range ch.Message.ToolCalls {
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{Name: tc.Function.Name, Args: json.RawMessage(tc.Function.Arguments)})
		}
	}
	return resp, nil
}

func (p *openAIProvider) chatStream(ctx context.Context, t Target, url string, payload map[string]any) (*ChatResponse, error) {
	var textB strings.Builder
	resp := &ChatResponse{}
	toolAcc := map[int]*struct {
		name string
		args strings.Builder
	}{}

	rawLog, err := p.client.postSSE(ctx, url, p.headers(t), payload, func(ev sseEvent) error {
		if strings.TrimSpace(ev.Data) == "[DONE]" {
			return nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int `json:"index"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(ev.Data), &chunk); err != nil {
			return nil // 容忍无法解析的心跳/注释块
		}
		resp.Chunks++
		if chunk.Usage != nil {
			resp.Usage = Usage{chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens, chunk.Usage.TotalTokens}
		}
		for _, ch := range chunk.Choices {
			textB.WriteString(ch.Delta.Content)
			if ch.FinishReason != "" {
				resp.FinishReason = ch.FinishReason
			}
			for _, tc := range ch.Delta.ToolCalls {
				acc := toolAcc[tc.Index]
				if acc == nil {
					acc = &struct {
						name string
						args strings.Builder
					}{}
					toolAcc[tc.Index] = acc
				}
				if tc.Function.Name != "" {
					acc.name = tc.Function.Name
				}
				acc.args.WriteString(tc.Function.Arguments)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp.Text = textB.String()
	resp.Raw = rawLog
	for _, idx := range sortedKeys(toolAcc) {
		acc := toolAcc[idx]
		resp.ToolCalls = append(resp.ToolCalls, ToolCall{Name: acc.name, Args: json.RawMessage(acc.args.String())})
	}
	return resp, nil
}

func (p *openAIProvider) completion(ctx context.Context, t Target, req ChatRequest) (*ChatResponse, error) {
	url := trimBase(t.BaseURL) + "/v1/completions"
	payload := map[string]any{"model": req.Model, "prompt": req.Prompt, "temperature": req.Temperature}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	raw, err := p.client.postJSON(ctx, url, p.headers(t), payload)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Choices []struct {
			Text         string `json:"text"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	resp := &ChatResponse{Raw: raw, Usage: Usage(parsed.Usage)}
	if len(parsed.Choices) > 0 {
		resp.Text = parsed.Choices[0].Text
		resp.FinishReason = parsed.Choices[0].FinishReason
	}
	return resp, nil
}

func (p *openAIProvider) Embedding(ctx context.Context, t Target, req EmbeddingRequest) (*EmbeddingResponse, error) {
	url := trimBase(t.BaseURL) + "/v1/embeddings"
	payload := map[string]any{"model": req.Model, "input": req.Input}
	raw, err := p.client.postJSON(ctx, url, p.headers(t), payload)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	sort.Slice(parsed.Data, func(i, j int) bool { return parsed.Data[i].Index < parsed.Data[j].Index })
	resp := &EmbeddingResponse{Raw: raw, Usage: Usage{PromptTokens: parsed.Usage.PromptTokens, TotalTokens: parsed.Usage.TotalTokens}}
	for _, d := range parsed.Data {
		resp.Vectors = append(resp.Vectors, d.Embedding)
	}
	return resp, nil
}

func sortedKeys(m map[int]*struct {
	name string
	args strings.Builder
}) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}
