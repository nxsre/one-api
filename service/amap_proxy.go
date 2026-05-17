package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/songquanpeng/one-api/common/client"
)

// UpstreamHTTPSnap 可选；非 nil 时在 HTTP 往返后调用（用于审计上游请求/响应头）。
type UpstreamHTTPSnap func(req *http.Request, resp *http.Response)

const amapRESTBase = "https://restapi.amap.com"

// 允许通过 one-api 代理的高德 REST 路径前缀（客户端令牌鉴权后由服务端注入 Key）。
// 涵盖：路径规划 v3/v4、路径规划 2.0（v5/direction）、POI 2.0（v5/place）。
var amapRelayPathPrefixes = []string{
	"/v3/direction/",
	"/v4/direction/",
	"/v5/direction/",
	"/v5/place/",
}

// NormalizeAmapRelayPath 规范化客户端传入的路径（必须为绝对路径，禁止逃逸）。
func NormalizeAmapRelayPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = path.Clean(p)
	if p == "/" || p == "." {
		return ""
	}
	return p
}

// IsAllowedAmapRelayPath 是否允许代理到该路径。
func IsAllowedAmapRelayPath(p string) bool {
	n := NormalizeAmapRelayPath(p)
	if n == "" {
		return false
	}
	for _, pre := range amapRelayPathPrefixes {
		if strings.HasPrefix(n, pre) {
			return true
		}
	}
	return false
}

func cloneURLValues(v url.Values) url.Values {
	if v == nil {
		return url.Values{}
	}
	out := make(url.Values)
	for k, vs := range v {
		out[k] = append([]string(nil), vs...)
	}
	return out
}

func stripClientKey(q url.Values) {
	q.Del("key")
}

func mergeInjectKeyOutput(q url.Values, serverKey string) url.Values {
	v := cloneURLValues(q)
	stripClientKey(v)
	v.Set("key", serverKey)
	if strings.TrimSpace(v.Get("output")) == "" {
		v.Set("output", "json")
	}
	return v
}

func resolveOrUseAmapKey(serverKey string) (string, error) {
	k := strings.TrimSpace(serverKey)
	if k != "" {
		return k, nil
	}
	k = ResolveAmapWebServiceKey()
	if k == "" {
		return "", fmt.Errorf("高德 Web 服务 Key 未配置：请设置环境变量 AMAP_KEY、系统选项 AmapWebServiceSecret，或在后台添加已启用的「高德」渠道并填写高德 Web 服务 Key")
	}
	return k, nil
}

// CallAmapUpstreamGET 向 https://restapi.amap.com{relayPath} 发起 GET。params 不含 key，由本函数注入。
// serverKey 非空时优先使用，否则走 ResolveAmapWebServiceKey。
// snap 可选，用于记录上游 HTTP 头（如 requestaudit）。
func CallAmapUpstreamGET(ctx context.Context, relayPath string, params url.Values, serverKey string, snap ...UpstreamHTTPSnap) (status int, contentType string, body []byte, err error) {
	key, err := resolveOrUseAmapKey(serverKey)
	if err != nil {
		return 0, "", nil, err
	}
	p := NormalizeAmapRelayPath(relayPath)
	if !IsAllowedAmapRelayPath(p) {
		return 0, "", nil, fmt.Errorf("不允许代理的路径 %q：仅允许前缀 %v", relayPath, amapRelayPathPrefixes)
	}
	q := mergeInjectKeyOutput(params, key)
	reqURL := amapRESTBase + p + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, "", nil, err
	}
	var fn UpstreamHTTPSnap
	if len(snap) > 0 {
		fn = snap[0]
	}
	return doAmapUpstream(req, fn)
}

// CallAmapUpstreamPOSTForm 向高德发起 POST（application/x-www-form-urlencoded）。
// query 与解析后的 body 表单合并后注入 key；适用于 v5 驾车等「参数过长改用 POST」的场景。
func CallAmapUpstreamPOSTForm(ctx context.Context, relayPath string, query url.Values, formBody string, serverKey string, snap ...UpstreamHTTPSnap) (status int, contentType string, body []byte, err error) {
	key, err := resolveOrUseAmapKey(serverKey)
	if err != nil {
		return 0, "", nil, err
	}
	p := NormalizeAmapRelayPath(relayPath)
	if !IsAllowedAmapRelayPath(p) {
		return 0, "", nil, fmt.Errorf("不允许代理的路径 %q：仅允许前缀 %v", relayPath, amapRelayPathPrefixes)
	}
	values, err := url.ParseQuery(formBody)
	if err != nil {
		return 0, "", nil, fmt.Errorf("解析表单 body 失败: %w", err)
	}
	for k, vs := range query {
		for _, x := range vs {
			values.Add(k, x)
		}
	}
	encoded := mergeInjectKeyOutput(values, key).Encode()
	reqURL := amapRESTBase + p
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(encoded))
	if err != nil {
		return 0, "", nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var fn UpstreamHTTPSnap
	if len(snap) > 0 {
		fn = snap[0]
	}
	return doAmapUpstream(req, fn)
}

func doAmapUpstream(req *http.Request, snap UpstreamHTTPSnap) (status int, contentType string, body []byte, err error) {
	if client.HTTPClient == nil {
		return 0, "", nil, fmt.Errorf("HTTP 客户端未初始化")
	}
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		if snap != nil {
			snap(req, nil)
		}
		return 0, "", nil, err
	}
	defer resp.Body.Close()
	if snap != nil {
		snap(req, resp)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", nil, err
	}
	ct := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if ct == "" {
		ct = "application/json; charset=utf-8"
	}
	return resp.StatusCode, ct, b, nil
}
