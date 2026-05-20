package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
)

// collectRelayAvailableModels 返回当前令牌/分组下可用的模型列表（与 GET /v1/models 一致）。
func collectRelayAvailableModels(c *gin.Context) ([]OpenAIModels, error) {
	ctx := c.Request.Context()
	userId := c.GetInt(ctxkey.Id)
	userGroup, _ := model.CacheGetUserGroup(userId)
	var availableModels []string
	var err error
	if c.GetString(ctxkey.AvailableModels) != "" {
		availableModels = strings.Split(c.GetString(ctxkey.AvailableModels), ",")
	} else {
		availableModels, err = model.CacheGetGroupModels(ctx, userGroup)
		if err != nil {
			return nil, err
		}
	}
	u, uerr := model.GetUserById(userId, false)
	if uerr == nil {
		availableModels, err = filterGroupModelsByTenantSubUser(ctx, u, userGroup, availableModels)
		if err != nil {
			return nil, err
		}
	}
	modelSet := make(map[string]bool)
	for _, availableModel := range availableModels {
		if id := strings.TrimSpace(availableModel); id != "" {
			modelSet[id] = true
		}
	}
	availableOpenAIModels := make([]OpenAIModels, 0)
	for _, m := range mergedOpenAIModelsBestEffort() {
		if modelSet[m.Id] {
			modelSet[m.Id] = false
			availableOpenAIModels = append(availableOpenAIModels, m)
		}
	}
	for modelName, ok := range modelSet {
		if ok {
			availableOpenAIModels = append(availableOpenAIModels, OpenAIModels{
				Id:         modelName,
				Object:     "model",
				Created:    1626777600,
				OwnedBy:    "custom",
				Permission: defaultModelPermissions(),
				Root:       modelName,
				Parent:     nil,
			})
		}
	}
	return availableOpenAIModels, nil
}
