package routing

import (
	dbmodel "github.com/songquanpeng/one-api/model"
)

// SelectChannel 选取单个渠道（加权随机 / 一致性哈希 / 熔断跳过 / 排除集合）。
func SelectChannel(group, model string, ignoreFirstPriority bool, opts SelectOpts) (*dbmodel.Channel, error) {
	opts.RequestModel = model
	channels, err := dbmodel.LoadSortedChannelsForGroupModel(group, model)
	if err != nil {
		return nil, err
	}
	return PickChannel(channels, ignoreFirstPriority, opts)
}
