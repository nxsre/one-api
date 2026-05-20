package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GeminiModel 与 Google Generative Language API ListModels 单项结构对齐（CoWork / @google/genai 等客户端依赖）。
type GeminiModel struct {
	Name                       string   `json:"name"`
	DisplayName                string   `json:"displayName"`
	Version                    string   `json:"version,omitempty"`
	Description                string   `json:"description,omitempty"`
	InputTokenLimit            int      `json:"inputTokenLimit,omitempty"`
	OutputTokenLimit           int      `json:"outputTokenLimit,omitempty"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods,omitempty"`
}

var defaultGeminiGenerationMethods = []string{
	"generateContent",
	"countTokens",
	"embedContent",
	"batchEmbedContents",
}

func geminiResourceName(modelID string) string {
	id := strings.TrimSpace(modelID)
	id = strings.TrimPrefix(id, "models/")
	if id == "" {
		return ""
	}
	return "models/" + id
}

func toGeminiModel(m OpenAIModels) GeminiModel {
	id := strings.TrimSpace(m.Id)
	display := id
	if strings.Contains(m.OwnedBy, "·") {
		parts := strings.SplitN(m.OwnedBy, "·", 2)
		if dn := strings.TrimSpace(parts[1]); dn != "" {
			display = dn
		}
	}
	return GeminiModel{
		Name:                       geminiResourceName(id),
		DisplayName:                display,
		Version:                    "001",
		SupportedGenerationMethods: defaultGeminiGenerationMethods,
		InputTokenLimit:            1048576,
		OutputTokenLimit:           8192,
	}
}

// ListGeminiModels handles GET /v1beta/models（Gemini 原生模型列表，供 CoWork 等客户端 Refresh Models / Test Connection）。
func ListGeminiModels(c *gin.Context) {
	available, err := collectRelayAvailableModels(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    http.StatusInternalServerError,
				"message": err.Error(),
				"status":  "INTERNAL",
			},
		})
		return
	}
	models := make([]GeminiModel, 0, len(available))
	for _, m := range available {
		if gm := toGeminiModel(m); gm.Name != "" {
			models = append(models, gm)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"models":        models,
		"nextPageToken": "",
	})
}

// RetrieveGeminiModel handles GET /v1beta/models/{model}.
func RetrieveGeminiModel(c *gin.Context) {
	modelID := strings.TrimSpace(c.Param("model"))
	resource := geminiResourceName(modelID)
	available, err := collectRelayAvailableModels(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    http.StatusInternalServerError,
				"message": err.Error(),
				"status":  "INTERNAL",
			},
		})
		return
	}
	for _, m := range available {
		if geminiResourceName(m.Id) == resource {
			c.JSON(http.StatusOK, toGeminiModel(m))
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{
			"code":    http.StatusNotFound,
			"message": "Model not found: " + modelID,
			"status":  "NOT_FOUND",
		},
	})
}
