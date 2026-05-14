package common

import (
	"github.com/songquanpeng/one-api/common/config"
)

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
