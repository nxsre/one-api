package controller

import (
	"context"
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

func intersectSortedModelSlices(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, x := range b {
		set[x] = struct{}{}
	}
	out := make([]string, 0)
	for _, x := range a {
		if _, ok := set[x]; ok {
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

func filterGroupModelsByTenantSubUser(ctx context.Context, u *model.User, userGroup string, models []string) ([]string, error) {
	if u.TenantID == nil || u.Role != model.RoleCommonUser {
		return models, nil
	}
	tid := *u.TenantID
	chIDs := dedupePositiveIntsForController(u.AllowedChannelIDs)
	out := models
	if len(chIDs) > 0 {
		chModels, err := model.GetDistinctModelsForGroupTenantChannelIDs(userGroup, tid, chIDs)
		if err != nil {
			return nil, err
		}
		out = intersectSortedModelSlices(out, chModels)
	}
	if len(u.AllowedModels) > 0 {
		allow := make(map[string]struct{})
		for _, m := range u.AllowedModels {
			t := strings.TrimSpace(m)
			if t != "" {
				allow[t] = struct{}{}
			}
		}
		if len(allow) > 0 {
			filtered := make([]string, 0, len(out))
			for _, m := range out {
				if _, ok := allow[m]; ok {
					filtered = append(filtered, m)
				}
			}
			out = filtered
		}
	}
	return out, nil
}

func dedupePositiveIntsForController(ids []int) []int {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int]struct{})
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
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
	availableOpenAIModels, err := collectRelayAvailableModels(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"object":  "error",
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"object": "list",
		"data":   availableOpenAIModels,
	})
}

func RetrieveModel(c *gin.Context) {
	ctx := c.Request.Context()
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
		userId := c.GetInt(ctxkey.Id)
		userGroup, _ := model.CacheGetUserGroup(userId)
		u, uerr := model.GetUserById(userId, false)
		if uerr == nil {
			allowed, ferr := filterGroupModelsByTenantSubUser(ctx, u, userGroup, []string{modelId})
			if ferr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": relaymodel.Error{
						Message: ferr.Error(),
						Type:    "api_error",
					},
				})
				return
			}
			if len(allowed) == 0 {
				found = nil
			}
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

// resolveUserAvailableModelIDs 解析某用户「可用模型 ID 列表」，与 GetUserAvailableModels /
// 令牌 relay 入口完全一致：平台管理员=全部启用模型；否则=分组模型，租户子账号再过
// AllowedModels/AllowedChannelIDs 白名单。供 /available_models 与 /model_square 复用。
func resolveUserAvailableModelIDs(ctx context.Context, id int) ([]string, error) {
	if model.IsAdmin(id) {
		models, err := model.ListDistinctEnabledModels()
		if err != nil {
			return nil, err
		}
		if len(models) == 0 {
			models = distinctSortedModelIDsFromMerged()
		}
		return models, nil
	}
	userGroup, err := model.CacheGetUserGroup(id)
	if err != nil {
		return nil, err
	}
	models, err := model.CacheGetGroupModels(ctx, userGroup)
	if err != nil {
		return nil, err
	}
	if u, uerr := model.GetUserById(id, false); uerr == nil {
		models, err = filterGroupModelsByTenantSubUser(ctx, u, userGroup, models)
		if err != nil {
			return nil, err
		}
	}
	return models, nil
}

func GetUserAvailableModels(c *gin.Context) {
	id := c.GetInt(ctxkey.Id)
	models, err := resolveUserAvailableModelIDs(c.Request.Context(), id)
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
}

// GetUserAvailableModelsDetail GET /user/available_models_detail
// 返回模型与可提供该模型的渠道列表，供令牌「模型范围」下拉展示与按渠道批量勾选。
func GetUserAvailableModelsDetail(c *gin.Context) {
	id := c.GetInt(ctxkey.Id)
	if model.IsAdmin(id) {
		rows, err := model.QueryAvailableModelChannelRowsAdmin()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		rows, err = model.ApplyClientFacingModelNames(rows)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"items": rows}})
		return
	}
	userGroup, err := model.CacheGetUserGroup(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if strings.TrimSpace(userGroup) == "" {
		userGroup = "default"
	}
	ut := 0
	u, uerr := model.GetUserById(id, false)
	if uerr == nil && u.Role == model.RoleCommonUser && u.TenantID != nil {
		ut = *u.TenantID
	}
	rows, err := model.QueryAvailableModelChannelRowsScoped(userGroup, ut)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if uerr == nil {
		rows = model.FilterModelChannelRowsForTenantSubUser(u, rows)
	}
	rows, err = model.ApplyClientFacingModelNames(rows)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"items": rows}})
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
