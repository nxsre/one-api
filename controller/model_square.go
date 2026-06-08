package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
)

// GetUserModelSquare GET /api/user/model_square
//
// 返回当前用户「可用模型」+ 模型目录元数据（provider/family/modalities/context），供「模型广场」
// 卡片展示与按分类筛选。可用范围与 /api/user/available_models 完全一致（平台用户=分组模型；租户
// 子账号再过 AllowedModels/AllowedChannelIDs 白名单；平台管理员=全部启用模型）。
//
// 查询参数：
//   - category：模型广场分类（language|reasoning|multimodal|code|image），空=全部；
//   - keyword：模糊搜索（模型 ID / 名称 / provider / family 等）。
//
// 行为：无分类过滤时，目录里缺元数据的可用模型也会补成「裸卡片」（仅 model_id），保证广场覆盖
// 全部可用模型；带分类时仅返回目录命中项（裸模型没有分类信息）。
func GetUserModelSquare(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.GetInt(ctxkey.Id)
	ids, err := resolveUserAvailableModelIDs(ctx, id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	category := strings.TrimSpace(c.Query("category"))
	search := strings.TrimSpace(c.Query("keyword"))

	cards, err := model.GetModelSquareCards(ids, category, search)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	if category == "" {
		seen := make(map[string]struct{}, len(cards))
		for _, card := range cards {
			seen[card.ModelID] = struct{}{}
		}
		lowerSearch := strings.ToLower(search)
		for _, mid := range ids {
			if _, ok := seen[mid]; ok {
				continue
			}
			if search != "" && !strings.Contains(strings.ToLower(mid), lowerSearch) {
				continue
			}
			cards = append(cards, model.ModelSquareCard{ModelID: mid, ModelName: mid})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"items": cards, "total": len(cards)},
	})
}
