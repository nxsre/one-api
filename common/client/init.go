package client

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
)

var HTTPClient *http.Client
var ImpatientHTTPClient *http.Client
var UserContentRequestHTTPClient *http.Client

type outboundCheckTransport struct {
	*http.Transport
}

func (o *outboundCheckTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req != nil && req.URL != nil {
		if err := common.RunGlobalOutboundURLCheck(req.URL.String()); err != nil {
			return nil, err
		}
	}
	return o.Transport.RoundTrip(req)
}

func wrapRelayTransport(t *http.Transport) http.RoundTripper {
	if t == nil {
		return nil
	}
	return &outboundCheckTransport{Transport: t}
}

func Init() {
	if config.UserContentRequestProxy != "" {
		logger.SysLog(fmt.Sprintf("using %s as proxy to fetch user content", config.UserContentRequestProxy))
		proxyURL, err := url.Parse(config.UserContentRequestProxy)
		if err != nil {
			logger.FatalLog(fmt.Sprintf("USER_CONTENT_REQUEST_PROXY set but invalid: %s", config.UserContentRequestProxy))
		}
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		UserContentRequestHTTPClient = &http.Client{
			Transport: transport,
			Timeout:   time.Second * time.Duration(config.UserContentRequestTimeout),
		}
	} else {
		UserContentRequestHTTPClient = &http.Client{}
	}

	base := http.DefaultTransport.(*http.Transport).Clone()
	if config.RelayProxy != "" {
		logger.SysLog(fmt.Sprintf("using %s as api relay proxy", config.RelayProxy))
		proxyURL, err := url.Parse(config.RelayProxy)
		if err != nil {
			logger.FatalLog(fmt.Sprintf("RELAY_PROXY set but invalid: %s", config.RelayProxy))
		}
		base.Proxy = http.ProxyURL(proxyURL)
	}
	relayTransport := wrapRelayTransport(base)

	if config.RelayTimeout == 0 {
		HTTPClient = &http.Client{Transport: relayTransport}
	} else {
		HTTPClient = &http.Client{
			Timeout:   time.Duration(config.RelayTimeout) * time.Second,
			Transport: relayTransport,
		}
	}

	ImpatientHTTPClient = &http.Client{
		Timeout:   5 * time.Second,
		Transport: relayTransport,
	}
}
