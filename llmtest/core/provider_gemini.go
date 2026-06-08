package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type geminiProvider struct{ client *Client }

func (p *geminiProvider) Protocol() Protocol { return ProtocolGemini }

// Gemini 在本工具范围内仅覆盖 generateContent（对话）。
func (p *geminiProvider) Supports(k EndpointKind) bool { return k == KindChat }

func (p *geminiProvider) headers(t Target) map[string]string {
	h := map[string]string{}
	for k, v := range t.Headers {
		h[k] = v
	}
	return h
}

func (p *geminiProvider) buildContents(req ChatRequest) []map[string]any {
	var contents []map[string]any
	for _, m := range req.Messages {
		if m.Role == "system" {
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		var parts []any
		if m.Text != "" {
			parts = append(parts, map[string]any{"text": m.Text})
		}
		if m.ImageB64 != "" {
			// 注意：网关侧 Gemini proto 解析较严格，须用 camelCase 键（inlineData/mimeType）。
			parts = append(parts, map[string]any{"inlineData": map[string]any{
				"mimeType": m.ImageTyp, "data": m.ImageB64,
			}})
		}
		contents = append(contents, map[string]any{"role": role, "parts": parts})
	}
	return contents
}

func (p *geminiProvider) buildPayload(req ChatRequest) map[string]any {
	payload := map[string]any{"contents": p.buildContents(req)}
	if req.System != "" {
		payload["systemInstruction"] = map[string]any{"parts": []any{map[string]any{"text": req.System}}}
	}
	genCfg := map[string]any{"temperature": req.Temperature}
	if req.MaxTokens > 0 {
		// Gemini 2.5/3 为思考型模型，内部推理会消耗 maxOutputTokens。给一个下限，
		// 避免预算被思考耗尽导致可见文本为空（场景本身只需简短答案，放大上限不影响正确性）。
		out := req.MaxTokens
		if out < 1024 {
			out = 1024
		}
		genCfg["maxOutputTokens"] = out
	}
	if req.JSONMode {
		genCfg["responseMimeType"] = "application/json"
	}
	payload["generationConfig"] = genCfg
	if len(req.Tools) > 0 {
		decls := make([]map[string]any, 0, len(req.Tools))
		for _, tl := range req.Tools {
			decls = append(decls, map[string]any{
				"name": tl.Name, "description": tl.Description, "parameters": tl.Parameters,
			})
		}
		// camelCase：网关 proto 要求 functionDeclarations。
		payload["tools"] = []map[string]any{{"functionDeclarations": decls}}
	}
	return payload
}

// candidate 描述 Gemini 响应中单个候选的结构。
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text         string `json:"text"`
				FunctionCall *struct {
					Name string          `json:"name"`
					Args json.RawMessage `json:"args"`
				} `json:"functionCall"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func (r *geminiResponse) apply(resp *ChatResponse, textB *strings.Builder) {
	if r.UsageMetadata.TotalTokenCount > 0 || r.UsageMetadata.PromptTokenCount > 0 {
		resp.Usage = Usage{r.UsageMetadata.PromptTokenCount, r.UsageMetadata.CandidatesTokenCount, r.UsageMetadata.TotalTokenCount}
	}
	for _, cand := range r.Candidates {
		if cand.FinishReason != "" {
			resp.FinishReason = cand.FinishReason
		}
		for _, part := range cand.Content.Parts {
			textB.WriteString(part.Text)
			if part.FunctionCall != nil {
				args := part.FunctionCall.Args
				if len(args) == 0 {
					args = json.RawMessage("{}")
				}
				resp.ToolCalls = append(resp.ToolCalls, ToolCall{Name: part.FunctionCall.Name, Args: args})
			}
		}
	}
}

func (p *geminiProvider) Chat(ctx context.Context, t Target, req ChatRequest) (*ChatResponse, error) {
	base := trimBase(t.BaseURL)
	q := ""
	if t.APIKey != "" {
		q = "?key=" + url.QueryEscape(t.APIKey)
	}
	payload := p.buildPayload(req)

	if req.Stream {
		sep := "?"
		if q != "" {
			sep = "&"
		}
		streamURL := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent%s%salt=sse", base, req.Model, q, sep)
		return p.chatStream(ctx, t, streamURL, payload)
	}
	genURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent%s", base, req.Model, q)
	raw, err := p.client.postJSON(ctx, genURL, p.headers(t), payload)
	if err != nil {
		return nil, err
	}
	var parsed geminiResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w (原始: %s)", err, truncate(string(raw), 200))
	}
	resp := &ChatResponse{Raw: raw}
	var textB strings.Builder
	parsed.apply(resp, &textB)
	resp.Text = textB.String()
	return resp, nil
}

func (p *geminiProvider) chatStream(ctx context.Context, t Target, streamURL string, payload map[string]any) (*ChatResponse, error) {
	resp := &ChatResponse{}
	var textB strings.Builder
	rawLog, err := p.client.postSSE(ctx, streamURL, p.headers(t), payload, func(ev sseEvent) error {
		var chunk geminiResponse
		if err := json.Unmarshal([]byte(ev.Data), &chunk); err != nil {
			return nil
		}
		resp.Chunks++
		chunk.apply(resp, &textB)
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp.Text = textB.String()
	resp.Raw = rawLog
	return resp, nil
}

func (p *geminiProvider) Embedding(ctx context.Context, t Target, req EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("gemini 协议（本工具范围）不支持 embedding")
}
