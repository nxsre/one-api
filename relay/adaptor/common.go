package adaptor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/requestaudit"
	"github.com/songquanpeng/one-api/relay/meta"
)

func SetupCommonRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) {
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	req.Header.Set("Accept", c.Request.Header.Get("Accept"))
	if meta.IsStream && c.Request.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/event-stream")
	}
}

func DoRequestHelper(a Adaptor, c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	var fullRequestURL string
	var err error
	if meta.OverrideRequestURL != "" {
		fullRequestURL = meta.OverrideRequestURL
	} else {
		fullRequestURL, err = a.GetRequestURL(meta)
		if err != nil {
			return nil, fmt.Errorf("get request url failed: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(requestContext(c), c.Request.Method, fullRequestURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("new request failed: %w", err)
	}
	err = a.SetupRequestHeader(c, req, meta)
	if err != nil {
		return nil, fmt.Errorf("setup request header failed: %w", err)
	}
	exclusive := false
	if v, ok := c.Get(ctxkey.ParamOverrideHeadersExclusive); ok {
		exclusive, _ = v.(bool)
	}
	key := meta.ChannelKey
	if key == "" {
		key = meta.APIKey
	}
	if exclusive {
		if raw, ok := c.Get(ctxkey.ParamOverrideRuntimeHeaders); ok {
			if hdrs, ok := raw.(map[string]string); ok {
				for k, v := range hdrs {
					if k == "" {
						continue
					}
					req.Header.Set(k, strings.ReplaceAll(v, "{api_key}", key))
				}
			}
		}
	} else if raw, ok := c.Get(ctxkey.ChannelHeaderOverride); ok {
		if hdrs, ok := raw.(map[string]string); ok {
			for k, v := range hdrs {
				if k == "" {
					continue
				}
				req.Header.Set(k, strings.ReplaceAll(v, "{api_key}", key))
			}
		}
	}
	resp, err := DoRequest(c, req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}
	return resp, nil
}

func DoRequest(c *gin.Context, req *http.Request) (*http.Response, error) {
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		requestaudit.SnapUpstreamHTTP(c, req, nil)
		return nil, err
	}
	if resp == nil {
		requestaudit.SnapUpstreamHTTP(c, req, nil)
		return nil, errors.New("resp is nil")
	}
	requestaudit.SnapUpstreamHTTP(c, req, resp)
	_ = req.Body.Close()
	_ = c.Request.Body.Close()
	return resp, nil
}

func requestContext(c *gin.Context) context.Context {
	if c == nil || c.Request == nil || c.Request.Context() == nil {
		return context.Background()
	}
	return c.Request.Context()
}
