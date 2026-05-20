package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/setting/billing_setting"
)

// ListPricingEntryBlocks GET /api/pricing_entries/blocks
func ListPricingEntryBlocks(c *gin.Context) {
	summary, err := billing_setting.ListBlocksSummary()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	for i, item := range summary {
		blockID, _ := item["block_id"].(string)
		if blockID == "" {
			continue
		}
		count, err := model.CountPricingEntries(blockID)
		if err == nil {
			summary[i]["entry_count"] = count
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

// ListPricingEntries GET /api/pricing_entries
func ListPricingEntries(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	result, err := model.ListPricingEntriesPaged(model.PricingEntryListParams{
		BlockId:  c.Query("block_id"),
		Page:     page,
		PageSize: pageSize,
		Search:   c.Query("q"),
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

type pricingEntryUpsertRequest struct {
	BlockId   string `json:"block_id"`
	EntryKey  string `json:"entry_key"`
	SubKey    string `json:"sub_key"`
	ValueText string `json:"value_text"`
}

// CreatePricingEntry POST /api/pricing_entries
func CreatePricingEntry(c *gin.Context) {
	var req pricingEntryUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}
	row, err := model.UpsertPricingEntry(req.BlockId, req.EntryKey, req.SubKey, req.ValueText)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

type pricingEntryUpdateRequest struct {
	ValueText string `json:"value_text"`
}

// UpdatePricingEntry PUT /api/pricing_entries/:id
func UpdatePricingEntry(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效 id"})
		return
	}
	var req pricingEntryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}
	row, err := model.UpdatePricingEntryByID(id, req.ValueText)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

// DeletePricingEntry DELETE /api/pricing_entries/:id
func DeletePricingEntry(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效 id"})
		return
	}
	if err := model.DeletePricingEntryByID(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
