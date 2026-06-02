package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
)

// allChannels 所有受支持的在线支付渠道（目前仅微信扫码）。
var allChannels = []string{"wxpay"}

// globallyEnabledChannels 返回「全局已启用且配置完整」的渠道集合。
func globallyEnabledChannels() []string {
	out := make([]string, 0, len(allChannels))
	if wechatPayEnabled() {
		if _, _, err := wechatPayClient(); err == nil {
			out = append(out, "wxpay")
		}
	}
	return out
}

// GetUserPaymentChannels GET /api/user/pay/channels
// 返回当前用户「可用」的支付渠道 = 全局已启用 ∩ 被授权（默认全关）。
func GetUserPaymentChannels(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	granted, _ := model.ResolveUserAllowedPaymentChannels(userId)
	enabled := globallyEnabledChannels()
	channels := make([]string, 0, len(enabled))
	for _, ch := range enabled {
		if containsStr(granted, ch) {
			channels = append(channels, ch)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"channels":       channels,
			"quota_per_yuan": quotaPerYuan(),
			"discount":       paymentDiscount(),
		},
	})
}

// ---- 后台（超管）：支付渠道授权管理 ----

// ListPaymentAccess GET /api/payment/access —— 列出所有授权记录。
func ListPaymentAccess(c *gin.Context) {
	rows, err := model.ListPaymentChannelAccess()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items":        rows,
			"all_channels": allChannels,
			"enabled":      globallyEnabledChannels(),
		},
	})
}

type setPaymentAccessRequest struct {
	ScopeType string   `json:"scope_type"` // user | tenant
	ScopeId   int      `json:"scope_id"`
	Channels  []string `json:"channels"`
}

// SetPaymentAccess PUT /api/payment/access —— upsert 某用户/租户的授权渠道。
func SetPaymentAccess(c *gin.Context) {
	var req setPaymentAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	req.ScopeType = strings.TrimSpace(req.ScopeType)
	if req.ScopeType != model.PaymentScopeUser && req.ScopeType != model.PaymentScopeTenant {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "scope_type 必须为 user 或 tenant"})
		return
	}
	if req.ScopeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "scope_id 无效"})
		return
	}
	// 仅接受受支持的渠道。
	clean := make([]string, 0, len(req.Channels))
	for _, ch := range req.Channels {
		ch = strings.TrimSpace(ch)
		if containsStr(allChannels, ch) {
			clean = append(clean, ch)
		}
	}
	if err := model.SetPaymentChannelAccess(req.ScopeType, req.ScopeId, clean); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// DeletePaymentAccess DELETE /api/payment/access?scope_type=user&scope_id=5
func DeletePaymentAccess(c *gin.Context) {
	scopeType := strings.TrimSpace(c.Query("scope_type"))
	scopeId := 0
	if v := strings.TrimSpace(c.Query("scope_id")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			scopeId = n
		}
	}
	if scopeType == "" || scopeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数无效"})
		return
	}
	if err := model.DeletePaymentChannelAccess(scopeType, scopeId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
