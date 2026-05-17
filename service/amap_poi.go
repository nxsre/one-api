package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/model"
)

// ResolveAmapWebServiceKey 解析顺序：环境变量 AMAP_KEY → 系统选项 AmapWebServiceSecret → 首个启用的「高德」渠道密钥。
func ResolveAmapWebServiceKey() string {
	if k := strings.TrimSpace(os.Getenv("AMAP_KEY")); k != "" {
		return k
	}
	config.OptionMapRWMutex.RLock()
	var secret string
	if config.OptionMap != nil {
		secret = strings.TrimSpace(config.OptionMap["AmapWebServiceSecret"])
	}
	config.OptionMapRWMutex.RUnlock()
	if secret != "" {
		return secret
	}
	return model.PickAmapWebServiceKeyFromEnabledChannel()
}

// CallAmapPlaceV5 调用高德 POI 搜索 2.0 Web 服务（GET）。suffix 为 text、around、polygon、detail；params 不含 key/output，由上游注入。
func CallAmapPlaceV5(ctx context.Context, key string, suffix string, params url.Values) (status int, body []byte, err error) {
	suffix = strings.TrimPrefix(strings.TrimSpace(suffix), "/")
	switch suffix {
	case "text", "around", "polygon", "detail":
	default:
		return 0, nil, fmt.Errorf("无效的 place 接口: %q", suffix)
	}
	st, _, b, err := CallAmapUpstreamGET(ctx, "/v5/place/"+suffix, params, key)
	return st, b, err
}
