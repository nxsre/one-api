package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/i18n"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/rbac"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

// TenantConsoleTenantIDHeader 平台管理员代管租户控制台时在请求头指定目标租户 ID。
const TenantConsoleTenantIDHeader = "X-Tenant-Console-Tenant-Id"

func GetTenantConsoleTenantID(c *gin.Context, requiredPerm string) (int, error) {
	uid := c.GetInt(ctxkey.Id)
	if uid <= 0 {
		return 0, fmt.Errorf("无权访问")
	}
	if !model.IsPlatformConsoleOperator(uid) {
		var user model.User
		err := model.DB.Where("id = ?", uid).Select("role", "tenant_id", "tenant_permissions").Find(&user).Error
		if err != nil || user.TenantID == nil {
			return 0, fmt.Errorf("无权访问")
		}
		if user.Role == model.RoleTenantAdmin {
			return *user.TenantID, nil
		}
		if user.Role == model.RoleCommonUser {
			if requiredPerm == "" {
				if len(user.TenantPermissions) > 0 {
					return *user.TenantID, nil
				}
				return 0, fmt.Errorf("无权访问")
			}
			for _, p := range user.TenantPermissions {
				if p == requiredPerm {
					return *user.TenantID, nil
				}
			}
			return 0, fmt.Errorf("您没有权限执行此操作 (%s)", requiredPerm)
		}
		return 0, fmt.Errorf("无权访问")
	}
	raw := strings.TrimSpace(c.GetHeader(TenantConsoleTenantIDHeader))
	if raw == "" {
		return 0, fmt.Errorf("请先在页面顶部选择要代管的租户")
	}
	tid64, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || tid64 <= 0 || tid64 > int64(math.MaxInt) {
		return 0, fmt.Errorf("选择的租户ID无效")
	}
	tid := int(tid64)
	if _, err := model.GetTenantByID(tid); err != nil {
		return 0, fmt.Errorf("选择的租户不存在")
	}
	return tid, nil
}

// GetTenantConsoleTenantIDForSubuserTokenRoutes 管理子账号的令牌：委派子账号具备 manage_users 或 manage_tokens 之一即可（与侧栏展示「子账号」入口的条件一致）。
func GetTenantConsoleTenantIDForSubuserTokenRoutes(c *gin.Context) (int, error) {
	tid, err := GetTenantConsoleTenantID(c, "manage_tokens")
	if err == nil {
		return tid, nil
	}
	return GetTenantConsoleTenantID(c, "manage_users")
}

func dedupePositiveInts(ids []int) []int {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int]struct{})
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizeAllowedModelSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{})
	for _, m := range in {
		t := strings.TrimSpace(m)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func TenantConsoleListUsers(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_users")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	order := c.DefaultQuery("order", "")
	users, err := model.GetTenantSubUsers(tid, p*config.ItemsPerPage, config.ItemsPerPage, order)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": users})
}

func TenantConsoleSearchUsers(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_users")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	users, err := model.SearchTenantSubUsers(tid, c.Query("keyword"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": users})
}

// TenantConsoleGetUserProfile 返回本子账号详情（供租户控制台编辑页）。
func TenantConsoleGetUserProfile(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_users")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	pk, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	u, err := model.GetUserById(pk, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if u.TenantID == nil || *u.TenantID != tid || (u.Role != model.RoleCommonUser && u.Role != model.RoleTenantAdmin) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权查看该用户"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":               u.Uid,
			"username":              u.Username,
			"display_name":          u.DisplayName,
			"role":                  u.Role,
			"status":                u.Status,
			"group":                 u.Group,
			"quota":                 u.Quota,
			"email":                 u.Email,
			"allowed_models":        u.AllowedModels,
			"allowed_channel_ids":   u.AllowedChannelIDs,
			"tenant_permissions":    u.TenantPermissions,
		},
	})
}

func TenantConsoleCreateUser(c *gin.Context) {
	ctx := c.Request.Context()
	tid, err := GetTenantConsoleTenantID(c, "manage_users")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	var u model.User
	if err := json.NewDecoder(c.Request.Body).Decode(&u); err != nil || u.Username == "" || u.Password == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": i18n.Translate(c, "invalid_parameter")})
		return
	}
	if err := common.Validate.Struct(&u); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": i18n.Translate(c, "invalid_input")})
		return
	}
	if u.DisplayName == "" {
		u.DisplayName = u.Username
	}
	tidAssign := tid
	clean := model.User{
		Username:          u.Username,
		Password:          u.Password,
		DisplayName:       u.DisplayName,
		Group:             u.Group,
		Quota:             u.Quota,
		Role:              u.Role,
		Status:            model.UserStatusEnabled,
		TenantID:          &tidAssign,
		TenantPermissions: u.TenantPermissions,
	}
	if clean.Role != model.RoleTenantAdmin {
		clean.Role = model.RoleCommonUser
	}
	if strings.TrimSpace(clean.Group) == "" {
		clean.Group = "default"
	}
	chIDs := dedupePositiveInts(u.AllowedChannelIDs)
	if err := model.ValidateChannelIDsForTenantSubUser(tid, clean.Group, chIDs); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	clean.AllowedChannelIDs = chIDs
	clean.AllowedModels = normalizeAllowedModelSlice(u.AllowedModels)
	if err := clean.Insert(ctx, 0); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	// 确保 tenant_id 落库（个别驱动/历史表结构下 Create 可能未写入关联字段）
	if err := model.DB.Model(&model.User{}).Where("id = ?", clean.Id).UpdateColumn("tenant_id", tid).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户已创建但写入租户关联失败: " + err.Error()})
		return
	}
	_ = rbac.SyncUser(clean.Id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TenantConsoleMetaTenant GET /tenant_console/meta/tenant — 当前登录上下文下的租户 ID（便于子账号核对租户登录页填写）。
func TenantConsoleMetaTenant(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	t, err := model.GetTenantByID(tid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tenant_id": t.Id,
			"name":      t.Name,
			"slug":      t.Slug,
		},
	})
}

func TenantConsoleUpdateUser(c *gin.Context) {
	ctx := c.Request.Context()
	tid, err := GetTenantConsoleTenantID(c, "manage_users")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": i18n.Translate(c, "invalid_parameter")})
		return
	}
	_, hasAllowedModels := raw["allowed_models"]
	_, hasAllowedChannelIDs := raw["allowed_channel_ids"]
	_, hasTenantPermissions := raw["tenant_permissions"]
	var updated model.User
	if err := json.Unmarshal(body, &updated); err != nil || strings.TrimSpace(updated.Uid) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": i18n.Translate(c, "invalid_parameter")})
		return
	}
	origin, err := model.GetUserByPublicID(updated.Uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if origin.TenantID == nil || *origin.TenantID != tid || (origin.Role != model.RoleCommonUser && origin.Role != model.RoleTenantAdmin) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "只能管理本租户下的子账号"})
		return
	}
	updated.Id = origin.Id
	updated.TenantID = origin.TenantID
	if updated.Role != model.RoleTenantAdmin {
		updated.Role = model.RoleCommonUser
	}
	if !hasAllowedModels {
		updated.AllowedModels = origin.AllowedModels
	}
	if !hasAllowedChannelIDs {
		updated.AllowedChannelIDs = origin.AllowedChannelIDs
	}
	if !hasTenantPermissions {
		updated.TenantPermissions = origin.TenantPermissions
	}
	if updated.Password == "" {
		updated.Password = "$I_LOVE_U"
	}
	if err := common.Validate.Struct(&updated); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": i18n.Translate(c, "invalid_input")})
		return
	}
	if updated.Password == "$I_LOVE_U" {
		updated.Password = ""
	}
	if hasAllowedChannelIDs {
		chIDs := dedupePositiveInts(updated.AllowedChannelIDs)
		subGroup := strings.TrimSpace(updated.Group)
		if subGroup == "" {
			subGroup = strings.TrimSpace(origin.Group)
		}
		if subGroup == "" {
			subGroup = "default"
		}
		if err := model.ValidateChannelIDsForTenantSubUser(tid, subGroup, chIDs); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		updated.AllowedChannelIDs = chIDs
	}
	if hasAllowedModels {
		updated.AllowedModels = normalizeAllowedModelSlice(updated.AllowedModels)
	}
	updatePassword := updated.Password != ""
	if err := updated.Update(updatePassword); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	_ = rbac.SyncUser(updated.Id)
	if origin.Quota != updated.Quota {
		model.RecordLog(ctx, origin.Id, model.LogTypeManage, fmt.Sprintf("租户管理员将用户额度从 %s修改为 %s", common.LogQuota(origin.Quota), common.LogQuota(updated.Quota)))
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func TenantConsoleDeleteUser(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_users")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	pk, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	u, err := model.GetUserById(pk, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if u.TenantID == nil || *u.TenantID != tid || (u.Role != model.RoleCommonUser && u.Role != model.RoleTenantAdmin) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权删除该用户"})
		return
	}
	if err := u.Delete(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	rbac.RemoveSubject(u.Id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func TenantConsoleListChannels(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_channels")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	pageSize := config.ItemsPerPage
	if ps := c.Query("page_size"); ps != "" {
		if v, e := strconv.Atoi(ps); e == nil && v > 0 && v <= 500 {
			pageSize = v
		}
	}
	idSort, _ := strconv.ParseBool(c.Query("id_sort"))
	typeF := -1
	if t := strings.TrimSpace(c.Query("type")); t != "" {
		typeF, _ = strconv.Atoi(t)
	}
	statusF := -1
	switch strings.ToLower(strings.TrimSpace(c.Query("status"))) {
	case "enabled", "1":
		statusF = model.ChannelStatusEnabled
	case "disabled", "0":
		statusF = 0
	}
	channels, _, err := model.GetChannelsPage(model.ChannelPageOptions{
		Offset:       p * pageSize,
		Limit:        pageSize,
		Scope:        "limited",
		TypeFilter:   typeF,
		StatusFilter: statusF,
		SortBy:       c.Query("sort_by"),
		SortOrder:    c.Query("sort_order"),
		IdSort:       idSort,
		TenantIDEq:   &tid,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": channels})
}

func TenantConsoleSearchChannels(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_channels")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	chs, err := model.SearchTenantChannels(tid, c.Query("keyword"), c.Query("group"), c.Query("model"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": chs})
}

// TenantConsoleChannelsForACL GET /tenant_console/meta/channels_for_acl?group=
// 子账号「渠道白名单」可选范围：本租户渠道 + 平台渠道（分组命中）。
func TenantConsoleChannelsForACL(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	group := strings.TrimSpace(c.Query("group"))
	if group == "" {
		group = "default"
	}
	chs, err := model.ListChannelsForTenantSubUserACL(tid, group)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": chs})
}

func TenantConsoleAddChannel(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_channels")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	channel := model.Channel{}
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	channel.TenantID = &tid
	channel.CreatedTime = helper.GetTimestamp()
	keys := strings.Split(channel.Key, "\n")
	channels := make([]model.Channel, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		localChannel := channel
		localChannel.Key = key
		if err := channeltype.ValidateChannelBaseURL(localChannel.Type, localChannel.GetBaseURL()); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		channels = append(channels, localChannel)
	}
	if err := model.BatchInsertChannels(channels); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func TenantConsoleUpdateChannel(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_channels")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	channel := model.Channel{}
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	existing, err := model.GetChannelById(channel.Id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if existing.TenantID == nil || *existing.TenantID != tid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权修改该渠道"})
		return
	}
	channel.TenantID = &tid
	t2, base := effectiveChannelTypeBaseURL(existing, &channel)
	if err := channeltype.ValidateChannelBaseURL(t2, base); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := channel.Update(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func TenantConsoleDeleteChannel(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_channels")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	ch, err := model.GetChannelById(id, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if ch.TenantID == nil || *ch.TenantID != tid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权删除该渠道"})
		return
	}
	if err := (&model.Channel{Id: id}).Delete(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
