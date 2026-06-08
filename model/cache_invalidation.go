package model

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
)

// 跨实例缓存失效（Redis Pub/Sub）。
//
// 原本配置（OptionMap）与渠道缓存（group2model2channels）只靠每 SyncFrequency 秒
// （默认 600s）的轮询同步，意味着一个实例改了配置 / 渠道，其它实例最长要 10 分钟
// 才生效，且每次都做全表扫描。这里在写入后发布一条失效事件，所有实例（含自身）
// 立即重载对应缓存，把传播延迟降到亚秒级；轮询保留为兜底。
//
// Redis 未启用时发布与订阅都是空操作，行为退回到纯轮询，单实例不受影响。
const cacheInvalidationChannel = "one-api:cache-invalidate"

const (
	cacheKindOptions  = "options"
	cacheKindChannels = "channels"
)

// PublishOptionsChanged 在配置写入后调用，通知各实例重载 OptionMap。
func PublishOptionsChanged() { publishCacheInvalidation(cacheKindOptions) }

// PublishChannelsChanged 在渠道 / abilities 写入后调用，通知各实例重建渠道缓存。
func PublishChannelsChanged() { publishCacheInvalidation(cacheKindChannels) }

func publishCacheInvalidation(kind string) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	if err := common.RDB.Publish(context.Background(), cacheInvalidationChannel, kind).Err(); err != nil {
		logger.SysError("publish cache invalidation (" + kind + ") failed: " + err.Error())
	}
}

// StartCacheInvalidationSubscriber 订阅失效事件并在收到时重载缓存。Redis 未启用时为空操作。
func StartCacheInvalidationSubscriber() {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	client, ok := common.RDB.(redis.UniversalClient)
	if !ok {
		logger.SysError("cache invalidation subscriber disabled: redis client does not support pub/sub")
		return
	}
	go runCacheInvalidationSubscriber(client)
}

func runCacheInvalidationSubscriber(client redis.UniversalClient) {
	ctx := context.Background()
	sub := client.Subscribe(ctx, cacheInvalidationChannel)
	defer sub.Close()
	ch := sub.Channel()
	logger.SysLog("cache invalidation subscriber started")

	// debounce 合并突发：批量导入渠道会发出很多事件，300ms 内的同类事件只触发一次重建。
	const debounce = 300 * time.Millisecond
	var optTimer, chanTimer <-chan time.Time
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			switch msg.Payload {
			case cacheKindOptions:
				if optTimer == nil {
					optTimer = time.After(debounce)
				}
			case cacheKindChannels:
				if chanTimer == nil {
					chanTimer = time.After(debounce)
				}
			}
		case <-optTimer:
			optTimer = nil
			loadOptionsFromDatabase()
			logger.SysLog("options reloaded via pubsub invalidation")
		case <-chanTimer:
			chanTimer = nil
			InitChannelCache()
			logger.SysLog("channels reloaded via pubsub invalidation")
		}
	}
}
