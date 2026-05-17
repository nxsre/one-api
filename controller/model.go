package controller

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	relay "github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/apitype"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

// https://platform.openai.com/docs/api-reference/models/list

type OpenAIModelPermission struct {
	Id                 string  `json:"id"`
	Object             string  `json:"object"`
	Created            int     `json:"created"`
	AllowCreateEngine  bool    `json:"allow_create_engine"`
	AllowSampling      bool    `json:"allow_sampling"`
	AllowLogprobs      bool    `json:"allow_logprobs"`
	AllowSearchIndices bool    `json:"allow_search_indices"`
	AllowView          bool    `json:"allow_view"`
	AllowFineTuning    bool    `json:"allow_fine_tuning"`
	Organization       string  `json:"organization"`
	Group              *string `json:"group"`
	IsBlocking         bool    `json:"is_blocking"`
}

type OpenAIModels struct {
	Id         string                  `json:"id"`
	Object     string                  `json:"object"`
	Created    int                     `json:"created"`
	OwnedBy    string                  `json:"owned_by"`
	Permission []OpenAIModelPermission `json:"permission"`
	Root       string                  `json:"root"`
	Parent     *string                 `json:"parent"`
}

var models []OpenAIModels
var channelId2Models map[int][]string

func dedupeBuiltinOpenAIModelsByID(in []OpenAIModels) []OpenAIModels {
	seen := make(map[string]struct{}, len(in))
	out := make([]OpenAIModels, 0, len(in))
	for _, m := range in {
		id := strings.TrimSpace(m.Id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, m)
	}
	return out
}

func init() {
	permission := defaultModelPermissions()
	// https://platform.openai.com/docs/models/model-endpoint-compatibility
	for i := 0; i < apitype.Dummy; i++ {
		if i == apitype.AIProxyLibrary {
			continue
		}
		adaptor := relay.GetAdaptor(i)
		channelName := adaptor.GetChannelName()
		modelNames := adaptor.GetModelList()
		for _, modelName := range modelNames {
			models = append(models, OpenAIModels{
				Id:         modelName,
				Object:     "model",
				Created:    1626777600,
				OwnedBy:    channelName,
				Permission: permission,
				Root:       modelName,
				Parent:     nil,
			})
		}
	}
	for _, channelType := range openai.CompatibleChannels {
		if channelType == channeltype.Azure {
			continue
		}
		channelName, channelModelList := openai.GetCompatibleChannelMeta(channelType)
		for _, modelName := range channelModelList {
			models = append(models, OpenAIModels{
				Id:         modelName,
				Object:     "model",
				Created:    1626777600,
				OwnedBy:    channelName,
				Permission: permission,
				Root:       modelName,
				Parent:     nil,
			})
		}
	}
	models = dedupeBuiltinOpenAIModelsByID(models)
	channelId2Models = make(map[int][]string)
	for i := 1; i < channeltype.Dummy; i++ {
		if channeltype.ToAPIType(i) == apitype.AIProxyLibrary {
			continue
		}
		adaptor := relay.GetAdaptor(channeltype.ToAPIType(i))
		meta := &meta.Meta{
			ChannelType: i,
		}
		adaptor.Init(meta)
		channelId2Models[i] = adaptor.GetModelList()
	}
}

func DashboardListModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channelId2Models,
	})
}

func ListAllModels(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"object":  "list",
		"data":    mergedOpenAIModelsBestEffort(),
	})
}

func ListModels(c *gin.Context) {
	ctx := c.Request.Context()
	var availableModels []string
	if c.GetString(ctxkey.AvailableModels) != "" {
		availableModels = strings.Split(c.GetString(ctxkey.AvailableModels), ",")
	} else {
		userId := c.GetInt(ctxkey.Id)
		userGroup, _ := model.CacheGetUserGroup(userId)
		availableModels, _ = model.CacheGetGroupModels(ctx, userGroup)
	}
	modelSet := make(map[string]bool)
	for _, availableModel := range availableModels {
		modelSet[availableModel] = true
	}
	availableOpenAIModels := make([]OpenAIModels, 0)
	for _, model := range mergedOpenAIModelsBestEffort() {
		if _, ok := modelSet[model.Id]; ok {
			modelSet[model.Id] = false
			availableOpenAIModels = append(availableOpenAIModels, model)
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
	c.JSON(200, gin.H{
		"object": "list",
		"data":   availableOpenAIModels,
	})
}

func RetrieveModel(c *gin.Context) {
	modelId := c.Param("model")
	var found *OpenAIModels
	for _, m := range mergedOpenAIModelsBestEffort() {
		if m.Id == modelId {
			mm := m
			found = &mm
			break
		}
	}
	if found != nil {
		c.JSON(200, *found)
	} else {
		Error := relaymodel.Error{
			Message: fmt.Sprintf("The model '%s' does not exist", modelId),
			Type:    "invalid_request_error",
			Param:   "model",
			Code:    "model_not_found",
		}
		c.JSON(200, gin.H{
			"error": Error,
		})
	}
}

func GetUserAvailableModels(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.GetInt(ctxkey.Id)
	if model.IsAdmin(id) {
		models, err := model.ListDistinctEnabledModels()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		if len(models) == 0 {
			models = distinctSortedModelIDsFromMerged()
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data":    models,
		})
		return
	}
	userGroup, err := model.CacheGetUserGroup(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	models, err := model.CacheGetGroupModels(ctx, userGroup)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    models,
	})
	return
}

func distinctSortedModelIDsFromMerged() []string {
	mm := mergedOpenAIModelsBestEffort()
	seen := make(map[string]struct{}, len(mm))
	out := make([]string, 0, len(mm))
	for _, m := range mm {
		id := strings.TrimSpace(m.Id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
