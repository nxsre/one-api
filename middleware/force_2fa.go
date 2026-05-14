package middleware

import (
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/model"

	"github.com/gin-gonic/gin"
)

func sessionInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case float32:
		return int(x), true
	default:
		return 0, false
	}
}

func isForce2FAExemptRoute(method, path string) bool {
	switch {
	case method == http.MethodGet && path == "/api/user/logout":
		return true
	case method == http.MethodGet && strings.HasPrefix(path, "/api/user/self"):
		return true
	case method == http.MethodGet && path == "/api/user/2fa/status":
		return true
	case method == http.MethodPost && path == "/api/user/2fa/setup":
		return true
	case method == http.MethodPost && path == "/api/user/2fa/enable":
		return true
	case method == http.MethodPost && path == "/api/user/2fa/disable":
		return true
	case method == http.MethodPost && path == "/api/user/2fa/backup_codes":
		return true
	case method == http.MethodPost && path == "/api/user/login/2fa":
		return true
	case method == http.MethodPost && path == "/api/user/login":
		return true
	case method == http.MethodGet && path == "/api/user/login/request-proof":
		return true
	case method == http.MethodGet && path == "/api/user/login/captcha":
		return true
	}
	return false
}

// CheckForce2FAWebSession 控制台会话：强制全员 2FA 时拦截未启用 TOTP 的请求。
func CheckForce2FAWebSession(c *gin.Context, userId, role int) bool {
	if !common.Force2FAForAllUsers {
		return true
	}
	if model.UserHasSecondFactor(userId) {
		return true
	}
	if isForce2FAExemptRoute(c.Request.Method, c.Request.URL.Path) {
		return true
	}
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"code":    "force_2fa_required",
		"message": "根据安全策略，请先完成两步验证（TOTP）设置",
	})
	c.Abort()
	return false
}

// CheckForce2FATokenAuth API Key 转发：未启用 TOTP 则拒绝。
func CheckForce2FATokenAuth(c *gin.Context, userId int) bool {
	if !common.Force2FAForAllUsers {
		return true
	}
	if model.UserHasSecondFactor(userId) {
		return true
	}
	abortWithMessage(c, http.StatusForbidden, "根据安全策略，使用 API 前须启用两步验证（TOTP）")
	return false
}
