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
	"github.com/songquanpeng/one-api/common/random"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func stopReasonOpenAI2Claude(fr string) *string {
	if fr == "" {
		return nil
	}
	var s string
	switch fr {
	case "stop":
		s = "end_turn"
	case "length":
		s = "max_tokens"
	case "tool_calls":
		s = "tool_use"
	default:
		s = fr
	}
	return &s
}

// AnthropicJSONFromOpenAIChatCompletion 将 OpenAI chat.completion JSON 转为 Anthropic message 响应 JSON。
func AnthropicJSONFromOpenAIChatCompletion(openAIBody []byte, displayModel string) ([]byte, *relaymodel.Usage, error) {
	var probe map[string]json.RawMessage
	if json.Unmarshal(openAIBody, &probe) == nil {
		if _, ok := probe["error"]; ok {
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
	var blocks []anthropic.Content
	if txt := strings.TrimSpace(msg.StringContent()); txt != "" {
		blocks = append(blocks, anthropic.Content{Type: "text", Text: txt})
	}
	for _, tc := range msg.ToolCalls {
		var input map[string]any
		_ = json.Unmarshal([]byte(conv.AsString(tc.Function.Arguments)), &input)
		blocks = append(blocks, anthropic.Content{
			Type:  "tool_use",
			Id:    tc.Id,
			Name:  tc.Function.Name,
			Input: input,
		})
	}
	out := anthropic.Response{
		Id:         strings.TrimPrefix(tr.Id, "chatcmpl-"),
		Type:       "message",
		Role:       "assistant",
		Content:    blocks,
		Model:      displayModel,
		StopReason: stopReasonOpenAI2Claude(tr.Choices[0].FinishReason),
		Usage: anthropic.Usage{
			InputTokens:  tr.Usage.PromptTokens,
			OutputTokens: tr.Usage.CompletionTokens,
		},
	}
	if out.Id == "" {
		out.Id = "msg_" + random.GetUUID()
	}
	raw, err := json.Marshal(out)
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

// WriteAnthropicSSEFromOpenAISSE 将 OpenAI chat.completion.chunk SSE 转为 Anthropic Messages SSE 写给客户端。
func WriteAnthropicSSEFromOpenAISSE(c *gin.Context, openAISSE []byte, displayModel string) (*relaymodel.Usage, error) {
	common.SetEventStreamHeaders(c)
	c.Writer.WriteHeader(http.StatusOK)
	msgID := "msg_" + random.GetUUID()
	var inputTok, outputTok int
	var finish *string
	var accTools []relaymodel.Tool
	textBlockClosed := false
	nextAnthropicIdx := 1
	openAIToolToAnthropic := map[int]int{}
	toolHeadEmitted := map[int]bool{}
	lastArgLen := map[int]int{}
	var totalText strings.Builder

	write := func(obj any) {
		b, _ := json.Marshal(obj)
		_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
		if f, ok := c.Writer.(interface{ Flush() }); ok {
			f.Flush()
		}
	}

	write(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":      msgID,
			"type":    "message",
			"role":    "assistant",
			"content": []any{},
			"model":   displayModel,
			"usage": map[string]int{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	})
	write(map[string]any{
		"type":           "content_block_start",
		"index":          0,
		"content_block": map[string]any{"type": "text", "text": ""},
	})

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
		if ch.FinishReason != nil && *ch.FinishReason != "" {
			finish = stopReasonOpenAI2Claude(*ch.FinishReason)
		}

		mergeOpenAIStreamToolCalls(&accTools, ch.Delta.ToolCalls)

		piece := conv.AsString(ch.Delta.Content)
		if piece != "" {
			totalText.WriteString(piece)
			write(map[string]any{
				"type":  "content_block_delta",
				"index": 0,
				"delta": map[string]string{"type": "text_delta", "text": piece},
			})
		}

		for i := 0; i < len(accTools); i++ {
			tc := accTools[i]
			if tc.Id != "" && tc.Function.Name != "" && !toolHeadEmitted[i] {
				if !textBlockClosed {
					write(map[string]any{"type": "content_block_stop", "index": 0})
					textBlockClosed = true
				}
				abi := nextAnthropicIdx
				nextAnthropicIdx++
				openAIToolToAnthropic[i] = abi
				toolHeadEmitted[i] = true
				lastArgLen[i] = 0
				write(map[string]any{
					"type":  "content_block_start",
					"index": abi,
					"content_block": map[string]any{
						"type":  "tool_use",
						"id":    tc.Id,
						"name":  tc.Function.Name,
						"input": map[string]any{},
					},
				})
			}
			if toolHeadEmitted[i] {
				s := conv.AsString(tc.Function.Arguments)
				if len(s) > lastArgLen[i] {
					frag := s[lastArgLen[i]:]
					write(map[string]any{
						"type":  "content_block_delta",
						"index": openAIToolToAnthropic[i],
						"delta": map[string]string{
							"type":         "input_json_delta",
							"partial_json": frag,
						},
					})
					lastArgLen[i] = len(s)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if !textBlockClosed {
		write(map[string]any{"type": "content_block_stop", "index": 0})
		textBlockClosed = true
	}
	for i := 0; i < len(accTools); i++ {
		if !toolHeadEmitted[i] {
			continue
		}
		write(map[string]any{
			"type":  "content_block_stop",
			"index": openAIToolToAnthropic[i],
		})
	}

	if finish == nil {
		if len(accTools) > 0 {
			x := "tool_use"
			finish = &x
		} else {
			x := "end_turn"
			finish = &x
		}
	}
	write(map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":     finish,
			"stop_sequence": nil,
		},
		"usage": map[string]int{
			"input_tokens":  inputTok,
			"output_tokens": outputTok,
		},
	})
	write(map[string]any{"type": "message_stop"})

	u := &relaymodel.Usage{
		PromptTokens:     inputTok,
		CompletionTokens: outputTok,
		TotalTokens:      inputTok + outputTok,
	}
	if u.TotalTokens == 0 && totalText.Len() > 0 {
		u.CompletionTokens = openai.CountTokenText(totalText.String(), displayModel)
	}
	return u, nil
}
