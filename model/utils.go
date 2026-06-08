package model

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
)

const (
	BatchUpdateTypeUserQuota = iota
	BatchUpdateTypeTokenQuota
	BatchUpdateTypeUsedQuota
	BatchUpdateTypeChannelUsedQuota
	BatchUpdateTypeRequestCount
	BatchUpdateTypeCount // if you add a new type, you need to add a new map and a new lock
)

var batchUpdateStores []map[int]int64
var batchUpdateLocks []sync.Mutex

func init() {
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateStores = append(batchUpdateStores, make(map[int]int64))
		batchUpdateLocks = append(batchUpdateLocks, sync.Mutex{})
	}
}

func InitBatchUpdater() {
	go func() {
		for {
			time.Sleep(time.Duration(config.BatchUpdateInterval) * time.Second)
			batchUpdate()
		}
	}()
}

// FlushBatchUpdates 立即把累积的配额增量落库。供优雅关闭调用，确保进程退出前
// 不丢失尚未 flush 的配额数据。未开启批量更新时累积区为空，调用无副作用。
func FlushBatchUpdates() {
	batchUpdate()
}

func addNewRecord(type_ int, id int, value int64) {
	batchUpdateLocks[type_].Lock()
	defer batchUpdateLocks[type_].Unlock()
	if _, ok := batchUpdateStores[type_][id]; !ok {
		batchUpdateStores[type_][id] = value
	} else {
		batchUpdateStores[type_][id] += value
	}
}

func batchUpdate() {
	logger.SysLog("batch update started")
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateLocks[i].Lock()
		store := batchUpdateStores[i]
		batchUpdateStores[i] = make(map[int]int64)
		batchUpdateLocks[i].Unlock()
		if len(store) == 0 {
			continue
		}
		// 把同一类型下所有 id 的增量合并成一条 CASE WHEN 语句，避免逐 id 单独 UPDATE。
		var err error
		switch i {
		case BatchUpdateTypeUserQuota:
			err = batchIncreaseColumn("users", "quota", store)
		case BatchUpdateTypeTokenQuota:
			err = batchIncreaseTokenQuota(store)
		case BatchUpdateTypeUsedQuota:
			err = batchIncreaseColumn("users", "used_quota", store)
		case BatchUpdateTypeRequestCount:
			err = batchIncreaseColumn("users", "request_count", store)
		case BatchUpdateTypeChannelUsedQuota:
			err = batchIncreaseColumn("channels", "used_quota", store)
		}
		if err != nil {
			logger.SysError(fmt.Sprintf("failed to batch update type %d (%d rows): %s", i, len(store), err.Error()))
		}
	}
	logger.SysLog("batch update finished")
}

// buildIncrementCase 把 {id: delta} 拼成 `CASE id WHEN ? THEN ? ... ELSE 0 END` 片段、
// 对应的占位参数，以及 id 列表（供 WHERE id IN ? 使用）。
func buildIncrementCase(deltas map[int]int64) (caseExpr string, caseArgs []any, ids []int) {
	var b strings.Builder
	b.WriteString("CASE id")
	caseArgs = make([]any, 0, len(deltas)*2)
	ids = make([]int, 0, len(deltas))
	for id, delta := range deltas {
		b.WriteString(" WHEN ? THEN ?")
		caseArgs = append(caseArgs, id, delta)
		ids = append(ids, id)
	}
	b.WriteString(" ELSE 0 END")
	return b.String(), caseArgs, ids
}

// batchIncreaseColumn 在 table 上对 column 按 {id: delta} 批量自增，单条 SQL 完成。
// 占位符 `?` 由 GORM 按方言转换（MySQL/SQLite 原样、PostgreSQL 转 $n），可移植。
func batchIncreaseColumn(table, column string, deltas map[int]int64) error {
	if len(deltas) == 0 {
		return nil
	}
	caseExpr, args, ids := buildIncrementCase(deltas)
	sql := fmt.Sprintf("UPDATE %s SET %s = %s + (%s) WHERE id IN ?", table, column, column, caseExpr)
	args = append(args, ids)
	return DB.Exec(sql, args...).Error
}

// batchIncreaseTokenQuota 对应原 increaseTokenQuota：remain_quota += delta、
// used_quota -= delta、accessed_time = now，并在落库后逐 id 失效缓存。
func batchIncreaseTokenQuota(deltas map[int]int64) error {
	if len(deltas) == 0 {
		return nil
	}
	caseExpr, caseArgs, ids := buildIncrementCase(deltas)
	sql := fmt.Sprintf(
		"UPDATE tokens SET remain_quota = remain_quota + (%s), used_quota = used_quota - (%s), accessed_time = ? WHERE id IN ?",
		caseExpr, caseExpr)
	args := make([]any, 0, len(caseArgs)*2+2)
	args = append(args, caseArgs...) // 第一个 CASE：remain_quota
	args = append(args, caseArgs...) // 第二个 CASE：used_quota（同样的 id→delta 对）
	args = append(args, helper.GetTimestamp())
	args = append(args, ids)
	if err := DB.Exec(sql, args...).Error; err != nil {
		return err
	}
	for _, id := range ids {
		CacheInvalidateTokenById(id)
	}
	return nil
}
