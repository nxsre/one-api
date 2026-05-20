package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/setting/billing_setting"
)

// GetPublicRatioConfig GET /api/ratio_config — 对外暴露倍率价目（需开启 ExposeRatioEnabled）。
func GetPublicRatioConfig(c *gin.Context) {
	if !config.IsExposeRatioEnabled() {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "ratio config is not exposed",
		})
		return
	}
	data := billing_setting.GetPricingSyncData(map[string]any(billingratio.GetExposedData()))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}
