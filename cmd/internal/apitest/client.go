// Package apitest 供 cmd/* 探测脚本调用 one-api HTTP 接口（仅能被 cmd 下的包导入）。
package apitest

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client 使用与用户/SDK 相同的方式访问 one-api（Bearer 令牌）。
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

const defaultHTTPTimeout = 180 * time.Second

// Options 汇总客户端可选项，便于在不破坏既有签名的前提下扩展。
type Options struct {
	SkipTLSVerify bool          // 跳过 HTTPS 服务端证书校验（仅调试或内网自签证书）
	Timeout       time.Duration // HTTP 超时；<=0 时用默认值
	// Proxy 代理地址，支持 http/https/socks5（如 http://127.0.0.1:7890、socks5://127.0.0.1:1080）。
	// 为空时回退到 HTTP_PROXY/HTTPS_PROXY/NO_PROXY 环境变量（http.ProxyFromEnvironment）。
	Proxy string
}

// New 构造客户端；baseURL 为空时用 http://127.0.0.1:3000。
// skipTLSVerify 为 true 时跳过 HTTPS 服务端证书校验（仅调试或内网自签证书；勿在生产滥用）。
func New(baseURL, token string, skipTLSVerify bool) *Client {
	return NewWithTimeout(baseURL, token, skipTLSVerify, defaultHTTPTimeout)
}

// NewWithTimeout 与 New 相同，但可指定 HTTP 超时。
func NewWithTimeout(baseURL, token string, skipTLSVerify bool, timeout time.Duration) *Client {
	cli, _ := NewWithOptions(baseURL, token, Options{SkipTLSVerify: skipTLSVerify, Timeout: timeout})
	return cli
}

// NewWithOptions 用 Options 构造客户端；Proxy 非法时返回 error（客户端仍可用，但不会启用代理）。
func NewWithOptions(baseURL, token string, opts Options) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "http://127.0.0.1:3000"
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}

	// 以 DefaultTransport 为基底克隆，保留连接复用/超时等默认值，再叠加代理与 TLS 设置。
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = http.ProxyFromEnvironment

	var proxyErr error
	if p := strings.TrimSpace(opts.Proxy); p != "" {
		u, err := url.Parse(p)
		if err != nil || u.Scheme == "" || u.Host == "" {
			if err == nil {
				err = fmt.Errorf("代理地址需包含 scheme 与 host，如 http://127.0.0.1:7890")
			}
			proxyErr = fmt.Errorf("解析代理地址 %q 失败: %w", p, err)
		} else {
			tr.Proxy = http.ProxyURL(u)
		}
	}
	if opts.SkipTLSVerify {
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		}
		tr.TLSClientConfig.InsecureSkipVerify = true
	}

	hc := &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   strings.TrimSpace(token),
		HTTP:    hc,
	}, proxyErr
}

// PostJSON POST JSON，path 须以 / 开头（可省略，会自动补全）。
func (c *Client) PostJSON(path string, body any) (status int, respBody []byte, err error) {
	return c.PostJSONWithHeaders(path, body, nil)
}

// PostJSONWithHeaders 在 PostJSON 基础上附加请求头（如 Anthropic-Version）。
func (c *Client) PostJSONWithHeaders(path string, body any, headers map[string]string) (status int, respBody []byte, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, nil, fmt.Errorf("empty path")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	auth := c.Token
	if auth != "" && !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		auth = "Bearer " + auth
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	for k, v := range headers {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, v)
		}
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	return res.StatusCode, b, err
}

// ChatCompletionSnippet 从 OpenAI 兼容响应中提取首条 assistant content；若有 error 字段则返回错误。
func ChatCompletionSnippet(body []byte) (string, error) {
	var v struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		return "", fmt.Errorf("parse JSON: %w; body=%s", err, truncate(string(body), 500))
	}
	if v.Error != nil && strings.TrimSpace(v.Error.Message) != "" {
		return "", fmt.Errorf("%s", v.Error.Message)
	}
	if len(v.Choices) == 0 {
		return "", fmt.Errorf("empty choices; body=%s", truncate(string(body), 500))
	}
	return truncate(strings.TrimSpace(v.Choices[0].Message.Content), 800), nil
}

// AnthropicMessageSnippet 从 Anthropic Messages 响应提取文本；若有 error 字段则返回错误。
func AnthropicMessageSnippet(body []byte) (string, error) {
	var v struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		return "", fmt.Errorf("parse JSON: %w; body=%s", err, truncate(string(body), 500))
	}
	if v.Error != nil && strings.TrimSpace(v.Error.Message) != "" {
		return "", fmt.Errorf("%s", v.Error.Message)
	}
	var b strings.Builder
	for _, c := range v.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", fmt.Errorf("empty content; body=%s", truncate(string(body), 500))
	}
	return truncate(text, 800), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
