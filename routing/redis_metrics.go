package routing

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/common"
)

const metricsTTL = 50 * time.Hour

// RecordRelayMetric 渠道 / Provider / Token / 分钟桶多维聚合。
func RecordRelayMetric(channelID int, model string, tokenID int, provider string, ok bool, latencyMs int64) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	model = trimModel(model)
	prov := strings.TrimSpace(provider)
	if prov == "" {
		prov = "default"
	}
	day := time.Now().UTC().Format("20060102")
	ctx := context.Background()
	rdb := common.RDB
	base := fmt.Sprintf("oa:rt:m:%s:%d:%s", day, channelID, model)
	pipe := rdb.Pipeline()
	if ok {
		pipe.HIncrBy(ctx, base, "ok", 1)
	} else {
		pipe.HIncrBy(ctx, base, "fail", 1)
	}
	if latencyMs >= 0 {
		pipe.HIncrBy(ctx, base, "lat_sum_ms", latencyMs)
		pipe.HIncrBy(ctx, base, "lat_n", 1)
	}
	if tokenID > 0 {
		if ok {
			pipe.HIncrBy(ctx, base, fmt.Sprintf("tok:%d:ok", tokenID), 1)
		} else {
			pipe.HIncrBy(ctx, base, fmt.Sprintf("tok:%d:fail", tokenID), 1)
		}
	}

	pk := fmt.Sprintf("oa:rt:d:%s:%s:%s", day, prov, model)
	if ok {
		pipe.HIncrBy(ctx, pk, "ok", 1)
	} else {
		pipe.HIncrBy(ctx, pk, "fail", 1)
	}
	if latencyMs >= 0 {
		pipe.HIncrBy(ctx, pk, "lat_sum_ms", latencyMs)
		pipe.HIncrBy(ctx, pk, "lat_n", 1)
	}
	_, _ = pipe.Exec(ctx)
	_ = rdb.Expire(ctx, base, metricsTTL).Err()
	_ = rdb.Expire(ctx, pk, metricsTTL).Err()

	minBucket := time.Now().Unix() / 60
	zk := fmt.Sprintf("oa:rt:z:%d:%s:%d", channelID, modelMinuteKey(model), minBucket)
	pipe2 := rdb.Pipeline()
	if ok {
		pipe2.HIncrBy(ctx, zk, "ok", 1)
	} else {
		pipe2.HIncrBy(ctx, zk, "fail", 1)
	}
	if latencyMs >= 0 {
		pipe2.HIncrBy(ctx, zk, "lat_sum_ms", latencyMs)
		pipe2.HIncrBy(ctx, zk, "lat_n", 1)
	}
	_, _ = pipe2.Exec(ctx)
	_ = rdb.Expire(ctx, zk, 90*time.Hour).Err()

	touchKey := fmt.Sprintf("oa:rt:auto:touch:%s", day)
	_ = rdb.SAdd(ctx, touchKey, strconv.Itoa(channelID)).Err()
	_ = rdb.Expire(ctx, touchKey, metricsTTL).Err()
}

func trimModel(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 160 {
		return s[:160]
	}
	return s
}

func modelMinuteKey(s string) string {
	s = strings.ReplaceAll(trimModel(s), ":", "_")
	s = strings.ReplaceAll(s, "/", "_")
	if len(s) > 80 {
		return s[:80]
	}
	return s
}

// ChannelMetricAgg Redis Hash 聚合快照。
type ChannelMetricAgg struct {
	RedisKey string `json:"redis_key"`
	OK       int64  `json:"ok"`
	Fail     int64  `json:"fail"`
	LatSumMs int64  `json:"lat_sum_ms"`
	LatN     int64  `json:"lat_n"`
	Tokens   any    `json:"tokens_breakdown,omitempty"`
}

// ScanChannelMetricsDay 扫描某日的渠道+模型指标（用于运维大屏）。
func ScanChannelMetricsDay(day string) ([]ChannelMetricAgg, error) {
	if !common.RedisEnabled || common.RDB == nil {
		return nil, nil
	}
	ctx := context.Background()
	pat := fmt.Sprintf("oa:rt:m:%s:*", day)
	var cursor uint64
	var out []ChannelMetricAgg
	for {
		keys, next, err := common.RDB.Scan(ctx, cursor, pat, 200).Result()
		if err != nil {
			return out, err
		}
		for _, key := range keys {
			h, err := common.RDB.HGetAll(ctx, key).Result()
			if err != nil || len(h) == 0 {
				continue
			}
			agg := ChannelMetricAgg{RedisKey: key}
			tok := map[string]map[string]int64{}
			for mk, mv := range h {
				v, _ := strconv.ParseInt(mv, 10, 64)
				switch mk {
				case "ok":
					agg.OK = v
				case "fail":
					agg.Fail = v
				case "lat_sum_ms":
					agg.LatSumMs = v
				case "lat_n":
					agg.LatN = v
				default:
					if strings.HasPrefix(mk, "tok:") && strings.HasSuffix(mk, ":ok") {
						id := strings.TrimSuffix(strings.TrimPrefix(mk, "tok:"), ":ok")
						if tok[id] == nil {
							tok[id] = map[string]int64{}
						}
						tok[id]["ok"] = v
					}
					if strings.HasPrefix(mk, "tok:") && strings.HasSuffix(mk, ":fail") {
						id := strings.TrimSuffix(strings.TrimPrefix(mk, "tok:"), ":fail")
						if tok[id] == nil {
							tok[id] = map[string]int64{}
						}
						tok[id]["fail"] = v
					}
				}
			}
			if len(tok) > 0 {
				agg.Tokens = tok
			}
			out = append(out, agg)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return out, nil
}
