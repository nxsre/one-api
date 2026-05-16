package routing

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/songquanpeng/one-api/common"
)

const fuseLogKey = "oa:rt:fuse:events"

// FuseEvent 熔断记录（运维可视化）。
type FuseEvent struct {
	ChannelID   int    `json:"channel_id"`
	Model       string `json:"model,omitempty"`
	State       string `json:"state"`
	UnixMilli   int64  `json:"unix_ms"`
	Reason      string `json:"reason,omitempty"`
	CooldownSec int    `json:"cooldown_sec,omitempty"`
}

// AppendFuseEvent 写入最近熔断/恢复事件（LIST，保留约 500 条）。
func AppendFuseEvent(ev FuseEvent) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	ev.UnixMilli = time.Now().UnixMilli()
	b, _ := json.Marshal(ev)
	ctx := context.Background()
	rdb := common.RDB
	_ = rdb.LPush(ctx, fuseLogKey, string(b)).Err()
	_ = rdb.LTrim(ctx, fuseLogKey, 0, 499).Err()
}

// ListRecentFuseEvents 读取最近事件。
func ListRecentFuseEvents(limit int) ([]FuseEvent, error) {
	if !common.RedisEnabled || common.RDB == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	ctx := context.Background()
	raw, err := common.RDB.LRange(ctx, fuseLogKey, 0, int64(limit-1)).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	var out []FuseEvent
	for _, s := range raw {
		var ev FuseEvent
		if json.Unmarshal([]byte(s), &ev) == nil {
			out = append(out, ev)
		}
	}
	return out, nil
}
