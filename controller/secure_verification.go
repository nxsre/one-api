package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/model"
)

type universalVerifyRequest struct {
	Method string `json:"method"` // 目前仅支持 "2fa"
	Code   string `json:"code,omitempty"`
}

// validateTwoFactorAuthStepUp 与 new-api 一致：6 位 TOTP 或备用码。
func validateTwoFactorAuthStepUp(twoFA *model.TwoFA, code string) bool {
	if cleanCode, err := common.ValidateNumericCode(code); err == nil {
		if ok, _ := twoFA.ValidateTOTPAndUpdateUsage(cleanCode); ok {
			return true
		}
	}
	if ok, err := twoFA.ValidateBackupCodeAndUpdateUsage(code); err == nil && ok {
		return true
	}
	return false
}

// UniversalVerify 已登录用户完成二次验证后，在 session 中写入短期有效标记（用于查看密钥等敏感操作）。one-api 无 Passkey，仅支持 method=2fa。
func UniversalVerify(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}

	var req universalVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误: " + err.Error()})
		return
	}

	user := &model.User{Id: userId}
	if err := user.FillUserById(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取用户信息失败"})
		return
	}
	if user.Status != model.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该用户已被禁用"})
		return
	}

	twoFA, err := model.GetTwoFAByUserId(userId)
	if err != nil || twoFA == nil || !twoFA.IsEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户未启用两步验证（TOTP）"})
		return
	}

	switch req.Method {
	case "2fa":
		if req.Code == "" {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "验证码不能为空"})
			return
		}
		if !validateTwoFactorAuthStepUp(twoFA, req.Code) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "验证失败，请检查验证码或备用码"})
			return
		}
	default:
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("不支持的验证方式: %s", req.Method)})
		return
	}

	session := sessions.Default(c)
	now := time.Now().Unix()
	session.Set(middleware.SecureVerificationSessionKey, now)
	session.Set(middleware.SecureVerificationMethodSessionKey, "2fa")
	if err := session.Save(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "保存验证状态失败"})
		return
	}

	model.RecordLog(c.Request.Context(), userId, model.LogTypeSystem, "通用安全验证成功 (验证方式: 2FA)")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证成功",
		"data": gin.H{
			"verified":   true,
			"expires_at": now + middleware.SecureVerificationTimeout,
		},
	})
}
