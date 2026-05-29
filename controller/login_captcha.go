package controller

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// resolveLoginCaptchaMode picks the concrete captcha mode for this request.
// The legacy click(点选) captcha was removed; only the slide / rotate modes are
// offered. "random" (and any legacy value) rotates between slide and rotate.
func resolveLoginCaptchaMode(_ *gin.Context) string {
	switch config.LoginCaptchaMode {
	case config.LoginCaptchaModeSlide, config.LoginCaptchaModeRotate:
		return config.LoginCaptchaMode
	default:
		modes := []string{
			config.LoginCaptchaModeSlide,
			config.LoginCaptchaModeRotate,
		}
		return modes[rand.Intn(len(modes))]
	}
}

// LoginCaptchaChallenge 生成登录验证码（与 Turnstile 互斥）。支持 click / slide / rotate 多模式。
func LoginCaptchaChallenge(c *gin.Context) {
	if !config.PasswordLoginEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "密码登录已关闭"})
		return
	}
	if config.TurnstileCheckEnabled || !common.LoginMathCaptchaEnabled || !config.LoginCaptchaEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前未启用图形验证码登录"})
		return
	}

	mode := resolveLoginCaptchaMode(c)
	challenge, err := service.GenerateLoginCaptcha(mode)
	if err != nil {
		// Slide/rotate need bundled assets; if unavailable fall back to click.
		if mode != config.LoginCaptchaModeClick {
			challenge, err = service.GenerateLoginCaptcha(config.LoginCaptchaModeClick)
		}
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "验证码生成失败"})
			return
		}
	}

	var proofID string
	var proofTs int64
	var proofSig string
	var encKeyB64 string
	if config.SecurePasswordLoginEnabled {
		proofID, proofTs, proofSig, encKeyB64, err = prepareLoginRequestProof(c)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "凭证生成失败"})
			return
		}
	}

	stored := challenge.StoredJSON()
	var captchaID string
	if common.RedisEnabled && common.RDB != nil {
		captchaID, err = service.StoreLoginCaptchaRedis(stored)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "验证码生成失败"})
			return
		}
	}

	session := sessions.Default(c)
	session.Delete("login_click_captcha_dots")
	session.Delete("pending_login_captcha_id")
	if captchaID != "" {
		session.Set("pending_login_captcha_id", captchaID)
	} else {
		session.Set("login_click_captcha_dots", string(stored))
	}
	if err := session.Save(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无法保存会话"})
		return
	}

	resp := gin.H{
		"mode":         challenge.Mode,
		"master_image": challenge.MasterImage,
		"thumb_image":  challenge.ThumbImage,
	}
	switch challenge.Mode {
	case config.LoginCaptchaModeClick:
		resp["dot_num"] = challenge.DotNum
	case config.LoginCaptchaModeSlide:
		resp["tile_x"] = challenge.TileX
		resp["tile_y"] = challenge.TileY
		resp["tile_width"] = challenge.TileWidth
		resp["tile_height"] = challenge.TileHeight
	case config.LoginCaptchaModeRotate:
		resp["thumb_size"] = challenge.ThumbSize
	}
	if config.SecurePasswordLoginEnabled {
		resp["login_request_id"] = proofID
		resp["login_request_ts"] = proofTs
		resp["login_request_sig"] = proofSig
		resp["login_enc_key"] = encKeyB64
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

// parseLoginCaptchaAnswer parses the mode-specific solution JSON into an answer.
func parseLoginCaptchaAnswer(mode string, raw []byte) (service.LoginCaptchaAnswer, error) {
	switch mode {
	case config.LoginCaptchaModeSlide:
		var a struct {
			X int `json:"x"`
			Y int `json:"y"`
		}
		if err := json.Unmarshal(raw, &a); err != nil {
			return service.LoginCaptchaAnswer{}, err
		}
		if a.X < 0 || a.X > 32767 || a.Y < 0 || a.Y > 32767 {
			return service.LoginCaptchaAnswer{}, fmt.Errorf("invalid slide coords")
		}
		return service.LoginCaptchaAnswer{Mode: mode, X: a.X, Y: a.Y}, nil
	case config.LoginCaptchaModeRotate:
		var a struct {
			Angle int `json:"angle"`
		}
		if err := json.Unmarshal(raw, &a); err != nil {
			return service.LoginCaptchaAnswer{}, err
		}
		if a.Angle < 0 || a.Angle > 360 {
			return service.LoginCaptchaAnswer{}, fmt.Errorf("invalid rotate angle")
		}
		return service.LoginCaptchaAnswer{Mode: mode, Angle: a.Angle}, nil
	default:
		pts, err := parseLoginCaptchaPointsJSON(raw)
		if err != nil {
			return service.LoginCaptchaAnswer{}, err
		}
		return service.LoginCaptchaAnswer{Mode: config.LoginCaptchaModeClick, Dots: pts}, nil
	}
}

func asStringSession(v interface{}) string {
	s, _ := v.(string)
	return s
}
