package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/i18n"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const (
	loginReqProofKeyPrefix = "login_req:v1:"
	loginReqProofTTL       = 10 * time.Minute
)

var loginProofConsumeScript = redis.NewScript(`
local v = redis.call("GET", KEYS[1])
if v == false then
  return 0
end
if v ~= ARGV[1] then
  return -1
end
redis.call("DEL", KEYS[1])
return 1
`)

func prepareLoginRequestProof(c *gin.Context) (id string, ts int64, sigB64, encKeyB64 string, err error) {
	id = uuid.New().String()
	ts = time.Now().Unix()
	sigB64, err = common.SignLoginRequestProof(id, ts)
	if err != nil {
		return "", 0, "", "", err
	}
	encKey, encKeyB64, err := common.NewLoginEncKey()
	if err != nil {
		return "", 0, "", "", err
	}
	if err = common.StoreLoginEncKey(c, id, encKey); err != nil {
		return "", 0, "", "", err
	}
	if common.RedisEnabled && common.RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		key := loginReqProofKeyPrefix + id
		if err = common.RDB.Set(ctx, key, fmt.Sprintf("%d", ts), loginReqProofTTL).Err(); err != nil {
			return "", 0, "", "", err
		}
		return id, ts, sigB64, encKeyB64, nil
	}
	sess := sessions.Default(c)
	sess.Set("pending_login_req_proof_id", id)
	sess.Set("pending_login_req_proof_ts", ts)
	return id, ts, sigB64, encKeyB64, nil
}

func consumeLoginRequestProofRedis(id string, ts int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	key := loginReqProofKeyPrefix + id
	n, err := loginProofConsumeScript.Run(ctx, common.RDB, []string{key}, fmt.Sprintf("%d", ts)).Int()
	if err != nil {
		return false
	}
	return n == 1
}

func consumeLoginRequestProofSession(c *gin.Context, id string, ts int64) bool {
	sess := sessions.Default(c)
	sid, ok := sess.Get("pending_login_req_proof_id").(string)
	if !ok || sid != id {
		return false
	}
	sts, ok := asInt64Session(sess.Get("pending_login_req_proof_ts"))
	if !ok || sts != ts {
		return false
	}
	sess.Delete("pending_login_req_proof_id")
	sess.Delete("pending_login_req_proof_ts")
	_ = sess.Save()
	return true
}

func asInt64Session(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

func consumeLoginRequestProof(c *gin.Context, id string, ts int64, sigB64 string) bool {
	if id == "" || sigB64 == "" {
		return false
	}
	if err := common.VerifyLoginRequestProof(id, ts, sigB64); err != nil {
		return false
	}
	now := time.Now().Unix()
	if ts < now-300 || ts > now+120 {
		return false
	}
	if common.RedisEnabled && common.RDB != nil {
		return consumeLoginRequestProofRedis(id, ts)
	}
	return consumeLoginRequestProofSession(c, id, ts)
}

// LoginRequestProofIssue 登录前签发一次性防重放凭证与 AES 密钥（密码须用 login_enc_key 加密）。
func LoginRequestProofIssue(c *gin.Context) {
	if !config.PasswordLoginEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员关闭了密码登录",
		})
		return
	}
	if !config.SecurePasswordLoginEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "未启用安全登录，无需获取登录凭证",
		})
		return
	}
	id, ts, sig, encKeyB64, err := prepareLoginRequestProof(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_parameter"),
		})
		return
	}
	if common.RDB == nil {
		if err := sessions.Default(c).Save(); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法保存会话信息，请重试",
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"login_request_id":  id,
			"login_request_ts":  ts,
			"login_request_sig": sig,
			"login_enc_key":     encKeyB64,
		},
	})
}
