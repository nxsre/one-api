package routing

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	dbmodel "github.com/songquanpeng/one-api/model"

	"github.com/songquanpeng/one-api/common"
)

// TimeseriesPoint 分钟桶延迟与错误。
type TimeseriesPoint struct {
	MinuteUnix int64   `json:"minute_unix"`
	OK         int64   `json:"ok"`
	Fail       int64   `json:"fail"`
	AvgLatency float64 `json:"avg_latency_ms"`
	ErrRatio   float64 `json:"err_ratio"`
}

// QueryTimeseries 读取指定渠道 + 模型 的近若干小时分钟桶曲线。
func QueryTimeseries(channelID int, model string, hours int) ([]TimeseriesPoint, error) {
	if !common.RedisEnabled || common.RDB == nil {
		return nil, nil
	}
	if hours <= 0 || hours > 168 {
		hours = 24
	}
	mk := modelMinuteKey(model)
	nowMin := time.Now().Unix() / 60
	start := nowMin - int64(hours*60)

	ctx := context.Background()
	rdb := common.RDB
	var pts []TimeseriesPoint
	for m := start; m <= nowMin; m++ {
		key := fmt.Sprintf("oa:rt:z:%d:%s:%d", channelID, mk, m)
		h, err := rdb.HGetAll(ctx, key).Result()
		if err != nil || len(h) == 0 {
			continue
		}
		parse := func(s string) int64 {
			v, _ := strconv.ParseInt(s, 10, 64)
			return v
		}
		ok := parse(h["ok"])
		fail := parse(h["fail"])
		latSum := parse(h["lat_sum_ms"])
		latN := parse(h["lat_n"])
		avg := 0.0
		if latN > 0 {
			avg = float64(latSum) / float64(latN)
		}
		total := ok + fail
		errR := 0.0
		if total > 0 {
			errR = float64(fail) / float64(total)
		}
		if ok == 0 && fail == 0 {
			continue
		}
		pts = append(pts, TimeseriesPoint{
			MinuteUnix: m * 60,
			OK:         ok,
			Fail:       fail,
			AvgLatency: avg,
			ErrRatio:   errR,
		})
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].MinuteUnix < pts[j].MinuteUnix })
	return pts, nil
}

func aggregateChannelDayTraffic(day string, channelID int) (ok int64, fail int64) {
	if !common.RedisEnabled || common.RDB == nil {
		return 0, 0
	}
	ctx := context.Background()
	pat := fmt.Sprintf("oa:rt:m:%s:%d:*", day, channelID)
	var cursor uint64
	for {
		keys, next, err := common.RDB.Scan(ctx, cursor, pat, 200).Result()
		if err != nil {
			break
		}
		for _, key := range keys {
			h, err := common.RDB.HGetAll(ctx, key).Result()
			if err != nil {
				continue
			}
			for mk, mv := range h {
				if mk != "ok" && mk != "fail" {
					continue
				}
				v, _ := strconv.ParseInt(mv, 10, 64)
				if mk == "ok" {
					ok += v
				} else {
					fail += v
				}
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return ok, fail
}

func runAdaptiveAdjust(p RoutingPolicy) {
	ctx := context.Background()
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	day := time.Now().UTC().Format("20060102")
	key := fmt.Sprintf("oa:rt:auto:touch:%s", day)
	members, err := common.RDB.SMembers(ctx, key).Result()
	if err != nil {
		return
	}
	for _, sid := range members {
		id, err := strconv.Atoi(sid)
		if err != nil || id <= 0 {
			continue
		}
		ch, err := dbmodel.GetChannelById(id, false)
		if err != nil || ch == nil {
			continue
		}
		if !ch.RoutingAdaptiveEffective() {
			continue
		}
		ok, fail := aggregateChannelDayTraffic(day, id)
		total := ok + fail
		if total < 8 {
			continue
		}
		ratio := float64(fail) / float64(total)
		mul := GetAutoWeightMultiplier(id)
		if ratio >= p.AdaptiveErrRatioThreshold {
			mul *= p.AdaptiveAggressivePenalty
		} else {
			mul *= p.AdaptiveGentleBoost
		}
		if mul < p.AutoWeightMin {
			mul = p.AutoWeightMin
		}
		if mul > p.AutoWeightMax {
			mul = p.AutoWeightMax
		}
		_ = SetAutoWeightMultiplier(id, mul)
	}
}
