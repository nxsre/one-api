package client

import (
	"net/http"
	"time"

	"github.com/songquanpeng/one-api/common"
)

type outboundRoundTripper struct {
	base http.RoundTripper
}

func (o *outboundRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req != nil && req.URL != nil {
		if err := common.ValidateOutboundURL(req.URL.String()); err != nil {
			return nil, err
		}
	}
	base := o.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// WrapOutboundRoundTripper 为任意 RoundTripper 叠加统一出站 URL 校验。
func WrapOutboundRoundTripper(base http.RoundTripper) http.RoundTripper {
	return &outboundRoundTripper{base: base}
}

// NewOutboundHTTPClient 创建带统一出站白名单/SSRF 校验的 HTTP 客户端。
func NewOutboundHTTPClient(timeout time.Duration) *http.Client {
	return NewOutboundHTTPClientWithTransport(nil, timeout)
}

// NewOutboundHTTPClientWithTransport 基于自定义 Transport（如 ratio_sync 的 DialContext）创建出站客户端。
func NewOutboundHTTPClientWithTransport(base *http.Transport, timeout time.Duration) *http.Client {
	var inner http.RoundTripper
	if base != nil {
		inner = base
	} else {
		inner = http.DefaultTransport.(*http.Transport).Clone()
	}
	c := &http.Client{Transport: WrapOutboundRoundTripper(inner)}
	if timeout > 0 {
		c.Timeout = timeout
	}
	return c
}
