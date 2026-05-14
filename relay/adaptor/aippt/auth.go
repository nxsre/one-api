package aippt

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

// 以下鉴权与 openclaw aipptskill/scripts/aippt.sh 保持一致。
//
// generate_signature() {
//   local sign_str="${1}@${2}@${3}"
//   echo -n "$sign_str" | openssl dgst -sha1 -hmac "$SECRET_KEY" -binary | base64
// }
//
// 仅申请 token 的 GET /api/grant/token/ 需要签名字段；其余业务请求为 x-api-key + x-channel(空) + x-token，见 api_get / api_post。

// GrantTokenPath 必须与签名中的 path 一致（含尾斜杠），与脚本中
//
//	generate_signature "GET" "/api/grant/token/" "$ts"
//
// 的第二个参数相同。
const GrantTokenPath = "/api/grant/token/"

// DefaultTimeExpireSeconds 对应脚本中 data.get('time_expire', 259200) 的缺省值（秒）
const DefaultTimeExpireSeconds = 259200

// GenerateGrantSignature 对应 aippt.sh 的 generate_signature，用于拉取 access token。
// method 为 "GET"，path 为 GrantTokenPath，ts 为 Unix 秒，与 x-timestamp 相同。
func GenerateGrantSignature(secretKey, method, path string, ts int64) string {
	signStr := method + "@" + path + "@" + strconv.FormatInt(ts, 10)
	mac := hmac.New(sha1.New, []byte(secretKey))
	_, _ = mac.Write([]byte(signStr))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// tokenCache 复刻 .token_cache.json 行为：{ token, expire_time }，expire_time 为 Unix 秒
type tokenCache struct {
	mu     sync.Mutex
	token  string
	expire int64 // 过期时间 Unix 秒，与 bash 中 $(date +%s)+time_expire 一致
}

func (c *tokenCache) get() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == "" {
		return "", false
	}
	now := time.Now().Unix()
	if c.expire > now {
		return c.token, true
	}
	return "", false
}

func (c *tokenCache) set(token string, body []byte) {
	ttl := gjson.GetBytes(body, "data.time_expire").Int()
	if ttl <= 0 {
		ttl = int64(DefaultTimeExpireSeconds)
	}
	expire := time.Now().Unix() + ttl
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
	c.expire = expire
}

// GrantSuccessCode 与脚本 check_resp 一致，code 为 0 表示成功（JSON 为数字或字符串 "0"）
func GrantSuccessCode(body []byte) bool {
	v := gjson.GetBytes(body, "code")
	return v.Int() == 0 || strings.TrimSpace(v.String()) == "0"
}
