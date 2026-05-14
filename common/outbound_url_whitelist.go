package common

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
)

var (
	globalOutboundURLCheck func(string) error
	outboundCheckMu        sync.RWMutex
)

func SetGlobalOutboundURLCheck(fn func(string) error) {
	outboundCheckMu.Lock()
	defer outboundCheckMu.Unlock()
	globalOutboundURLCheck = fn
}

func RunGlobalOutboundURLCheck(urlStr string) error {
	outboundCheckMu.RLock()
	fn := globalOutboundURLCheck
	outboundCheckMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn(urlStr)
}

func IsIpInCIDRList(ip net.IP, cidrList []string) bool {
	for _, cidr := range cidrList {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			if whitelistIP := net.ParseIP(cidr); whitelistIP != nil {
				if ip.Equal(whitelistIP) {
					return true
				}
			}
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func MatchURLHostAgainstOutboundWhitelist(urlStr string, enabled bool, domainList, ipList []string) error {
	if !enabled {
		return nil
	}
	if len(domainList) == 0 && len(ipList) == 0 {
		return fmt.Errorf("全局出站 URL 白名单已开启但未配置任何域名或 IP 段")
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "ws" && u.Scheme != "wss" {
		return fmt.Errorf("仅允许 http、https、ws、wss 协议，当前为 %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url 中缺少 host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if len(ipList) == 0 {
			return fmt.Errorf("使用 IP 直链的 URL 未在白名单内: %s", ip)
		}
		if IsIpInCIDRList(ip, ipList) {
			return nil
		}
		return fmt.Errorf("IP %s 不在出站白名单的 IP/CIDR 列表中", ip)
	}
	if len(domainList) > 0 && outboundHostMatchesDomainList(host, domainList) {
		return nil
	}
	if len(ipList) > 0 {
		ips, lerr := net.LookupIP(host)
		if lerr != nil {
			return fmt.Errorf("解析域名失败: %w", lerr)
		}
		for _, a := range ips {
			if IsIpInCIDRList(a, ipList) {
				return nil
			}
		}
	}
	return fmt.Errorf("主机名 %q 未命中出站白名单的域名或解析 IP 段", host)
}

func outboundHostMatchesDomainList(domain string, list []string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	for _, item := range list {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if domain == item {
			return true
		}
		if strings.HasPrefix(item, "*.") {
			suffix := strings.TrimPrefix(item, "*.")
			if strings.HasSuffix(domain, "."+suffix) || domain == suffix {
				return true
			}
		}
	}
	return false
}

func splitOutboundList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ';'
	}) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
