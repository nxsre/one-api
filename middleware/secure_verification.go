package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	SecureVerificationSessionKey       = "secure_verified_at"
	SecureVerificationMethodSessionKey = "secure_verified_method"
	SecureVerificationTimeout          = 300 // 秒
)

// SecureVerificationRequired 检查用户是否在有效期内完成过安全验证（与 controller 包 session key 一致）。
func SecureVerificationRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetInt("id")
		if userId == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "未登录",
			})
			c.Abort()
			return
		}

		session := sessions.Default(c)
		verifiedAtRaw := session.Get(SecureVerificationSessionKey)
		if verifiedAtRaw == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "需要安全验证",
				"code":    "VERIFICATION_REQUIRED",
			})
			c.Abort()
			return
		}

		verifiedAt, ok := verifiedAtRaw.(int64)
		if !ok {
			clearSecureVerificationSession(session)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "验证状态异常，请重新验证",
				"code":    "VERIFICATION_INVALID",
			})
			c.Abort()
			return
		}

		if time.Now().Unix()-verifiedAt >= SecureVerificationTimeout {
			clearSecureVerificationSession(session)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "验证已过期，请重新验证",
				"code":    "VERIFICATION_EXPIRED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func clearSecureVerificationSession(session sessions.Session) {
	session.Delete(SecureVerificationSessionKey)
	session.Delete(SecureVerificationMethodSessionKey)
	_ = session.Save()
}
