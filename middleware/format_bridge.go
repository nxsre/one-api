package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

// ResponsesRealtimeFormatBridge 将 Chat Completions 风格字段转换为 Responses / Realtime sessions 兼容字段。
func ResponsesRealtimeFormatBridge() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		p := relaymode.NormalizeAPIPath(c.Request.URL.Path)
		isResponses := strings.HasSuffix(p, "/responses")
		isRTSessions := strings.HasSuffix(p, "/realtime/sessions")
		if !isResponses && !isRTSessions {
			c.Next()
			return
		}
		body, err := io.ReadAll(c.Request.Body)
		if err != nil || len(body) == 0 {
			c.Next()
			return
		}
		var nb []byte
		var changed bool
		if isResponses {
			nb, changed = openai.TryNormalizeResponsesRequest(body)
		} else {
			nb, changed = openai.TryNormalizeRealtimeSessionRequest(body)
		}
		if !changed || len(nb) == 0 {
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			c.Next()
			return
		}
		c.Request.Header.Set("Content-Length", strconv.Itoa(len(nb)))
		c.Request.Body = io.NopCloser(bytes.NewReader(nb))
		c.Next()
	}
}
