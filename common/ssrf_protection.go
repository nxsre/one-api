package common

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// SSRFProtection SSRF防护配置
type SSRFProtection struct {
	AllowPrivateIp         bool
	DomainFilterMode       bool     // true: 白名单, false: 黑名单
	DomainList             []string // domain format, e.g. example.com, *.example.com
	IpFilterMode           bool     // true: 白名单, false: 黑名单
	IpList                 []string // CIDR or single IP
	AllowedPorts           []int    // 允许的端口范围
	ApplyIPFilterForDomain bool     // 对域名启用IP过滤
}

// DefaultSSRFProtection 默认SSRF防护配置
var DefaultSSRFProtection = &SSRFProtection{
	AllowPrivateIp:   false,
	DomainFilterMode: true,
	DomainList:       []string{},
	IpFilterMode:     true,
	IpList:           []string{},
	AllowedPorts:     []int{},
}

// privateIPv4Nets IPv4 私有/保留/特殊用途网段
// 参考 IANA IPv4 Special-Purpose Address Registry
// https://www.iana.org/assignments/iana-ipv4-special-registry/
var privateIPv4Nets = []net.IPNet{
	{IP: net.IPv4(0, 0, 0, 0), Mask: net.CIDRMask(8, 32)},       // 0.0.0.0/8 ("This network" / 未指定)
	{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},      // 10.0.0.0/8 (私有)
	{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)},   // 100.64.0.0/10 (运营商级 NAT / CGNAT)
	{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},     // 127.0.0.0/8 (回环)
	{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)},  // 169.254.0.0/16 (链路本地)
	{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},   // 172.16.0.0/12 (私有)
	{IP: net.IPv4(192, 0, 0, 0), Mask: net.CIDRMask(24, 32)},    // 192.0.0.0/24 (IETF 协议分配)
	{IP: net.IPv4(192, 0, 2, 0), Mask: net.CIDRMask(24, 32)},    // 192.0.2.0/24 (TEST-NET-1)
	{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},  // 192.168.0.0/16 (私有)
	{IP: net.IPv4(198, 18, 0, 0), Mask: net.CIDRMask(15, 32)},   // 198.18.0.0/15 (基准测试)
	{IP: net.IPv4(198, 51, 100, 0), Mask: net.CIDRMask(24, 32)}, // 198.51.100.0/24 (TEST-NET-2)
	{IP: net.IPv4(203, 0, 113, 0), Mask: net.CIDRMask(24, 32)},  // 203.0.113.0/24 (TEST-NET-3)
	{IP: net.IPv4(224, 0, 0, 0), Mask: net.CIDRMask(4, 32)},     // 224.0.0.0/4 (组播)
	{IP: net.IPv4(240, 0, 0, 0), Mask: net.CIDRMask(4, 32)},     // 240.0.0.0/4 (保留)
	{IP: net.IPv4(255, 255, 255, 255), Mask: net.CIDRMask(32, 32)}, // 255.255.255.255/32 (受限广播)
}

// privateIPv6Nets IPv6 私有/保留/特殊用途网段
// 参考 IANA IPv6 Special-Purpose Address Registry
// https://www.iana.org/assignments/iana-ipv6-special-registry/
var privateIPv6Nets = func() []net.IPNet {
	cidrs := []string{
		"::/128",        // 未指定地址
		"::1/128",       // 回环
		"::ffff:0:0/96", // IPv4-mapped
		"64:ff9b::/96",  // IPv4/IPv6 translation
		"100::/64",      // Discard-Only
		"2001::/23",     // IETF Protocol Assignments
		"2001:db8::/32", // 文档
		"fc00::/7",      // Unique Local Address (ULA)
		"fe80::/10",     // 链路本地
		"ff00::/8",      // 组播
	}
	nets := make([]net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil && n != nil {
			nets = append(nets, *n)
		}
	}
	return nets
}()

// IsProhibitedSSRFIP 判断 IP 是否属于在 SSRF 场景下应禁止连接的目标
//（私有网段、回环、链路本地、元数据/文档地址等，与 isPrivateIP 一致，供包外如 Dial 钩子等使用）
func IsProhibitedSSRFIP(ip net.IP) bool {
	return isPrivateIP(ip)
}

// isPrivateIP 检查IP是否为私有/保留/特殊用途地址
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// 未指定地址 (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return true
	}
	// 回环、链路本地 (unicast/multicast)
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// 接口本地组播 (IPv6 ff01::/16 等)
	if ip.IsInterfaceLocalMulticast() {
		return true
	}

	if v4 := ip.To4(); v4 != nil {
		for _, privateNet := range privateIPv4Nets {
			if privateNet.Contains(v4) {
				return true
			}
		}
		return false
	}

	// IPv6 检查
	for _, privateNet := range privateIPv6Nets {
		if privateNet.Contains(ip) {
			return true
		}
	}
	// 兜底: Go 标准库识别的其他私有地址
	if ip.IsPrivate() {
		return true
	}
	return false
}

// parsePortRanges 解析端口范围配置
// 支持格式: "80", "443", "8000-9000"
func parsePortRanges(portConfigs []string) ([]int, error) {
	var ports []int

	for _, config := range portConfigs {
		config = strings.TrimSpace(config)
		if config == "" {
			continue
		}

		if strings.Contains(config, "-") {
			// 处理端口范围 "8000-9000"
			parts := strings.Split(config, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid port range format: %s", config)
			}

			startPort, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start port in range %s: %v", config, err)
			}

			endPort, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end port in range %s: %v", config, err)
			}

			if startPort > endPort {
				return nil, fmt.Errorf("invalid port range %s: start port cannot be greater than end port", config)
			}

			if startPort < 1 || startPort > 65535 || endPort < 1 || endPort > 65535 {
				return nil, fmt.Errorf("port range %s contains invalid port numbers (must be 1-65535)", config)
			}

			// 添加范围内的所有端口
			for port := startPort; port <= endPort; port++ {
				ports = append(ports, port)
			}
		} else {
			// 处理单个端口 "80"
			port, err := strconv.Atoi(config)
			if err != nil {
				return nil, fmt.Errorf("invalid port number: %s", config)
			}

			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("invalid port number %d (must be 1-65535)", port)
			}

			ports = append(ports, port)
		}
	}

	return ports, nil
}

// isAllowedPort 检查端口是否被允许
func (p *SSRFProtection) isAllowedPort(port int) bool {
	if len(p.AllowedPorts) == 0 {
		return true // 如果没有配置端口限制，则允许所有端口
	}

	for _, allowedPort := range p.AllowedPorts {
		if port == allowedPort {
			return true
		}
	}
	return false
}

// isDomainWhitelisted 检查域名是否在白名单中
func isDomainListed(domain string, list []string) bool {
	if len(list) == 0 {
		return false
	}

	domain = strings.ToLower(domain)
	for _, item := range list {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		// 精确匹配
		if domain == item {
			return true
		}
		// 通配符匹配 (*.example.com)
		if strings.HasPrefix(item, "*.") {
			suffix := strings.TrimPrefix(item, "*.")
			if strings.HasSuffix(domain, "."+suffix) || domain == suffix {
				return true
			}
		}
	}
	return false
}

func (p *SSRFProtection) isDomainAllowed(domain string) bool {
	listed := isDomainListed(domain, p.DomainList)
	if p.DomainFilterMode { // 白名单
		return listed
	}
	// 黑名单
	return !listed
}

// isIPWhitelisted 检查IP是否在白名单中

func isIPListed(ip net.IP, list []string) bool {
	if len(list) == 0 {
		return false
	}

	return IsIpInCIDRList(ip, list)
}

// IsIPAccessAllowed 检查IP是否允许访问
func (p *SSRFProtection) IsIPAccessAllowed(ip net.IP) bool {
	// 私有IP限制
	if isPrivateIP(ip) && !p.AllowPrivateIp {
		return false
	}

	listed := isIPListed(ip, p.IpList)
	if p.IpFilterMode { // 白名单
		return listed
	}
	// 黑名单
	return !listed
}

// ValidateURL 验证URL是否安全
func (p *SSRFProtection) ValidateURL(urlStr string) error {
	// 解析URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// 只允许HTTP/HTTPS协议
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported protocol: %s (only http/https allowed)", u.Scheme)
	}

	// 解析主机和端口
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		// 没有端口，使用默认端口
		host = u.Hostname()
		if u.Scheme == "https" {
			portStr = "443"
		} else {
			portStr = "80"
		}
	}

	// 验证端口
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %s", portStr)
	}

	if !p.isAllowedPort(port) {
		return fmt.Errorf("port %d is not allowed", port)
	}

	// 如果 host 是 IP，则跳过域名检查
	if ip := net.ParseIP(host); ip != nil {
		if !p.IsIPAccessAllowed(ip) {
			if isPrivateIP(ip) {
				return fmt.Errorf("private IP address not allowed: %s", ip.String())
			}
			if p.IpFilterMode {
				return fmt.Errorf("ip not in whitelist: %s", ip.String())
			}
			return fmt.Errorf("ip in blacklist: %s", ip.String())
		}
		return nil
	}

	// 先进行域名过滤
	if !p.isDomainAllowed(host) {
		if p.DomainFilterMode {
			return fmt.Errorf("domain not in whitelist: %s", host)
		}
		return fmt.Errorf("domain in blacklist: %s", host)
	}

	// 解析主机名对应 IP 并做校验。
	// 当 ApplyIPFilterForDomain 为 false 时，历史实现会跳过 DNS 解析，导致
	// localhost 等可解析为回环/内网的主机名绕过私有地址检查。此处对主机名
	// 始终做解析，并至少实施「非 AllowPrivateIp 则禁止内网/保留 IP」。
	// ApplyIPFilterForDomain 为 true 时，再对解析结果应用 IpList 白/黑名单。
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS resolution failed for %s: %v", host, err)
	}
	for _, ip := range ips {
		if p.ApplyIPFilterForDomain {
			if !p.IsIPAccessAllowed(ip) {
				if isPrivateIP(ip) && !p.AllowPrivateIp {
					return fmt.Errorf("private IP address not allowed: %s resolves to %s", host, ip.String())
				}
				if p.IpFilterMode {
					return fmt.Errorf("ip not in whitelist: %s resolves to %s", host, ip.String())
				}
				return fmt.Errorf("ip in blacklist: %s resolves to %s", host, ip.String())
			}
		} else {
			if isPrivateIP(ip) && !p.AllowPrivateIp {
				return fmt.Errorf("private IP address not allowed: %s resolves to %s", host, ip.String())
			}
		}
	}
	return nil
}

// ValidateURLForRatioSyncFetch 专用于 /api/ratio_sync/fetch 出站请求预检：
// - 不随「关闭全局 SSRF 防护」而跳过校验（即始终做协议与目标检查）；
// - 始终禁止访问内网/保留/特殊目标（忽略 fetch 中的 allow_private_ip）；
// - 仅允许 http、https 协议（见 ValidateURL）；
// 其它（域名/端口/ IP 名单、是否对解析 IP 做名单过滤）仍与 Fetch 系统设置一致，便于在管理端用白名单收紧公网域。
func ValidateURLForRatioSyncFetch(urlStr string, domainFilterMode bool, ipFilterMode bool, domainList, ipList, allowedPorts []string, applyIPFilterForDomain bool) error {
	if err := RunGlobalOutboundURLCheck(urlStr); err != nil {
		return err
	}
	allowedPortInts, err := parsePortRanges(allowedPorts)
	if err != nil {
		return fmt.Errorf("request reject - invalid port configuration: %v", err)
	}
	protection := &SSRFProtection{
		AllowPrivateIp:         false, // 本接口禁止向内网/保留网段发请求
		DomainFilterMode:       domainFilterMode,
		DomainList:             domainList,
		IpFilterMode:           ipFilterMode,
		IpList:                 ipList,
		AllowedPorts:           allowedPortInts,
		ApplyIPFilterForDomain: applyIPFilterForDomain,
	}
	return protection.ValidateURL(urlStr)
}

// ValidateURLWithFetchSetting 使用FetchSetting配置验证URL
func ValidateURLWithFetchSetting(urlStr string, enableSSRFProtection, allowPrivateIp bool, domainFilterMode bool, ipFilterMode bool, domainList, ipList, allowedPorts []string, applyIPFilterForDomain bool) error {
	if err := RunGlobalOutboundURLCheck(urlStr); err != nil {
		return err
	}
	// 如果SSRF防护被禁用，直接返回成功
	if !enableSSRFProtection {
		return nil
	}

	// 解析端口范围配置
	allowedPortInts, err := parsePortRanges(allowedPorts)
	if err != nil {
		return fmt.Errorf("request reject - invalid port configuration: %v", err)
	}

	protection := &SSRFProtection{
		AllowPrivateIp:         allowPrivateIp,
		DomainFilterMode:       domainFilterMode,
		DomainList:             domainList,
		IpFilterMode:           ipFilterMode,
		IpList:                 ipList,
		AllowedPorts:           allowedPortInts,
		ApplyIPFilterForDomain: applyIPFilterForDomain,
	}
	return protection.ValidateURL(urlStr)
}

// RatioSyncDialContext 在 ratio_sync 拉取时于 TCP 层再次拒绝连向私网/保留 IP，并与 github.io 的 IPv4 优先行为兼容。
func RatioSyncDialContext(d *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}
		if IsOutboundSSRFSProtectionEnabled() {
			if ip := net.ParseIP(host); ip != nil && IsProhibitedSSRFIP(ip) {
				return nil, fmt.Errorf("ssrf: connection to %s is not allowed (private or reserved address)", ip)
			}
		}
		if strings.HasSuffix(host, "github.io") {
			if conn, e := d.DialContext(ctx, "tcp4", addr); e == nil {
				return conn, nil
			}
			return d.DialContext(ctx, "tcp6", addr)
		}
		return d.DialContext(ctx, network, addr)
	}
}
