package controller

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/model"
)

// sessionUserLoggedIn 判断 Web 会话是否已登录（/api/status 与 UserAuth 一致，仅看 session cookie）。
func sessionUserLoggedIn(c *gin.Context) bool {
	sess := sessions.Default(c)
	id := sess.Get("id")
	if id == nil {
		return false
	}
	switch v := id.(type) {
	case int:
		return v > 0
	case int64:
		return v > 0
	case float64:
		return int(v) > 0
	default:
		return false
	}
}

// buildPublicStatusData 未登录可见：登录/注册页 UI、OAuth 入口、品牌信息。
func buildPublicStatusData() gin.H {
	return gin.H{
		"status_scope":       "public",
		"version":            common.Version,
		"system_name":        config.SystemName,
		"logo":               config.Logo,
		"footer_html":        config.Footer,
		"email_verification": config.EmailVerificationEnabled,
		"github_oauth":       config.GitHubOAuthEnabled,
		"github_client_id":   config.GitHubClientId,
		"lark_client_id":     config.LarkClientId,
		"wechat_qrcode":      config.WeChatAccountQRCodeImageURL,
		"wechat_login":       config.WeChatAuthEnabled,
		"turnstile_check":    config.TurnstileCheckEnabled,
		"turnstile_site_key": config.TurnstileSiteKey,
		"oidc":               config.OidcEnabled,
		"oidc_client_id":     config.OidcClientId,
		"oidc_authorization_endpoint": config.OidcAuthorizationEndpoint,
		"login_math_captcha":          common.LoginMathCaptchaEnabled && !config.TurnstileCheckEnabled && config.PasswordLoginEnabled,
		"secure_password_login":       config.SecurePasswordLoginEnabled,
	}
}

// buildAuthenticatedStatusData 已登录用户额外字段（在 public 基础上合并）。
func buildAuthenticatedStatusData() gin.H {
	config.OptionMapRWMutex.RLock()
	outboundURLWhitelistEnabled := config.OptionMap["OutboundURLWhitelistEnabled"] == "true"
	config.OptionMapRWMutex.RUnlock()

	data := buildPublicStatusData()
	data["status_scope"] = "authenticated"
	data["start_time"] = common.StartTime
	data["server_address"] = config.ServerAddress
	data["top_up_link"] = config.TopUpLink
	data["chat_link"] = config.ChatLink
	data["quota_per_unit"] = config.QuotaPerUnit
	data["display_in_currency"] = config.DisplayInCurrencyEnabled
	data["oidc_well_known"] = config.OidcWellKnown
	data["oidc_token_endpoint"] = config.OidcTokenEndpoint
	data["oidc_userinfo_endpoint"] = config.OidcUserinfoEndpoint
	data["force_2fa_for_all_users"] = common.Force2FAForAllUsers
	data["secure_step_up_verify"] = true
	data["global_access_mode"] = model.GetGlobalAccessMode()
	data["outbound_url_whitelist_enabled"] = outboundURLWhitelistEnabled
	data["nacos_enabled"] = config.IsNacosEnabled()
	return data
}
