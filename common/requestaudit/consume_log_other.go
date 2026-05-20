package requestaudit

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/relayctx"
)

// ConsumeLogOtherJSON 写入消费日志 other：HTTP 头快照、IP、body 大小、用户输入摘要等。
func ConsumeLogOtherJSON(c *gin.Context) string {
	m := map[string]interface{}{}
	if c != nil {
		if c.Writer != nil && c.Writer.Status() > 0 {
			m["http_status"] = c.Writer.Status()
		}
		if c.Request != nil {
			m["http_method"] = c.Request.Method
			if c.Request.URL != nil {
				m["http_path"] = c.Request.URL.Path
				if q := c.Request.URL.RawQuery; q != "" {
					m["http_path"] = c.Request.URL.Path + "?" + q
				}
			}
			if hdrs := AuditHeaderSnapshot(c.Request.Header); len(hdrs) > 0 {
				m["client_request_headers"] = hdrs
			}
		}
		if v, ok := c.Get(ctxkey.ClientResponseHeadersLog); ok {
			if hdrs, ok := v.(map[string][]string); ok && len(hdrs) > 0 {
				m["client_response_headers"] = hdrs
			}
		}
		if v, ok := c.Get(ctxkey.ResponseBodyBytesLog); ok {
			if n, ok := v.(int); ok && n >= 0 {
				m["response_body_bytes"] = n
			}
		}
		if raw, ok := c.Get(ctxkey.KeyRequestBody); ok && raw != nil {
			if b, ok := raw.([]byte); ok {
				m["request_body_bytes"] = len(b)
			}
		}
		ip := relayctx.ClientIP(c.Request.Context())
		if ip == "" && c.Request != nil {
			ip = c.ClientIP()
		}
		if ip != "" {
			m["client_ip"] = ip
		}
		if xff := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); xff != "" {
			m["xff"] = xff
		}
	}
	if hdrJSON := UpstreamHeadersJSONForLog(c); strings.TrimSpace(hdrJSON) != "" {
		var upstream map[string]interface{}
		if err := json.Unmarshal([]byte(hdrJSON), &upstream); err == nil {
			for k, v := range upstream {
				m[k] = v
			}
		}
	}
	if c != nil {
		if v, ok := c.Get(ctxkey.UpstreamRequestMetaLog); ok && v != nil {
			m["upstream_request_meta"] = v
		}
		if v, ok := c.Get(ctxkey.UpstreamResponseMetaLog); ok && v != nil {
			m["upstream_response_meta"] = v
		}
	}
	if config.LogConsumeIncludeUserInput && c != nil {
		if raw, ok := c.Get(ctxkey.KeyRequestBody); ok && raw != nil {
			if b, ok := raw.([]byte); ok && len(b) > 0 {
				max := config.LogConsumeUserInputMaxBytes
				if max <= 0 {
					max = 8192
				}
				if summary := ExtractUserInputSummary(b, max); summary != "" {
					m["user_input"] = summary
				}
			}
		}
	}
	if len(m) == 0 {
		return ""
	}
	out, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(out)
}

// SnapClientResponse 快照返回给客户端的 HTTP 响应头（脱敏）。
func SnapClientResponse(c *gin.Context, w http.ResponseWriter) {
	if c == nil || w == nil {
		return
	}
	c.Set(ctxkey.ClientResponseHeadersLog, AuditHeaderSnapshot(w.Header()))
}
