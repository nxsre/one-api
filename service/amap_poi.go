package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/config"
)

const (
	amapPlaceV5Base = "https://restapi.amap.com/v5/place"
)

// ResolveAmapWebServiceKey 优先环境变量 AMAP_KEY，其次系统选项 AmapWebServiceSecret（不在 GetOptions 中展示）。
func ResolveAmapWebServiceKey() string {
	if k := strings.TrimSpace(os.Getenv("AMAP_KEY")); k != "" {
		return k
	}
	config.OptionMapRWMutex.RLock()
	defer config.OptionMapRWMutex.RUnlock()
	if config.OptionMap == nil {
		return ""
	}
	return strings.TrimSpace(config.OptionMap["AmapWebServiceSecret"])
}

// CallAmapPlaceV5 调用高德 POI 2.0 Web 服务（GET）。suffix 为 text、around、polygon、detail；params 不含 key/output，由本函数追加。
func CallAmapPlaceV5(ctx context.Context, key string, suffix string, params url.Values) (status int, body []byte, err error) {
	if key == "" {
		return 0, nil, fmt.Errorf("高德 Web 服务 Key 未配置（请设置环境变量 AMAP_KEY 或系统选项 AmapWebServiceSecret）")
	}
	suffix = strings.TrimPrefix(strings.TrimSpace(suffix), "/")
	switch suffix {
	case "text", "around", "polygon", "detail":
	default:
		return 0, nil, fmt.Errorf("无效的 place 接口: %q", suffix)
	}
	u, err := url.Parse(amapPlaceV5Base + "/" + suffix)
	if err != nil {
		return 0, nil, err
	}
	q := make(url.Values)
	if params != nil {
		for k, vs := range params {
			q[k] = append(q[k], vs...)
		}
	}
	q.Set("key", key)
	q.Set("output", "json")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, nil, err
	}
	if client.HTTPClient == nil {
		return 0, nil, fmt.Errorf("HTTP 客户端未初始化")
	}
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, b, nil
}
