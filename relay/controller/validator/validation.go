package validator

import (
	"errors"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
	"math"
	"strings"
)

func ValidateTextRequest(textRequest *model.GeneralOpenAIRequest, relayMode int) error {
	if textRequest.MaxTokens < 0 || textRequest.MaxTokens > math.MaxInt32/2 {
		return errors.New("max_tokens is invalid")
	}
	if textRequest.Model == "" {
		return errors.New("model is required")
	}
	switch relayMode {
	case relaymode.Completions:
		if textRequest.Prompt == "" {
			return errors.New("field prompt is required")
		}
	case relaymode.ChatCompletions:
		if textRequest.Messages == nil || len(textRequest.Messages) == 0 {
			return errors.New("field messages is required")
		}
	case relaymode.Embeddings:
		if embeddingInputEmpty(textRequest.Input) {
			return errors.New("field input is required")
		}
	case relaymode.Moderations:
		if textRequest.Input == "" {
			return errors.New("field input is required")
		}
	case relaymode.Edits:
		if textRequest.Instruction == "" {
			return errors.New("field instruction is required")
		}
	case relaymode.OpenAIResponses:
	case relaymode.OpenAIRealtimeSessions:
		return nil
	}
	return nil
}

func embeddingInputEmpty(input any) bool {
	if input == nil {
		return true
	}
	switch v := input.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []string:
		for _, s := range v {
			if strings.TrimSpace(s) != "" {
				return false
			}
		}
		return len(v) == 0
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				return false
			}
		}
		return len(v) == 0
	default:
		return false
	}
}
