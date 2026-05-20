package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/model"
)

// GetModelCatalogProviders GET /api/model_catalog/providers?q=&limit=
func GetModelCatalogProviders(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	rows, err := model.ListModelCatalogProviders(strings.TrimSpace(c.Query("q")), limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": rows})
}

// GetModelCatalogModelIDs GET /api/model_catalog/model_ids
// 查询参数：filter_provider_key（精确 provider_key）、filter_provider（模糊）、filter_family 等；仅返回 status=current 且 enabled=true 的 model_id。
func GetModelCatalogModelIDs(c *gin.Context) {
	ids, err := model.ListModelCatalogEnabledModelIDs(model.ModelCatalogListParams{
		FilterModelID:       strings.TrimSpace(c.Query("filter_model_id")),
		FilterModelName:     strings.TrimSpace(c.Query("filter_model_name")),
		FilterProviderKey:   strings.TrimSpace(c.Query("filter_provider_key")),
		FilterProvider:      strings.TrimSpace(c.Query("filter_provider")),
		FilterFamily:        strings.TrimSpace(c.Query("filter_family")),
		FilterModalitiesIn:  strings.TrimSpace(c.Query("filter_modalities_in")),
		FilterModalitiesOut: strings.TrimSpace(c.Query("filter_modalities_out")),
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"model_ids": ids,
			"total":     len(ids),
		},
	})
}
