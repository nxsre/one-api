package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
)

// PlatformGetTenantBillingRules GET /platform/tenants/:id/billing_rules
func PlatformGetTenantBillingRules(c *gin.Context) {
	tenantId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}

	rules, err := model.GetTenantBillingRules(tenantId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": rules})
}

// PlatformCreateTenantBillingRule POST /platform/tenants/:id/billing_rules
func PlatformCreateTenantBillingRule(c *gin.Context) {
	tenantId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}

	var req struct {
		ChannelId int     `json:"channel_id"`
		RuleType  int     `json:"rule_type"`
		Value     float64 `json:"value"`
		StartTime int64   `json:"start_time"`
		EndTime   int64   `json:"end_time"`
		Status    int     `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的参数"})
		return
	}

	rule := &model.TenantBillingRule{
		TenantId:  tenantId,
		ChannelId: req.ChannelId,
		RuleType:  req.RuleType,
		Value:     req.Value,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Status:    req.Status,
	}

	if err := model.InsertTenantBillingRule(rule); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// PlatformUpdateTenantBillingRule PUT /platform/tenants/:id/billing_rules/:rule_id
func PlatformUpdateTenantBillingRule(c *gin.Context) {
	tenantId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的租户ID"})
		return
	}

	ruleId, err := strconv.Atoi(c.Param("rule_id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的规则ID"})
		return
	}

	var req struct {
		ChannelId int     `json:"channel_id"`
		RuleType  int     `json:"rule_type"`
		Value     float64 `json:"value"`
		StartTime int64   `json:"start_time"`
		EndTime   int64   `json:"end_time"`
		Status    int     `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的参数"})
		return
	}

	rule, err := model.GetTenantBillingRuleById(ruleId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "规则不存在"})
		return
	}
	if rule.TenantId != tenantId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权操作该规则"})
		return
	}

	rule.ChannelId = req.ChannelId
	rule.RuleType = req.RuleType
	rule.Value = req.Value
	rule.StartTime = req.StartTime
	rule.EndTime = req.EndTime
	rule.Status = req.Status

	if err := model.UpdateTenantBillingRule(rule); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// PlatformDeleteTenantBillingRule DELETE /platform/tenants/:id/billing_rules/:rule_id
func PlatformDeleteTenantBillingRule(c *gin.Context) {
	tenantId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的租户ID"})
		return
	}

	ruleId, err := strconv.Atoi(c.Param("rule_id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的规则ID"})
		return
	}

	rule, err := model.GetTenantBillingRuleById(ruleId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "规则不存在"})
		return
	}
	if rule.TenantId != tenantId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权操作该规则"})
		return
	}

	if err := model.DeleteTenantBillingRule(ruleId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}