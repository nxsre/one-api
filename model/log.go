package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/relayctx"
)

// logGroupCol 在 initLogGroupColumn 中按日志库方言设置（`group` 为保留字）。
var logGroupCol = "`group`"

func initLogGroupColumn() {
	if LOG_DB == nil {
		return
	}
	switch LOG_DB.Dialector.Name() {
	case "postgres":
		logGroupCol = `"group"`
	default:
		logGroupCol = "`group`"
	}
}

type Log struct {
	Id                int    `json:"id"`
	UserId            int    `json:"-" gorm:"column:user_id;index"`
	UserPublicID      string `json:"user_id" gorm:"-"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint;index:idx_created_at_type"`
	Type              int    `json:"type" gorm:"index:idx_created_at_type"`
	Content           string `json:"content"`
	Username          string `json:"username" gorm:"index:index_username_model_name,priority:2;default:''"`
	TokenName         string `json:"token_name" gorm:"index;default:''"`
	ModelName         string `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota             int    `json:"quota" gorm:"default:0"`
	PromptTokens      int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens  int    `json:"completion_tokens" gorm:"default:0"`
	UseTime           int    `json:"use_time" gorm:"default:0"` // 秒
	ChannelId         int    `json:"channel" gorm:"column:channel_id;index"`
	ChannelName       string `json:"channel_name" gorm:"-"`
	TokenId           int    `json:"token_id" gorm:"default:0;index"`
	Group             string `json:"group" gorm:"column:group;index;default:''"`
	Ip                string `json:"ip" gorm:"index;default:''"`
	RequestId         string `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	Other             string `json:"other" gorm:"type:text"`
	ElapsedTime       int64  `json:"elapsed_time" gorm:"default:0"` // 毫秒，保留兼容
	IsStream          bool   `json:"is_stream" gorm:"default:false"`
	SystemPromptReset bool   `json:"system_prompt_reset" gorm:"default:false"`
}

const (
	LogTypeUnknown = iota
	LogTypeTopup
	LogTypeConsume
	LogTypeManage
	LogTypeSystem
	LogTypeTest
	// LogTypeError、LogTypeRefund 类型码与常见派生实现一致，且不占用原 LogTypeTest=5。
	LogTypeError  = 6
	LogTypeRefund = 7
)

const logSearchCountLimit = 10000

func jsonStrToMap(s string) map[string]interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil || m == nil {
		return nil
	}
	return m
}

func jsonMapToStr(m map[string]interface{}) string {
	if m == nil || len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

// sanitizeLikePattern 清洗 LIKE 通配，避免恶意模式。
func sanitizeLikePattern(input string) (string, error) {
	input = strings.ReplaceAll(input, "!", "!!")
	input = strings.ReplaceAll(input, `_`, `!_`)
	if strings.Contains(input, "%%") {
		return "", errors.New("搜索模式中不允许包含连续的 % 通配符")
	}
	count := strings.Count(input, "%")
	if count > 2 {
		return "", errors.New("搜索模式中最多允许包含 2 个 % 通配符")
	}
	if count > 0 {
		stripped := strings.ReplaceAll(input, "%", "")
		if len(stripped) < 2 {
			return "", errors.New("使用模糊搜索时，关键词长度至少为 2 个字符")
		}
		return input, nil
	}
	return input, nil
}

func formatUserLogs(logs []*Log, startIdx int) {
	for i := range logs {
		logs[i].ChannelName = ""
		otherMap := jsonStrToMap(logs[i].Other)
		if otherMap != nil {
			delete(otherMap, "admin_info")
			delete(otherMap, "stream_status")
		}
		logs[i].Other = jsonMapToStr(otherMap)
		logs[i].Id = startIdx + i + 1
	}
}

func recordLogHelper(ctx context.Context, log *Log) {
	requestId := helper.GetRequestID(ctx)
	if log.RequestId == "" {
		log.RequestId = requestId
	}
	var err error
	if config.LogShardByDay {
		tbl := logShardTableFromUnix(log.CreatedAt)
		if err = ensureLogShardTable(tbl); err != nil {
			logger.Error(ctx, "failed to ensure log shard table: "+err.Error())
			return
		}
		err = LOG_DB.Table(tbl).Create(log).Error
	} else {
		err = LOG_DB.Create(log).Error
	}
	if err != nil {
		logger.Error(ctx, "failed to record log: "+err.Error())
		return
	}
	logger.Infof(ctx, "record log: %+v", log)
}

func RecordLog(ctx context.Context, userId int, logType int, content string) {
	if logType == LogTypeConsume && !config.LogConsumeEnabled {
		return
	}
	log := &Log{
		UserId:    userId,
		Username:  GetUsernameById(userId),
		CreatedAt: helper.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	recordLogHelper(ctx, log)
}

func RecordTopupLog(ctx context.Context, userId int, content string, quota int) {
	log := &Log{
		UserId:    userId,
		Username:  GetUsernameById(userId),
		CreatedAt: helper.GetTimestamp(),
		Type:      LogTypeTopup,
		Content:   content,
		Quota:     quota,
	}
	recordLogHelper(ctx, log)
}

// applyLogTypeScope 列表查询：type=0 为「全部」时默认排除失败记录，includeErrors 为 true 时包含。
func applyLogTypeScope(db *gorm.DB, logType int, includeErrors bool) *gorm.DB {
	if logType != LogTypeUnknown {
		return db.Where("type = ?", logType)
	}
	if includeErrors {
		return db
	}
	return db.Where("type <> ?", LogTypeError)
}

func RecordErrorLog(ctx context.Context, log *Log) {
	if !config.ErrorLogEnabled {
		return
	}
	log.Username = GetUsernameById(log.UserId)
	log.CreatedAt = helper.GetTimestamp()
	log.Type = LogTypeError
	if log.Ip == "" {
		log.Ip = relayctx.ClientIP(ctx)
	}
	if log.UseTime <= 0 && log.ElapsedTime > 0 {
		log.UseTime = int(log.ElapsedTime / 1000)
		if log.UseTime == 0 {
			log.UseTime = 1
		}
	}
	recordLogHelper(ctx, log)
}

func RecordConsumeLog(ctx context.Context, log *Log) {
	if !config.LogConsumeEnabled {
		return
	}
	log.Username = GetUsernameById(log.UserId)
	log.CreatedAt = helper.GetTimestamp()
	log.Type = LogTypeConsume
	if log.Ip == "" {
		log.Ip = relayctx.ClientIP(ctx)
	}
	if log.UseTime <= 0 && log.ElapsedTime > 0 {
		log.UseTime = int(log.ElapsedTime / 1000)
		if log.UseTime == 0 {
			log.UseTime = 1
		}
	}
	extra := jsonStrToMap(log.Other)
	if log.SystemPromptReset {
		if extra == nil {
			extra = map[string]interface{}{}
		}
		extra["system_prompt_reset"] = true
	}
	if len(extra) > 0 {
		log.Other = jsonMapToStr(extra)
	}
	recordLogHelper(ctx, log)
}

func RecordTestLog(ctx context.Context, log *Log) {
	log.CreatedAt = helper.GetTimestamp()
	log.Type = LogTypeTest
	if log.UseTime <= 0 && log.ElapsedTime > 0 {
		log.UseTime = int(log.ElapsedTime / 1000)
		if log.UseTime == 0 {
			log.UseTime = 1
		}
	}
	recordLogHelper(ctx, log)
}

func fillChannelNames(logs []*Log) {
	if len(logs) == 0 {
		return
	}
	seen := map[int]struct{}{}
	var ids []int
	for _, lg := range logs {
		if lg != nil && lg.ChannelId != 0 {
			if _, ok := seen[lg.ChannelId]; !ok {
				seen[lg.ChannelId] = struct{}{}
				ids = append(ids, lg.ChannelId)
			}
		}
	}
	if len(ids) == 0 {
		return
	}
	var rows []struct {
		Id   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := DB.Table("channels").Select("id", "name").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		logger.SysError("fillChannelNames: " + err.Error())
		return
	}
	m := make(map[int]string, len(rows))
	for _, r := range rows {
		m[r.Id] = r.Name
	}
	for _, lg := range logs {
		if lg == nil {
			continue
		}
		if name := m[lg.ChannelId]; name != "" {
			lg.ChannelName = name
		}
	}
}

func GetLogByTokenId(tokenId int) (logs []*Log, err error) {
	if config.LogShardByDay {
		tables, e := discoverLogPhysicalTables()
		if e != nil {
			return nil, e
		}
		p := &shardFilterParams{TokenId: tokenId}
		logs, err = unionSelectLogs(tables, p, config.MaxRecentItems, 0)
	} else {
		err = LOG_DB.Model(&Log{}).Where("token_id = ?", tokenId).Order("id desc").Limit(config.MaxRecentItems).Find(&logs).Error
	}
	formatUserLogs(logs, 0)
	return logs, err
}

func GetAllLogs(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, startIdx int, num int, channel int, group string, requestId string, includeErrors bool) (logs []*Log, total int64, err error) {
	if config.LogShardByDay {
		tables, err := tablesForLogQuery(startTimestamp, endTimestamp)
		if err != nil {
			return nil, 0, err
		}
		p := &shardFilterParams{
			LogType:        logType,
			IncludeErrors:  includeErrors,
			ModelName:      modelName,
			Username:       username,
			TokenName:      tokenName,
			RequestID:      requestId,
			StartTimestamp: startTimestamp,
			EndTimestamp:   endTimestamp,
			Channel:        channel,
			Group:          group,
		}
		total, err = unionCountLogs(tables, p)
		if err != nil {
			return nil, 0, err
		}
		logs, err = unionSelectLogs(tables, p, num, startIdx)
		if err != nil {
			return nil, 0, err
		}
		AttachPublicUserIDToLogs(logs)
		fillChannelNames(logs)
		return logs, total, err
	}

	tx := applyLogTypeScope(LOG_DB.Model(&Log{}), logType, includeErrors)
	if modelName != "" {
		modelNamePattern, perr := sanitizeLikePattern(modelName)
		if perr != nil {
			return nil, 0, perr
		}
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("request_id = ?", requestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
	}
	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	AttachPublicUserIDToLogs(logs)
	fillChannelNames(logs)
	return logs, total, err
}

func GetUserLogs(userId int, logType int, startTimestamp int64, endTimestamp int64, modelName string, tokenName string, startIdx int, num int, group string, requestId string, includeErrors bool) (logs []*Log, total int64, err error) {
	if config.LogShardByDay {
		tables, err := tablesForLogQuery(startTimestamp, endTimestamp)
		if err != nil {
			return nil, 0, errors.New("查询日志失败")
		}
		p := &shardFilterParams{
			LogType:        logType,
			IncludeErrors:  includeErrors,
			UserID:         &userId,
			ModelName:      modelName,
			TokenName:      tokenName,
			RequestID:      requestId,
			StartTimestamp: startTimestamp,
			EndTimestamp:   endTimestamp,
			Group:          group,
		}
		total, err = unionCountLogs(tables, p)
		if err != nil {
			logger.SysError("failed to count user logs: " + err.Error())
			return nil, 0, errors.New("查询日志失败")
		}
		logs, err = unionSelectLogs(tables, p, num, startIdx)
		if err != nil {
			logger.SysError("failed to search user logs: " + err.Error())
			return nil, 0, errors.New("查询日志失败")
		}
		AttachPublicUserIDToLogs(logs)
		formatUserLogs(logs, startIdx)
		return logs, total, err
	}

	tx := applyLogTypeScope(LOG_DB.Where("user_id = ?", userId), logType, includeErrors)
	if modelName != "" {
		modelNamePattern, perr := sanitizeLikePattern(modelName)
		if perr != nil {
			return nil, 0, perr
		}
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("request_id = ?", requestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
	}
	if err = tx.Model(&Log{}).Limit(logSearchCountLimit).Count(&total).Error; err != nil {
		logger.SysError("failed to count user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		logger.SysError("failed to search user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}
	AttachPublicUserIDToLogs(logs)
	formatUserLogs(logs, startIdx)
	return logs, total, err
}

func SearchAllLogs(keyword string) (logs []*Log, err error) {
	if config.LogShardByDay {
		tables, err := discoverLogPhysicalTables()
		if err != nil {
			return nil, err
		}
		p := &shardFilterParams{
			SearchKeyword:       keyword,
			SearchContentPrefix: keyword + "%",
		}
		logs, err = unionSelectLogs(tables, p, config.MaxRecentItems, 0)
	} else {
		err = LOG_DB.Where("type = ? or content LIKE ?", keyword, keyword+"%").Order("id desc").Limit(config.MaxRecentItems).Find(&logs).Error
	}
	if err != nil {
		return logs, err
	}
	AttachPublicUserIDToLogs(logs)
	return logs, err
}

func SearchUserLogs(userId int, keyword string) (logs []*Log, err error) {
	if config.LogShardByDay {
		tables, err := discoverLogPhysicalTables()
		if err != nil {
			return nil, err
		}
		uid := userId
		p := &shardFilterParams{
			UserID:       &uid,
			TypeStringEq: keyword,
		}
		logs, err = unionSelectLogs(tables, p, config.MaxRecentItems, 0)
	} else {
		err = LOG_DB.Where("user_id = ? and type = ?", userId, keyword).Order("id desc").Limit(config.MaxRecentItems).Find(&logs).Error
	}
	if err != nil {
		return logs, err
	}
	AttachPublicUserIDToLogs(logs)
	formatUserLogs(logs, 0)
	return logs, err
}

// AttachPublicUserIDToLogs 为日志列表填充对外用户 ID（users.uid），不修改数据库。
func AttachPublicUserIDToLogs(logs []*Log) {
	if len(logs) == 0 {
		return
	}
	seen := make(map[int]struct{})
	var ids []int
	for _, lg := range logs {
		if lg == nil {
			continue
		}
		if _, ok := seen[lg.UserId]; !ok {
			seen[lg.UserId] = struct{}{}
			ids = append(ids, lg.UserId)
		}
	}
	if len(ids) == 0 {
		return
	}
	type row struct {
		Id  int
		Uid string
	}
	var rows []row
	if err := DB.Table("users").Select("id", "uid").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		logger.SysError("AttachPublicUserIDToLogs: " + err.Error())
		return
	}
	m := make(map[int]string, len(rows))
	for _, r := range rows {
		m[r.Id] = r.Uid
	}
	for _, lg := range logs {
		if lg == nil {
			continue
		}
		if u, ok := m[lg.UserId]; ok {
			lg.UserPublicID = u
		}
	}
}

type LogUsageStat struct {
	Quota int `json:"quota"`
	Rpm   int `json:"rpm"`
	Tpm   int `json:"tpm"`
}

type quotaSumRow struct {
	Quota int `gorm:"column:quota"`
}

type rpmTpmRow struct {
	Rpm int `gorm:"column:rpm"`
	Tpm int `gorm:"column:tpm"`
}

func SumUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int, group string) (stat LogUsageStat, err error) {
	var qSel, rtSel string
	if common.UsingPostgreSQL {
		qSel = "COALESCE(sum(quota),0) AS quota"
		rtSel = "count(*) AS rpm, COALESCE(sum(prompt_tokens),0) + COALESCE(sum(completion_tokens),0) AS tpm"
	} else {
		qSel = "ifnull(sum(quota),0) AS quota"
		rtSel = "count(*) AS rpm, ifnull(sum(prompt_tokens),0) + ifnull(sum(completion_tokens),0) AS tpm"
	}

	if config.LogShardByDay {
		tables, err := tablesForLogQuery(startTimestamp, endTimestamp)
		if err != nil {
			logger.SysError("failed to query log stat: " + err.Error())
			return stat, errors.New("查询统计数据失败")
		}
		rpmSince := time.Now().Add(-60 * time.Second).Unix()
		rtTables := filterLogTablesByTimeRange(tables, rpmSince, time.Now().Unix())
		for _, tbl := range tables {
			tx := LOG_DB.Table(tbl).Select(qSel)
			tx, err = applyLogConsumeStatFilters(tx, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
			if err != nil {
				return stat, err
			}
			tx = tx.Where("type = ?", LogTypeConsume)
			var qRow quotaSumRow
			if err = tx.Scan(&qRow).Error; err != nil {
				logger.SysError("failed to query log stat: " + err.Error())
				return stat, errors.New("查询统计数据失败")
			}
			stat.Quota += qRow.Quota
		}
		for _, tbl := range rtTables {
			rtx := LOG_DB.Table(tbl).Select(rtSel)
			rtx, err = applyLogConsumeStatFilters(rtx, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
			if err != nil {
				return stat, err
			}
			rtx = rtx.Where("type = ?", LogTypeConsume).Where("created_at >= ?", rpmSince)
			var rtRow rpmTpmRow
			if err = rtx.Scan(&rtRow).Error; err != nil {
				logger.SysError("failed to query rpm/tpm stat: " + err.Error())
				return stat, errors.New("查询统计数据失败")
			}
			stat.Rpm += rtRow.Rpm
			stat.Tpm += rtRow.Tpm
		}
		return stat, nil
	}

	tx := LOG_DB.Table("logs").Select(qSel)
	rpmTpmQuery := LOG_DB.Table("logs").Select(rtSel)

	if username != "" {
		tx = tx.Where("username = ?", username)
		rpmTpmQuery = rpmTpmQuery.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
		rpmTpmQuery = rpmTpmQuery.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
		rpmTpmQuery = rpmTpmQuery.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
		rpmTpmQuery = rpmTpmQuery.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		modelNamePattern, perr := sanitizeLikePattern(modelName)
		if perr != nil {
			return stat, perr
		}
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
		rpmTpmQuery = rpmTpmQuery.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
		rpmTpmQuery = rpmTpmQuery.Where("channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
		rpmTpmQuery = rpmTpmQuery.Where(logGroupCol+" = ?", group)
	}

	tx = tx.Where("type = ?", LogTypeConsume)
	rpmTpmQuery = rpmTpmQuery.Where("type = ?", LogTypeConsume)
	rpmTpmQuery = rpmTpmQuery.Where("created_at >= ?", time.Now().Add(-60*time.Second).Unix())

	var qRow quotaSumRow
	if err = tx.Scan(&qRow).Error; err != nil {
		logger.SysError("failed to query log stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	var rtRow rpmTpmRow
	if err = rpmTpmQuery.Scan(&rtRow).Error; err != nil {
		logger.SysError("failed to query rpm/tpm stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	stat.Quota = qRow.Quota
	stat.Rpm = rtRow.Rpm
	stat.Tpm = rtRow.Tpm
	return stat, nil
}

func SumUsedToken(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string) (token int) {
	ifnull := "ifnull"
	if common.UsingPostgreSQL {
		ifnull = "COALESCE"
	}
	sel := fmt.Sprintf("%s(sum(prompt_tokens),0) + %s(sum(completion_tokens),0)", ifnull, ifnull)

	if config.LogShardByDay {
		tables, err := tablesForLogQuery(startTimestamp, endTimestamp)
		if err != nil {
			return 0
		}
		for _, tbl := range tables {
			tx := LOG_DB.Table(tbl).Select(sel)
			tx = applyLogConsumeTokenSumFilters(tx, startTimestamp, endTimestamp, modelName, username, tokenName)
			var part int
			tx.Where("type = ?", LogTypeConsume).Scan(&part)
			token += part
		}
		return token
	}

	tx := LOG_DB.Table("logs").Select(sel)
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
	tx.Where("type = ?", LogTypeConsume).Scan(&token)
	return token
}

func DeleteOldLog(targetTimestamp int64) (int64, error) {
	if config.LogShardByDay {
		tables, err := discoverLogPhysicalTables()
		if err != nil {
			return 0, err
		}
		var n int64
		for _, tbl := range tables {
			r := LOG_DB.Table(tbl).Where("created_at < ?", targetTimestamp).Delete(&Log{})
			if r.Error != nil {
				return n, r.Error
			}
			n += r.RowsAffected
		}
		return n, nil
	}
	result := LOG_DB.Where("created_at < ?", targetTimestamp).Delete(&Log{})
	return result.RowsAffected, result.Error
}

type LogStatistic struct {
	Day              string `gorm:"column:day"`
	ModelName        string `gorm:"column:model_name"`
	RequestCount     int    `gorm:"column:request_count"`
	Quota            int    `gorm:"column:quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens"`
}

func SearchLogsByDayAndModel(userId, start, end int) (LogStatistics []*LogStatistic, err error) {
	groupSelect := "DATE_FORMAT(FROM_UNIXTIME(created_at), '%Y-%m-%d') as day"

	if common.UsingPostgreSQL {
		groupSelect = "TO_CHAR(date_trunc('day', to_timestamp(created_at)), 'YYYY-MM-DD') as day"
	}

	if common.UsingSQLite {
		groupSelect = "strftime('%Y-%m-%d', datetime(created_at, 'unixepoch')) as day"
	}

	if config.LogShardByDay {
		tables, err := tablesForLogQuery(int64(start), int64(end))
		if err != nil {
			return nil, err
		}
		merged := make(map[string]*LogStatistic)
		innerSQL := `SELECT ` + groupSelect + `,
		model_name, count(1) as request_count,
		sum(quota) as quota,
		sum(prompt_tokens) as prompt_tokens,
		sum(completion_tokens) as completion_tokens
		FROM %s
		WHERE type=?
		AND user_id= ?
		AND created_at BETWEEN ? AND ?
		GROUP BY day, model_name`
		for _, tbl := range tables {
			q := fmt.Sprintf(innerSQL, quoteLogIdent(tbl))
			var batch []LogStatistic
			if err := LOG_DB.Raw(q, LogTypeConsume, userId, start, end).Scan(&batch).Error; err != nil {
				return nil, err
			}
			for _, row := range batch {
				key := row.Day + "\x00" + row.ModelName
				if ex, ok := merged[key]; ok {
					ex.RequestCount += row.RequestCount
					ex.Quota += row.Quota
					ex.PromptTokens += row.PromptTokens
					ex.CompletionTokens += row.CompletionTokens
				} else {
					cp := row
					merged[key] = &cp
				}
			}
		}
		out := make([]*LogStatistic, 0, len(merged))
		for _, v := range merged {
			out = append(out, v)
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Day != out[j].Day {
				return out[i].Day < out[j].Day
			}
			return out[i].ModelName < out[j].ModelName
		})
		return out, nil
	}

	err = LOG_DB.Raw(`
		SELECT `+groupSelect+`,
		model_name, count(1) as request_count,
		sum(quota) as quota,
		sum(prompt_tokens) as prompt_tokens,
		sum(completion_tokens) as completion_tokens
		FROM logs
		WHERE type=2
		AND user_id= ?
		AND created_at BETWEEN ? AND ?
		GROUP BY day, model_name
		ORDER BY day, model_name
	`, userId, start, end).Scan(&LogStatistics).Error

	return LogStatistics, err
}
