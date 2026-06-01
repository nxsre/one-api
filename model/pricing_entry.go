package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/setting/billing_setting"
	"gorm.io/gorm"
)

const (
	pricingEntryKeySep     = "\x00"
	pricingEntryMarker     = `{"__entries__":true}`
	pricingEntryBatchSize  = 500
)

// PricingEntry 单条价目配置（按 block + key 独立入库）。
type PricingEntry struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	BlockId     string `json:"block_id" gorm:"uniqueIndex:idx_pe_block_key,priority:1;size:64;index"`
	EntryKey    string `json:"entry_key" gorm:"uniqueIndex:idx_pe_block_key,priority:2;size:512"`
	SubKey      string `json:"sub_key" gorm:"size:512;default:''"`
	ValueText   string `json:"value_text" gorm:"type:text"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

// PricingVersionEntry 价目版本快照（按条存储，避免大 JSON）。
type PricingVersionEntry struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	BlockId     string `json:"block_id" gorm:"uniqueIndex:idx_pve_ver_key,priority:1;size:64;index"`
	VersionId   int    `json:"version_id" gorm:"uniqueIndex:idx_pve_ver_key,priority:2;index"`
	EntryKey    string `json:"entry_key" gorm:"uniqueIndex:idx_pve_ver_key,priority:3;size:512"`
	SubKey      string `json:"sub_key" gorm:"size:512;default:''"`
	ValueText   string `json:"value_text" gorm:"type:text"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
}

// PricingEntryListParams 分页查询参数。
type PricingEntryListParams struct {
	BlockId  string
	Page     int
	PageSize int
	Search   string
}

type PricingEntryListResult struct {
	Items      []PricingEntry `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

func pricingBlockDef(blockID string) (billing_setting.PricingBlockDef, bool) {
	return billing_setting.LookupBlockByID(blockID)
}

func isGroupGroupBlock(blockID string) bool {
	return blockID == "group_group_ratio"
}

func isStringValueBlock(blockID string) bool {
	return blockID == "billing_mode" || blockID == "billing_expr"
}

func composeEntryKey(blockID, key, subKey string) (string, string, error) {
	key = strings.TrimSpace(key)
	subKey = strings.TrimSpace(subKey)
	if isGroupGroupBlock(blockID) {
		if key == "" || subKey == "" {
			return "", "", fmt.Errorf("分组间倍率需填写用户分组与使用分组")
		}
		return key, subKey, nil
	}
	if key == "" {
		return "", "", fmt.Errorf("键名不能为空")
	}
	return key, "", nil
}

func CountPricingEntries(blockID string) (int64, error) {
	var n int64
	err := DB.Model(&PricingEntry{}).Where("block_id = ?", blockID).Count(&n).Error
	return n, err
}

func ListPricingEntriesPaged(p PricingEntryListParams) (*PricingEntryListResult, error) {
	blockID := strings.TrimSpace(p.BlockId)
	if blockID == "" {
		return nil, fmt.Errorf("block_id 必填")
	}
	if _, ok := pricingBlockDef(blockID); !ok {
		return nil, fmt.Errorf("unknown block %s", blockID)
	}
	page := p.Page
	if page < 1 {
		page = 1
	}
	pageSize := p.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	q := DB.Model(&PricingEntry{}).Where("block_id = ?", blockID)
	search := strings.TrimSpace(p.Search)
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("entry_key LIKE ? OR sub_key LIKE ? OR value_text LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []PricingEntry
	offset := (page - 1) * pageSize
	if err := q.Order("entry_key asc, sub_key asc").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, err
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}
	return &PricingEntryListResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func UpsertPricingEntry(blockID, key, subKey, valueText string) (*PricingEntry, error) {
	blockID = strings.TrimSpace(blockID)
	entryKey, sub, err := composeEntryKey(blockID, key, subKey)
	if err != nil {
		return nil, err
	}
	valueText = strings.TrimSpace(valueText)
	if valueText == "" {
		return nil, fmt.Errorf("值不能为空")
	}
	if err := validateEntryValue(blockID, valueText); err != nil {
		return nil, err
	}
	now := helper.GetTimestamp()
	var row PricingEntry
	err = DB.Where("block_id = ? AND entry_key = ? AND sub_key = ?", blockID, entryKey, sub).
		First(&row).Error
	if err != nil {
		row = PricingEntry{
			BlockId:     blockID,
			EntryKey:    entryKey,
			SubKey:      sub,
			ValueText:   valueText,
			UpdatedTime: now,
		}
		if err := DB.Create(&row).Error; err != nil {
			return nil, err
		}
	} else {
		row.ValueText = valueText
		row.UpdatedTime = now
		if err := DB.Save(&row).Error; err != nil {
			return nil, err
		}
	}
	if err := patchRuntimeEntry(blockID, entryKey, sub, valueText); err != nil {
		return nil, err
	}
	syncPricingOptionMarker(blockID)
	return &row, nil
}

func UpdatePricingEntryByID(id int, valueText string) (*PricingEntry, error) {
	var row PricingEntry
	if err := DB.First(&row, id).Error; err != nil {
		return nil, err
	}
	valueText = strings.TrimSpace(valueText)
	if valueText == "" {
		return nil, fmt.Errorf("值不能为空")
	}
	if err := validateEntryValue(row.BlockId, valueText); err != nil {
		return nil, err
	}
	row.ValueText = valueText
	row.UpdatedTime = helper.GetTimestamp()
	if err := DB.Save(&row).Error; err != nil {
		return nil, err
	}
	if err := patchRuntimeEntry(row.BlockId, row.EntryKey, row.SubKey, row.ValueText); err != nil {
		return nil, err
	}
	syncPricingOptionMarker(row.BlockId)
	return &row, nil
}

func DeletePricingEntryByID(id int) error {
	var row PricingEntry
	if err := DB.First(&row, id).Error; err != nil {
		return err
	}
	if err := DB.Delete(&row).Error; err != nil {
		return err
	}
	if err := removeRuntimeEntry(row.BlockId, row.EntryKey, row.SubKey); err != nil {
		return err
	}
	syncPricingOptionMarker(row.BlockId)
	return nil
}

func ApplyPricingPatches(blockID string, patches map[string]any) error {
	blockID = strings.TrimSpace(blockID)
	if _, ok := pricingBlockDef(blockID); !ok {
		return fmt.Errorf("unknown block %s", blockID)
	}
	if len(patches) == 0 {
		return nil
	}
	now := helper.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		for k, v := range patches {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			valueText := fmt.Sprint(v)
			if strings.TrimSpace(valueText) == "" {
				continue
			}
			if err := validateEntryValue(blockID, valueText); err != nil {
				return err
			}
			var row PricingEntry
			err := tx.Where("block_id = ? AND entry_key = ? AND sub_key = ?", blockID, key, "").
				First(&row).Error
			if err != nil {
				row = PricingEntry{
					BlockId:     blockID,
					EntryKey:    key,
					ValueText:   valueText,
					UpdatedTime: now,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			} else {
				row.ValueText = valueText
				row.UpdatedTime = now
				if err := tx.Save(&row).Error; err != nil {
					return err
				}
			}
		if err := patchRuntimeEntry(blockID, key, "", valueText); err != nil {
			return err
		}
	}
	syncPricingOptionMarker(blockID)
	return nil
})
}

func validateEntryValue(blockID, valueText string) error {
	if isStringValueBlock(blockID) {
		return nil
	}
	if _, err := strconv.ParseFloat(valueText, 64); err != nil {
		return fmt.Errorf("无效数值: %s", valueText)
	}
	return nil
}

func syncPricingOptionMarker(blockID string) {
	def, ok := pricingBlockDef(blockID)
	if !ok {
		return
	}
	count, err := CountPricingEntries(blockID)
	if err != nil || count == 0 {
		return
	}
	marker := fmt.Sprintf(`{"__entries__":true,"count":%d}`, count)
	configUpdateOptionValueOnly(def.OptionKey, marker)
}

func configUpdateOptionValueOnly(key, value string) {
	option := Option{Key: key}
	DB.FirstOrCreate(&option, Option{Key: key})
	if option.Value == value {
		return
	}
	option.Value = value
	DB.Save(&option)
	config.OptionMapRWMutex.Lock()
	config.OptionMap[key] = value
	config.OptionMapRWMutex.Unlock()
	PublishOptionsChanged()
}

// InitPricingEntryStore 迁移 JSON 价目为分条记录，并灌入运行时。
func InitPricingEntryStore() error {
	if err := DB.AutoMigrate(&PricingEntry{}, &PricingVersionEntry{}); err != nil {
		return err
	}
	if err := InitUpstreamSyncStore(); err != nil {
		return err
	}
	for _, def := range billing_setting.PricingBlockDefs {
		count, err := CountPricingEntries(def.BlockID)
		if err != nil {
			return err
		}
		if count == 0 {
			if err := migrateBlockFromOptionJSON(def.BlockID); err != nil {
				logger.SysError("migrate pricing entries " + def.BlockID + ": " + err.Error())
			}
		}
		if err := HydrateBlockRuntimeFromEntries(def.BlockID); err != nil {
			logger.SysError("hydrate pricing block " + def.BlockID + ": " + err.Error())
		}
	}
	return nil
}

func migrateBlockFromOptionJSON(blockID string) error {
	def, ok := pricingBlockDef(blockID)
	if !ok {
		return fmt.Errorf("unknown block")
	}
	config.OptionMapRWMutex.RLock()
	raw := strings.TrimSpace(config.OptionMap[def.OptionKey])
	config.OptionMapRWMutex.RUnlock()
	if raw == "" || raw == "{}" || strings.Contains(raw, `"__entries__"`) {
		return nil
	}
	now := helper.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		rows := make([]PricingEntry, 0)
		switch {
		case isGroupGroupBlock(blockID):
			var m map[string]map[string]float64
			if err := json.Unmarshal([]byte(raw), &m); err != nil {
				return err
			}
			for outer, innerMap := range m {
				for inner, val := range innerMap {
					rows = append(rows, PricingEntry{
						BlockId:     blockID,
						EntryKey:    strings.TrimSpace(outer),
						SubKey:      strings.TrimSpace(inner),
						ValueText:   strconv.FormatFloat(val, 'f', -1, 64),
						UpdatedTime: now,
					})
				}
			}
		case isStringValueBlock(blockID):
			var m map[string]string
			if err := json.Unmarshal([]byte(raw), &m); err != nil {
				return err
			}
			for k, v := range m {
				rows = append(rows, PricingEntry{
					BlockId:     blockID,
					EntryKey:    strings.TrimSpace(k),
					ValueText:   v,
					UpdatedTime: now,
				})
			}
		default:
			var m map[string]float64
			if err := json.Unmarshal([]byte(raw), &m); err != nil {
				return err
			}
			for k, v := range m {
				rows = append(rows, PricingEntry{
					BlockId:     blockID,
					EntryKey:    strings.TrimSpace(k),
					ValueText:   strconv.FormatFloat(v, 'f', -1, 64),
					UpdatedTime: now,
				})
			}
		}
		for i := 0; i < len(rows); i += pricingEntryBatchSize {
			end := i + pricingEntryBatchSize
			if end > len(rows) {
				end = len(rows)
			}
			if err := tx.CreateInBatches(rows[i:end], pricingEntryBatchSize).Error; err != nil {
				return err
			}
		}
		if len(rows) > 0 {
			marker := fmt.Sprintf(`{"__entries__":true,"count":%d}`, len(rows))
			configUpdateOptionValueOnly(def.OptionKey, marker)
		}
		return nil
	})
}

func HydrateBlockRuntimeFromEntries(blockID string) error {
	var rows []PricingEntry
	if err := DB.Where("block_id = ?", blockID).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	payload, err := entriesToPayloadJSON(blockID, rows)
	if err != nil {
		return err
	}
	def, ok := pricingBlockDef(blockID)
	if !ok {
		return fmt.Errorf("unknown block")
	}
	return updateOptionMapMemoryOnly(def.OptionKey, payload)
}

func entriesToPayloadJSON(blockID string, rows []PricingEntry) (string, error) {
	switch {
	case isGroupGroupBlock(blockID):
		m := make(map[string]map[string]float64)
		for _, r := range rows {
			if m[r.EntryKey] == nil {
				m[r.EntryKey] = make(map[string]float64)
			}
			f, err := strconv.ParseFloat(r.ValueText, 64)
			if err != nil {
				return "", err
			}
			m[r.EntryKey][r.SubKey] = f
		}
		b, err := json.Marshal(m)
		return string(b), err
	case isStringValueBlock(blockID):
		m := make(map[string]string, len(rows))
		for _, r := range rows {
			m[r.EntryKey] = r.ValueText
		}
		b, err := json.Marshal(m)
		return string(b), err
	default:
		m := make(map[string]float64, len(rows))
		for _, r := range rows {
			f, err := strconv.ParseFloat(r.ValueText, 64)
			if err != nil {
				return "", err
			}
			m[r.EntryKey] = f
		}
		b, err := json.Marshal(m)
		return string(b), err
	}
}

func updateOptionMapMemoryOnly(key, payload string) error {
	config.OptionMapRWMutex.Lock()
	config.OptionMap[key] = payload
	config.OptionMapRWMutex.Unlock()
	return applyRuntimePayload(key, payload)
}

func applyRuntimePayload(key, payload string) error {
	switch key {
	case "ModelRatio":
		return billingratio.UpdateModelRatioByJSONString(payload)
	case "CompletionRatio":
		return billingratio.UpdateCompletionRatioByJSONString(payload)
	case "ModelPrice":
		return billingratio.UpdateModelPriceByJSONString(payload)
	case "CacheRatio":
		return billingratio.UpdateCacheRatioByJSONString(payload)
	case "CreateCacheRatio":
		return billingratio.UpdateCreateCacheRatioByJSONString(payload)
	case "ImageRatio":
		return billingratio.UpdateImageRatioByJSONString(payload)
	case "AudioRatio":
		return billingratio.UpdateAudioRatioByJSONString(payload)
	case "AudioCompletionRatio":
		return billingratio.UpdateAudioCompletionRatioByJSONString(payload)
	case "BillingMode":
		return billing_setting.UpdateBillingModeByJSONString(payload)
	case "BillingExpr":
		return billing_setting.UpdateBillingExprByJSONString(payload)
	case "GroupRatio":
		return billingratio.UpdateGroupRatioByJSONString(payload)
	case "GroupGroupRatio":
		return billingratio.UpdateGroupGroupRatioByJSONString(payload)
	case "TopupGroupRatio":
		return billingratio.UpdateTopupGroupRatioByJSONString(payload)
	default:
		return nil
	}
}

func patchRuntimeEntry(blockID, entryKey, subKey, valueText string) error {
	switch {
	case isGroupGroupBlock(blockID):
		m := billingratio.GetGroupGroupRatioCopy()
		if m == nil {
			m = make(map[string]map[string]float64)
		}
		if m[entryKey] == nil {
			m[entryKey] = make(map[string]float64)
		}
		f, err := strconv.ParseFloat(valueText, 64)
		if err != nil {
			return err
		}
		m[entryKey][subKey] = f
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}
		return billingratio.UpdateGroupGroupRatioByJSONString(string(b))
	case isStringValueBlock(blockID):
		var m map[string]string
		switch blockID {
		case "billing_mode":
			raw := billing_setting.BillingMode2JSONString()
			_ = json.Unmarshal([]byte(raw), &m)
		case "billing_expr":
			raw := billing_setting.BillingExpr2JSONString()
			_ = json.Unmarshal([]byte(raw), &m)
		}
		if m == nil {
			m = make(map[string]string)
		}
		m[entryKey] = valueText
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}
		if blockID == "billing_mode" {
			return billing_setting.UpdateBillingModeByJSONString(string(b))
		}
		return billing_setting.UpdateBillingExprByJSONString(string(b))
	default:
		updater, getter := flatRatioRuntimeAccessors(blockID)
		if updater == nil {
			return fmt.Errorf("unknown block %s", blockID)
		}
		m := getter()
		if m == nil {
			m = make(map[string]float64)
		}
		f, err := strconv.ParseFloat(valueText, 64)
		if err != nil {
			return err
		}
		m[entryKey] = f
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}
		return updater(string(b))
	}
}

func removeRuntimeEntry(blockID, entryKey, subKey string) error {
	switch {
	case isGroupGroupBlock(blockID):
		m := billingratio.GetGroupGroupRatioCopy()
		if m == nil {
			return nil
		}
		if inner, ok := m[entryKey]; ok {
			delete(inner, subKey)
			if len(inner) == 0 {
				delete(m, entryKey)
			}
		}
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}
		return billingratio.UpdateGroupGroupRatioByJSONString(string(b))
	case isStringValueBlock(blockID):
		var m map[string]string
		switch blockID {
		case "billing_mode":
			raw := billing_setting.BillingMode2JSONString()
			_ = json.Unmarshal([]byte(raw), &m)
		case "billing_expr":
			raw := billing_setting.BillingExpr2JSONString()
			_ = json.Unmarshal([]byte(raw), &m)
		}
		if m == nil {
			return nil
		}
		delete(m, entryKey)
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}
		if blockID == "billing_mode" {
			return billing_setting.UpdateBillingModeByJSONString(string(b))
		}
		return billing_setting.UpdateBillingExprByJSONString(string(b))
	default:
		updater, getter := flatRatioRuntimeAccessors(blockID)
		if updater == nil {
			return fmt.Errorf("unknown block %s", blockID)
		}
		m := getter()
		if m == nil {
			return nil
		}
		delete(m, entryKey)
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}
		return updater(string(b))
	}
}

func flatRatioRuntimeAccessors(blockID string) (func(string) error, func() map[string]float64) {
	switch blockID {
	case "model_ratio":
		return billingratio.UpdateModelRatioByJSONString, billingratio.GetModelRatioCopy
	case "completion_ratio":
		return billingratio.UpdateCompletionRatioByJSONString, billingratio.GetCompletionRatioCopy
	case "model_price":
		return billingratio.UpdateModelPriceByJSONString, billingratio.GetModelPriceCopy
	case "cache_ratio":
		return billingratio.UpdateCacheRatioByJSONString, billingratio.GetCacheRatioCopy
	case "create_cache_ratio":
		return billingratio.UpdateCreateCacheRatioByJSONString, billingratio.GetCreateCacheRatioCopy
	case "image_ratio":
		return billingratio.UpdateImageRatioByJSONString, billingratio.GetImageRatioCopy
	case "audio_ratio":
		return billingratio.UpdateAudioRatioByJSONString, billingratio.GetAudioRatioCopy
	case "audio_completion_ratio":
		return billingratio.UpdateAudioCompletionRatioByJSONString, billingratio.GetAudioCompletionRatioCopy
	case "group_ratio":
		return billingratio.UpdateGroupRatioByJSONString, billingratio.GetGroupRatioCopy
	case "topup_group_ratio":
		return billingratio.UpdateTopupGroupRatioByJSONString, billingratio.GetTopupGroupRatioCopy
	default:
		return nil, nil
	}
}

func SnapshotPricingEntriesToVersion(blockID string, versionID int) error {
	blockID = strings.TrimSpace(blockID)
	if versionID <= 0 {
		return fmt.Errorf("invalid version id")
	}
	var rows []PricingEntry
	if err := DB.Where("block_id = ?", blockID).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	now := helper.GetTimestamp()
	snapshots := make([]PricingVersionEntry, 0, len(rows))
	for _, r := range rows {
		snapshots = append(snapshots, PricingVersionEntry{
			BlockId:     blockID,
			VersionId:   versionID,
			EntryKey:    r.EntryKey,
			SubKey:      r.SubKey,
			ValueText:   r.ValueText,
			CreatedTime: now,
		})
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("block_id = ? AND version_id = ?", blockID, versionID).
			Delete(&PricingVersionEntry{}).Error; err != nil {
			return err
		}
		for i := 0; i < len(snapshots); i += pricingEntryBatchSize {
			end := i + pricingEntryBatchSize
			if end > len(snapshots) {
				end = len(snapshots)
			}
			if err := tx.CreateInBatches(snapshots[i:end], pricingEntryBatchSize).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func RestorePricingEntriesFromVersion(blockID string, versionID int) error {
	blockID = strings.TrimSpace(blockID)
	var rows []PricingVersionEntry
	if err := DB.Where("block_id = ? AND version_id = ?", blockID, versionID).
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("版本快照为空")
	}
	now := helper.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("block_id = ?", blockID).Delete(&PricingEntry{}).Error; err != nil {
			return err
		}
		entries := make([]PricingEntry, 0, len(rows))
		for _, r := range rows {
			entries = append(entries, PricingEntry{
				BlockId:     blockID,
				EntryKey:    r.EntryKey,
				SubKey:      r.SubKey,
				ValueText:   r.ValueText,
				UpdatedTime: now,
			})
		}
		for i := 0; i < len(entries); i += pricingEntryBatchSize {
			end := i + pricingEntryBatchSize
			if end > len(entries) {
				end = len(entries)
			}
			if err := tx.CreateInBatches(entries[i:end], pricingEntryBatchSize).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func IsPricingOptionKey(key string) bool {
	_, ok := billing_setting.LookupBlockByOptionKey(key)
	return ok
}

func UsePricingEntryStore(optionKey string) bool {
	def, ok := billing_setting.LookupBlockByOptionKey(optionKey)
	if !ok {
		return false
	}
	n, err := CountPricingEntries(def.BlockID)
	return err == nil && n > 0
}
