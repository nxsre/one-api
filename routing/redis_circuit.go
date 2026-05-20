package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/metrics"
)

type circuitState struct {
	State       string `json:"state"` // closed | open | half_open
	FailStreak  int    `json:"fail_streak"`
	OpenUntilMs int64  `json:"open_until_ms"`
}

func redisCli() redis.Cmdable {
	return common.RDB
}

// IsCircuitOpen 若熔断打开（未到恢复时间）返回 true。
func IsCircuitOpen(channelID int) bool {
	if !common.RedisEnabled || common.RDB == nil {
		return false
	}
	p := CurrentRoutingPolicy()
	if p.CircuitFailThreshold <= 0 {
		return false
	}
	ctx := context.Background()
	key := fmt.Sprintf("oa:rt:circuit:%d", channelID)
	s, err := redisCli().Get(ctx, key).Result()
	if err == redis.Nil || s == "" {
		return false
	}
	if err != nil {
		return false
	}
	var st circuitState
	if json.Unmarshal([]byte(s), &st) != nil {
		return false
	}
	now := time.Now().UnixMilli()
	switch st.State {
	case "open":
		return now < st.OpenUntilMs
	case "half_open":
		return false
	default:
		return false
	}
}

// RecordCircuitSuccess 记录成功，重置失败计数或关闭半开。
func RecordCircuitSuccess(channelID int) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	p := CurrentRoutingPolicy()
	if p.CircuitFailThreshold <= 0 {
		return
	}
	ctx := context.Background()
	key := fmt.Sprintf("oa:rt:circuit:%d", channelID)
	raw, err := redisCli().Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return
	}
	st := circuitState{State: "closed", FailStreak: 0}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &st)
	}
	prevState := st.State
	if st.State == "half_open" || st.State == "open" {
		st = circuitState{State: "closed", FailStreak: 0}
	} else {
		st.FailStreak = 0
		st.State = "closed"
	}
	b, _ := json.Marshal(st)
	_ = redisCli().Set(ctx, key, string(b), time.Duration(p.CircuitCooldownSec*3)*time.Second).Err()
	if prevState == "half_open" || prevState == "open" {
		AppendFuseEvent(FuseEvent{
			ChannelID: channelID,
			State:     "closed",
			Reason:    "success_recovery",
		})
	}
}

// RecordCircuitFailure 累计失败并可能在达到阈值时打开熔断。
func RecordCircuitFailure(channelID int) {
	if channelID > 0 {
		metrics.CircuitFailureTotal.WithLabelValues(fmt.Sprintf("%d", channelID)).Inc()
	}
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	p := CurrentRoutingPolicy()
	if p.CircuitFailThreshold <= 0 {
		return
	}
	ctx := context.Background()
	key := fmt.Sprintf("oa:rt:circuit:%d", channelID)
	raw, err := redisCli().Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return
	}
	st := circuitState{State: "closed", FailStreak: 0}
	if err == nil && raw != "" {
		_ = json.Unmarshal([]byte(raw), &st)
	}
	now := time.Now().UnixMilli()
	if st.State == "open" && now < st.OpenUntilMs {
		return
	}
	if st.State == "open" && now >= st.OpenUntilMs {
		st.State = "half_open"
		st.FailStreak = 0
	}
	st.FailStreak++
	th := p.CircuitFailThreshold
	openedNow := false
	if st.State == "half_open" {
		// 半开探测失败立即再打打开
		st.State = "open"
		st.OpenUntilMs = now + int64(p.CircuitCooldownSec)*1000
		st.FailStreak = 0
		openedNow = true
	} else if st.FailStreak >= th {
		st.State = "open"
		st.OpenUntilMs = now + int64(p.CircuitCooldownSec)*1000
		st.FailStreak = 0
		openedNow = true
	}
	b, _ := json.Marshal(st)
	ttl := time.Duration(p.CircuitCooldownSec*4) * time.Second
	if ttl < time.Minute {
		ttl = time.Minute
	}
	_ = redisCli().Set(ctx, key, string(b), ttl).Err()
	if openedNow {
		AppendFuseEvent(FuseEvent{
			ChannelID:   channelID,
			State:       "open",
			Reason:      "failure_threshold",
			CooldownSec: p.CircuitCooldownSec,
		})
	}
}

// ReadCircuitRaw 返回熔断 Redis 原文（运维调试）。
func ReadCircuitRaw(channelID int) string {
	if !common.RedisEnabled || common.RDB == nil {
		return ""
	}
	ctx := context.Background()
	key := fmt.Sprintf("oa:rt:circuit:%d", channelID)
	s, err := redisCli().Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	return s
}

// GetManualWeightMultiplier Redis 手工覆盖倍率，默认 1。
func GetManualWeightMultiplier(channelID int) float64 {
	if !common.RedisEnabled || common.RDB == nil {
		return 1
	}
	ctx := context.Background()
	key := fmt.Sprintf("oa:rt:wmul:%d", channelID)
	s, err := redisCli().Get(ctx, key).Result()
	if err != nil || s == "" {
		return 1
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 {
		return 1
	}
	return v
}

// SetManualWeightMultiplier 运营调权：倍率写入 Redis（跨实例生效）。
func SetManualWeightMultiplier(channelID int, multiplier float64) error {
	if !common.RedisEnabled || common.RDB == nil {
		return nil
	}
	ctx := context.Background()
	key := fmt.Sprintf("oa:rt:wmul:%d", channelID)
	if multiplier <= 0 {
		return redisCli().Del(ctx, key).Err()
	}
	return redisCli().Set(ctx, key, strconv.FormatFloat(multiplier, 'f', 4, 64), 0).Err()
}
