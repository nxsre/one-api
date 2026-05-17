package model

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common/logger"
)

const logShardPrefix = "logs_"

var (
	logShardMigrateMu    sync.Mutex
	logShardMigratedTbls = map[string]bool{}
	logShardNameRe       = regexp.MustCompile(`^logs_[0-9]{8}$`)
)

// LogShardTableUTC 返回 UTC 日历日对应的物理表名：logs_YYYYMMDD。
func LogShardTableUTC(t time.Time) string {
	t = t.UTC()
	return fmt.Sprintf("%s%04d%02d%02d", logShardPrefix, t.Year(), int(t.Month()), t.Day())
}

func logShardTableFromUnix(sec int64) string {
	return LogShardTableUTC(time.Unix(sec, 0).UTC())
}

func quoteLogIdent(name string) string {
	switch LOG_DB.Dialector.Name() {
	case "postgres":
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	default:
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
}

func logSelectColumnsSQL() string {
	return "id, user_id, created_at, type, content, username, token_name, model_name, quota, prompt_tokens, completion_tokens, use_time, channel_id, token_id, " +
		logGroupCol + ", ip, request_id, other, elapsed_time, is_stream, system_prompt_reset"
}

func shardMigrateCacheKey(db *gorm.DB, table string) string {
	return fmt.Sprintf("%p:%s", db, table)
}

func ensureLogShardTable(name string) error {
	if LOG_DB == nil {
		return errors.New("LOG_DB is nil")
	}
	return ensureLogShardTableOn(LOG_DB, name)
}

func ensureLogShardTableOn(db *gorm.DB, name string) error {
	if db == nil {
		return errors.New("db is nil")
	}
	key := shardMigrateCacheKey(db, name)
	logShardMigrateMu.Lock()
	defer logShardMigrateMu.Unlock()
	if logShardMigratedTbls[key] {
		return nil
	}
	if err := db.Table(name).AutoMigrate(&Log{}); err != nil {
		return err
	}
	logShardMigratedTbls[key] = true
	return nil
}

func tableExistsLogDB(table string) bool {
	if LOG_DB == nil {
		return false
	}
	var n int64
	var err error
	switch LOG_DB.Dialector.Name() {
	case "postgres":
		err = LOG_DB.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?`, table).Scan(&n).Error
	case "mysql":
		err = LOG_DB.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&n).Error
	default:
		err = LOG_DB.Raw(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n).Error
	}
	if err != nil {
		logger.SysError("tableExistsLogDB: " + err.Error())
		return false
	}
	return n > 0
}

// discoverLogPhysicalTables 返回可用于查询的日志表：logs_YYYYMMDD（降序：新→旧），若存在旧版单表 logs 则排在最后。
func discoverLogPhysicalTables() ([]string, error) {
	if LOG_DB == nil {
		return nil, errors.New("LOG_DB is nil")
	}
	var names []string
	switch LOG_DB.Dialector.Name() {
	case "postgres":
		var rows []struct {
			Tablename string `gorm:"column:tablename"`
		}
		err := LOG_DB.Raw(`SELECT tablename FROM pg_tables WHERE schemaname = current_schema() AND tablename ~ '^logs_[0-9]{8}$'`).Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			names = append(names, r.Tablename)
		}
	case "mysql":
		var rows []struct {
			TableName string `gorm:"column:TABLE_NAME"`
		}
		err := LOG_DB.Raw(`SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME REGEXP '^logs_[0-9]{8}$'`).Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			names = append(names, r.TableName)
		}
	default:
		err := LOG_DB.Raw(`SELECT name FROM sqlite_master WHERE type = 'table' AND LENGTH(name) = 13 AND SUBSTR(name, 1, 5) = 'logs_' AND SUBSTR(name, 6, 8) GLOB '[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]'`).Scan(&names).Error
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] > names[j]
	})
	var out []string
	for _, n := range names {
		if logShardNameRe.MatchString(n) {
			out = append(out, n)
		}
	}
	if tableExistsLogDB("logs") {
		out = append(out, "logs")
	}
	return out, nil
}

func parseShardTableDayUnix(table string) (dayStart, dayEnd int64, ok bool) {
	if !strings.HasPrefix(table, logShardPrefix) || len(table) != len(logShardPrefix)+8 {
		return 0, 0, false
	}
	suf := table[len(logShardPrefix):]
	y, err1 := strconv.Atoi(suf[0:4])
	mo, err2 := strconv.Atoi(suf[4:6])
	d, err3 := strconv.Atoi(suf[6:8])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, false
	}
	t0 := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(24 * time.Hour)
	return t0.Unix(), t1.Unix() - 1, true
}

// filterLogTablesByTimeRange 按 created_at 区间筛选可能命中的表（UTC 按日对齐）。
func filterLogTablesByTimeRange(tables []string, startSec, endSec int64) []string {
	if len(tables) == 0 {
		return nil
	}
	var out []string
	for _, tbl := range tables {
		if tbl == "logs" {
			out = append(out, tbl)
			continue
		}
		ds, de, ok := parseShardTableDayUnix(tbl)
		if !ok {
			continue
		}
		if startSec != 0 && de < startSec {
			continue
		}
		if endSec != 0 && ds > endSec {
			continue
		}
		out = append(out, tbl)
	}
	return out
}

type shardFilterParams struct {
	LogType        int
	TypeStringEq   string // 非空时按字符串绑定 type（兼容 SearchUserLogs）
	UserID         *int
	TokenId        int // 0 = 不按 token_id 过滤
	ModelName      string
	Username       string
	TokenName      string
	RequestID      string
	StartTimestamp int64
	EndTimestamp   int64
	Channel        int
	Group          string
	// 全文检索类（SearchAllLogs）：与单表条件一致
	SearchKeyword       string
	SearchContentPrefix string
}

func appendShardWhereSQL(sb *strings.Builder, p *shardFilterParams, args *[]interface{}) error {
	if p.TypeStringEq != "" {
		sb.WriteString(" AND type = ?")
		*args = append(*args, p.TypeStringEq)
	} else if p.LogType != LogTypeUnknown {
		sb.WriteString(" AND type = ?")
		*args = append(*args, p.LogType)
	}
	if p.UserID != nil {
		sb.WriteString(" AND user_id = ?")
		*args = append(*args, *p.UserID)
	}
	if p.TokenId != 0 {
		sb.WriteString(" AND token_id = ?")
		*args = append(*args, p.TokenId)
	}
	if p.ModelName != "" {
		modelNamePattern, perr := sanitizeLikePattern(p.ModelName)
		if perr != nil {
			return perr
		}
		sb.WriteString(" AND model_name LIKE ? ESCAPE '!'")
		*args = append(*args, modelNamePattern)
	}
	if p.Username != "" {
		sb.WriteString(" AND username = ?")
		*args = append(*args, p.Username)
	}
	if p.TokenName != "" {
		sb.WriteString(" AND token_name = ?")
		*args = append(*args, p.TokenName)
	}
	if p.RequestID != "" {
		sb.WriteString(" AND request_id = ?")
		*args = append(*args, p.RequestID)
	}
	if p.StartTimestamp != 0 {
		sb.WriteString(" AND created_at >= ?")
		*args = append(*args, p.StartTimestamp)
	}
	if p.EndTimestamp != 0 {
		sb.WriteString(" AND created_at <= ?")
		*args = append(*args, p.EndTimestamp)
	}
	if p.Channel != 0 {
		sb.WriteString(" AND channel_id = ?")
		*args = append(*args, p.Channel)
	}
	if p.Group != "" {
		sb.WriteString(" AND " + logGroupCol + " = ?")
		*args = append(*args, p.Group)
	}
	if p.SearchKeyword != "" {
		sb.WriteString(" AND (type = ? OR content LIKE ?)")
		*args = append(*args, p.SearchKeyword, p.SearchContentPrefix)
	}
	return nil
}

func cloneRepeatArgs(perTable []interface{}, nTables int) []interface{} {
	if nTables <= 0 {
		return nil
	}
	out := make([]interface{}, 0, len(perTable)*nTables)
	for i := 0; i < nTables; i++ {
		out = append(out, perTable...)
	}
	return out
}

func unionCountLogs(tables []string, p *shardFilterParams) (int64, error) {
	if len(tables) == 0 {
		return 0, nil
	}
	var sb strings.Builder
	sb.WriteString("SELECT COUNT(*) FROM (")
	perArgs := []interface{}{}
	wsb := &strings.Builder{}
	if err := appendShardWhereSQL(wsb, p, &perArgs); err != nil {
		return 0, err
	}
	whereSQL := wsb.String()
	for i, tbl := range tables {
		if i > 0 {
			sb.WriteString(" UNION ALL ")
		}
		sb.WriteString("(SELECT 1 FROM ")
		sb.WriteString(quoteLogIdent(tbl))
		sb.WriteString(" WHERE 1=1")
		sb.WriteString(whereSQL)
		sb.WriteString(")")
	}
	sb.WriteString(") AS _u")
	args := cloneRepeatArgs(perArgs, len(tables))
	var total int64
	err := LOG_DB.Raw(sb.String(), args...).Scan(&total).Error
	return total, err
}

func unionSelectLogs(tables []string, p *shardFilterParams, limit, offset int) ([]*Log, error) {
	if len(tables) == 0 {
		return nil, nil
	}
	cols := logSelectColumnsSQL()
	var sb strings.Builder
	perArgs := []interface{}{}
	wsb := &strings.Builder{}
	if err := appendShardWhereSQL(wsb, p, &perArgs); err != nil {
		return nil, err
	}
	whereSQL := wsb.String()
	sb.WriteString("SELECT * FROM (")
	for i, tbl := range tables {
		if i > 0 {
			sb.WriteString(" UNION ALL ")
		}
		sb.WriteString("(SELECT ")
		sb.WriteString(cols)
		sb.WriteString(" FROM ")
		sb.WriteString(quoteLogIdent(tbl))
		sb.WriteString(" WHERE 1=1")
		sb.WriteString(whereSQL)
		sb.WriteString(")")
	}
	sb.WriteString(") AS _union_logs ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?")
	args := cloneRepeatArgs(perArgs, len(tables))
	args = append(args, limit, offset)
	var logs []*Log
	err := LOG_DB.Raw(sb.String(), args...).Scan(&logs).Error
	return logs, err
}

func tablesForLogQuery(startTimestamp, endTimestamp int64) ([]string, error) {
	all, err := discoverLogPhysicalTables()
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, nil
	}
	return filterLogTablesByTimeRange(all, startTimestamp, endTimestamp), nil
}

// migrateLogShardTablesBoot 在指定 DB 上创建当日与昨日分表（UTC），减少首次写入时的迁移延迟。
func migrateLogShardTablesBoot(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	now := time.Now().UTC()
	for _, d := range []time.Time{now, now.AddDate(0, 0, -1)} {
		tbl := LogShardTableUTC(d)
		if err := ensureLogShardTableOn(db, tbl); err != nil {
			return err
		}
	}
	return nil
}

// applyLogConsumeStatFilters 与 SumUsedQuota 筛选一致（model_name 使用 LIKE）。
func applyLogConsumeStatFilters(tx *gorm.DB, startTimestamp, endTimestamp int64, modelName, username, tokenName string, channel int, group string) (*gorm.DB, error) {
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		modelNamePattern, perr := sanitizeLikePattern(modelName)
		if perr != nil {
			return nil, perr
		}
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
	}
	return tx, nil
}

// applyLogConsumeTokenSumFilters 与 SumUsedToken 筛选一致（model_name 精确匹配）。
func applyLogConsumeTokenSumFilters(tx *gorm.DB, startTimestamp, endTimestamp int64, modelName, username, tokenName string) *gorm.DB {
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	return tx
}