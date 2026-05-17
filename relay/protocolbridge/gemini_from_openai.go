package protocolbridge

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/conv"
	"github.com/songquanpeng/one-api/relay/adaptor/gemini"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

// GeminiJSONFromOpenAIChatCompletion 将 OpenAI chat.completion JSON 转为 Gemini ChatResponse JSON。
func GeminiJSONFromOpenAIChatCompletion(openAIBody []byte, displayModel string) ([]byte, *relaymodel.Usage, error) {
	_ = displayModel
	var probe map[string]json.RawMessage
	if json.Unmarshal(openAIBody, &probe) == nil {
		if _, ok := probe["error"]; ok && probe["choices"] == nil {
			return nil, nil, fmt.Errorf("openai error payload")
		}
	}
	var tr openai.TextResponse
	if err := json.Unmarshal(openAIBody, &tr); err != nil {
		return nil, nil, err
	}
	if len(tr.Choices) == 0 {
		return nil, nil, fmt.Errorf("no choices")
	}
	msg := tr.Choices[0].Message
	text := strings.TrimSpace(msg.StringContent())
	var parts []gemini.Part
	if text != "" {
		parts = append(parts, gemini.Part{Text: text})
	}
	for _, tc := range msg.ToolCalls {
		var args any
		rawArgs := conv.AsString(tc.Function.Arguments)
		if rawArgs != "" {
			_ = json.Unmarshal([]byte(rawArgs), &args)
		}
		parts = append(parts, gemini.Part{
			FunctionCall: &gemini.FunctionCall{
				FunctionName: tc.Function.Name,
				Arguments:    args,
			},
		})
	}
	if len(parts) == 0 {
		parts = []gemini.Part{{Text: ""}}
	}
	gr := gemini.ChatResponse{
		Candidates: []gemini.ChatCandidate{{
			Content: gemini.ChatContent{
				Role:  "model",
				Parts: parts,
			},
			FinishReason: tr.Choices[0].FinishReason,
		}},
	}
	raw, err := json.Marshal(gr)
	if err != nil {
		return nil, nil, err
	}
	u := &relaymodel.Usage{
		PromptTokens:     tr.Usage.PromptTokens,
		CompletionTokens: tr.Usage.CompletionTokens,
		TotalTokens:      tr.Usage.TotalTokens,
	}
	return raw, u, nil
}

// WriteGeminiSSEFromOpenAISSE 将 OpenAI chunk SSE 转为 Gemini streamGenerateContent 风格的 SSE 行。
func WriteGeminiSSEFromOpenAISSE(c *gin.Context, openAISSE []byte, modelName string) (*relaymodel.Usage, error) {
	common.SetEventStreamHeaders(c)
	c.Writer.WriteHeader(http.StatusOK)
	var inputTok, outputTok int
	var totalText strings.Builder
	var accTools []relaymodel.Tool
	toolsEmitted := false

	writeData := func(obj any) {
		b, _ := json.Marshal(obj)
		_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
		if f, ok := c.Writer.(interface{ Flush() }); ok {
			f.Flush()
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader(openAISSE))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk openai.ChatCompletionsStreamResponse
		if json.Unmarshal([]byte(payload), &chunk) != nil {
			continue
		}
		if chunk.Usage != nil {
			if chunk.Usage.PromptTokens > 0 {
				inputTok = chunk.Usage.PromptTokens
			}
			if chunk.Usage.CompletionTokens > 0 {
				outputTok = chunk.Usage.CompletionTokens
			}
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		ch := chunk.Choices[0]
		mergeOpenAIStreamToolCalls(&accTools, ch.Delta.ToolCalls)

		piece := conv.AsString(ch.Delta.Content)
		if piece != "" {
			totalText.WriteString(piece)
			gr := gemini.ChatResponse{
				Candidates: []gemini.ChatCandidate{{
					Content: gemini.ChatContent{
						Role:  "model",
						Parts: []gemini.Part{{Text: piece}},
					},
				}},
			}
			writeData(gr)
		}

		if ch.FinishReason != nil {
			fr := *ch.FinishReason
			if !toolsEmitted && len(accTools) > 0 && (fr == "tool_calls" || fr == "stop") {
				for _, tc := range accTools {
					if tc.Function.Name == "" {
						continue
					}
					var args any
					rawArgs := conv.AsString(tc.Function.Arguments)
					if rawArgs != "" {
						_ = json.Unmarshal([]byte(rawArgs), &args)
					}
					writeData(gemini.ChatResponse{
						Candidates: []gemini.ChatCandidate{{
							Content: gemini.ChatContent{
								Role: "model",
								Parts: []gemini.Part{{
									FunctionCall: &gemini.FunctionCall{
										FunctionName: tc.Function.Name,
										Arguments:    args,
									},
								}},
							},
							FinishReason: fr,
						}},
					})
				}
				toolsEmitted = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	u := &relaymodel.Usage{
		PromptTokens:     inputTok,
		CompletionTokens: outputTok,
		TotalTokens:      inputTok + outputTok,
	}
	if u.TotalTokens == 0 && totalText.Len() > 0 {
		u.CompletionTokens = openai.CountTokenText(totalText.String(), modelName)
	}
	return u, nil
}
