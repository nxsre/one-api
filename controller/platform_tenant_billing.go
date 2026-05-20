package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
)

// PlatformUpdateTenantBilling PUT /platform/tenants/:id/billing
func PlatformUpdateTenantBilling(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}
	var req struct {
		DiscountRatio     float64 `json:"discount_ratio"`
		PricePer1kApiCall float64 `json:"price_per_1k_api_call"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的参数"})
		return
	}

	t, err := model.GetTenantByID(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "租户不存在"})
		return
	}

	t.DiscountRatio = req.DiscountRatio
	t.PricePer1kApiCall = req.PricePer1kApiCall

	if err := model.DB.Save(t).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}