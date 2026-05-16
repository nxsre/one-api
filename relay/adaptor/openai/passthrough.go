package openai

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/relay/model"
)

// PassthroughUpstreamResponse 将上游 HTTP 响应原样写给客户端（用于 Responses / Realtime sessions 等形态与 Chat Completions 解析器不兼容的接口）。
func PassthroughUpstreamResponse(c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		return nil, ErrorWrapper(err, "upstream_passthrough_failed", http.StatusBadGateway)
	}
	// 透明转发时不解析 usage；计费侧 total_tokens=0 会退回预扣（见 postConsumeQuota）。
	return &model.Usage{}, nil
}
