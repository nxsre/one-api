package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/dto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	UpstreamSyncStatusRunning   = "running"
	UpstreamSyncStatusCompleted = "completed"
	UpstreamSyncStatusFailed    = "failed"

	upstreamSyncDiffBatchSize = 500
)

// UpstreamSyncBatch 上游价目拉取批次。
type UpstreamSyncBatch struct {
	Id             int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Status         string `json:"status" gorm:"size:32;index"`
	ChannelIds     string `json:"channel_ids" gorm:"type:text"`
	TestResults    string `json:"test_results" gorm:"type:text"`
	LocalSnapshot  string `json:"local_snapshot" gorm:"type:longtext"`
	UpstreamData   string `json:"upstream_data" gorm:"type:longtext"`
	DiffCount      int    `json:"diff_count"`
	SelectionCount int    `json:"selection_count"`
	ErrorMessage   string `json:"error_message" gorm:"type:text"`
	CreatedTime    int64  `json:"created_time" gorm:"bigint;index"`
	CompletedTime  int64  `json:"completed_time" gorm:"bigint"`
}

// UpstreamSyncDiff 批次内扁平化差异行。
type UpstreamSyncDiff struct {
	Id           int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	BatchId      int64  `json:"batch_id" gorm:"uniqueIndex:idx_usd_batch_model_field,priority:1;index"`
	Model        string `json:"model" gorm:"uniqueIndex:idx_usd_batch_model_field,priority:2;size:512;index"`
	Field        string `json:"field" gorm:"uniqueIndex:idx_usd_batch_model_field,priority:3;size:64"`
	CurrentValue string `json:"current_value" gorm:"type:text"`
	Upstreams    string `json:"upstreams" gorm:"type:text"`
	Confidence   string `json:"confidence" gorm:"type:text"`
	CreatedTime  int64  `json:"created_time" gorm:"bigint"`
}

// UpstreamSyncSelection 批次内用户勾选临时表。
type UpstreamSyncSelection struct {
	Id           int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	BatchId      int64  `json:"batch_id" gorm:"uniqueIndex:idx_uss_batch_model_field,priority:1;index"`
	Model        string `json:"model" gorm:"uniqueIndex:idx_uss_batch_model_field,priority:2;size:512"`
	Field        string `json:"field" gorm:"uniqueIndex:idx_uss_batch_model_field,priority:3;size:64"`
	UpstreamName string `json:"upstream_name" gorm:"size:256"`
	Selected     bool   `json:"selected"`
	UpdatedTime  int64  `json:"updated_time" gorm:"bigint"`
}

type UpstreamSyncDiffListParams struct {
	BatchId  int64
	Page     int
	PageSize int
	Model    string
	Selected *bool
}

type UpstreamSyncDiffListResult struct {
	Items      []UpstreamSyncDiffRow `json:"items"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

type UpstreamSyncDiffRow struct {
	Id           int64                  `json:"id"`
	Model        string                 `json:"model"`
	Field        string                 `json:"field"`
	Current      interface{}            `json:"current"`
	Upstreams    map[string]interface{} `json:"upstreams"`
	Confidence   map[string]bool        `json:"confidence"`
	Selected     bool                   `json:"selected"`
	UpstreamName string                 `json:"upstream_name"`
}

type UpstreamSnapshot struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data"`
}

func InitUpstreamSyncStore() error {
	return DB.AutoMigrate(&UpstreamSyncBatch{}, &UpstreamSyncDiff{}, &UpstreamSyncSelection{})
}

func CreateUpstreamSyncBatch(channelIDs []int64) (*UpstreamSyncBatch, error) {
	raw, _ := json.Marshal(channelIDs)
	now := time.Now().Unix()
	batch := &UpstreamSyncBatch{
		Status:      UpstreamSyncStatusRunning,
		ChannelIds:  string(raw),
		CreatedTime: now,
	}
	if err := DB.Create(batch).Error; err != nil {
		return nil, err
	}
	return batch, nil
}

func GetUpstreamSyncBatch(id int64) (*UpstreamSyncBatch, error) {
	var batch UpstreamSyncBatch
	if err := DB.First(&batch, id).Error; err != nil {
		return nil, err
	}
	return &batch, nil
}

func ListUpstreamSyncBatches(limit int) ([]UpstreamSyncBatch, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var batches []UpstreamSyncBatch
	err := DB.Order("id desc").Limit(limit).Find(&batches).Error
	return batches, err
}

func GetLatestUpstreamSyncBatch() (*UpstreamSyncBatch, error) {
	var batch UpstreamSyncBatch
	err := DB.Where("status = ?", UpstreamSyncStatusCompleted).Order("id desc").First(&batch).Error
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func FinishUpstreamSyncBatch(id int64, status string, testResults []dto.TestResult, localSnapshot, upstreamData map[string]any, diffCount int, errMsg string) error {
	testRaw, _ := json.Marshal(testResults)
	localRaw, _ := json.Marshal(localSnapshot)
	upRaw, _ := json.Marshal(upstreamData)
	now := time.Now().Unix()
	return DB.Model(&UpstreamSyncBatch{}).Where("id = ?", id).Updates(map[string]any{
		"status":          status,
		"test_results":    string(testRaw),
		"local_snapshot":  string(localRaw),
		"upstream_data":   string(upRaw),
		"diff_count":      diffCount,
		"error_message":   errMsg,
		"completed_time":  now,
	}).Error
}

func upstreamSyncDiffRowKey(model, field string) string {
	return strings.TrimSpace(model) + "\x00" + strings.TrimSpace(field)
}

func ReplaceUpstreamSyncDiffs(batchID int64, differences map[string]map[string]dto.DifferenceItem) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("batch_id = ?", batchID).Delete(&UpstreamSyncDiff{}).Error; err != nil {
			return err
		}
		now := time.Now().Unix()
		seen := make(map[string]struct{})
		rows := make([]UpstreamSyncDiff, 0, 256)
		for modelName, fieldMap := range differences {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" {
				continue
			}
			for field, item := range fieldMap {
				field = strings.TrimSpace(field)
				if field == "" {
					continue
				}
				key := upstreamSyncDiffRowKey(modelName, field)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				curRaw, _ := json.Marshal(item.Current)
				upRaw, _ := json.Marshal(item.Upstreams)
				confRaw, _ := json.Marshal(item.Confidence)
				rows = append(rows, UpstreamSyncDiff{
					BatchId:      batchID,
					Model:        modelName,
					Field:        field,
					CurrentValue: string(curRaw),
					Upstreams:    string(upRaw),
					Confidence:   string(confRaw),
					CreatedTime:  now,
				})
			}
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "batch_id"},
				{Name: "model"},
				{Name: "field"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"current_value", "upstreams", "confidence", "created_time",
			}),
		}).CreateInBatches(rows, upstreamSyncDiffBatchSize).Error
	})
}

func DecodeUpstreamSyncDiffRow(row UpstreamSyncDiff, sel *UpstreamSyncSelection) UpstreamSyncDiffRow {
	return decodeDiffRow(row, sel)
}

func LoadVersionEntriesAsFieldMap(blockID string, versionID int) (map[string]any, error) {
	var rows []PricingVersionEntry
	if err := DB.Where("block_id = ? AND version_id = ?", blockID, versionID).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("版本快照为空")
	}
	out := make(map[string]any, len(rows))
	switch {
	case isGroupGroupBlock(blockID):
		for _, r := range rows {
			sub, ok := out[r.EntryKey].(map[string]any)
			if !ok {
				sub = make(map[string]any)
				out[r.EntryKey] = sub
			}
			if f, err := strconv.ParseFloat(r.ValueText, 64); err == nil {
				sub[r.SubKey] = f
			}
		}
	case isStringValueBlock(blockID):
		for _, r := range rows {
			out[r.EntryKey] = r.ValueText
		}
	default:
		for _, r := range rows {
			if f, err := strconv.ParseFloat(r.ValueText, 64); err == nil {
				out[r.EntryKey] = f
			}
		}
	}
	return out, nil
}

func decodeDiffRow(row UpstreamSyncDiff, sel *UpstreamSyncSelection) UpstreamSyncDiffRow {
	out := UpstreamSyncDiffRow{
		Id:    row.Id,
		Model: row.Model,
		Field: row.Field,
	}
	_ = json.Unmarshal([]byte(row.CurrentValue), &out.Current)
	_ = json.Unmarshal([]byte(row.Upstreams), &out.Upstreams)
	_ = json.Unmarshal([]byte(row.Confidence), &out.Confidence)
	if out.Upstreams == nil {
		out.Upstreams = map[string]interface{}{}
	}
	if out.Confidence == nil {
		out.Confidence = map[string]bool{}
	}
	if sel != nil {
		out.Selected = sel.Selected
		out.UpstreamName = sel.UpstreamName
	}
	return out
}

func ListUpstreamSyncDiffs(params UpstreamSyncDiffListParams) (*UpstreamSyncDiffListResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 50
	}
	if params.PageSize > 200 {
		params.PageSize = 200
	}
	q := DB.Model(&UpstreamSyncDiff{}).Where("batch_id = ?", params.BatchId)
	if m := strings.TrimSpace(params.Model); m != "" {
		q = q.Where("model LIKE ?", "%"+m+"%")
	}
	if params.Selected != nil {
		sub := DB.Model(&UpstreamSyncSelection{}).
			Select("1").
			Where("upstream_sync_selections.batch_id = upstream_sync_diffs.batch_id").
			Where("upstream_sync_selections.model = upstream_sync_diffs.model").
			Where("upstream_sync_selections.field = upstream_sync_diffs.field").
			Where("upstream_sync_selections.selected = ?", *params.Selected)
		q = q.Where("EXISTS (?)", sub)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []UpstreamSyncDiff
	offset := (params.Page - 1) * params.PageSize
	if err := q.Order("model asc, field asc").Offset(offset).Limit(params.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	selMap, err := loadSelectionMap(params.BatchId)
	if err != nil {
		return nil, err
	}
	items := make([]UpstreamSyncDiffRow, 0, len(rows))
	for _, row := range rows {
		key := selectionKey(row.Model, row.Field)
		items = append(items, decodeDiffRow(row, selMap[key]))
	}
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize != 0 {
		totalPages++
	}
	return &UpstreamSyncDiffListResult{
		Items:      items,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

func selectionKey(model, field string) string {
	return model + "\x00" + field
}

func loadSelectionMap(batchID int64) (map[string]*UpstreamSyncSelection, error) {
	var sels []UpstreamSyncSelection
	if err := DB.Where("batch_id = ?", batchID).Find(&sels).Error; err != nil {
		return nil, err
	}
	out := make(map[string]*UpstreamSyncSelection, len(sels))
	for i := range sels {
		key := selectionKey(sels[i].Model, sels[i].Field)
		out[key] = &sels[i]
	}
	return out, nil
}

func SaveUpstreamSyncSelections(batchID int64, items []dto.UpstreamSyncSelectionItem) (int, error) {
	if len(items) == 0 {
		return countUpstreamSyncSelections(batchID)
	}
	now := time.Now().Unix()
	count := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			modelName := strings.TrimSpace(item.Model)
			field := strings.TrimSpace(item.Field)
			if modelName == "" || field == "" {
				continue
			}
			sel := UpstreamSyncSelection{
				BatchId:      batchID,
				Model:        modelName,
				Field:        field,
				UpstreamName: strings.TrimSpace(item.UpstreamName),
				Selected:     item.Selected,
				UpdatedTime:  now,
			}
			var existing UpstreamSyncSelection
			err := tx.Where("batch_id = ? AND model = ? AND field = ?", batchID, modelName, field).First(&existing).Error
			if err == nil {
				if err := tx.Model(&existing).Updates(map[string]any{
					"upstream_name": sel.UpstreamName,
					"selected":      sel.Selected,
					"updated_time":  now,
				}).Error; err != nil {
					return err
				}
			} else if err == gorm.ErrRecordNotFound {
				if err := tx.Create(&sel).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		var selectedTotal int64
		if err := tx.Model(&UpstreamSyncSelection{}).
			Where("batch_id = ? AND selected = ?", batchID, true).
			Count(&selectedTotal).Error; err != nil {
			return err
		}
		count = int(selectedTotal)
		return tx.Model(&UpstreamSyncBatch{}).Where("id = ?", batchID).Update("selection_count", count).Error
	})
	return count, err
}

func countUpstreamSyncSelections(batchID int64) (int, error) {
	var selectedTotal int64
	if err := DB.Model(&UpstreamSyncSelection{}).
		Where("batch_id = ? AND selected = ?", batchID, true).
		Count(&selectedTotal).Error; err != nil {
		return 0, err
	}
	count := int(selectedTotal)
	_ = DB.Model(&UpstreamSyncBatch{}).Where("id = ?", batchID).Update("selection_count", count).Error
	return count, nil
}

func SelectAllUpstreamSyncDiffs(batchID int64, upstreamName string, selected bool) (int, error) {
	var diffs []UpstreamSyncDiff
	if err := DB.Where("batch_id = ?", batchID).Find(&diffs).Error; err != nil {
		return 0, err
	}
	items := make([]dto.UpstreamSyncSelectionItem, 0, len(diffs))
	for _, d := range diffs {
		row := decodeDiffRow(d, nil)
		name := strings.TrimSpace(upstreamName)
		if name == "" {
			for n, v := range row.Upstreams {
				if v != nil && v != "same" {
					name = n
					break
				}
			}
		}
		items = append(items, dto.UpstreamSyncSelectionItem{
			Model:        d.Model,
			Field:        d.Field,
			UpstreamName: name,
			Selected:     selected,
		})
	}
	return SaveUpstreamSyncSelections(batchID, items)
}

func ListUpstreamSyncSelections(batchID int64) ([]UpstreamSyncSelection, error) {
	var sels []UpstreamSyncSelection
	err := DB.Where("batch_id = ? AND selected = ?", batchID, true).Find(&sels).Error
	return sels, err
}

func LoadUpstreamSnapshots(batch *UpstreamSyncBatch) ([]UpstreamSnapshot, error) {
	if batch == nil || strings.TrimSpace(batch.UpstreamData) == "" {
		return nil, nil
	}
	var wrapper struct {
		Snapshots []UpstreamSnapshot `json:"snapshots"`
	}
	if err := json.Unmarshal([]byte(batch.UpstreamData), &wrapper); err == nil && len(wrapper.Snapshots) > 0 {
		return wrapper.Snapshots, nil
	}
	var snaps []UpstreamSnapshot
	if err := json.Unmarshal([]byte(batch.UpstreamData), &snaps); err != nil {
		return nil, err
	}
	return snaps, nil
}

func LoadLocalSnapshot(batch *UpstreamSyncBatch) (map[string]any, error) {
	if batch == nil || strings.TrimSpace(batch.LocalSnapshot) == "" {
		return nil, fmt.Errorf("empty local snapshot")
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(batch.LocalSnapshot), &data); err != nil {
		return nil, err
	}
	return data, nil
}

func ParseBatchTestResults(batch *UpstreamSyncBatch) []dto.TestResult {
	if batch == nil || strings.TrimSpace(batch.TestResults) == "" {
		return nil
	}
	var out []dto.TestResult
	_ = json.Unmarshal([]byte(batch.TestResults), &out)
	return out
}
