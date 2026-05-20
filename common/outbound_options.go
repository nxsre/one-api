package common

import (
	"github.com/songquanpeng/one-api/common/config"
)

// IsOutboundSSRFSProtectionEnabled 是否启用出站 SSRF 防护（禁止内网/保留地址等）。
// 未配置时默认开启。
func IsOutboundSSRFSProtectionEnabled() bool {
	config.OptionMapRWMutex.RLock()
	defer config.OptionMapRWMutex.RUnlock()
	v, ok := config.OptionMap["OutboundSSRFSProtectionEnabled"]
	if !ok {
		return true
	}
	return v == "true"
}

// RefreshOutboundWhitelistFromOptions 根据 options 表刷新出站 URL 校验函数。
func RefreshOutboundWhitelistFromOptions() {
	config.OptionMapRWMutex.RLock()
	enabled := config.OptionMap["OutboundURLWhitelistEnabled"] == "true"
	domains := splitOutboundList(config.OptionMap["OutboundURLWhitelistDomains"])
	ips := splitOutboundList(config.OptionMap["OutboundURLWhitelistIPs"])
	config.OptionMapRWMutex.RUnlock()

	SetGlobalOutboundURLCheck(func(u string) error {
		return MatchURLHostAgainstOutboundWhitelist(u, enabled, domains, ips)
	})
}
