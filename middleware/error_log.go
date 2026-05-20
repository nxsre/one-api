package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/relayctx"
	"github.com/songquanpeng/one-api/common/requestaudit"
	"github.com/songquanpeng/one-api/model"
)

func bindTokenIdentity(c *gin.Context, token *model.Token) {
	if c == nil || token == nil {
		return
	}
	c.Set(ctxkey.Id, token.UserId)
	c.Set(ctxkey.TokenId, token.Id)
	c.Set(ctxkey.TokenName, token.Name)
	c.Set(ctxkey.UserTenantID, model.GetUserTenantIDNumeric(token.UserId))
}

func requestAPIPath(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return ""
	}
	p := c.Request.URL.Path
	if q := c.Request.URL.RawQuery; q != "" {
		return p + "?" + q
	}
	return p
}

func formatAbortLog(c *gin.Context, statusCode int, message string) string {
	parts := []string{message}
	parts = append(parts, fmt.Sprintf("status=%d", statusCode))
	if uid := c.GetInt(ctxkey.Id); uid > 0 {
		parts = append(parts, fmt.Sprintf("user_id=%d", uid))
		if name := model.GetUsernameById(uid); name != "" {
			parts = append(parts, fmt.Sprintf("username=%s", name))
		}
	}
	if tid := c.GetInt(ctxkey.TokenId); tid > 0 {
		parts = append(parts, fmt.Sprintf("token_id=%d", tid))
	}
	if tn := strings.TrimSpace(c.GetString(ctxkey.TokenName)); tn != "" {
		parts = append(parts, fmt.Sprintf("token_name=%s", tn))
	}
	if m := strings.TrimSpace(c.GetString(ctxkey.RequestModel)); m != "" {
		parts = append(parts, fmt.Sprintf("model=%s", m))
	}
	if c != nil && c.Request != nil {
		parts = append(parts, fmt.Sprintf("api=%s %s", c.Request.Method, requestAPIPath(c)))
	}
	ip := ""
	if c != nil {
		ip = relayctx.ClientIP(c.Request.Context())
		if ip == "" {
			ip = c.ClientIP()
		}
	}
	if ip != "" {
		parts = append(parts, fmt.Sprintf("client_ip=%s", ip))
	}
	if c != nil {
		if xff := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); xff != "" {
			parts = append(parts, fmt.Sprintf("xff=%s", xff))
		}
	}
	return strings.Join(parts, " | ")
}

// recordAPIRejectLog 将网关层拒绝（鉴权/模型白名单等）写入操作日志 type=错误。
func recordAPIRejectLog(c *gin.Context, statusCode int, message string) {
	if c == nil {
		return
	}
	userId := c.GetInt(ctxkey.Id)
	content := fmt.Sprintf("[%d] %s", statusCode, strings.TrimSpace(message))
	if content == "" || content == fmt.Sprintf("[%d] ", statusCode) {
		content = fmt.Sprintf("[%d] API rejected", statusCode)
	}
	api := ""
	if c.Request != nil {
		api = fmt.Sprintf("%s %s", c.Request.Method, requestAPIPath(c))
		if api != " " {
			content = fmt.Sprintf("%s | api=%s", content, strings.TrimSpace(api))
		}
	}
	if tn := strings.TrimSpace(c.GetString(ctxkey.TokenName)); tn != "" {
		content = fmt.Sprintf("%s | token=%s", content, tn)
	}
	if tid := c.GetInt(ctxkey.TokenId); tid > 0 {
		content = fmt.Sprintf("%s | token_id=%d", content, tid)
	}
	model.RecordErrorLog(c.Request.Context(), &model.Log{
		UserId:      userId,
		Content:     content,
		ModelName:   strings.TrimSpace(c.GetString(ctxkey.RequestModel)),
		TokenName:   c.GetString(ctxkey.TokenName),
		TokenId:     c.GetInt(ctxkey.TokenId),
		Group:       c.GetString(ctxkey.Group),
		UseTime:     1,
		ElapsedTime: 0,
		Other:       requestaudit.ConsumeLogOtherJSON(c),
	})
}
