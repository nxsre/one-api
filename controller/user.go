package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/i18n"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/random"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/rbac"
	"github.com/songquanpeng/one-api/service"
)

// sessionKeyTenantLoginGate 密码登录在 SetupLogin 前写入，用于阻止 OAuth 等绕过租户登录校验。
const sessionKeyTenantLoginGate = "tenant_login_gate"

type LoginRequest struct {
	Username        string          `json:"username"`
	Password        string          `json:"password"`
	CaptchaID       string          `json:"captcha_id"`
	CaptchaMode     string          `json:"captcha_mode"`
	CaptchaDotsEnc  string          `json:"captcha_dots_enc"`
	LoginRequestID  string          `json:"login_request_id"`
	LoginRequestTs  int64           `json:"login_request_ts"`
	LoginRequestSig string          `json:"login_request_sig"`
	TenantIDRaw     json.RawMessage `json:"tenant_id"`
}

func Login(c *gin.Context) {
	if !config.PasswordLoginEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "管理员关闭了密码登录",
			"success": false,
		})
		return
	}
	var loginRequest LoginRequest
	err := json.NewDecoder(c.Request.Body).Decode(&loginRequest)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": i18n.Translate(c, "invalid_parameter"),
			"success": false,
		})
		return
	}
	username := loginRequest.Username
	password := loginRequest.Password
	if username == "" || password == "" {
		c.JSON(http.StatusOK, gin.H{
			"message": i18n.Translate(c, "invalid_parameter"),
			"success": false,
		})
		return
	}
	loginTenantID, tenantClaimed, tenantIDBad := parseLoginTenantID(loginRequest.TenantIDRaw)
	if tenantIDBad {
		logger.Warnf(c.Request.Context(), "login denied: invalid tenant_id format username=%q ip=%s", username, common.ResolveClientIPForLoginBrute(c.GetHeader("X-Forwarded-For"), c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{
			"message": "租户ID格式无效",
			"success": false,
		})
		return
	}
	password, captchaAnswer, resolveErr := resolveLoginPasswordAndCaptcha(c, loginRequest)
	if resolveErr != "" {
		c.JSON(http.StatusOK, gin.H{
			"message": resolveErr,
			"success": false,
		})
		return
	}

	var loginCaptchaRedisCleaned bool
	if common.LoginMathCaptchaEnabled && config.LoginCaptchaEnabled && !config.TurnstileCheckEnabled {
		sess := sessions.Default(c)
		if common.RedisEnabled && common.RDB != nil {
			pendingRaw := sess.Get("pending_login_captcha_id")
			if !service.ConsumeLoginCaptchaRedis(loginRequest.CaptchaID, asStringSession(pendingRaw), captchaAnswer) {
				c.JSON(http.StatusOK, gin.H{"message": "验证码错误", "success": false})
				return
			}
			sess.Delete("pending_login_captcha_id")
			loginCaptchaRedisCleaned = true
		} else {
			rawVal := sess.Get("login_click_captcha_dots")
			sess.Delete("login_click_captcha_dots")
			_ = sess.Save()
			jsonStr, ok := rawVal.(string)
			if !ok || jsonStr == "" {
				c.JSON(http.StatusOK, gin.H{"message": "验证码错误", "success": false})
				return
			}
			if !service.ValidateLoginCaptchaSession([]byte(jsonStr), captchaAnswer) {
				c.JSON(http.StatusOK, gin.H{"message": "验证码错误", "success": false})
				return
			}
		}
	}

	clientIP := common.ResolveClientIPForLoginBrute(c.GetHeader("X-Forwarded-For"), c.ClientIP())
	if common.IsLoginBruteLocked(clientIP, username) {
		if loginCaptchaRedisCleaned {
			_ = sessions.Default(c).Save()
		}
		c.JSON(http.StatusOK, gin.H{"message": "登录尝试过于频繁，请稍后再试", "success": false})
		return
	}

	user := model.User{
		Username: username,
		Password: password,
	}
	err = user.ValidateAndFill()
	if err != nil {
		if loginCaptchaRedisCleaned {
			_ = sessions.Default(c).Save()
		}
		common.RecordLoginBruteFailure(clientIP, username)
		var cred *model.LoginCredentialError
		if errors.As(err, &cred) && cred != nil {
			logger.Warnf(c.Request.Context(), "login denied: reason=%s username=%q ip=%s tenant_claimed=%v tenant_id_in_request=%v",
				cred.Reason, username, clientIP, tenantClaimed, loginTenantID)
		} else {
			logger.Warnf(c.Request.Context(), "login denied: reason=%q username=%q ip=%s tenant_claimed=%v tenant_id_in_request=%v",
				err.Error(), username, clientIP, tenantClaimed, loginTenantID)
		}
		c.JSON(http.StatusOK, gin.H{
			"message": err.Error(),
			"success": false,
		})
		return
	}

	// 租户内账号（子账号 / 租户管理员）必须与「租户登录」一并提交 tenant_id，与普通用户入口完全隔离。
	if loginTenantID != nil {
		if user.TenantID == nil || *user.TenantID != *loginTenantID {
			if loginCaptchaRedisCleaned {
				_ = sessions.Default(c).Save()
			}
			common.RecordLoginBruteFailure(clientIP, username)
			ut := 0
			if user.TenantID != nil {
				ut = *user.TenantID
			}
			logger.Warnf(c.Request.Context(), "login denied: reason=tenant_mismatch username=%q user_id=%d ip=%s tenant_id_in_request=%d user_belongs_to_tenant_id=%d",
				username, user.Id, clientIP, *loginTenantID, ut)
			c.JSON(http.StatusOK, gin.H{
				"message": "用户不属于该租户或租户 ID 错误。请在「租户登录」页填写管理员提供的租户 ID（可在租户控制台首页查看），勿与用户 ID 混淆。",
				"success": false,
			})
			return
		}
	} else if user.TenantID != nil {
		if loginCaptchaRedisCleaned {
			_ = sessions.Default(c).Save()
		}
		logger.Warnf(c.Request.Context(), "login denied: reason=tenant_user_main_login username=%q user_id=%d ip=%s tenant_id=%d",
			username, user.Id, clientIP, *user.TenantID)
		c.JSON(http.StatusOK, gin.H{
			"message": "该账号隶属于租户空间，请使用「租户登录」页面并填写管理员提供的租户 ID 后再登录，勿在普通登录入口登录。",
			"success": false,
		})
		return
	}

	common.ClearLoginBruteState(clientIP, username)

	if model.IsTwoFAEnabled(user.Id) {
		session := sessions.Default(c)
		session.Clear()
		session.Set("pending_username", user.Username)
		session.Set("pending_user_id", user.Id)
		if user.TenantID != nil {
			session.Set(sessionKeyTenantLoginGate, *user.TenantID)
		}
		if err := session.Save(); err != nil {
			c.JSON(http.StatusOK, gin.H{"message": "无法保存会话信息，请重试", "success": false})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "",
			"success": true,
			"data": gin.H{
				"require_2fa": true,
			},
		})
		return
	}

	if user.TenantID != nil {
		sess := sessions.Default(c)
		sess.Set(sessionKeyTenantLoginGate, *user.TenantID)
		if err := sess.Save(); err != nil {
			c.JSON(http.StatusOK, gin.H{"message": "无法保存会话信息，请重试", "success": false})
			return
		}
	}

	SetupLogin(&user, c)
}

// parseLoginTenantID 解析登录请求中的 tenant_id（支持 JSON 数字或字符串，避免前端 Number 大整数精度丢失）。
func parseLoginTenantID(raw json.RawMessage) (id *int, claimed bool, invalid bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, false, false
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, false, true
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, false, false
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil || v < 0 {
			return nil, true, true
		}
		i := int(v)
		if int64(i) != v {
			return nil, true, true
		}
		return &i, true, false
	}
	var v int64
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false, true
	}
	if v < 0 {
		return nil, true, true
	}
	i := int(v)
	if int64(i) != v {
		return nil, true, true
	}
	return &i, true, false
}

// intFromSession 解析 session 中存储的整型（兼容 int / int64 / float64）。
func intFromSession(v interface{}) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case json.Number:
		n, err := x.Int64()
		return int(n), err == nil
	default:
		return 0, false
	}
}

// setup session & cookies and then return user info
func SetupLogin(user *model.User, c *gin.Context) {
	session := sessions.Default(c)
	if user.TenantID != nil {
		gate, ok := intFromSession(session.Get(sessionKeyTenantLoginGate))
		if !ok || gate != *user.TenantID {
			c.JSON(http.StatusOK, gin.H{
				"message": "该账号隶属于租户空间，请通过「租户登录」完成密码登录；不支持从普通登录或第三方绑定直接进入。",
				"success": false,
			})
			return
		}
		session.Delete(sessionKeyTenantLoginGate)
		_ = session.Save()
	}
	session.Clear()
	session.Set("id", user.Id)
	session.Set("username", user.Username)
	session.Set("role", user.Role)
	session.Set("status", user.Status)
	session.Set("group", user.Group)
	err := session.Save()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "无法保存会话信息，请重试",
			"success": false,
		})
		return
	}
	data := gin.H{
		"user_id":      user.Uid,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"role":         user.Role,
		"status":       user.Status,
		"group":        user.Group,
	}
	if user.TenantID != nil {
		data["tenant_id"] = *user.TenantID
	}
	if len(user.TenantPermissions) > 0 {
		data["tenant_permissions"] = user.TenantPermissions
	}
	if common.Force2FAForAllUsers && !model.UserHasSecondFactor(user.Id) {
		data["require_force_2fa_setup"] = true
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "",
		"success": true,
		"data":    data,
	})
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	err := session.Save()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": err.Error(),
			"success": false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "",
		"success": true,
	})
}

func Register(c *gin.Context) {
	ctx := c.Request.Context()
	if !config.RegisterEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "管理员关闭了新用户注册",
			"success": false,
		})
		return
	}
	if !config.PasswordRegisterEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "管理员关闭了通过密码进行注册，请使用第三方账户验证的形式进行注册",
			"success": false,
		})
		return
	}
	var user model.User
	err := json.NewDecoder(c.Request.Body).Decode(&user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_parameter"),
		})
		return
	}
	if err := common.Validate.Struct(&user); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_input"),
		})
		return
	}
	if config.EmailVerificationEnabled {
		if user.Email == "" || user.VerificationCode == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员开启了邮箱验证，请输入邮箱地址和验证码",
			})
			return
		}
		if !common.VerifyCodeWithKey(user.Email, user.VerificationCode, common.EmailVerificationPurpose) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "验证码错误或已过期",
			})
			return
		}
	}
	affCode := user.AffCode // this code is the inviter's code, not the user's own code
	inviterId, _ := model.GetUserIdByAffCode(affCode)
	cleanUser := model.User{
		Username:    user.Username,
		Password:    user.Password,
		DisplayName: user.Username,
		InviterId:   inviterId,
	}
	if config.EmailVerificationEnabled {
		cleanUser.Email = user.Email
	}
	if err := cleanUser.Insert(ctx, inviterId); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	_ = rbac.SyncUser(cleanUser.Id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func GetUsersPaged(c *gin.Context) {
	var opts model.UserQueryOptions
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	opts.Page = p
	opts.PageSize, _ = strconv.Atoi(c.Query("page_size"))
	if opts.PageSize <= 0 {
		opts.PageSize = config.ItemsPerPage
	}
	opts.Order = c.DefaultQuery("order", "")
	opts.Scope = c.DefaultQuery("scope", "independent")
	if tID, err := strconv.Atoi(c.Query("tenant_id")); err == nil {
		opts.TenantID = &tID
	}
	if rolesStr := c.Query("roles"); rolesStr != "" {
		for _, rs := range strings.Split(rolesStr, ",") {
			if r, err := strconv.Atoi(rs); err == nil {
				opts.Roles = append(opts.Roles, r)
			}
		}
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			opts.Status = &s
		}
	}
	opts.Keyword = c.Query("keyword")

	users, total, err := model.ListUsersPaged(opts)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items":     users,
			"total":     total,
			"page":      opts.Page,
			"page_size": opts.PageSize,
			"scope":     opts.Scope,
			"tenant_id": opts.TenantID,
		},
	})
}

func PromoteUserToSuperAdmin(c *gin.Context) {
	pk, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	targetUser, err := model.GetUserById(pk, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if targetUser.TenantID != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "只能将非租户独立用户升级为超级管理员"})
		return
	}
	if targetUser.Role >= model.RoleSuperAdmin {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "目标用户已是超级管理员或更高等级"})
		return
	}
	targetUser.Role = model.RoleSuperAdmin
	if err := targetUser.Update(false); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	_ = rbac.SyncUser(targetUser.Id)
	model.RecordLog(c.Request.Context(), c.GetInt(ctxkey.Id), model.LogTypeManage, fmt.Sprintf("将用户 %s 升级为超级管理员", targetUser.Username))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func DemoteSuperAdmin(c *gin.Context) {
	pk, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	targetUser, err := model.GetUserById(pk, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if targetUser.Role == model.RoleRootUser {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "不可降级 Root 超级管理员"})
		return
	}
	if targetUser.Role != model.RoleSuperAdmin {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "目标用户不是超级管理员"})
		return
	}
	targetUser.Role = model.RoleAdminUser
	if err := targetUser.Update(false); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	_ = rbac.SyncUser(targetUser.Id)
	model.RecordLog(c.Request.Context(), c.GetInt(ctxkey.Id), model.LogTypeManage, fmt.Sprintf("将用户 %s 从超级管理员降级为普通管理员", targetUser.Username))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func GetAllUsers(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}

	order := c.DefaultQuery("order", "")
	users, err := model.GetAllUsers(p*config.ItemsPerPage, config.ItemsPerPage, order)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    users,
	})
}

func SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	users, err := model.SearchUsers(keyword)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    users,
	})
	return
}

func GetUser(c *gin.Context) {
	user, err := model.GetUserByPublicID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	myRole := c.GetInt(ctxkey.Role)
	if myRole <= user.Role && myRole != model.RoleRootUser {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权获取同级或更高等级用户的信息",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user,
	})
	return
}

func GetUserDashboard(c *gin.Context) {
	id := c.GetInt(ctxkey.Id)
	now := time.Now()
	startOfDay := now.Truncate(24*time.Hour).AddDate(0, 0, -6).Unix()
	endOfDay := now.Truncate(24 * time.Hour).Add(24*time.Hour - time.Second).Unix()

	dashboards, err := model.SearchLogsByDayAndModel(id, int(startOfDay), int(endOfDay))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无法获取统计信息",
			"data":    nil,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dashboards,
	})
	return
}

func GenerateAccessToken(c *gin.Context) {
	id := c.GetInt(ctxkey.Id)
	user, err := model.GetUserById(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user.AccessToken = random.GetUUID()

	if model.DB.Where("access_token = ?", user.AccessToken).First(user).RowsAffected != 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "请重试，系统生成的 UUID 竟然重复了！",
		})
		return
	}

	if err := user.Update(false); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user.AccessToken,
	})
	return
}

func GetAffCode(c *gin.Context) {
	id := c.GetInt(ctxkey.Id)
	user, err := model.GetUserById(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if user.AffCode == "" {
		user.AffCode = random.GetRandomString(4)
		if err := user.Update(false); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user.AffCode,
	})
	return
}

func GetSelf(c *gin.Context) {
	id := c.GetInt(ctxkey.Id)
	user, err := model.GetUserById(id, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	data := gin.H{
		"user_id":         user.Uid,
		"username":        user.Username,
		"display_name":    user.DisplayName,
		"role":            user.Role,
		"status":          user.Status,
		"group":           user.Group,
		"quota":           user.Quota,
		"used_quota":      user.UsedQuota,
		"s3_site_enabled": common.S3SiteOpen(),
		"s3_region":       common.S3Region,
		"s3_enabled":      user.S3Enabled,
	}
	if user.S3AccessKey != nil && *user.S3AccessKey != "" {
		data["s3_access_key"] = *user.S3AccessKey
	}
	if common.Force2FAForAllUsers && !model.UserHasSecondFactor(user.Id) {
		data["require_force_2fa_setup"] = true
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
	return
}

func UpdateUser(c *gin.Context) {
	ctx := c.Request.Context()
	var updatedUser model.User
	err := json.NewDecoder(c.Request.Body).Decode(&updatedUser)
	if err != nil || strings.TrimSpace(updatedUser.Uid) == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_parameter"),
		})
		return
	}
	originUser, err := model.GetUserByPublicID(updatedUser.Uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	updatedUser.Id = originUser.Id
	if updatedUser.Password == "" {
		updatedUser.Password = "$I_LOVE_U" // make Validator happy :)
	}
	if err := common.Validate.Struct(&updatedUser); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_input"),
		})
		return
	}
	myRole := c.GetInt(ctxkey.Role)
	if myRole <= originUser.Role && myRole != model.RoleRootUser {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权更新同权限等级或更高权限等级的用户信息",
		})
		return
	}
	if myRole <= updatedUser.Role && myRole != model.RoleRootUser {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权将其他用户权限等级提升到大于等于自己的权限等级",
		})
		return
	}
	if updatedUser.Password == "$I_LOVE_U" {
		updatedUser.Password = "" // rollback to what it should be
	}
	updatePassword := updatedUser.Password != ""
	if err := updatedUser.Update(updatePassword); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	_ = rbac.SyncUser(updatedUser.Id)
	if originUser.Quota != updatedUser.Quota {
		model.RecordLog(ctx, originUser.Id, model.LogTypeManage, fmt.Sprintf("管理员将用户额度从 %s修改为 %s", common.LogQuota(originUser.Quota), common.LogQuota(updatedUser.Quota)))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateSelf(c *gin.Context) {
	var user model.User
	err := json.NewDecoder(c.Request.Body).Decode(&user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_parameter"),
		})
		return
	}
	if user.Password == "" {
		user.Password = "$I_LOVE_U" // make Validator happy :)
	}
	if err := common.Validate.Struct(&user); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "输入不合法 " + err.Error(),
		})
		return
	}

	cleanUser := model.User{
		Id:          c.GetInt(ctxkey.Id),
		Username:    user.Username,
		Password:    user.Password,
		DisplayName: user.DisplayName,
	}
	if user.Password == "$I_LOVE_U" {
		user.Password = "" // rollback to what it should be
		cleanUser.Password = ""
	}
	updatePassword := user.Password != ""
	if err := cleanUser.Update(updatePassword); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteUser(c *gin.Context) {
	pk, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	originUser, err := model.GetUserById(pk, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	myRole := c.GetInt("role")
	if myRole <= originUser.Role {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权删除同权限等级或更高权限等级的用户",
		})
		return
	}
	err = model.DeleteUserById(pk)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	rbac.RemoveSubject(pk)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func DeleteSelf(c *gin.Context) {
	id := c.GetInt("id")
	user, _ := model.GetUserById(id, false)

	if user.Role == model.RoleRootUser {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "不能删除超级管理员账户",
		})
		return
	}

	err := model.DeleteUserById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	rbac.RemoveSubject(id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func CreateUser(c *gin.Context) {
	ctx := c.Request.Context()
	var user model.User
	err := json.NewDecoder(c.Request.Body).Decode(&user)
	if err != nil || user.Username == "" || user.Password == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_parameter"),
		})
		return
	}
	if err := common.Validate.Struct(&user); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_input"),
		})
		return
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}
	myRole := c.GetInt("role")
	if user.Role >= myRole {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无法创建权限大于等于自己的用户",
		})
		return
	}
	// Even for admin users, we cannot fully trust them!
	cleanUser := model.User{
		Username:    user.Username,
		Password:    user.Password,
		DisplayName: user.DisplayName,
	}
	if err := cleanUser.Insert(ctx, 0); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	_ = rbac.SyncUser(cleanUser.Id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

type ManageRequest struct {
	Username string `json:"username"`
	Action   string `json:"action"`
}

// ManageUser Only admin user can do this
func ManageUser(c *gin.Context) {
	var req ManageRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_parameter"),
		})
		return
	}
	user := model.User{
		Username: req.Username,
	}
	// Fill attributes
	model.DB.Where(&user).First(&user)
	if user.Id == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}
	myRole := c.GetInt("role")
	if myRole <= user.Role && myRole != model.RoleRootUser {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权更新同权限等级或更高权限等级的用户信息",
		})
		return
	}
	switch req.Action {
	case "disable":
		user.Status = model.UserStatusDisabled
		if user.Role == model.RoleRootUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法禁用超级管理员用户",
			})
			return
		}
	case "enable":
		user.Status = model.UserStatusEnabled
	case "delete":
		if user.Role == model.RoleRootUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法删除超级管理员用户",
			})
			return
		}
		if err := user.Delete(); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		rbac.RemoveSubject(user.Id)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": model.UserManageResult{
				Role:   user.Role,
				Status: user.Status,
			},
		})
		return
	case "promote":
		if myRole != model.RoleRootUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "普通管理员用户无法提升其他用户为管理员",
			})
			return
		}
		if user.Role >= model.RoleAdminUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "该用户已经是管理员",
			})
			return
		}
		user.Role = model.RoleAdminUser
	case "demote":
		if user.Role == model.RoleRootUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法降级超级管理员用户",
			})
			return
		}
		if user.Role == model.RoleCommonUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "该用户已经是普通用户",
			})
			return
		}
		user.Role = model.RoleCommonUser
	}

	if err := user.Update(false); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	_ = rbac.SyncUser(user.Id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": model.UserManageResult{
			Role:   user.Role,
			Status: user.Status,
		},
	})
	return
}

func EmailBind(c *gin.Context) {
	email := c.Query("email")
	code := c.Query("code")
	if !common.VerifyCodeWithKey(email, code, common.EmailVerificationPurpose) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "验证码错误或已过期",
		})
		return
	}
	id := c.GetInt("id")
	user := model.User{
		Id: id,
	}
	err := user.FillUserById()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user.Email = email
	// no need to check if this email already taken, because we have used verification code to check it
	err = user.Update(false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if user.Role == model.RoleRootUser {
		config.RootUserEmail = email
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

type topUpRequest struct {
	Key string `json:"key"`
}

func TopUp(c *gin.Context) {
	ctx := c.Request.Context()
	req := topUpRequest{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	id := c.GetInt("id")
	quota, err := model.Redeem(ctx, req.Key, id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    quota,
	})
	return
}

type adminTopUpRequest struct {
	UserID string `json:"user_id"`
	Quota  int    `json:"quota"`
	Remark string `json:"remark"`
}

func AdminTopUp(c *gin.Context) {
	ctx := c.Request.Context()
	req := adminTopUpRequest{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	target, err := model.GetUserByPublicID(strings.TrimSpace(req.UserID))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = model.IncreaseUserQuota(target.Id, int64(req.Quota))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if req.Remark == "" {
		req.Remark = fmt.Sprintf("通过 API 充值 %s", common.LogQuota(int64(req.Quota)))
	}
	model.RecordTopupLog(ctx, target.Id, req.Remark, req.Quota)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}
