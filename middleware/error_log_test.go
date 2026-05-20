package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
)

func TestFormatAbortLog_includesUserTokenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.1")
	c.Set(ctxkey.Id, 42)
	c.Set(ctxkey.TokenId, 7)
	c.Set(ctxkey.TokenName, "my-token")
	c.Set(ctxkey.RequestModel, "claude-sonnet-4-6")

	out := formatAbortLog(c, 403, "该令牌无权使用模型：claude-sonnet-4-6")
	for _, sub := range []string{
		"user_id=42",
		"token_id=7",
		"token_name=my-token",
		"model=claude-sonnet-4-6",
		"api=POST /v1/chat/completions",
		"status=403",
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in %q", sub, out)
		}
	}
}
