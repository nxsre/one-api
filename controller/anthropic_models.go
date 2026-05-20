package controller

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
)

type anthropicCapabilitySupport struct {
	Supported bool `json:"supported"`
}

type anthropicThinkingTypes struct {
	Enabled  anthropicCapabilitySupport `json:"enabled"`
	Adaptive anthropicCapabilitySupport `json:"adaptive"`
}

type anthropicThinkingCapability struct {
	Supported bool                   `json:"supported"`
	Types     anthropicThinkingTypes `json:"types"`
}

type anthropicModelCapabilities struct {
	ImageInput anthropicCapabilitySupport  `json:"image_input"`
	Thinking   anthropicThinkingCapability `json:"thinking"`
}

type AnthropicModelInfo struct {
	ID             string                     `json:"id"`
	Type           string                     `json:"type"`
	DisplayName    string                     `json:"display_name"`
	CreatedAt      string                     `json:"created_at"`
	MaxInputTokens int                        `json:"max_input_tokens"`
	MaxTokens      int                        `json:"max_tokens"`
	Capabilities   anthropicModelCapabilities `json:"capabilities"`
}

func defaultAnthropicCapabilities(thinking bool) anthropicModelCapabilities {
	on := anthropicCapabilitySupport{Supported: true}
	off := anthropicCapabilitySupport{Supported: false}
	c := anthropicModelCapabilities{ImageInput: on}
	if thinking {
		c.Thinking = anthropicThinkingCapability{
			Supported: true,
			Types: anthropicThinkingTypes{
				Enabled:  on,
				Adaptive: on,
			},
		}
	} else {
		c.Thinking = anthropicThinkingCapability{
			Supported: false,
			Types: anthropicThinkingTypes{
				Enabled:  off,
				Adaptive: off,
			},
		}
	}
	return c
}

func toAnthropicModelInfo(m OpenAIModels) AnthropicModelInfo {
	display, maxIn, maxOut, thinking := anthropicModelMeta(m.Id)
	if strings.Contains(m.OwnedBy, "·") {
		parts := strings.SplitN(m.OwnedBy, "·", 2)
		if dn := strings.TrimSpace(parts[1]); dn != "" {
			display = dn
		}
	}
	return AnthropicModelInfo{
		ID:             m.Id,
		Type:           "model",
		DisplayName:    display,
		CreatedAt:      time.Unix(int64(m.Created), 0).UTC().Format(time.RFC3339),
		MaxInputTokens: maxIn,
		MaxTokens:      maxOut,
		Capabilities:   defaultAnthropicCapabilities(thinking),
	}
}

func expandAnthropicModelsForClaudeCode(available []OpenAIModels) []AnthropicModelInfo {
	data := make([]AnthropicModelInfo, 0, len(available)*2)
	for _, m := range available {
		data = append(data, toAnthropicModelInfo(m))
		for _, suffix := range anthropic.ClientContextVariants(m.Id) {
			v := toAnthropicModelInfo(m)
			v.ID = m.Id + suffix
			if suffix == "[1m]" {
				v.MaxInputTokens = 1000000
			}
			data = append(data, v)
		}
	}
	return data
}

func anthropicModelMeta(id string) (displayName string, maxInput, maxOutput int, thinking bool) {
	lower := strings.ToLower(id)
	switch {
	case strings.Contains(lower, "haiku"):
		return "Claude Haiku", 200000, 8192, false
	case strings.Contains(lower, "opus"):
		return "Claude Opus", 200000, 32000, true
	default:
		if strings.Contains(lower, "sonnet-4-6") || strings.Contains(lower, "4-6") {
			return "Claude Sonnet 4.6", 1000000, 64000, true
		}
		return "Claude Sonnet", 200000, 64000, true
	}
}

// ListAnthropicModels handles GET /v1/models (Anthropic Models API).
func ListAnthropicModels(c *gin.Context) {
	available, err := collectRelayAvailableModels(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"type": "api_error",
			"error": gin.H{
				"type":    "api_error",
				"message": err.Error(),
			},
		})
		return
	}
	data := expandAnthropicModelsForClaudeCode(available)
	firstID, lastID := "", ""
	if len(data) > 0 {
		firstID = data[0].ID
		lastID = data[len(data)-1].ID
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      data,
		"first_id":  firstID,
		"last_id":   lastID,
		"has_more":  false,
	})
}

// RetrieveAnthropicModel handles GET /v1/models/{model_id}.
func RetrieveAnthropicModel(c *gin.Context) {
	modelID := strings.TrimSpace(c.Param("model_id"))
	available, err := collectRelayAvailableModels(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"type": "api_error",
			"error": gin.H{
				"type":    "api_error",
				"message": err.Error(),
			},
		})
		return
	}
	for _, m := range available {
		if m.Id == modelID {
			c.JSON(http.StatusOK, toAnthropicModelInfo(m))
			return
		}
	}
	if base, variant := anthropic.SplitClientModelVariant(modelID); variant != "" {
		for _, m := range available {
			if m.Id == base {
				info := toAnthropicModelInfo(m)
				info.ID = modelID
				if variant == "[1m]" {
					info.MaxInputTokens = 1000000
				}
				c.JSON(http.StatusOK, info)
				return
			}
		}
	}
	c.JSON(http.StatusNotFound, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    "not_found_error",
			"message": "model: " + modelID,
		},
	})
}
