package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/dto"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/setting/billing_setting"
)

func applyPricingOption(optionKey, payload string) error {
	return model.UpdateOption(optionKey, payload)
}

// ListPricingVersionBlocks GET /api/ratio_sync/versions
func ListPricingVersionBlocks(c *gin.Context) {
	summary, err := billing_setting.ListBlocksSummary()
	if err != nil {
		logger.SysError("list pricing version blocks: " + err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

// ListPricingBlockVersions GET /api/ratio_sync/versions/:block_id
func ListPricingBlockVersions(c *gin.Context) {
	blockID := strings.TrimSpace(c.Param("block_id"))
	if blockID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "缺少 block_id"})
		return
	}
	versions, active, err := billing_setting.ListBlockVersionsMeta(blockID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"block_id":       blockID,
			"active_version": active,
			"versions":       versions,
		},
	})
}

// ActivatePricingVersion POST /api/ratio_sync/versions/activate
func ActivatePricingVersion(c *gin.Context) {
	var req dto.PricingVersionActivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}
	blockID := strings.TrimSpace(req.BlockID)
	if blockID == "" || req.VersionID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "block_id 与 version_id 必填"})
		return
	}
	versions, _, err := billing_setting.ListBlockVersions(blockID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	var target *billing_setting.PricingVersion
	for _, v := range versions {
		if v.ID == req.VersionID {
			target = v
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("version %d not found", req.VersionID)})
		return
	}
	if strings.TrimSpace(target.Payload) == "" {
		if restoreErr := model.RestorePricingEntriesFromVersion(blockID, req.VersionID); restoreErr != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": restoreErr.Error()})
			return
		}
		if hydrateErr := model.HydrateBlockRuntimeFromEntries(blockID); hydrateErr != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": hydrateErr.Error()})
			return
		}
		if setErr := billing_setting.SetActiveVersionOnly(blockID, req.VersionID); setErr != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": setErr.Error()})
			return
		}
	} else if err := billing_setting.ActivatePricingVersion(blockID, req.VersionID, applyPricingOption); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已切换生效版本"})
}

// ApplyPricingSync POST /api/ratio_sync/apply
func ApplyPricingSync(c *gin.Context) {
	var req dto.PricingSyncApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请至少勾选一项差异"})
		return
	}

	blockPatches := make(map[string]map[string]any)
	for _, item := range req.Items {
		modelName := strings.TrimSpace(item.Model)
		field := strings.TrimSpace(item.Field)
		if modelName == "" || field == "" {
			continue
		}
		def, ok := billing_setting.LookupBlockBySyncField(field)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": fmt.Sprintf("未知字段 %s", field)})
			return
		}
		if blockPatches[def.BlockID] == nil {
			blockPatches[def.BlockID] = make(map[string]any)
		}
		blockPatches[def.BlockID][modelName] = item.Value
	}
	if len(blockPatches) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无有效差异项"})
		return
	}

	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "上游同步"
	}
	source := strings.TrimSpace(req.Source)
	note := strings.TrimSpace(req.Note)
	activate := req.Activate

	created := make(map[string]int)
	for blockID, patches := range blockPatches {
		if err := model.ApplyPricingPatches(blockID, patches); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		versionID, err := billing_setting.CreatePricingVersionEntryBased(blockID, label, source, note, activate)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		created[blockID] = versionID
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已写入本地价目版本",
		"data": gin.H{
			"created_versions": created,
			"activated":        activate,
		},
	})
}
