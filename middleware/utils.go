package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func abortWithMessage(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"message": helper.MessageWithRequestId(message, c.GetString(helper.RequestIdKey)),
			"type":    "one_api_error",
		},
	})
	c.Abort()
	logger.Error(c.Request.Context(), message)
}

func getRequestModel(c *gin.Context) (string, error) {
	path := relaymode.NormalizeAPIPath(c.Request.URL.Path)

	if c.Request.Method == http.MethodGet {
		p := strings.TrimSuffix(path, "/")
		if p == "/v1/realtime" {
			m := strings.TrimSpace(c.Query("model"))
			if m != "" {
				return m, nil
			}
			return "", fmt.Errorf("realtime websocket 需要在查询参数中指定 model")
		}
	}
	if strings.HasPrefix(path, "/v1/responses") && (c.Request.Method == http.MethodGet || c.Request.Method == http.MethodDelete) {
		if m := strings.TrimSpace(c.Query("model")); m != "" {
			return m, nil
		}
	}

	if strings.HasPrefix(path, "/v1beta/models/") {
		rest := strings.TrimPrefix(path, "/v1beta/models/")
		rest = strings.TrimPrefix(rest, "/")
		if i := strings.LastIndex(rest, ":"); i > 0 {
			modelName := rest[:i]
			if modelName != "" {
				return modelName, nil
			}
		}
		return "", fmt.Errorf("cannot parse model id from path /v1beta/models/…")
	}
	// /gemini/models/{model}:method → Normalize 后为 /models/…
	if strings.HasPrefix(path, "/models/") {
		rest := strings.TrimPrefix(path, "/models/")
		rest = strings.TrimPrefix(rest, "/")
		if i := strings.LastIndex(rest, ":"); i > 0 {
			modelName := rest[:i]
			if modelName != "" {
				return modelName, nil
			}
		}
		return "", fmt.Errorf("cannot parse model id from path /models/…")
	}

	var modelRequest ModelRequest
	err := common.UnmarshalBodyReusable(c, &modelRequest)
	if err != nil {
		return "", fmt.Errorf("common.UnmarshalBodyReusable failed: %w", err)
	}
	if strings.HasPrefix(path, "/v1/moderations") {
		if modelRequest.Model == "" {
			modelRequest.Model = "text-moderation-stable"
		}
	}
	if strings.HasSuffix(path, "embeddings") {
		if modelRequest.Model == "" {
			modelRequest.Model = c.Param("model")
		}
	}
	if strings.HasPrefix(path, "/v1/images/generations") {
		if modelRequest.Model == "" {
			modelRequest.Model = "dall-e-2"
		}
	}
	if strings.HasPrefix(path, "/v1/audio/transcriptions") || strings.HasPrefix(path, "/v1/audio/translations") {
		if modelRequest.Model == "" {
			modelRequest.Model = "whisper-1"
		}
	}
	return modelRequest.Model, nil
}

func isModelInList(modelName string, models string) bool {
	modelList := strings.Split(models, ",")
	for _, model := range modelList {
		if modelName == model {
			return true
		}
	}
	return false
}
