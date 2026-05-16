package routing

import (
	"context"
	"fmt"
	"strconv"

	"github.com/songquanpeng/one-api/common"
)

func autoWeightKey(channelID int) string {
	return fmt.Sprintf("oa:rt:auto:wmul:%d", channelID)
}

// GetAutoWeightMultiplier 自适应倍率（与手工倍率相乘）。
func GetAutoWeightMultiplier(channelID int) float64 {
	if !common.RedisEnabled || common.RDB == nil {
		return 1
	}
	s, err := common.RDB.Get(context.Background(), autoWeightKey(channelID)).Result()
	if err != nil || s == "" {
		return 1
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 {
		return 1
	}
	return v
}

// SetAutoWeightMultiplier 运维或自适应 Worker 写入自动倍率。
func SetAutoWeightMultiplier(channelID int, multiplier float64) error {
	if !common.RedisEnabled || common.RDB == nil {
		return nil
	}
	ctx := context.Background()
	if multiplier <= 0 {
		return common.RDB.Del(ctx, autoWeightKey(channelID)).Err()
	}
	return common.RDB.Set(ctx, autoWeightKey(channelID), strconv.FormatFloat(multiplier, 'f', 6, 64), 0).Err()
}
