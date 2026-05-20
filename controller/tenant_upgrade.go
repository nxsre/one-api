package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/model"
)

func ApplyTenantUpgrade(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	var body struct {
		Name   string `json:"name"`
		Slug   string `json:"slug"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Slug = strings.TrimSpace(body.Slug)
	if body.Name == "" || body.Slug == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "名称和Slug不能为空"})
		return
	}
	if _, err := model.GetTenantBySlug(body.Slug); err == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Slug 已存在，请更换一个"})
		return
	}

	req := &model.TenantUpgradeRequest{
		UserId: userId,
		Name:   body.Name,
		Slug:   body.Slug,
		Remark: strings.TrimSpace(body.Remark),
		Status: model.TenantUpgradeStatusPending,
	}

	if err := model.InsertTenantUpgradeRequest(req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func PlatformListTenantUpgrades(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	requests, err := model.GetTenantUpgradeRequests(p*100, 100)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": requests})
}

func PlatformApproveTenantUpgrade(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}

	req, err := model.GetTenantUpgradeRequestById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请求不存在"})
		return
	}

	if req.Status != model.TenantUpgradeStatusPending {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该请求已处理"})
		return
	}

	if _, err := model.GetTenantBySlug(req.Slug); err == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Slug 已被使用"})
		return
	}

	user, err := model.GetUserById(req.UserId, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return
	}

	t := &model.Tenant{
		Name:        req.Name,
		Slug:        req.Slug,
		Remark:      req.Remark,
		Status:      model.TenantStatusEnabled,
		CreatedTime: helper.GetTimestamp(),
	}

	if err := model.InsertTenant(t); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	user.Role = model.RoleTenantAdmin
	user.TenantID = &t.Id
	if err := model.DB.Save(user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "修改用户角色失败: " + err.Error()})
		return
	}

	if err := model.UpdateTenantUpgradeRequestStatus(id, model.TenantUpgradeStatusApproved); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新请求状态失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "审批通过并已创建租户"})
}

func PlatformRejectTenantUpgrade(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}

	req, err := model.GetTenantUpgradeRequestById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请求不存在"})
		return
	}

	if req.Status != model.TenantUpgradeStatusPending {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该请求已处理"})
		return
	}

	if err := model.UpdateTenantUpgradeRequestStatus(id, model.TenantUpgradeStatusRejected); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已拒绝"})
}