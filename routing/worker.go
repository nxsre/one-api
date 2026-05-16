package routing

import (
	"time"

	"github.com/songquanpeng/one-api/common"
)

// StartBackgroundJobs 自适应权重调节周期任务（依赖 Redis）。
func StartBackgroundJobs() {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	go func() {
		var lastAdaptive int64
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			p := CurrentRoutingPolicy()
			if !p.AutoAdaptiveEnabled {
				continue
			}
			now := time.Now().Unix()
			interval := int64(p.AdaptiveIntervalSec)
			if interval < 10 {
				interval = 10
			}
			if now-lastAdaptive < interval {
				continue
			}
			lastAdaptive = now
			runAdaptiveAdjust(p)
		}
	}()
}
