package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

// ChannelTypeOption 管理端「创建/编辑渠道」类型下拉项（字段保持精简，供各前端 Select 使用）
type ChannelTypeOption struct {
	Key             int    `json:"key"`
	Value           int    `json:"value"`
	Text            string `json:"text"`
	Color           string `json:"color,omitempty"`
	Tip             string `json:"tip,omitempty"`
	Description     string `json:"description,omitempty"`
	Protocol        string `json:"protocol"`
	TypeKind        string `json:"type_kind"`
	AutoImported    bool     `json:"auto_imported"`
	ImportSourceURL string   `json:"import_source_url,omitempty"`
	DefaultBaseURL  string   `json:"default_base_url,omitempty"`
	BuiltinModels   []string `json:"builtin_models,omitempty"` // 渠道类型内置模型（适配器 GetModelList，目录表外），供前端并入下拉/自动填入
}

// GetModelCatalogEditorOptions GET /api/model_catalog/editor_options
// channel_types 顺序与 BuiltinEditorTypes 一致；default_base_urls 为 channelDefaultBaseURL 全表（硬编码）；不再查询模型目录。
func GetModelCatalogEditorOptions(c *gin.Context) {
	builtins := channeltype.BuiltinEditorTypes()

	typeIDs := channeltype.SortedTypesInDefaultBaseURLRegistry()
	defaultBaseURLs := make(map[string]string, len(typeIDs))
	for _, id := range typeIDs {
		defaultBaseURLs[strconv.Itoa(id)] = channeltype.DefaultBaseURL(id)
	}

	opts := make([]ChannelTypeOption, 0, len(builtins))
	for _, b := range builtins {
		id := b.TypeValue
		tip := strings.TrimSpace(b.Tip)
		typeKind := strings.TrimSpace(b.TypeKind)
		if typeKind == "" {
			typeKind = model.ChannelEditorTypeKindBasic
		}
		defURL := strings.TrimSpace(channeltype.DefaultBaseURL(id))

		// 内置模型（如高德 amap、AiPPT、深知 deep-research）只在适配器代码里，不入模型目录表；
		// 一并返回，前端选中类型时并入下拉选项并在模型为空时自动填入。
		var builtinModels []string
		if ms, ok := channelId2Models[id]; ok {
			for _, m := range ms {
				if m = strings.TrimSpace(m); m != "" {
					builtinModels = append(builtinModels, m)
				}
			}
		}

		desc := tip
		if desc == "" {
			if defURL != "" {
				desc = "内置默认 API 根地址：" + defURL
			} else {
				desc = "无内置默认 API 根地址，须填写 Base URL。"
			}
		}

		opts = append(opts, ChannelTypeOption{
			Key:             id,
			Value:           id,
			Text:            b.Text,
			Tip:             tip,
			Description:     desc,
			Protocol:        b.Protocol,
			TypeKind:        typeKind,
			AutoImported:    false,
			ImportSourceURL: "",
			DefaultBaseURL:  defURL,
			BuiltinModels:   builtinModels,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"channel_types":     opts,
			"default_base_urls": defaultBaseURLs,
		},
	})
}
