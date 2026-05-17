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
	ModelId     string `json:"model_id" gorm:"uniqueIndex:uk_model_catalog_mid;size:191;column:model_id"`
	OwnedBy     string `json:"owned_by" gorm:"size:256"`
	Enabled     bool   `json:"enabled" gorm:"default:true;index"`
	Source      string `json:"source" gorm:"size:64"`
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
}

func GetAllModelCatalog() ([]ModelCatalog, error) {
	var rows []ModelCatalog
	err := DB.Order("model_id asc").Find(&rows).Error
	return rows, err
}

// ModelCatalogListParams GET /api/model_catalog 服务端分页参数。
type ModelCatalogListParams struct {
	Page     int
	PageSize int
	Search   string
	SortBy   string
	SortDesc bool
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

func modelCatalogSearchQuery(tx *gorm.DB, rawSearch string) *gorm.DB {
	terms := strings.Fields(strings.TrimSpace(rawSearch))
	for _, term := range terms {
		t := strings.TrimSpace(term)
		t = strings.ReplaceAll(t, "%", "")
		t = strings.ReplaceAll(t, "_", "")
		t = strings.ReplaceAll(t, "\\", "")
		if t == "" {
			continue
		}
		pat := "%" + t + "%"
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
	if err := DB.Model(&ModelCatalog{}).Count(&grandTotal).Error; err != nil {
		return nil, 0, 0, err
	}

	cntTx := modelCatalogSearchQuery(DB.Model(&ModelCatalog{}), p.Search)
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
	findTx := modelCatalogSearchQuery(DB.Model(&ModelCatalog{}), p.Search)
	err := findTx.Order(orderExpr).Offset(off).Limit(p.PageSize).Find(&rows).Error
	return rows, total, grandTotal, err
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

// BatchUpsertModelCatalogForSync 按 model_id 插入或更新（同步专用，不删除已有行）。
func BatchUpsertModelCatalogForSync(entries []ModelCatalog) (added int, updated int, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return 0, 0, tx.Error
	}

	for _, e := range entries {
		e.ModelId = strings.TrimSpace(e.ModelId)
		if e.ModelId == "" {
			continue
		}
		var existing ModelCatalog
		er := tx.Where("model_id = ?", e.ModelId).First(&existing).Error
		now := helper.GetTimestamp()
		if errors.Is(er, gorm.ErrRecordNotFound) {
			row := e
			row.CreatedTime = now
			row.UpdatedTime = now
			if err := tx.Create(&row).Error; err != nil {
				_ = tx.Rollback()
				return 0, 0, err
			}
			added++
			continue
		}
		if er != nil {
			_ = tx.Rollback()
			return 0, 0, er
		}
		existing.OwnedBy = e.OwnedBy
		existing.Enabled = e.Enabled
		existing.Source = e.Source
		existing.UpdatedTime = now
		// 仅 Models.dev 同步写入扩展列，避免 OpenAI 等同步把已有元数据清空
		if strings.HasPrefix(e.Source, "sync_models_dev") {
			existing.ModelName = e.ModelName
			existing.ProviderKey = e.ProviderKey
			existing.ProviderDisplay = e.ProviderDisplay
			existing.Family = e.Family
			existing.NpmPackage = e.NpmPackage
			existing.APIBase = e.APIBase
			existing.DocURL = e.DocURL
			existing.ModalitiesIn = e.ModalitiesIn
			existing.ModalitiesOut = e.ModalitiesOut
			existing.ContextLimit = e.ContextLimit
			existing.OutputLimit = e.OutputLimit
			existing.CostInput = e.CostInput
			existing.CostOutput = e.CostOutput
			existing.CostCacheRead = e.CostCacheRead
			existing.CostCacheWrite = e.CostCacheWrite
			existing.Reasoning = e.Reasoning
			existing.ToolCall = e.ToolCall
			existing.TemperatureOK = e.TemperatureOK
			existing.AttachmentOK = e.AttachmentOK
			existing.OpenWeights = e.OpenWeights
			existing.KnowledgeCutoff = e.KnowledgeCutoff
			existing.ReleaseDate = e.ReleaseDate
			existing.LastUpdatedDev = e.LastUpdatedDev
		}
		if strings.TrimSpace(e.Notes) != "" {
			existing.Notes = e.Notes
		}
		if err := tx.Save(&existing).Error; err != nil {
			_ = tx.Rollback()
			return 0, 0, err
		}
		updated++
	}
	if err := tx.Commit().Error; err != nil {
		return 0, 0, err
	}
	return added, updated, nil
}
