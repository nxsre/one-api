package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/model"
)

// GetModelCatalogAdmin GET /api/model_catalog
// 查询参数：page（默认 1）、page_size（默认 20，最大 100）、search（空格分词 AND）、sort、order（asc|desc）。
// 响应 data：{ items, total, grand_total, page, page_size }；若不带分页参数仍兼容旧逻辑时可不传——此处始终返回分页结构。
func GetModelCatalogAdmin(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	search := strings.TrimSpace(c.Query("search"))
	sortBy := strings.TrimSpace(c.Query("sort"))
	order := strings.TrimSpace(strings.ToLower(c.DefaultQuery("order", "asc")))
	sortDesc := order == "desc"

	rows, total, grandTotal, err := model.ListModelCatalogPaged(model.ModelCatalogListParams{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
		SortBy:   sortBy,
		SortDesc: sortDesc,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items":       rows,
			"total":       total,
			"grand_total": grandTotal,
			"page":        page,
			"page_size":   pageSize,
		},
	})
}

type modelCatalogCreate struct {
	ModelId string `json:"model_id"`
	OwnedBy string `json:"owned_by"`
	Enabled *bool  `json:"enabled"`
	Notes   string `json:"notes"`
}

// PostModelCatalogAdmin POST /api/model_catalog
func PostModelCatalogAdmin(c *gin.Context) {
	var body modelCatalogCreate
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	body.ModelId = strings.TrimSpace(body.ModelId)
	if body.ModelId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "model_id 不能为空"})
		return
	}
	en := true
	if body.Enabled != nil {
		en = *body.Enabled
	}
	row := model.ModelCatalog{
		ModelId: body.ModelId,
		OwnedBy: strings.TrimSpace(body.OwnedBy),
		Enabled: en,
		Source:  "manual",
		Notes:   strings.TrimSpace(body.Notes),
	}
	if err := row.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": row})
}

type modelCatalogUpdate struct {
	Id      int    `json:"id"`
	ModelId string `json:"model_id"`
	OwnedBy string `json:"owned_by"`
	Enabled bool   `json:"enabled"`
	Source  string `json:"source"`
	Notes   string `json:"notes"`
}

// PutModelCatalogAdmin PUT /api/model_catalog
func PutModelCatalogAdmin(c *gin.Context) {
	var body modelCatalogUpdate
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if body.Id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效 id"})
		return
	}
	existing, err := model.GetModelCatalogByPk(body.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	body.ModelId = strings.TrimSpace(body.ModelId)
	if body.ModelId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "model_id 不能为空"})
		return
	}
	existing.ModelId = body.ModelId
	existing.OwnedBy = strings.TrimSpace(body.OwnedBy)
	existing.Enabled = body.Enabled
	existing.Notes = strings.TrimSpace(body.Notes)
	if strings.TrimSpace(body.Source) != "" {
		existing.Source = strings.TrimSpace(body.Source)
	}
	if err := existing.SaveUpdates(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": existing})
}

// DeleteModelCatalogAdmin DELETE /api/model_catalog/:id
func DeleteModelCatalogAdmin(c *gin.Context) {
	id, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效 id"})
		return
	}
	if err := model.DeleteModelCatalogByPk(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
