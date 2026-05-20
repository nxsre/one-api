package billing_setting

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/songquanpeng/one-api/common/config"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
)

const PricingVersionStoreOptionKey = "PricingConfigVersionStore"

const (
	VersionSourceOperationSave = "operation_save"
	VersionLabelOperationSave  = "运营设置保存"
)

var versionRecordSkipOnSave int32

// PricingBlockDef 一块可独立版本化的运营价目配置（对应 options 表中的一个 key）。
type PricingBlockDef struct {
	BlockID      string
	OptionKey    string
	Title        string
	SyncFieldKey string // ratio_sync 字段名；空表示不参与上游对比
}

var PricingBlockDefs = []PricingBlockDef{
	{BlockID: "model_ratio", OptionKey: "ModelRatio", Title: "模型倍率", SyncFieldKey: "model_ratio"},
	{BlockID: "completion_ratio", OptionKey: "CompletionRatio", Title: "补全倍率", SyncFieldKey: "completion_ratio"},
	{BlockID: "model_price", OptionKey: "ModelPrice", Title: "模型价格", SyncFieldKey: "model_price"},
	{BlockID: "cache_ratio", OptionKey: "CacheRatio", Title: "缓存倍率", SyncFieldKey: "cache_ratio"},
	{BlockID: "create_cache_ratio", OptionKey: "CreateCacheRatio", Title: "缓存创建倍率", SyncFieldKey: "create_cache_ratio"},
	{BlockID: "image_ratio", OptionKey: "ImageRatio", Title: "图片倍率", SyncFieldKey: "image_ratio"},
	{BlockID: "audio_ratio", OptionKey: "AudioRatio", Title: "音频倍率", SyncFieldKey: "audio_ratio"},
	{BlockID: "audio_completion_ratio", OptionKey: "AudioCompletionRatio", Title: "音频补全倍率", SyncFieldKey: "audio_completion_ratio"},
	{BlockID: "billing_mode", OptionKey: "BillingMode", Title: "计费模式", SyncFieldKey: BillingModeField},
	{BlockID: "billing_expr", OptionKey: "BillingExpr", Title: "分层计费表达式", SyncFieldKey: BillingExprField},
	{BlockID: "group_ratio", OptionKey: "GroupRatio", Title: "分组倍率", SyncFieldKey: ""},
	{BlockID: "group_group_ratio", OptionKey: "GroupGroupRatio", Title: "分组间倍率", SyncFieldKey: ""},
	{BlockID: "topup_group_ratio", OptionKey: "TopupGroupRatio", Title: "充值分组倍率", SyncFieldKey: ""},
}

// PricingVersion 单条历史版本（Payload 与对应 Option 的 Value 格式一致；分条存储时可为空）。
type PricingVersion struct {
	ID        int    `json:"id"`
	Label     string `json:"label"`
	Source    string `json:"source"`
	Note      string `json:"note,omitempty"`
	CreatedAt int64  `json:"created_at"`
	Payload   string `json:"payload,omitempty"`
}

type pricingBlockState struct {
	ActiveVersion int                        `json:"active_version"`
	NextVersion   int                        `json:"next_version"`
	Versions      map[string]*PricingVersion `json:"versions"`
}

// PricingVersionStore 全部价目块的版本仓库。
type PricingVersionStore struct {
	Blocks map[string]*pricingBlockState `json:"blocks"`
}

var (
	pricingVersionMu     sync.Mutex
	pricingBlockByID     map[string]PricingBlockDef
	pricingBlockByOption map[string]PricingBlockDef
	pricingBlockBySync   map[string]PricingBlockDef
)

func init() {
	pricingBlockByID = make(map[string]PricingBlockDef, len(PricingBlockDefs))
	pricingBlockByOption = make(map[string]PricingBlockDef, len(PricingBlockDefs))
	pricingBlockBySync = make(map[string]PricingBlockDef, len(PricingBlockDefs))
	for _, d := range PricingBlockDefs {
		pricingBlockByID[d.BlockID] = d
		pricingBlockByOption[d.OptionKey] = d
		if d.SyncFieldKey != "" {
			pricingBlockBySync[d.SyncFieldKey] = d
		}
	}
}

func LookupBlockByID(blockID string) (PricingBlockDef, bool) {
	d, ok := pricingBlockByID[blockID]
	return d, ok
}

func LookupBlockBySyncField(field string) (PricingBlockDef, bool) {
	d, ok := pricingBlockBySync[field]
	return d, ok
}

func LookupBlockByOptionKey(optionKey string) (PricingBlockDef, bool) {
	d, ok := pricingBlockByOption[optionKey]
	return d, ok
}

func skipVersionRecordOnSave() func() {
	atomic.AddInt32(&versionRecordSkipOnSave, 1)
	return func() { atomic.AddInt32(&versionRecordSkipOnSave, -1) }
}

func shouldSkipVersionRecordOnSave() bool {
	return atomic.LoadInt32(&versionRecordSkipOnSave) > 0
}

func currentOptionPayload(optionKey string) string {
	config.OptionMapRWMutex.RLock()
	defer config.OptionMapRWMutex.RUnlock()
	return strings.TrimSpace(config.OptionMap[optionKey])
}

func payloadFromRuntime(def PricingBlockDef) (string, error) {
	switch def.OptionKey {
	case "ModelRatio":
		return billingratio.ModelRatio2JSONString(), nil
	case "CompletionRatio":
		return billingratio.CompletionRatio2JSONString(), nil
	case "ModelPrice":
		return billingratio.ModelPrice2JSONString(), nil
	case "CacheRatio":
		return billingratio.CacheRatio2JSONString(), nil
	case "CreateCacheRatio":
		return billingratio.CreateCacheRatio2JSONString(), nil
	case "ImageRatio":
		return billingratio.ImageRatio2JSONString(), nil
	case "AudioRatio":
		return billingratio.AudioRatio2JSONString(), nil
	case "AudioCompletionRatio":
		return billingratio.AudioCompletionRatio2JSONString(), nil
	case "BillingMode":
		return BillingMode2JSONString(), nil
	case "BillingExpr":
		return BillingExpr2JSONString(), nil
	case "GroupRatio":
		return billingratio.GroupRatio2JSONString(), nil
	case "GroupGroupRatio":
		return billingratio.GroupGroupRatio2JSONString(), nil
	case "TopupGroupRatio":
		return billingratio.TopupGroupRatio2JSONString(), nil
	default:
		return "", fmt.Errorf("unknown option key %s", def.OptionKey)
	}
}

func loadStoreRaw() (string, error) {
	config.OptionMapRWMutex.RLock()
	raw := config.OptionMap[PricingVersionStoreOptionKey]
	config.OptionMapRWMutex.RUnlock()
	return raw, nil
}

func LoadPricingVersionStore() (*PricingVersionStore, error) {
	pricingVersionMu.Lock()
	defer pricingVersionMu.Unlock()

	raw, err := loadStoreRaw()
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return migrateStoreLocked()
	}
	var store PricingVersionStore
	if err := json.Unmarshal([]byte(raw), &store); err != nil {
		return nil, fmt.Errorf("parse version store: %w", err)
	}
	if store.Blocks == nil {
		store.Blocks = make(map[string]*pricingBlockState)
	}
	return &store, nil
}

// RegisterPricingVersionStoreSaver 由 model 包 init 时注册，将版本库写入 DB 并刷新内存。
var RegisterPricingVersionStoreSaver func(key, value string) error

// RegisterPricingEntrySnapshot 由 model 包注册，将分条价目写入版本快照表。
var RegisterPricingEntrySnapshot func(blockID string, versionID int) error

func persistStore(store *PricingVersionStore) error {
	if RegisterPricingVersionStoreSaver == nil {
		return fmt.Errorf("pricing version saver not registered")
	}
	b, err := json.Marshal(store)
	if err != nil {
		return err
	}
	return RegisterPricingVersionStoreSaver(PricingVersionStoreOptionKey, string(b))
}

func migrateStoreLocked() (*PricingVersionStore, error) {
	store := &PricingVersionStore{Blocks: make(map[string]*pricingBlockState)}
	now := time.Now().Unix()
	for _, def := range PricingBlockDefs {
		payload, err := payloadFromRuntime(def)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(payload) == "" {
			payload = "{}"
		}
		store.Blocks[def.BlockID] = &pricingBlockState{
			ActiveVersion: 1,
			NextVersion:   2,
			Versions: map[string]*PricingVersion{
				"1": {
					ID:        1,
					Label:     "当前生效",
					Source:    "migrate",
					CreatedAt: now,
					Payload:   payload,
				},
			},
		}
	}
	if err := persistStore(store); err != nil {
		return nil, err
	}
	return store, nil
}

func ensureBlock(store *PricingVersionStore, blockID string) *pricingBlockState {
	if store.Blocks[blockID] == nil {
		store.Blocks[blockID] = &pricingBlockState{
			ActiveVersion: 0,
			NextVersion:   1,
			Versions:      make(map[string]*PricingVersion),
		}
	}
	if store.Blocks[blockID].Versions == nil {
		store.Blocks[blockID].Versions = make(map[string]*PricingVersion)
	}
	return store.Blocks[blockID]
}

// ListBlocksSummary 返回各块版本摘要（供 API）。
func ListBlocksSummary() ([]map[string]any, error) {
	store, err := LoadPricingVersionStore()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, def := range PricingBlockDefs {
		st := store.Blocks[def.BlockID]
		active := 0
		versionCount := 0
		if st != nil {
			active = st.ActiveVersion
			versionCount = len(st.Versions)
		}
		out = append(out, map[string]any{
			"block_id":       def.BlockID,
			"option_key":     def.OptionKey,
			"title":          def.Title,
			"sync_field":     def.SyncFieldKey,
			"active_version": active,
			"version_count":  versionCount,
		})
	}
	return out, nil
}

// ListBlockVersions 返回某块全部版本（按 id 升序）。
func ListBlockVersions(blockID string) ([]*PricingVersion, int, error) {
	if _, ok := pricingBlockByID[blockID]; !ok {
		return nil, 0, fmt.Errorf("unknown block %s", blockID)
	}
	store, err := LoadPricingVersionStore()
	if err != nil {
		return nil, 0, err
	}
	st := store.Blocks[blockID]
	if st == nil {
		return nil, 0, nil
	}
	ids := make([]int, 0, len(st.Versions))
	for k := range st.Versions {
		var id int
		fmt.Sscanf(k, "%d", &id)
		if id > 0 {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	list := make([]*PricingVersion, 0, len(ids))
	for _, id := range ids {
		if v := st.Versions[fmt.Sprintf("%d", id)]; v != nil {
			list = append(list, v)
		}
	}
	return list, st.ActiveVersion, nil
}

// ActivatePricingVersion 切换生效版本：写入 options 并刷新运行时。
func ActivatePricingVersion(blockID string, versionID int, applyFn func(optionKey, payload string) error) error {
	def, ok := pricingBlockByID[blockID]
	if !ok {
		return fmt.Errorf("unknown block %s", blockID)
	}
	pricingVersionMu.Lock()
	defer pricingVersionMu.Unlock()

	store, err := loadStoreUnlocked()
	if err != nil {
		return err
	}
	st := store.Blocks[blockID]
	if st == nil {
		return fmt.Errorf("block %s has no versions", blockID)
	}
	v := st.Versions[fmt.Sprintf("%d", versionID)]
	if v == nil {
		return fmt.Errorf("version %d not found", versionID)
	}
	if applyFn == nil {
		return fmt.Errorf("apply handler not set")
	}
	defer skipVersionRecordOnSave()()
	if err := applyFn(def.OptionKey, v.Payload); err != nil {
		return err
	}
	st.ActiveVersion = versionID
	return persistStore(store)
}

// RecordSavedOptionAsVersion 在 Option 已写入 DB/内存后追加一条版本并标为生效（不再重复 UpdateOption）。
func RecordSavedOptionAsVersion(optionKey, payload string) (int, error) {
	if shouldSkipVersionRecordOnSave() {
		return 0, nil
	}
	def, ok := LookupBlockByOptionKey(optionKey)
	if !ok {
		return 0, nil
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		payload = "{}"
	}
	if err := validateBlockPayload(def, payload); err != nil {
		return 0, err
	}

	pricingVersionMu.Lock()
	defer pricingVersionMu.Unlock()

	store, err := loadStoreUnlocked()
	if err != nil {
		return 0, err
	}
	st := ensureBlock(store, def.BlockID)
	if cur := st.Versions[fmt.Sprintf("%d", st.ActiveVersion)]; cur != nil && cur.Payload == payload {
		return st.ActiveVersion, nil
	}
	id := st.NextVersion
	st.NextVersion++
	st.Versions[fmt.Sprintf("%d", id)] = &PricingVersion{
		ID:        id,
		Label:     VersionLabelOperationSave,
		Source:    VersionSourceOperationSave,
		CreatedAt: time.Now().Unix(),
		Payload:   payload,
	}
	st.ActiveVersion = id
	if err := persistStore(store); err != nil {
		return 0, err
	}
	return id, nil
}

func loadStoreUnlocked() (*PricingVersionStore, error) {
	raw, err := loadStoreRaw()
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return migrateStoreLocked()
	}
	var store PricingVersionStore
	if err := json.Unmarshal([]byte(raw), &store); err != nil {
		return nil, err
	}
	if store.Blocks == nil {
		store.Blocks = make(map[string]*pricingBlockState)
	}
	return &store, nil
}

// CreatePricingVersion 新增版本；activate 为 true 时同时生效。
func CreatePricingVersion(blockID, payload, label, source, note string, activate bool, applyFn func(optionKey, payload string) error) (int, error) {
	def, ok := pricingBlockByID[blockID]
	if !ok {
		return 0, fmt.Errorf("unknown block %s", blockID)
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		payload = "{}"
	}
	if err := validateBlockPayload(def, payload); err != nil {
		return 0, err
	}

	pricingVersionMu.Lock()
	defer pricingVersionMu.Unlock()

	store, err := loadStoreUnlocked()
	if err != nil {
		return 0, err
	}
	st := ensureBlock(store, blockID)
	id := st.NextVersion
	st.NextVersion++
	label = strings.TrimSpace(label)
	if label == "" {
		label = fmt.Sprintf("v%d", id)
	}
	st.Versions[fmt.Sprintf("%d", id)] = &PricingVersion{
		ID:        id,
		Label:     label,
		Source:    strings.TrimSpace(source),
		Note:      strings.TrimSpace(note),
		CreatedAt: time.Now().Unix(),
		Payload:   payload,
	}
	if activate {
		if applyFn == nil {
			return 0, fmt.Errorf("apply handler not set")
		}
		defer skipVersionRecordOnSave()()
		if err := applyFn(def.OptionKey, payload); err != nil {
			return 0, err
		}
		st.ActiveVersion = id
	}
	if err := persistStore(store); err != nil {
		return 0, err
	}
	return id, nil
}

func validateBlockPayload(def PricingBlockDef, payload string) error {
	if def.OptionKey == "BillingMode" || def.OptionKey == "BillingExpr" {
		var o map[string]string
		if err := json.Unmarshal([]byte(payload), &o); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		return nil
	}
	if def.OptionKey == "GroupGroupRatio" {
		var o map[string]map[string]float64
		if err := json.Unmarshal([]byte(payload), &o); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		return nil
	}
	var o map[string]float64
	if err := json.Unmarshal([]byte(payload), &o); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// MergePatchIntoBlockPayload 在指定块的当前生效 payload 上合并键值，生成新 payload 字符串。
func MergePatchIntoBlockPayload(blockID string, patches map[string]any) (string, error) {
	def, ok := pricingBlockByID[blockID]
	if !ok {
		return "", fmt.Errorf("unknown block %s", blockID)
	}
	base := currentOptionPayload(def.OptionKey)
	if base == "" {
		base = "{}"
	}

	if def.OptionKey == "BillingMode" || def.OptionKey == "BillingExpr" {
		var m map[string]string
		if err := json.Unmarshal([]byte(base), &m); err != nil || m == nil {
			m = make(map[string]string)
		}
		for k, v := range patches {
			ks := strings.TrimSpace(k)
			if ks == "" {
				continue
			}
			m[ks] = fmt.Sprint(v)
		}
		b, err := json.Marshal(m)
		return string(b), err
	}

	var m map[string]float64
	if err := json.Unmarshal([]byte(base), &m); err != nil || m == nil {
		m = make(map[string]float64)
	}
	for k, v := range patches {
		ks := strings.TrimSpace(k)
		if ks == "" {
			continue
		}
		switch n := v.(type) {
		case float64:
			m[ks] = n
		case float32:
			m[ks] = float64(n)
		case int:
			m[ks] = float64(n)
		case int64:
			m[ks] = float64(n)
		case json.Number:
			f, _ := n.Float64()
			m[ks] = f
		default:
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
				var f float64
				if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
					m[ks] = f
				}
			}
		}
	}
	b, err := json.Marshal(m)
	return string(b), err
}

// CreatePricingVersionEntryBased 基于分条价目创建版本（payload 为空，快照写入 pricing_version_entries）。
func CreatePricingVersionEntryBased(blockID, label, source, note string, activate bool) (int, error) {
	if _, ok := pricingBlockByID[blockID]; !ok {
		return 0, fmt.Errorf("unknown block %s", blockID)
	}

	pricingVersionMu.Lock()
	defer pricingVersionMu.Unlock()

	store, err := loadStoreUnlocked()
	if err != nil {
		return 0, err
	}
	st := ensureBlock(store, blockID)
	id := st.NextVersion
	st.NextVersion++
	label = strings.TrimSpace(label)
	if label == "" {
		label = fmt.Sprintf("v%d", id)
	}
	st.Versions[fmt.Sprintf("%d", id)] = &PricingVersion{
		ID:        id,
		Label:     label,
		Source:    strings.TrimSpace(source),
		Note:      strings.TrimSpace(note),
		CreatedAt: time.Now().Unix(),
	}
	if activate {
		st.ActiveVersion = id
	}
	if err := persistStore(store); err != nil {
		return 0, err
	}
	if RegisterPricingEntrySnapshot != nil {
		if err := RegisterPricingEntrySnapshot(blockID, id); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// SetActiveVersionOnly 仅更新版本库中的生效版本号（不写入 options）。
func SetActiveVersionOnly(blockID string, versionID int) error {
	if _, ok := pricingBlockByID[blockID]; !ok {
		return fmt.Errorf("unknown block %s", blockID)
	}
	pricingVersionMu.Lock()
	defer pricingVersionMu.Unlock()
	store, err := loadStoreUnlocked()
	if err != nil {
		return err
	}
	st := store.Blocks[blockID]
	if st == nil || st.Versions[fmt.Sprintf("%d", versionID)] == nil {
		return fmt.Errorf("version %d not found", versionID)
	}
	st.ActiveVersion = versionID
	return persistStore(store)
}

// ListBlockVersionsMeta 返回版本元数据（不含 payload）。
func ListBlockVersionsMeta(blockID string) ([]*PricingVersion, int, error) {
	list, active, err := ListBlockVersions(blockID)
	if err != nil {
		return nil, 0, err
	}
	for _, v := range list {
		v.Payload = ""
	}
	return list, active, nil
}

// SnapshotCurrentAsVersion 将当前运行时保存为新版本（不切换，除非 activate）。
func SnapshotCurrentAsVersion(blockID, label, source, note string, activate bool, applyFn func(optionKey, payload string) error) (int, error) {
	def, ok := pricingBlockByID[blockID]
	if !ok {
		return 0, fmt.Errorf("unknown block %s", blockID)
	}
	payload, err := payloadFromRuntime(def)
	if err != nil {
		return 0, err
	}
	return CreatePricingVersion(blockID, payload, label, source, note, activate, applyFn)
}
