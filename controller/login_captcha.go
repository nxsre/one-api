package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// LoginCaptchaChallenge 生成点击验证码（与 Turnstile 互斥）。
func LoginCaptchaChallenge(c *gin.Context) {
	if !config.PasswordLoginEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "密码登录已关闭"})
		return
	}
	if config.TurnstileCheckEnabled || !common.LoginMathCaptchaEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前未启用图形验证码登录"})
		return
	}
	masterB64, thumbB64, dotNum, captchaID, legacyDots, err := service.GenerateLoginClickCaptcha()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "验证码生成失败"})
		return
	}
	proofID, proofTs, proofSig, err := prepareLoginRequestProof(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "凭证生成失败"})
		return
	}
	session := sessions.Default(c)
	session.Delete("login_click_captcha_dots")
	session.Delete("pending_login_captcha_id")
	if captchaID != "" {
		session.Set("pending_login_captcha_id", captchaID)
	} else {
		session.Set("login_click_captcha_dots", string(legacyDots))
	}
	if err := session.Save(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无法保存会话"})
		return
	}
	resp := gin.H{
		"master_image":      "data:image/jpeg;base64," + masterB64,
		"thumb_image":       "data:image/png;base64," + thumbB64,
		"dot_num":           dotNum,
		"login_request_id":  proofID,
		"login_request_ts":  proofTs,
		"login_request_sig": proofSig,
	}
	if captchaID != "" {
		resp["captcha_id"] = captchaID
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func parseLoginCaptchaPointsJSON(raw []byte) ([]service.LoginCaptchaClickPoint, error) {
	var pts []service.LoginCaptchaClickPoint
	if err := json.Unmarshal(raw, &pts); err != nil {
		return nil, err
	}
	if len(pts) == 0 || len(pts) > 32 {
		return nil, fmt.Errorf("invalid captcha dots length")
	}
	for _, p := range pts {
		if p.X < 0 || p.X > 32767 || p.Y < 0 || p.Y > 32767 {
			return nil, fmt.Errorf("invalid captcha coords")
		}
	}
	return pts, nil
}

func asStringSession(v interface{}) string {
	s, _ := v.(string)
	return s
}
