package controller

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
)

// channelModelTestHTTPDetail 单次模型测试的 HTTP 往返详情（供测试报告展示）。
type channelModelTestHTTPDetail struct {
	RequestMethod   string              `json:"request_method,omitempty"`
	RequestURL      string              `json:"request_url,omitempty"`
	RequestPath     string              `json:"request_path,omitempty"`
	RequestBody     string              `json:"request_body,omitempty"`
	RequestHeaders  map[string][]string `json:"request_headers,omitempty"`
	WireProtocol    string              `json:"wire_protocol,omitempty"`
	ResponseStatus  int                 `json:"response_status,omitempty"`
	ResponseHeaders map[string][]string `json:"response_headers,omitempty"`
	ResponseBody    string              `json:"response_body,omitempty"`
}

func captureChannelModelTestHTTPDetail(
	c *gin.Context,
	requestPath, requestBody, responseBody, wireProtocol string,
	httpStatus int,
	upstreamURL string,
) channelModelTestHTTPDetail {
	d := channelModelTestHTTPDetail{
		RequestMethod:  "POST",
		RequestPath:    requestPath,
		RequestURL:     upstreamURL,
		RequestBody:    requestBody,
		ResponseBody:   responseBody,
		ResponseStatus: httpStatus,
		WireProtocol:   wireProtocol,
	}
	if c == nil {
		return d
	}
	if v, ok := c.Get(ctxkey.UpstreamRequestMetaLog); ok {
		if m, ok := v.(map[string]interface{}); ok {
			if method, ok := m["method"].(string); ok && method != "" {
				d.RequestMethod = method
			}
			if url, ok := m["url"].(string); ok && url != "" {
				d.RequestURL = url
			}
		}
	}
	if v, ok := c.Get(ctxkey.UpstreamRequestHeadersLog); ok {
		if hdrs, ok := v.(map[string][]string); ok && len(hdrs) > 0 {
			d.RequestHeaders = hdrs
		}
	}
	if v, ok := c.Get(ctxkey.UpstreamResponseHeadersLog); ok {
		if hdrs, ok := v.(map[string][]string); ok && len(hdrs) > 0 {
			d.ResponseHeaders = hdrs
		}
	}
	if d.ResponseStatus == 0 {
		if v, ok := c.Get(ctxkey.UpstreamResponseMetaLog); ok {
			if m, ok := v.(map[string]interface{}); ok {
				if code, ok := m["status_code"].(int); ok {
					d.ResponseStatus = code
				}
			}
		}
	}
	return d
}

func marshalChannelModelTestDetail(d channelModelTestHTTPDetail) string {
	if d.RequestBody == "" && d.ResponseBody == "" && len(d.RequestHeaders) == 0 && len(d.ResponseHeaders) == 0 &&
		d.RequestURL == "" && d.RequestPath == "" && d.WireProtocol == "" {
		return ""
	}
	b, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	return string(b)
}

func parseChannelModelTestDetail(raw string) *channelModelTestHTTPDetail {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var d channelModelTestHTTPDetail
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return nil
	}
	return &d
}
