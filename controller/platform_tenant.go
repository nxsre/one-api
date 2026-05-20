package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/rbac"
)

// PlatformCreateTenant Root 创建租户并初始化首个租户管理员账号。
func PlatformCreateTenant(c *gin.Context) {
	var body struct {
		Name             string `json:"name"`
		Slug             string `json:"slug"`
		Remark           string `json:"remark"`
		AdminUsername    string `json:"admin_username"`
		AdminPassword    string `json:"admin_password"`
		AdminDisplayName string `json:"admin_display_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Slug = strings.TrimSpace(body.Slug)
	body.AdminUsername = strings.TrimSpace(body.AdminUsername)
	if body.Name == "" || body.Slug == "" || body.AdminUsername == "" || body.AdminPassword == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name、slug、admin_username、admin_password 必填"})
		return
	}
	if _, err := model.GetTenantBySlug(body.Slug); err == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "slug 已存在"})
		return
	}
	t := &model.Tenant{
		Name:        body.Name,
		Slug:        body.Slug,
		Remark:      strings.TrimSpace(body.Remark),
		Status:      model.TenantStatusEnabled,
		CreatedTime: helper.GetTimestamp(),
	}
	if err := model.InsertTenant(t); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	display := strings.TrimSpace(body.AdminDisplayName)
	if display == "" {
		display = body.AdminUsername
	}
	tid := t.Id
	u := &model.User{
		Username:    body.AdminUsername,
		Password:    body.AdminPassword,
		DisplayName: display,
		Role:        model.RoleTenantAdmin,
		Status:      model.UserStatusEnabled,
		TenantID:    &tid,
	}
	ctx := c.Request.Context()
	if err := u.Insert(ctx, 0); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	_ = rbac.SyncUser(u.Id)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"tenant_id": t.Id, "admin_user_id": u.Id}})
}

// PlatformListTenants GET /platform/tenants 平台管理员枚举租户（用于代管租户控制台时选择租户）。
func PlatformListTenants(c *gin.Context) {
	limit := 500
	if q := strings.TrimSpace(c.Query("limit")); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	rows, err := model.ListTenantsForPlatform(limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

// PlatformListTenantOptions GET /platform/tenants/options
func PlatformListTenantOptions(c *gin.Context) {
	type Option struct {
		Id     int    `json:"id"`
		Name   string `json:"name"`
		Status int    `json:"status"`
	}
	var opts []Option
	err := model.DB.Model(&model.Tenant{}).Select("id", "name", "status").Find(&opts).Error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": opts})
}
