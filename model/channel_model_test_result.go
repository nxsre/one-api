package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"gorm.io/gorm/clause"
)

// ChannelModelTestResult 渠道模型探测结果（按 channel + base_url + model 保留最后一次）。
type ChannelModelTestResult struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ChannelId   int    `json:"channel_id" gorm:"uniqueIndex:idx_cmtr_unique,priority:1;index"`
	BaseURL     string `json:"base_url" gorm:"size:768;not null;default:''"`
	BaseURLHash string `json:"-" gorm:"size:64;uniqueIndex:idx_cmtr_unique,priority:2;column:base_url_hash;not null;default:''"`
	ModelId     string `json:"model_id" gorm:"uniqueIndex:idx_cmtr_unique,priority:3;size:191;not null;column:model_id"`
	Success      bool   `json:"success" gorm:"not null"`
	Skipped      bool   `json:"skipped" gorm:"not null;default:0"`
	TimedOut     bool   `json:"timed_out" gorm:"not null;default:0"`
	TestKind     string `json:"test_kind" gorm:"size:32;not null;default:''"`
	TestProtocol string `json:"test_protocol" gorm:"size:64;not null;default:''"`
	Message      string `json:"message" gorm:"type:text"`
	DetailJSON   string `json:"-" gorm:"type:text;column:detail_json"`
	StartedAt    int64  `json:"started_at" gorm:"bigint;default:0"`
	ElapsedMs    int64  `json:"elapsed_ms" gorm:"default:0"`
	TestedAt     int64  `json:"tested_at" gorm:"bigint;index;not null"`
}

func channelTestBaseURLHash(baseURL string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(baseURL)))
	return hex.EncodeToString(sum[:])
}

// ChannelTestScopeKey 渠道 + Base URL 测试任务幂等键（channel_id=0 表示未保存渠道）。
func ChannelTestScopeKey(channelID int, baseURL string) string {
	h := channelTestBaseURLHash(baseURL)
	if channelID <= 0 {
		return "0:" + h
	}
	return strconv.Itoa(channelID) + ":" + h
}

func NormalizeChannelTestBaseURL(formBase string, channelType int) string {
	base := strings.TrimRight(strings.TrimSpace(formBase), "/")
	if base != "" {
		return base
	}
	if channelType > 0 && channelType < channeltype.Dummy {
		return strings.TrimRight(strings.TrimSpace(channeltype.DefaultBaseURL(channelType)), "/")
	}
	return ""
}

// UpsertChannelModelTestResult 写入/更新某渠道在某 Base URL 下某模型的最后一次测试结果。
func UpsertChannelModelTestResult(channelID int, baseURL, modelID string, success, skipped, timedOut bool, testKind, testProtocol, message, detailJSON string, startedAt, elapsedMs int64) error {
	if channelID <= 0 {
		return nil
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil
	}
	base := strings.TrimSpace(baseURL)
	row := ChannelModelTestResult{
		ChannelId:   channelID,
		BaseURL:     base,
		BaseURLHash: channelTestBaseURLHash(base),
		ModelId:     modelID,
		Success:      success,
		Skipped:      skipped,
		TimedOut:     timedOut,
		TestKind:     strings.TrimSpace(testKind),
		TestProtocol: strings.TrimSpace(testProtocol),
		Message:      strings.TrimSpace(message),
		DetailJSON:   strings.TrimSpace(detailJSON),
		StartedAt:    startedAt,
		ElapsedMs:    elapsedMs,
		TestedAt:     helper.GetTimestamp(),
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "channel_id"},
			{Name: "base_url_hash"},
			{Name: "model_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"base_url", "success", "skipped", "timed_out", "test_kind", "test_protocol", "message", "detail_json", "started_at", "elapsed_ms", "tested_at"}),
	}).Create(&row).Error
}

// ListChannelModelTestResults 列出某渠道在某 Base URL 下各模型的最后一次测试结果。
func ListChannelModelTestResults(channelID int, baseURL string) ([]ChannelModelTestResult, error) {
	if channelID <= 0 {
		return nil, nil
	}
	base := strings.TrimSpace(baseURL)
	var rows []ChannelModelTestResult
	err := DB.Where("channel_id = ? AND base_url_hash = ?", channelID, channelTestBaseURLHash(base)).
		Order("model_id asc").
		Find(&rows).Error
	return rows, err
}

// DeleteChannelModelTestResultsByChannel 删除渠道时清理探测记录。
func DeleteChannelModelTestResultsByChannel(channelID int) error {
	if channelID <= 0 {
		return nil
	}
	return DB.Where("channel_id = ?", channelID).Delete(&ChannelModelTestResult{}).Error
}
