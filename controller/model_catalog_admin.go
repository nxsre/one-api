package controller

import (
	"errors"
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
	includeExpired := c.Query("include_expired") == "true"

	rows, total, grandTotal, err := model.ListModelCatalogPaged(model.ModelCatalogListParams{
		Page:                page,
		PageSize:            pageSize,
		Search:              search,
		FilterModelID:       strings.TrimSpace(c.Query("filter_model_id")),
		FilterModelName:     strings.TrimSpace(c.Query("filter_model_name")),
		FilterProvider:      strings.TrimSpace(c.Query("filter_provider")),
		FilterProviderKey:   strings.TrimSpace(c.Query("filter_provider_key")),
		FilterFamily:        strings.TrimSpace(c.Query("filter_family")),
		FilterModalitiesIn:  strings.TrimSpace(c.Query("filter_modalities_in")),
		FilterModalitiesOut: strings.TrimSpace(c.Query("filter_modalities_out")),
		FilterCategory:      strings.TrimSpace(c.Query("filter_category")),
		SortBy:              sortBy,
		SortDesc:            sortDesc,
		IncludeExpired:      includeExpired,
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

// modelCatalogWriteBody 管理端新增/编辑可写字段（与 ModelCatalog 对齐）。
type modelCatalogWriteBody struct {
	Id              int     `json:"id"`
	ModelId         string  `json:"model_id"`
	ModelName       string  `json:"model_name"`
	OwnedBy         string  `json:"owned_by"`
	Enabled         *bool   `json:"enabled"`
	Source          string  `json:"source"`
	Notes           string  `json:"notes"`
	ProviderKey     string  `json:"provider_key"`
	ProviderDisplay string  `json:"provider_display"`
	Family          string  `json:"family"`
	NpmPackage      string  `json:"npm_package"`
	APIBase         string  `json:"api_base"`
	DocURL          string  `json:"doc_url"`
	ModalitiesIn    string  `json:"modalities_in"`
	ModalitiesOut   string  `json:"modalities_out"`
	ContextLimit    int     `json:"context_limit"`
	OutputLimit     int     `json:"output_limit"`
	CostInput       float64 `json:"cost_input"`
	CostOutput      float64 `json:"cost_output"`
	CostCacheRead   float64 `json:"cost_cache_read"`
	CostCacheWrite  float64 `json:"cost_cache_write"`
	Reasoning       bool    `json:"reasoning"`
	ToolCall        bool    `json:"tool_call"`
	TemperatureOK   bool    `json:"temperature_ok"`
	AttachmentOK    bool    `json:"attachment_ok"`
	OpenWeights     bool    `json:"open_weights"`
	KnowledgeCutoff string  `json:"knowledge_cutoff"`
	ReleaseDate     string  `json:"release_date"`
	LastUpdatedDev  string  `json:"last_updated"`
	Tags            string  `json:"tags"`
}

func trimCatalogStr(s string) string {
	return strings.TrimSpace(s)
}

func applyModelCatalogWriteBody(row *model.ModelCatalog, body *modelCatalogWriteBody, forCreate bool) error {
	if row == nil || body == nil {
		return nil
	}
	body.ModelId = trimCatalogStr(body.ModelId)
	if body.ModelId == "" {
		return errors.New("model_id 不能为空")
	}
	row.ModelId = body.ModelId
	row.ModelName = trimCatalogStr(body.ModelName)
	if row.ModelName == "" {
		row.ModelName = body.ModelId
	}
	row.OwnedBy = trimCatalogStr(body.OwnedBy)
	if body.Enabled != nil {
		row.Enabled = *body.Enabled
	}
	row.Notes = trimCatalogStr(body.Notes)
	row.ProviderKey = trimCatalogStr(body.ProviderKey)
	row.ProviderDisplay = trimCatalogStr(body.ProviderDisplay)
	row.Family = trimCatalogStr(body.Family)
	row.NpmPackage = trimCatalogStr(body.NpmPackage)
	row.APIBase = trimCatalogStr(body.APIBase)
	row.DocURL = trimCatalogStr(body.DocURL)
	row.ModalitiesIn = trimCatalogStr(body.ModalitiesIn)
	row.ModalitiesOut = trimCatalogStr(body.ModalitiesOut)
	row.ContextLimit = body.ContextLimit
	row.OutputLimit = body.OutputLimit
	row.CostInput = body.CostInput
	row.CostOutput = body.CostOutput
	row.CostCacheRead = body.CostCacheRead
	row.CostCacheWrite = body.CostCacheWrite
	row.Reasoning = body.Reasoning
	row.ToolCall = body.ToolCall
	row.TemperatureOK = body.TemperatureOK
	row.AttachmentOK = body.AttachmentOK
	row.OpenWeights = body.OpenWeights
	row.KnowledgeCutoff = trimCatalogStr(body.KnowledgeCutoff)
	row.ReleaseDate = trimCatalogStr(body.ReleaseDate)
	row.LastUpdatedDev = trimCatalogStr(body.LastUpdatedDev)
	row.Tags = trimCatalogStr(body.Tags)
	if src := trimCatalogStr(body.Source); src != "" {
		row.Source = src
	} else if forCreate {
		row.Source = "manual"
	}
	return nil
}

// PostModelCatalogAdmin POST /api/model_catalog
func PostModelCatalogAdmin(c *gin.Context) {
	var body modelCatalogWriteBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	en := true
	if body.Enabled == nil {
		body.Enabled = &en
	}
	row := model.ModelCatalog{}
	if err := applyModelCatalogWriteBody(&row, &body, true); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	tx := model.DB.Begin()
	if err := model.UpsertCatalogVersioned(tx, &row); err != nil {
		tx.Rollback()
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": row})
}

// PutModelCatalogAdmin PUT /api/model_catalog
func PutModelCatalogAdmin(c *gin.Context) {
	var body modelCatalogWriteBody
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
	enabled := existing.Enabled
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	body.Enabled = &enabled
	if err := applyModelCatalogWriteBody(existing, &body, false); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	tx := model.DB.Begin()
	if err := model.UpsertCatalogVersioned(tx, existing); err != nil {
		tx.Rollback()
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	tx.Commit()
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
