package common

// outboundSSRF 出站请求统一 SSRF 防护（禁止内网/保留地址），与全局出站白名单叠加使用。
var outboundSSRF = &SSRFProtection{
	AllowPrivateIp:         false,
	DomainFilterMode:       false,
	DomainList:             nil,
	IpFilterMode:           false,
	IpList:                 nil,
	AllowedPorts:           nil,
	ApplyIPFilterForDomain: false,
}

// ValidateOutboundURL 所有出站 HTTP(S) 请求的统一预检：SSRF（可关）+ 全局出站 URL 白名单（若已开启）。
func ValidateOutboundURL(urlStr string) error {
	if IsOutboundSSRFSProtectionEnabled() {
		if err := outboundSSRF.ValidateURL(urlStr); err != nil {
			return err
		}
	}
	return RunGlobalOutboundURLCheck(urlStr)
}
