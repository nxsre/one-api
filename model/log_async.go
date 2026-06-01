package model

import (
	"strconv"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
)

// 异步消费日志队列。
//
// 目的：把高频的 Log 写库从请求热路径剥离。RecordConsumeLog / RecordErrorLog 等
// 原本在请求返回路径上每条都 INSERT 一次，高 QPS 下 Log 表写入会最先成为瓶颈。
// 这里用一个带缓冲的 channel + 单 worker 批量落库（CreateInBatches）。
//
// 不丢日志保证：队列满时 enqueueLog 返回 false，调用方回退为同步写入（背压）。
var (
	logQueue      chan *Log
	logWorkerDone chan struct{}
	logAsyncOnce  sync.Once
)

// InitLogConsumer 启动异步日志 worker。需在 DB / LOG_DB 初始化之后调用。
// 未启用（LOG_ASYNC_ENABLED=false）时为空操作，recordLogHelper 走原有同步路径。
func InitLogConsumer() {
	if !config.LogAsyncEnabled {
		return
	}
	logAsyncOnce.Do(func() {
		bufferSize := config.LogAsyncBufferSize
		if bufferSize <= 0 {
			bufferSize = 10000
		}
		logQueue = make(chan *Log, bufferSize)
		logWorkerDone = make(chan struct{})
		go logConsumeWorker()
		logger.SysLog("async log consumer enabled, buffer size " + strconv.Itoa(bufferSize))
	})
}

// enqueueLog 尝试把日志放入异步队列。
// 返回 true 表示已入队（将由 worker 落库）；返回 false 表示未启用或队列已满，
// 调用方需自行同步写库。
func enqueueLog(log *Log) bool {
	if logQueue == nil {
		return false
	}
	select {
	case logQueue <- log:
		return true
	default:
		// 队列已满：回退同步写入，提供背压而非丢弃计费/消费日志。
		return false
	}
}

func logConsumeWorker() {
	defer close(logWorkerDone)

	batchSize := config.LogAsyncBatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	flushInterval := time.Duration(config.LogAsyncFlushIntervalMs) * time.Millisecond
	if flushInterval <= 0 {
		flushInterval = time.Second
	}

	batch := make([]*Log, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		flushLogBatch(batch)
		batch = batch[:0]
	}

	for {
		select {
		case log, ok := <-logQueue:
			if !ok {
				flush()
				return
			}
			batch = append(batch, log)
			if len(batch) >= batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// flushLogBatch 批量落库。LogShardByDay 开启时按分表分组后分别批量写入。
func flushLogBatch(batch []*Log) {
	if LOG_DB == nil {
		logger.SysError("async log: LOG_DB is nil, dropping " + strconv.Itoa(len(batch)) + " logs")
		return
	}
	if config.LogShardByDay {
		groups := make(map[string][]*Log)
		for _, log := range batch {
			tbl := logShardTableFromUnix(log.CreatedAt)
			groups[tbl] = append(groups[tbl], log)
		}
		for tbl, logs := range groups {
			if err := ensureLogShardTable(tbl); err != nil {
				logger.SysError("async log: ensure shard table " + tbl + " failed: " + err.Error())
				continue
			}
			if err := LOG_DB.Table(tbl).CreateInBatches(logs, len(logs)).Error; err != nil {
				logger.SysError("async log: batch insert into " + tbl + " failed: " + err.Error())
			}
		}
		return
	}
	if err := LOG_DB.CreateInBatches(batch, len(batch)).Error; err != nil {
		logger.SysError("async log: batch insert failed: " + err.Error())
	}
}

// FlushLogQueue 关闭队列并等待 worker 把剩余日志全部落库。
// 供优雅关闭（graceful shutdown）调用，确保进程退出前不丢未落库的日志。
// 调用后不应再写入消费日志。多次调用安全。
func FlushLogQueue() {
	if logQueue == nil {
		return
	}
	done := logWorkerDone
	closeLogQueueOnce.Do(func() {
		close(logQueue)
	})
	if done != nil {
		<-done
	}
}

var closeLogQueueOnce sync.Once
