package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestHandlerWritesBodyVerbatim 验证非流式 Handler 把上游响应体原样写回客户端，
// 并正确解析出 usage —— 保证「直接 Write」改动与原 reset+io.Copy 行为一致。
func TestHandlerWritesBodyVerbatim(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const body = `{"id":"x","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":22,"total_tokens":33}}`

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	errResp, usage := Handler(c, resp, 11, "gpt-3.5-turbo")
	if errResp != nil {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
	if w.Code != http.StatusOK {
		t.Errorf("status: want 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != body {
		t.Errorf("body not proxied verbatim:\n want %s\n  got %s", body, got)
	}
	if usage == nil || usage.TotalTokens != 33 || usage.PromptTokens != 11 || usage.CompletionTokens != 22 {
		t.Errorf("usage mismatch: %+v", usage)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type header not propagated, got %q", ct)
	}
}
