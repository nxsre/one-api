package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/model"
)

// ListPaymentDiscount GET /api/payment/discount —— 列出折扣规则 + 全局默认折扣。
func ListPaymentDiscount(c *gin.Context) {
	rows, err := model.ListPaymentDiscountRules()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items":       rows,
			"global":      paymentDiscount(),
			"match_types": []string{model.DiscountMatchUser, model.DiscountMatchGroup, model.DiscountMatchTag},
		},
	})
}

type setPaymentDiscountRequest struct {
	MatchType  string  `json:"match_type"` // user | group | tag
	MatchValue string  `json:"match_value"`
	Discount   float64 `json:"discount"`
	Enabled    *bool   `json:"enabled"`
}

// SetPaymentDiscount PUT /api/payment/discount —— upsert 一条折扣规则。
func SetPaymentDiscount(c *gin.Context) {
	var req setPaymentDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	req.MatchType = strings.TrimSpace(req.MatchType)
	req.MatchValue = strings.TrimSpace(req.MatchValue)
	if req.MatchType != model.DiscountMatchUser && req.MatchType != model.DiscountMatchGroup && req.MatchType != model.DiscountMatchTag {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "match_type 必须为 user/group/tag"})
		return
	}
	if req.MatchValue == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "match_value 不能为空"})
		return
	}
	if req.Discount <= 0 || req.Discount > 1 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "discount 必须在 (0,1] 之间"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if err := model.SetPaymentDiscountRule(req.MatchType, req.MatchValue, req.Discount, enabled); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// DeletePaymentDiscount DELETE /api/payment/discount?match_type=user&match_value=alice
func DeletePaymentDiscount(c *gin.Context) {
	matchType := strings.TrimSpace(c.Query("match_type"))
	matchValue := strings.TrimSpace(c.Query("match_value"))
	if matchType == "" || matchValue == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数无效"})
		return
	}
	if err := model.DeletePaymentDiscountRule(matchType, matchValue); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
