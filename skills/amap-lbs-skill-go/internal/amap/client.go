package amap

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client 经网关的 POST /amap 访问高德能力（Bearer apikey）。
// 网关会剥离客户端 key 并注入高德 Web Service Key，因此本客户端无需持有高德 Key。
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// New 从配置文件构造客户端。configPath 为空时按默认顺序查找
// （--config > AMAP_SKILL_CONFIG > $HOME/.openclaw/.amap-lbs-skill.json > /home/node/.openclaw/.amap-lbs-skill.json > ./.amap-lbs-skill.json）。
func New(configPath string) (*Client, error) {
	cfg, path, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	cred, ok := cfg.credential()
	if !ok {
		src := path
		if src == "" {
			src = DefaultConfigPath()
		}
		return nil, fmt.Errorf("未配置凭证：请在 %s 写入 base_url/apikey，或设置环境变量 %s / %s",
			src, envBaseURL, envAPIKey)
	}
	hc := &http.Client{Timeout: 60 * time.Second}
	if cred.Insecure {
		tr := http.DefaultTransport.(*http.Transport).Clone()
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		}
		tr.TLSClientConfig.InsecureSkipVerify = true
		hc.Transport = tr
	}
	return &Client{BaseURL: cred.Base, Token: cred.Key, HTTP: hc}, nil
}

// amapRequest 与网关 /amap 的请求体约定对齐。
// 不要在 Query 里传 key——网关会丢弃并注入自己的高德 Key。
type amapRequest struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Query  map[string]string `json:"query,omitempty"`
	Body   string            `json:"body,omitempty"`
}

// callAmap 向网关 POST /amap，返回上游 HTTP 状态码与原始响应体。
// 当网关自身返回 {error:{...}} 信封时作为错误返回。
func (c *Client) callAmap(req amapRequest) (int, []byte, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return 0, nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.BaseURL+"/amap", bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	auth := c.Token
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		auth = "Bearer " + auth
	}
	httpReq.Header.Set("Authorization", auth)

	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return 0, nil, fmt.Errorf("请求失败: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 网关层错误信封：{"error":{"message":...,"type":...}}。
	// 高德上游成功响应不含顶层 error 字段，故可据此区分。
	var env struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(body, &env) == nil && len(env.Error) > 0 && string(env.Error) != "null" {
		return res.StatusCode, body, fmt.Errorf("网关错误: %s", gatewayErrMessage(env.Error))
	}
	return res.StatusCode, body, nil
}

func gatewayErrMessage(errRaw json.RawMessage) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(errRaw, &e) == nil && strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return string(errRaw)
}
