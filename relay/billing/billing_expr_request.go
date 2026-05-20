package billing

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/pkg/billingexpr"
)

func ResolveIncomingBillingExprRequestInput(c *gin.Context) (billingexpr.RequestInput, error) {
	if c == nil {
		return billingexpr.RequestInput{}, nil
	}
	if raw, ok := c.Get(ctxkey.BillingRequestInput); ok && raw != nil {
		if input, ok := raw.(*billingexpr.RequestInput); ok && input != nil {
			return cloneRequestInput(*input), nil
		}
	}
	input := billingexpr.RequestInput{}
	if raw, ok := c.Get(ctxkey.KeyRequestBody); ok && raw != nil {
		if b, ok := raw.([]byte); ok {
			input.Body = append([]byte(nil), b...)
		}
	}
	if c.Request != nil {
		input.Headers = requestHeadersFlat(c)
	}
	return input, nil
}

func requestHeadersFlat(c *gin.Context) map[string]string {
	h := make(map[string]string)
	if c == nil || c.Request == nil {
		return h
	}
	for k, vv := range c.Request.Header {
		if len(vv) > 0 {
			h[k] = vv[0]
		}
	}
	return h
}

func cloneRequestInput(src billingexpr.RequestInput) billingexpr.RequestInput {
	input := billingexpr.RequestInput{Headers: cloneStringMap(src.Headers)}
	if len(src.Body) > 0 {
		input.Body = append([]byte(nil), src.Body...)
	}
	return input
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func isJSONContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	return strings.HasPrefix(contentType, "application/json")
}
