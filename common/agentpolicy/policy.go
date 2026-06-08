// Package agentpolicy 定义"按 agent 客户端类型"的访问与限流策略，供全局（管理员 option）、
// 令牌、用户、租户四个层级共用同一份结构。
//
// 语义：
//   - AllowedClients 为空 = 放开所有客户端类型；非空 = 仅允许其中列出的类型（白名单）。
//   - Rules[client] 为某类型的细则：是否禁用、限流阈值。
//   - 层级优先级为"就近覆盖"：令牌 > 用户 > 租户（在 Effective* 函数中实现）。
package agentpolicy

import (
	"encoding/json"
	"strings"
	"sync/atomic"

	"github.com/songquanpeng/one-api/common/config"
)

// enabled 为进程级总开关：仅当全局/任一令牌/用户/租户配置了非空策略时为 true。
// 中继热路径据此快速短路，未启用本功能时零额外开销。启动时探测一次，保存策略时置位。
var enabled atomic.Bool

// SetEnabled 标记策略功能已启用（保存任意层级策略时调用）。
func SetEnabled(v bool) { enabled.Store(v) }

// Enabled 报告策略功能是否已启用。
func Enabled() bool { return enabled.Load() }

// OtherClient 是未识别为已知 agent 的兜底类别（普通 API/SDK 直连）。
// 配置白名单时包含它即可放行普通调用。
const OtherClient = "other"

// ClientRule 为单个客户端类型的细则。
type ClientRule struct {
	// Disabled 为 true 时直接拒绝该类型请求。
	Disabled bool `json:"disabled,omitempty"`
	// MaxRequests / WindowSec：窗口期内最大请求数。MaxRequests<=0 表示不限流。
	MaxRequests int `json:"max_requests,omitempty"`
	WindowSec   int `json:"window_sec,omitempty"`
}

// Policy 为某一层级（全局/令牌/用户/租户）的完整策略。
type Policy struct {
	// AllowedClients 为空表示放开所有类型；非空表示白名单。
	AllowedClients []string `json:"allowed_clients,omitempty"`
	// Rules：按客户端类型的禁用/限流细则（精确覆盖 Default）。
	Rules map[string]ClientRule `json:"rules,omitempty"`
	// Default：对未在 Rules 中单列的类型生效的默认细则（便于"本令牌限速 N 次/W 秒"这类简单配置）。
	Default *ClientRule `json:"default,omitempty"`
}

// IsZero 报告策略是否为空（未配置）。
func (p *Policy) IsZero() bool {
	return p == nil || (len(p.AllowedClients) == 0 && len(p.Rules) == 0 && p.Default == nil)
}

// EffectiveRule 返回某类型的生效细则：优先精确 Rules，其次 Default，再次零值。
func (p *Policy) EffectiveRule(client string) ClientRule {
	if p == nil {
		return ClientRule{}
	}
	if p.Rules != nil {
		if r, ok := p.Rules[client]; ok {
			return r
		}
	}
	if p.Default != nil {
		return *p.Default
	}
	return ClientRule{}
}

// Allows 报告该层级是否允许给定客户端类型。AllowedClients 为空视为放开所有。
func (p *Policy) Allows(client string) bool {
	if p == nil || len(p.AllowedClients) == 0 {
		return true
	}
	for _, c := range p.AllowedClients {
		if strings.EqualFold(strings.TrimSpace(c), client) || c == "*" {
			return true
		}
	}
	return false
}

// HasAllowList 报告该层级是否设置了白名单（用于"就近覆盖"判定）。
func (p *Policy) HasAllowList() bool {
	return p != nil && len(p.AllowedClients) > 0
}

// Rule 返回某客户端类型的细则（不存在则返回零值）。
func (p *Policy) Rule(client string) ClientRule {
	if p == nil || p.Rules == nil {
		return ClientRule{}
	}
	return p.Rules[client]
}

// Parse 从 JSON 字符串解析策略；空串或 "{}" 返回零值策略。
func Parse(raw string) *Policy {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" || raw == "null" {
		return &Policy{}
	}
	var p Policy
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return &Policy{}
	}
	return &p
}

// Global 返回全局策略（管理员在后台配置，存于 option AgentClientPolicy）。
func Global() *Policy {
	config.OptionMapRWMutex.RLock()
	raw := config.OptionMap[OptionKey]
	config.OptionMapRWMutex.RUnlock()
	return Parse(raw)
}

// OptionKey 为全局策略在 OptionMap / options 表中的键名。
const OptionKey = "AgentClientPolicy"

// Decision 为一次访问判定的结果。
type Decision struct {
	// Client 为最终归类的客户端类型（识别值或 OtherClient）。
	Client string
	// Blocked 为 true 表示应拒绝；Reason 给出原因（全局禁用/不在白名单）。
	Blocked bool
	Reason  string
}

// Evaluate 按"就近覆盖"合并各层级白名单，并叠加全局禁用，得出是否放行。
// token/user/tenant 任意可为 nil（表示该层级未配置）。limit 计算交由调用方按各层 Rule 处理。
func Evaluate(global, token, user, tenant *Policy, detected string) Decision {
	client := detected
	if client == "" {
		client = OtherClient
	}
	d := Decision{Client: client}

	// 全局禁用优先级最高。
	if global != nil {
		if global.EffectiveRule(client).Disabled {
			d.Blocked = true
			d.Reason = "globally disabled"
			return d
		}
		if !global.Allows(client) {
			d.Blocked = true
			d.Reason = "not in global allow list"
			return d
		}
	}

	// 就近覆盖：令牌 > 用户 > 租户，取第一个设置了白名单的层级判定。
	switch {
	case token.HasAllowList():
		if !token.Allows(client) {
			d.Blocked, d.Reason = true, "not allowed for this token"
		}
	case user.HasAllowList():
		if !user.Allows(client) {
			d.Blocked, d.Reason = true, "not allowed for this user"
		}
	case tenant.HasAllowList():
		if !tenant.Allows(client) {
			d.Blocked, d.Reason = true, "not allowed for this tenant"
		}
	}
	return d
}
