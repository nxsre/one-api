// Package routing 实现模型解析后的渠道选择、重试策略配置与 Redis 观测状态。
package routing

import (
	"encoding/json"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
)

// SelectionMode 在同优先级层内的选择方式。
type SelectionMode string

const (
	SelectionWeightedRandom SelectionMode = "weighted_random"
	SelectionConsistentHash SelectionMode = "consistent_hash"
)

// RoutingPolicy 全局路由（渠道层），通过 Option「RoutingPolicy」JSON 配置。
type RoutingPolicy struct {
	SelectionMode        SelectionMode `json:"selection_mode"`
	ConsistentHashSeed   string        `json:"consistent_hash_seed"`
	StickySource         string        `json:"sticky_source"` // token_id | user_id | none，默认 token_id
	DirectionProbeRatio  float64       `json:"direction_probe_ratio"`   // Provider 探测流量占比上限意图
	DirectionMinProbe    float64       `json:"direction_min_probe_ratio"` // 探测下限，避免完全贪心
	DirectionEnabled     bool          `json:"direction_enabled"`       // 启用 Direction（Provider）与 Route（渠道）双层
	DirectionLatencyW    float64       `json:"direction_latency_weight"`
	DirectionErrorW      float64       `json:"direction_error_weight"`
	ProbePickStrategy    string        `json:"probe_pick_strategy"` // secondary_uniform | worst_first

	AutoAdaptiveEnabled bool `json:"auto_adaptive_enabled"`
	AdaptiveIntervalSec int  `json:"adaptive_interval_sec"` // 自适应调节周期

	CircuitFailThreshold int `json:"circuit_fail_threshold"` // 连续失败打开熔断；0 关闭 Redis 熔断（默认）
	CircuitCooldownSec   int `json:"circuit_cooldown_sec"`
	HalfOpenProbeMs      int `json:"half_open_probe_ms"`

	AdaptiveAggressivePenalty float64 `json:"adaptive_aggressive_penalty"` // 失败率高时的乘性衰减，默认 0.92
	AdaptiveGentleBoost       float64 `json:"adaptive_gentle_boost"`     // 正常时的缓慢恢复，默认 1.01
	AdaptiveErrRatioThreshold float64 `json:"adaptive_err_ratio_threshold"`
	AutoWeightMin             float64 `json:"auto_weight_min"`
	AutoWeightMax             float64 `json:"auto_weight_max"`

	AggregationIntervalSec int `json:"aggregation_interval_sec"` // 写入分钟桶周期（与自适应可共用 tick）
}

// RelayRetryPolicy 通过 Option「RelayRetryPolicy」JSON，与 RetryTimes 叠加使用。
type RelayRetryPolicy struct {
	MaxRetriesOverride               *int  `json:"max_retries_override,omitempty"` // nil 表示沿用 RetryTimes
	BaseBackoffMs                    int   `json:"base_backoff_ms"`
	MaxBackoffMs                     int   `json:"max_backoff_ms"`
	RetryHTTPStatusAllowlist         []int `json:"retry_http_status_allowlist"`
	RetryHTTPStatusDenylist          []int `json:"retry_http_status_denylist"`
	ForceDifferentChannelEachAttempt bool  `json:"force_different_channel_each_attempt"`
}

// AliasTarget 别名指向的候选模型。
type AliasTarget struct {
	Model    string `json:"model"`
	Weight   int    `json:"weight"`
	Priority int64  `json:"priority"`
}

// ModelAliasPolicy Option「ModelAliasPolicy」：链式映射 + 多候选加权。
type ModelAliasPolicy struct {
	Chains          map[string]string                      `json:"chains"`
	Aliases         map[string][]AliasTarget               `json:"aliases"`
	Defaults        map[string][]AliasTarget               `json:"defaults"` // 并入 aliases
	GroupOverrides  map[string]GroupAliasFragment          `json:"group_overrides"`
	MaxSteps        int                                    `json:"max_chain_steps"`
}

// GroupAliasFragment 按用户分组覆盖别名片段（与全局合并时分组优先）。
type GroupAliasFragment struct {
	Chains  map[string]string           `json:"chains,omitempty"`
	Aliases map[string][]AliasTarget    `json:"aliases,omitempty"`
}

// RateLimitRule 模型维度限速。
type RateLimitRule struct {
	QPS          int   `json:"qps"`
	Burst        int   `json:"burst"`
	Concurrency  int   `json:"concurrency"`
	DailyQuota   int64 `json:"daily_quota"`
}

// ModelRateLimitPolicy Option「ModelRateLimitPolicy」。
type ModelRateLimitPolicy struct {
	Enabled bool                       `json:"enabled"`
	ByToken map[string]RateLimitRule   `json:"by_token"` // key 为模型名 glob，「*」默认
	ByUser  map[string]RateLimitRule   `json:"by_user"`
	ByGroup map[string]RateLimitRule   `json:"by_group"`
}

func routingPolicyJSON() string {
	config.OptionMapRWMutex.RLock()
	defer config.OptionMapRWMutex.RUnlock()
	if config.OptionMap == nil {
		return ""
	}
	return strings.TrimSpace(config.OptionMap["RoutingPolicy"])
}

func relayRetryPolicyJSON() string {
	config.OptionMapRWMutex.RLock()
	defer config.OptionMapRWMutex.RUnlock()
	if config.OptionMap == nil {
		return ""
	}
	return strings.TrimSpace(config.OptionMap["RelayRetryPolicy"])
}

func modelAliasPolicyJSON() string {
	config.OptionMapRWMutex.RLock()
	defer config.OptionMapRWMutex.RUnlock()
	if config.OptionMap == nil {
		return ""
	}
	return strings.TrimSpace(config.OptionMap["ModelAliasPolicy"])
}

func modelRateLimitPolicyJSON() string {
	config.OptionMapRWMutex.RLock()
	defer config.OptionMapRWMutex.RUnlock()
	if config.OptionMap == nil {
		return ""
	}
	return strings.TrimSpace(config.OptionMap["ModelRateLimitPolicy"])
}

// CurrentRoutingPolicy 返回解析后的路由策略（带默认值）。
func CurrentRoutingPolicy() RoutingPolicy {
	raw := routingPolicyJSON()
	var p RoutingPolicy
	if raw != "" && raw != "{}" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	if p.SelectionMode == "" {
		p.SelectionMode = SelectionWeightedRandom
	}
	if p.StickySource == "" {
		p.StickySource = "token_id"
	}
	if p.CircuitCooldownSec == 0 {
		p.CircuitCooldownSec = 60
	}
	if p.HalfOpenProbeMs == 0 {
		p.HalfOpenProbeMs = 8000
	}
	if p.DirectionLatencyW == 0 {
		p.DirectionLatencyW = 1
	}
	if p.DirectionErrorW == 0 {
		p.DirectionErrorW = 2
	}
	if p.ProbePickStrategy == "" {
		p.ProbePickStrategy = "secondary_uniform"
	}
	if p.AdaptiveIntervalSec <= 0 {
		p.AdaptiveIntervalSec = 60
	}
	if p.AggregationIntervalSec <= 0 {
		p.AggregationIntervalSec = 60
	}
	if p.AdaptiveAggressivePenalty <= 0 {
		p.AdaptiveAggressivePenalty = 0.92
	}
	if p.AdaptiveGentleBoost <= 0 {
		p.AdaptiveGentleBoost = 1.01
	}
	if p.AdaptiveErrRatioThreshold <= 0 {
		p.AdaptiveErrRatioThreshold = 0.25
	}
	if p.AutoWeightMin <= 0 {
		p.AutoWeightMin = 0.05
	}
	if p.AutoWeightMax <= 0 {
		p.AutoWeightMax = 3
	}
	return p
}

// CurrentRelayRetryPolicy 解析重试策略。
func CurrentRelayRetryPolicy() RelayRetryPolicy {
	raw := relayRetryPolicyJSON()
	var p RelayRetryPolicy
	if raw != "" && raw != "{}" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	if p.BaseBackoffMs <= 0 {
		p.BaseBackoffMs = 150
	}
	if p.MaxBackoffMs <= 0 {
		p.MaxBackoffMs = 4000
	}
	return p
}

// EffectiveMaxRetries 返回本轮请求允许的重试次数（不含首次）。
func EffectiveMaxRetries() int {
	p := CurrentRelayRetryPolicy()
	if p.MaxRetriesOverride != nil {
		return *p.MaxRetriesOverride
	}
	return config.RetryTimes
}

// CurrentModelAliasPolicy 别名 / 链配置。
func CurrentModelAliasPolicy() ModelAliasPolicy {
	raw := modelAliasPolicyJSON()
	var p ModelAliasPolicy
	if raw != "" && raw != "{}" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	if p.Chains == nil {
		p.Chains = map[string]string{}
	}
	if p.Aliases == nil {
		p.Aliases = map[string][]AliasTarget{}
	}
	if p.Defaults != nil {
		for k, v := range p.Defaults {
			p.Aliases[k] = append(p.Aliases[k], v...)
		}
	}
	if p.GroupOverrides == nil {
		p.GroupOverrides = map[string]GroupAliasFragment{}
	}
	if p.MaxSteps <= 0 {
		p.MaxSteps = 32
	}
	return p
}

// CurrentModelRateLimitPolicy 限速策略。
func CurrentModelRateLimitPolicy() ModelRateLimitPolicy {
	raw := modelRateLimitPolicyJSON()
	var p ModelRateLimitPolicy
	if raw != "" && raw != "{}" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	if p.ByToken == nil {
		p.ByToken = map[string]RateLimitRule{}
	}
	if p.ByUser == nil {
		p.ByUser = map[string]RateLimitRule{}
	}
	if p.ByGroup == nil {
		p.ByGroup = map[string]RateLimitRule{}
	}
	return p
}
