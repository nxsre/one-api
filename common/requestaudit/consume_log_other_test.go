package requestaudit

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/relayctx"
)

func TestConsumeLogOtherJSON_headersAndSizes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Authorization", "Bearer sk-secret")
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	c.Request.Header.Set("Host", "api.example.com")
	ctx := relayctx.WithClientIP(c.Request.Context(), "10.0.0.5")
	c.Request = c.Request.WithContext(ctx)
	c.Set(ctxkey.KeyRequestBody, []byte(`{"model":"gpt-4"}`))
	c.Set(ctxkey.ClientResponseHeadersLog, map[string][]string{
		"Content-Type": {"application/json"},
	})
	c.Set(ctxkey.ResponseBodyBytesLog, 128)
	c.Set(ctxkey.UpstreamRequestHeadersLog, map[string][]string{
		"Content-Type": {"application/json"},
	})
	c.Set(ctxkey.UpstreamResponseHeadersLog, map[string][]string{
		"X-Request-Id": {"req-abc"},
	})
	c.Set(ctxkey.UpstreamRequestMetaLog, map[string]interface{}{
		"method": "POST",
		"url":    "https://api.openai.com/v1/chat/completions",
	})
	c.Set(ctxkey.UpstreamResponseMetaLog, map[string]interface{}{
		"status":      "200 OK",
		"status_code": 200,
	})
	w.WriteHeader(200)

	out := ConsumeLogOtherJSON(c)
	if out == "" {
		t.Fatal("expected non-empty other json")
	}
	if strings.Contains(out, "sk-secret") {
		t.Fatal("authorization must be redacted")
	}
	for _, want := range []string{
		"client_request_headers",
		"client_response_headers",
		"upstream_request_headers",
		"upstream_response_headers",
		"client_ip",
		"xff",
		"request_body_bytes",
		"response_body_bytes",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
}
