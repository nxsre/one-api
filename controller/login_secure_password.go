package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/i18n"
	"github.com/songquanpeng/one-api/service"
)

// resolveLoginPasswordAndCaptcha 按系统设置解析登录密码与验证码坐标。
// 安全登录开启：校验 proof、AES 解密；关闭：密码与 captcha_dots_enc 为明文（后者为 JSON 字符串）。
func resolveLoginPasswordAndCaptcha(c *gin.Context, req LoginRequest) (password string, captchaDots []service.LoginCaptchaClickPoint, errMsg string) {
	if !config.SecurePasswordLoginEnabled {
		if req.LoginRequestID != "" || req.LoginRequestSig != "" {
			// 忽略多余字段，兼容旧客户端
		}
		password = req.Password
		if common.LoginMathCaptchaEnabled && !config.TurnstileCheckEnabled {
			if req.CaptchaDotsEnc == "" {
				return "", nil, i18n.Translate(c, "invalid_parameter")
			}
			pts, err := parseLoginCaptchaPointsJSON([]byte(req.CaptchaDotsEnc))
			if err != nil {
				return "", nil, i18n.Translate(c, "invalid_parameter")
			}
			return password, pts, ""
		}
		return password, nil, ""
	}

	if !consumeLoginRequestProof(c, req.LoginRequestID, req.LoginRequestTs, req.LoginRequestSig) {
		return "", nil, "登录凭证无效或已过期，请刷新页面后重试"
	}
	encKey, err := common.TakeLoginEncKey(c, req.LoginRequestID)
	if err != nil {
		return "", nil, "登录凭证无效或已过期，请刷新页面后重试"
	}
	plain, err := common.DecryptLoginPayloadAES(encKey, req.Password)
	if err != nil {
		return "", nil, i18n.Translate(c, "invalid_parameter")
	}
	password = plain
	if common.LoginMathCaptchaEnabled && !config.TurnstileCheckEnabled {
		if req.CaptchaDotsEnc == "" {
			return "", nil, i18n.Translate(c, "invalid_parameter")
		}
		dotsPlain, decErr := common.DecryptLoginPayloadAES(encKey, req.CaptchaDotsEnc)
		if decErr != nil {
			return "", nil, i18n.Translate(c, "invalid_parameter")
		}
		pts, decErr := parseLoginCaptchaPointsJSON([]byte(dotsPlain))
		if decErr != nil {
			return "", nil, i18n.Translate(c, "invalid_parameter")
		}
		return password, pts, ""
	}
	return password, nil, ""
}
