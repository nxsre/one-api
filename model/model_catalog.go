package model

import (
	"errors"
	"strings"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/helper"
	"gorm.io/gorm"
)

// ModelCatalog 管理员维护的模型目录：可增补模型名、覆盖分组标签（owned_by），或通过 enabled=false 从下拉中隐藏内置项。
// Models.dev 对齐字段在通过 sync_models_dev 同步时写入；其它来源或手工录入通常为空。
type ModelCatalog struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ModelId     string `json:"model_id" gorm:"uniqueIndex:idx_mc_src_mid_ver,priority:2;index:idx_mc_src_mid_status,priority:2;size:191;column:model_id"`
	OwnedBy     string `json:"owned_by" gorm:"size:256"`
	Enabled     bool   `json:"enabled" gorm:"default:true;index"`
	Source      string `json:"source" gorm:"uniqueIndex:idx_mc_src_mid_ver,priority:1;index:idx_mc_src_mid_status,priority:1;size:64"`
	Version     int    `json:"version" gorm:"uniqueIndex:idx_mc_src_mid_ver,priority:3;not null;default:1"`
	Status      string `json:"status" gorm:"index:idx_mc_src_mid_status,priority:3;size:16;not null;default:'current'"`
	Notes       string `json:"notes" gorm:"type:text"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`

	ModelName       string  `json:"model_name" gorm:"size:512"`
	ProviderKey     string  `json:"provider_key" gorm:"size:191;index"`
	ProviderDisplay string  `json:"provider_display" gorm:"size:512"`
	Family          string  `json:"family" gorm:"size:191"`
	NpmPackage      string  `json:"npm_package" gorm:"size:256"`
	APIBase         string  `json:"api_base" gorm:"size:768"`
	DocURL          string  `json:"doc_url" gorm:"size:768"`
	ModalitiesIn    string  `json:"modalities_in" gorm:"size:256"`
	ModalitiesOut   string  `json:"modalities_out" gorm:"size:256"`
	ContextLimit    int     `json:"context_limit" gorm:"default:0"`
	OutputLimit     int     `json:"output_limit" gorm:"default:0"`
	CostInput       float64 `json:"cost_input" gorm:"default:0"`
	CostOutput      float64 `json:"cost_output" gorm:"default:0"`
	CostCacheRead   float64 `json:"cost_cache_read" gorm:"default:0"`
	CostCacheWrite  float64 `json:"cost_cache_write" gorm:"default:0"`
	Reasoning       bool    `json:"reasoning" gorm:"default:false"`
	ToolCall        bool    `json:"tool_call" gorm:"default:false"`
	TemperatureOK   bool    `json:"temperature_ok" gorm:"default:false"`
	AttachmentOK    bool    `json:"attachment_ok" gorm:"default:false"`
	OpenWeights     bool    `json:"open_weights" gorm:"default:false"`
	KnowledgeCutoff string  `json:"knowledge_cutoff" gorm:"size:64"`
	ReleaseDate     string  `json:"release_date" gorm:"size:64"`
	LastUpdatedDev  string  `json:"last_updated" gorm:"size:64"`
	Tags            string  `json:"tags" gorm:"size:256"`
}

func GetAllModelCatalog() ([]ModelCatalog, error) {
	var rows []ModelCatalog
	err := DB.Order("model_id asc").Find(&rows).Error
	return rows, err
}

// ModelCatalogListParams GET /api/model_catalog 服务端分页参数。
type ModelCatalogListParams struct {
	Page           int
	PageSize       int
	Search         string
	FilterModelID       string
	FilterModelName     string
	FilterProvider      string
	FilterProviderKey   string
	FilterFamily        string
	FilterModalitiesIn  string
	FilterModalitiesOut string
	SortBy         string
	SortDesc       bool
	IncludeExpired bool
}

var modelCatalogSortColumns = map[string]string{
	"id":               "id",
	"model_id":         "model_id",
	"model_name":       "model_name",
	"provider_key":     "provider_key",
	"provider_display": "provider_display",
	"family":           "family",
	"modalities_in":    "modalities_in",
	"modalities_out":   "modalities_out",
	"context_limit":    "context_limit",
	"output_limit":     "output_limit",
	"cost_input":       "cost_input",
	"cost_output":      "cost_output",
	"cost_cache_read":  "cost_cache_read",
	"cost_cache_write": "cost_cache_write",
	"reasoning":        "reasoning",
	"tool_call":        "tool_call",
	"temperature_ok":   "temperature_ok",
	"attachment_ok":    "attachment_ok",
	"open_weights":     "open_weights",
	"knowledge_cutoff": "knowledge_cutoff",
	"release_date":     "release_date",
	"last_updated":     "last_updated_dev",
	"npm_package":      "npm_package",
	"api_base":         "api_base",
	"doc_url":          "doc_url",
	"owned_by":         "owned_by",
	"enabled":          "enabled",
	"source":           "source",
	"notes":            "notes",
	"created_time":     "created_time",
	"updated_time":     "updated_time",
}

func modelCatalogLikePattern(term string) string {
	t := strings.TrimSpace(term)
	t = strings.ReplaceAll(t, "%", "")
	t = strings.ReplaceAll(t, "_", "")
	t = strings.ReplaceAll(t, "\\", "")
	if t == "" {
		return ""
	}
	return "%" + t + "%"
}

func modelCatalogWhereColumnLike(tx *gorm.DB, column string, raw string) *gorm.DB {
	for _, term := range strings.Fields(strings.TrimSpace(raw)) {
		pat := modelCatalogLikePattern(term)
		if pat == "" {
			continue
		}
		if common.UsingPostgreSQL {
			tx = tx.Where(column+" ILIKE ?", pat)
		} else {
			tx = tx.Where(column+" LIKE ?", pat)
		}
	}
	return tx
}

func modelCatalogWhereProviderLike(tx *gorm.DB, raw string) *gorm.DB {
	for _, term := range strings.Fields(strings.TrimSpace(raw)) {
		pat := modelCatalogLikePattern(term)
		if pat == "" {
			continue
		}
		if common.UsingPostgreSQL {
			tx = tx.Where("provider_key ILIKE ? OR provider_display ILIKE ?", pat, pat)
		} else {
			tx = tx.Where("provider_key LIKE ? OR provider_display LIKE ?", pat, pat)
		}
	}
	return tx
}

func modelCatalogWhereProviderKeyExact(tx *gorm.DB, raw string) *gorm.DB {
	key := strings.TrimSpace(raw)
	if key == "" {
		return tx
	}
	if common.UsingPostgreSQL {
		return tx.Where("provider_key ILIKE ?", key)
	}
	return tx.Where("provider_key = ?", key)
}

func modelCatalogSearchQuery(tx *gorm.DB, rawSearch string) *gorm.DB {
	terms := strings.Fields(strings.TrimSpace(rawSearch))
	for _, term := range terms {
		pat := modelCatalogLikePattern(term)
		if pat == "" {
			continue
		}
		if common.UsingPostgreSQL {
			tx = tx.Where(
				`model_id ILIKE ? OR model_name ILIKE ? OR provider_key ILIKE ? OR provider_display ILIKE ? OR family ILIKE ? OR modalities_in ILIKE ? OR modalities_out ILIKE ? OR owned_by ILIKE ? OR source ILIKE ? OR notes ILIKE ? OR npm_package ILIKE ? OR api_base ILIKE ? OR doc_url ILIKE ?`,
				pat, pat, pat, pat, pat, pat, pat, pat, pat, pat, pat, pat, pat,
			)
			continue
		}
		tx = tx.Where(
			`model_id LIKE ? OR model_name LIKE ? OR provider_key LIKE ? OR provider_display LIKE ? OR family LIKE ? OR modalities_in LIKE ? OR modalities_out LIKE ? OR owned_by LIKE ? OR source LIKE ? OR notes LIKE ? OR npm_package LIKE ? OR api_base LIKE ? OR doc_url LIKE ?`,
			pat, pat, pat, pat, pat, pat, pat, pat, pat, pat, pat, pat, pat,
		)
	}
	return tx
}

func modelCatalogFilterQuery(tx *gorm.DB, p ModelCatalogListParams) *gorm.DB {
	if strings.TrimSpace(p.Search) != "" {
		tx = modelCatalogSearchQuery(tx, p.Search)
	}
	tx = modelCatalogWhereColumnLike(tx, "model_id", p.FilterModelID)
	tx = modelCatalogWhereColumnLike(tx, "model_name", p.FilterModelName)
	if strings.TrimSpace(p.FilterProviderKey) != "" {
		tx = modelCatalogWhereProviderKeyExact(tx, p.FilterProviderKey)
	} else {
		tx = modelCatalogWhereProviderLike(tx, p.FilterProvider)
	}
	tx = modelCatalogWhereColumnLike(tx, "family", p.FilterFamily)
	tx = modelCatalogWhereColumnLike(tx, "modalities_in", p.FilterModalitiesIn)
	tx = modelCatalogWhereColumnLike(tx, "modalities_out", p.FilterModalitiesOut)
	return tx
}

// ListModelCatalogPaged 分页列出目录；total 为筛选命中条数，grandTotal 为表总行数（不含筛选）。
func ListModelCatalogPaged(p ModelCatalogListParams) ([]ModelCatalog, int64, int64, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}

	var grandTotal int64
	grandTotalQ := DB.Model(&ModelCatalog{})
	if !p.IncludeExpired {
		grandTotalQ = grandTotalQ.Where("status = ?", "current")
	}
	if err := grandTotalQ.Count(&grandTotal).Error; err != nil {
		return nil, 0, 0, err
	}

	searchQ := DB.Model(&ModelCatalog{})
	if !p.IncludeExpired {
		searchQ = searchQ.Where("status = ?", "current")
	}
	cntTx := modelCatalogFilterQuery(searchQ, p)
	var total int64
	if err := cntTx.Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}

	col := "id"
	if c, ok := modelCatalogSortColumns[strings.TrimSpace(p.SortBy)]; ok {
		col = c
	}
	dir := "ASC"
	if p.SortDesc {
		dir = "DESC"
	}
	orderExpr := col + " " + dir

	off := (p.Page - 1) * p.PageSize
	if off < 0 {
		off = 0
	}

	var rows []ModelCatalog
	findTx := modelCatalogFilterQuery(searchQ, p)
	err := findTx.Order(orderExpr).Offset(off).Limit(p.PageSize).Find(&rows).Error
	return rows, total, grandTotal, err
}

// ModelCatalogProviderOption 提供方筛选项（去重后供前端补全）。
type ModelCatalogProviderOption struct {
	Key     string `json:"key"`
	Display string `json:"display"`
	Label   string `json:"label"`
}

// ListModelCatalogProviders 列出当前启用目录中的提供方（模糊匹配 key/display）。
func ListModelCatalogProviders(query string, limit int) ([]ModelCatalogProviderOption, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	q := DB.Model(&ModelCatalog{}).
		Where("status = ?", "current").
		Where("enabled = ?", true)
	if strings.TrimSpace(query) != "" {
		q = modelCatalogWhereProviderLike(q, query)
	}
	type providerRow struct {
		ProviderKey     string
		ProviderDisplay string
	}
	var rows []providerRow
	err := q.Select("provider_key", "provider_display").
		Distinct("provider_key", "provider_display").
		Order("provider_display asc, provider_key asc").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]ModelCatalogProviderOption, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		key := strings.TrimSpace(row.ProviderKey)
		display := strings.TrimSpace(row.ProviderDisplay)
		if key == "" && display == "" {
			continue
		}
		dedupeKey := key + "\x00" + display
		if _, ok := seen[dedupeKey]; ok {
			continue
		}
		seen[dedupeKey] = struct{}{}
		label := display
		if label == "" {
			label = key
		} else if key != "" && !strings.EqualFold(label, key) {
			label = display + " (" + key + ")"
		}
		out = append(out, ModelCatalogProviderOption{
			Key:     key,
			Display: display,
			Label:   label,
		})
	}
	return out, nil
}

// ListModelCatalogEnabledModelIDs 返回当前启用目录中的 model_id（可按提供方等筛选）。
func ListModelCatalogEnabledModelIDs(p ModelCatalogListParams) ([]string, error) {
	q := DB.Model(&ModelCatalog{}).
		Where("status = ?", "current").
		Where("enabled = ?", true)
	q = modelCatalogFilterQuery(q, p)
	var ids []string
	err := q.Distinct("model_id").Order("model_id asc").Pluck("model_id", &ids).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		s := strings.TrimSpace(id)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out, nil
}

func GetModelCatalogByPk(id int) (*ModelCatalog, error) {
	var row ModelCatalog
	err := DB.First(&row, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (c *ModelCatalog) Insert() error {
	now := helper.GetTimestamp()
	c.CreatedTime = now
	c.UpdatedTime = now
	c.ModelId = strings.TrimSpace(c.ModelId)
	return DB.Create(c).Error
}

func (c *ModelCatalog) SaveUpdates() error {
	c.UpdatedTime = helper.GetTimestamp()
	c.ModelId = strings.TrimSpace(c.ModelId)
	return DB.Save(c).Error
}

func DeleteModelCatalogByPk(id int) error {
	return DB.Delete(&ModelCatalog{}, "id = ?", id).Error
}

func GetCatalogVersions(source, modelID string) ([]ModelCatalog, error) {
	var rows []ModelCatalog
	err := DB.Where("source = ? AND model_id = ?", source, modelID).Order("version DESC").Find(&rows).Error
	return rows, err
}

func GetCurrentByModelID(modelID string) ([]*ModelCatalog, error) {
	var rows []*ModelCatalog
	err := DB.Where("model_id = ? AND status = ?", modelID, "current").Find(&rows).Error
	return rows, err
}

func UpsertCatalogVersioned(tx *gorm.DB, payload *ModelCatalog) error {
	now := helper.GetTimestamp()
	payload.ModelId = strings.TrimSpace(payload.ModelId)
	if payload.ModelId == "" {
		return errors.New("model_id is empty")
	}

	// 1. 将现有的 current 记录标记为 expired
	if err := tx.Model(&ModelCatalog{}).
		Where("source = ? AND model_id = ? AND status = ?", payload.Source, payload.ModelId, "current").
		Update("status", "expired").Error; err != nil {
		return err
	}

	// 2. 找到最大的 version
	var maxVersion int
	tx.Model(&ModelCatalog{}).
		Where("source = ? AND model_id = ?", payload.Source, payload.ModelId).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)

	// 3. 插入新记录
	payload.Version = maxVersion + 1
	payload.Status = "current"
	payload.CreatedTime = now
	payload.UpdatedTime = now
	return tx.Create(payload).Error
}

// BatchUpsertModelCatalogForSync 按 model_id 同步。使用版本化逻辑。
func BatchUpsertModelCatalogForSync(entries []ModelCatalog) (added int, updated int, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return 0, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	for _, e := range entries {
		e.ModelId = strings.TrimSpace(e.ModelId)
		if e.ModelId == "" {
			continue
		}
		
		// 判断是否有变更
		var existing ModelCatalog
		er := tx.Where("source = ? AND model_id = ? AND status = ?", e.Source, e.ModelId, "current").First(&existing).Error
		if errors.Is(er, gorm.ErrRecordNotFound) {
			if err := UpsertCatalogVersioned(tx, &e); err != nil {
				tx.Rollback()
				return 0, 0, err
			}
			added++
			continue
		} else if er != nil {
			tx.Rollback()
			return 0, 0, er
		}

		// 有变更时才 upsert，判断关键字段
		changed := existing.OwnedBy != e.OwnedBy || existing.Enabled != e.Enabled || existing.Notes != e.Notes
		if strings.HasPrefix(e.Source, "sync_models_dev") || e.Source == "basellm" || e.Source == "aliapi" || e.Source == "anyfast" || e.Source == "new_api_models" {
			changed = changed || existing.ModelName != e.ModelName || existing.ProviderKey != e.ProviderKey || existing.ProviderDisplay != e.ProviderDisplay || existing.Family != e.Family || existing.NpmPackage != e.NpmPackage || existing.APIBase != e.APIBase || existing.DocURL != e.DocURL || existing.ModalitiesIn != e.ModalitiesIn || existing.ModalitiesOut != e.ModalitiesOut || existing.ContextLimit != e.ContextLimit || existing.OutputLimit != e.OutputLimit || existing.CostInput != e.CostInput || existing.CostOutput != e.CostOutput || existing.CostCacheRead != e.CostCacheRead || existing.CostCacheWrite != e.CostCacheWrite || existing.Reasoning != e.Reasoning || existing.ToolCall != e.ToolCall || existing.TemperatureOK != e.TemperatureOK || existing.AttachmentOK != e.AttachmentOK || existing.OpenWeights != e.OpenWeights || existing.KnowledgeCutoff != e.KnowledgeCutoff || existing.ReleaseDate != e.ReleaseDate || existing.LastUpdatedDev != e.LastUpdatedDev || existing.Tags != e.Tags
		}

		if changed {
			if !strings.HasPrefix(e.Source, "sync_models_dev") && e.Source != "basellm" && e.Source != "aliapi" && e.Source != "anyfast" && e.Source != "new_api_models" {
				// 保留旧扩展列
				e.ModelName = existing.ModelName
				e.ProviderKey = existing.ProviderKey
				e.ProviderDisplay = existing.ProviderDisplay
				e.Family = existing.Family
				e.NpmPackage = existing.NpmPackage
				e.APIBase = existing.APIBase
				e.DocURL = existing.DocURL
				e.ModalitiesIn = existing.ModalitiesIn
				e.ModalitiesOut = existing.ModalitiesOut
				e.ContextLimit = existing.ContextLimit
				e.OutputLimit = existing.OutputLimit
				e.CostInput = existing.CostInput
				e.CostOutput = existing.CostOutput
				e.CostCacheRead = existing.CostCacheRead
				e.CostCacheWrite = existing.CostCacheWrite
				e.Reasoning = existing.Reasoning
				e.ToolCall = existing.ToolCall
				e.TemperatureOK = existing.TemperatureOK
				e.AttachmentOK = existing.AttachmentOK
				e.OpenWeights = existing.OpenWeights
				e.KnowledgeCutoff = existing.KnowledgeCutoff
				e.ReleaseDate = existing.ReleaseDate
				e.LastUpdatedDev = existing.LastUpdatedDev
				e.Tags = existing.Tags
			}
			if err := UpsertCatalogVersioned(tx, &e); err != nil {
				tx.Rollback()
				return 0, 0, err
			}
			updated++
		}
	}
	if err := tx.Commit().Error; err != nil {
		return 0, 0, err
	}
	return added, updated, nil
}
