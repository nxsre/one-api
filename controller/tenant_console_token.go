package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/agentpolicy"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/random"
	"github.com/songquanpeng/one-api/model"
)

func tenantResolveSubCommonUserPK(c *gin.Context, ownerPK int) (*model.User, error) {
	tid, err := GetTenantConsoleTenantIDForSubuserTokenRoutes(c)
	if err != nil {
		return nil, err
	}
	u, err := model.GetUserById(ownerPK, false)
	if err != nil {
		return nil, err
	}
	if u.TenantID == nil || *u.TenantID != tid || (u.Role != model.RoleCommonUser && u.Role != model.RoleTenantAdmin) {
		return nil, fmt.Errorf("只能管理本租户下的子账号")
	}
	return u, nil
}

func validateTenantSubUserTokenModels(ctx context.Context, u *model.User, modelsPtr *string) error {
	if modelsPtr == nil {
		return nil
	}
	raw := strings.TrimSpace(*modelsPtr)
	if raw == "" {
		return nil
	}
	userGroup, err := model.CacheGetUserGroup(u.Id)
	if err != nil {
		return err
	}
	allowed, err := model.CacheGetGroupModels(ctx, userGroup)
	if err != nil {
		return err
	}
	allowed, err = filterGroupModelsByTenantSubUser(ctx, u, userGroup, allowed)
	if err != nil {
		return err
	}
	allow := make(map[string]struct{}, len(allowed))
	for _, m := range allowed {
		t := strings.TrimSpace(m)
		if t != "" {
			allow[t] = struct{}{}
		}
	}
	mappings, _ := model.CollectGroupModelMappingsForUser(ctx, u.Id)
	for _, part := range strings.Split(raw, ",") {
		m := strings.TrimSpace(part)
		if m == "" {
			continue
		}
		if _, ok := allow[m]; ok {
			continue
		}
		ok := false
		for a := range allow {
			if model.AllowanceMatchRequestModel(m, a, mappings) {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("模型「%s」不在该子账号可用范围内（分组能力、租户渠道与账号模型白名单交集）", m)
		}
	}
	return nil
}

// TenantConsoleUserAvailableModels GET /tenant_console/users/:id/available_models
// 返回指定子账号在分组 + 租户渠道 + allowed_models 约束下可选的模型 ID 列表（用于令牌「模型范围」下拉）。
func TenantConsoleUserAvailableModels(c *gin.Context) {
	ctx := c.Request.Context()
	ownerPK, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	u, err := tenantResolveSubCommonUserPK(c, ownerPK)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	userGroup, err := model.CacheGetUserGroup(ownerPK)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	models, err := model.CacheGetGroupModels(ctx, userGroup)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	models, err = filterGroupModelsByTenantSubUser(ctx, u, userGroup, models)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": models})
}

// TenantConsoleUserAvailableModelsDetail GET /tenant_console/users/:id/available_models_detail
func TenantConsoleUserAvailableModelsDetail(c *gin.Context) {
	ownerPK, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	u, err := tenantResolveSubCommonUserPK(c, ownerPK)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	userGroup, err := model.CacheGetUserGroup(ownerPK)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if strings.TrimSpace(userGroup) == "" {
		userGroup = "default"
	}
	tid, err := GetTenantConsoleTenantID(c, "")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	rows, err := model.QueryAvailableModelChannelRowsScoped(userGroup, tid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	rows = model.FilterModelChannelRowsForTenantSubUser(u, rows)
	rows, err = model.ApplyClientFacingModelNames(rows)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"items": rows}})
}

// TenantConsoleListTokens GET /tenant_console/users/:id/tokens
func TenantConsoleListTokens(c *gin.Context) {
	ownerPK, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if _, err := tenantResolveSubCommonUserPK(c, ownerPK); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	order := c.Query("order")
	tokens, err := model.GetAllUserTokens(ownerPK, p*config.ItemsPerPage, config.ItemsPerPage, order)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tokens})
}

// TenantConsoleSearchTokens GET /tenant_console/users/:id/tokens/search
func TenantConsoleSearchTokens(c *gin.Context) {
	ownerPK, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if _, err := tenantResolveSubCommonUserPK(c, ownerPK); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	tokens, err := model.SearchUserTokens(ownerPK, c.Query("keyword"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tokens})
}

// TenantConsoleGetToken GET /tenant_console/users/:id/tokens/:tid
func TenantConsoleGetToken(c *gin.Context) {
	ownerPK, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if _, err := tenantResolveSubCommonUserPK(c, ownerPK); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	tid, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	token, err := model.GetTokenByIds(tid, ownerPK)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": token})
}

// TenantConsoleAddToken POST /tenant_console/users/:id/tokens
func TenantConsoleAddToken(c *gin.Context) {
	ownerPK, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ownerUser, err := tenantResolveSubCommonUserPK(c, ownerPK)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	token := model.Token{}
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := validateToken(c, token); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := validateTenantSubUserTokenModels(c.Request.Context(), ownerUser, token.Models); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	cleanToken := model.Token{
		UserId:            ownerPK,
		Name:              token.Name,
		Key:               random.GenerateKey(),
		CreatedTime:       helper.GetTimestamp(),
		AccessedTime:      helper.GetTimestamp(),
		ExpiredTime:       token.ExpiredTime,
		RemainQuota:       token.RemainQuota,
		UnlimitedQuota:    token.UnlimitedQuota,
		Models:            token.Models,
		Group:             token.Group,
		Subnet:            token.Subnet,
		AgentClientPolicy: token.AgentClientPolicy,
	}
	if cleanToken.AgentClientPolicy != nil && !cleanToken.AgentClientPolicy.IsZero() {
		agentpolicy.SetEnabled(true)
	}
	if err := cleanToken.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cleanToken})
}

// TenantConsoleUpdateToken PUT /tenant_console/users/:id/tokens
func TenantConsoleUpdateToken(c *gin.Context) {
	ownerPK, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ownerUser, err := tenantResolveSubCommonUserPK(c, ownerPK)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	statusOnly := c.Query("status_only")
	token := model.Token{}
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := validateToken(c, token); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if statusOnly == "" {
		if err := validateTenantSubUserTokenModels(c.Request.Context(), ownerUser, token.Models); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
	}
	cleanToken, err := model.GetTokenByIds(token.Id, ownerPK)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if token.Status == model.TokenStatusEnabled {
		if cleanToken.Status == model.TokenStatusExpired && cleanToken.ExpiredTime <= helper.GetTimestamp() && cleanToken.ExpiredTime != -1 {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "令牌已过期，无法启用，请先修改令牌过期时间，或者设置为永不过期"})
			return
		}
		if cleanToken.Status == model.TokenStatusExhausted && cleanToken.RemainQuota <= 0 && !cleanToken.UnlimitedQuota {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "令牌可用额度已用尽，无法启用，请先修改令牌剩余额度，或者设置为无限额度"})
			return
		}
	}
	if statusOnly != "" {
		cleanToken.Status = token.Status
	} else {
		cleanToken.Name = token.Name
		cleanToken.ExpiredTime = token.ExpiredTime
		cleanToken.RemainQuota = token.RemainQuota
		cleanToken.UnlimitedQuota = token.UnlimitedQuota
		cleanToken.Models = token.Models
		cleanToken.Subnet = token.Subnet
		cleanToken.Group = token.Group
		cleanToken.AgentClientPolicy = token.AgentClientPolicy
		if cleanToken.AgentClientPolicy != nil && !cleanToken.AgentClientPolicy.IsZero() {
			agentpolicy.SetEnabled(true)
		}
	}
	if err := cleanToken.Update(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cleanToken})
}

// TenantConsoleDeleteToken DELETE /tenant_console/users/:id/tokens/:tid
func TenantConsoleDeleteToken(c *gin.Context) {
	ownerPK, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if _, err := tenantResolveSubCommonUserPK(c, ownerPK); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	tid, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := model.DeleteTokenById(tid, ownerPK); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
