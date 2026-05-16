package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/model"
	"gorm.io/gorm"
)

func nacosMcpSpecBytes(r *model.NacosAIMcpServer) ([]byte, error) {
	return NacosReadRegistryPayload(r.SpecStorageKind, r.SpecStorageRef, r.SpecJSON)
}

func nacosA2ACardBytes(r *model.NacosAIA2AAgent) ([]byte, error) {
	return NacosReadRegistryPayload(r.CardStorageKind, r.CardStorageRef, r.CardJSON)
}

func nacosPromptVersionContentBytes(v *model.NacosAIPromptVersion) ([]byte, error) {
	return NacosReadRegistryPayload(v.ContentStorageKind, v.ContentStorageRef, v.ContentJSON)
}

// --- MCP ---

type NacosMcpListData struct {
	TotalCount     int            `json:"totalCount"`
	PageNumber     int            `json:"pageNumber"`
	PagesAvailable int            `json:"pagesAvailable"`
	PageItems      []NacosMcpItem `json:"pageItems"`
}

type NacosMcpItem struct {
	NamespaceId string            `json:"namespaceId"`
	ServerName  string            `json:"serverName"`
	Description string            `json:"description,omitempty"`
	Spec        json.RawMessage   `json:"spec,omitempty"`
	BizTags     string            `json:"bizTags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Scope       string            `json:"scope"`
	Enable      bool              `json:"enable"`
	UpdateTime  int64             `json:"updateTime"`
}

func NacosAIListMcp(namespace, nameFilter string, pageNo, pageSize int) (*NacosMcpListData, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	q := model.DB.Model(&model.NacosAIMcpServer{}).Where("namespace_id = ?", ns)
	if strings.TrimSpace(nameFilter) != "" {
		q = q.Where("server_name LIKE ?", "%"+strings.TrimSpace(nameFilter)+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var rows []model.NacosAIMcpServer
	offset := (pageNo - 1) * pageSize
	if err := q.Order("updated_at desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		pages = 0
	}
	items := make([]NacosMcpItem, 0, len(rows))
	for _, r := range rows {
		var spec json.RawMessage
		if b, err := nacosMcpSpecBytes(&r); err == nil && len(b) > 0 {
			spec = json.RawMessage(b)
		}
		items = append(items, NacosMcpItem{
			NamespaceId: r.NamespaceId,
			ServerName:  r.ServerName,
			Description: r.Description,
			Spec:        spec,
			BizTags:     r.BizTags,
			Labels:      parseArtifactLabelsJSON(r.LabelsJSON),
			Scope:       strings.TrimSpace(r.Scope),
			Enable:      r.Enable,
			UpdateTime:  r.UpdatedAt.UnixMilli(),
		})
	}
	return &NacosMcpListData{
		TotalCount:     int(total),
		PageNumber:     pageNo,
		PagesAvailable: pages,
		PageItems:      items,
	}, nil
}

func NacosAIDescribeMcp(namespace, serverName string) (*NacosMcpItem, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(serverName)
	if name == "" {
		return nil, errors.New("serverName 必填")
	}
	var r model.NacosAIMcpServer
	if err := model.DB.Where("namespace_id = ? AND server_name = ?", ns, name).First(&r).Error; err != nil {
		return nil, err
	}
	var spec json.RawMessage
	if b, err := nacosMcpSpecBytes(&r); err == nil && len(b) > 0 {
		spec = json.RawMessage(b)
	}
	return &NacosMcpItem{
		NamespaceId: r.NamespaceId,
		ServerName:  r.ServerName,
		Description: r.Description,
		Spec:        spec,
		BizTags:     r.BizTags,
		Labels:      parseArtifactLabelsJSON(r.LabelsJSON),
		Scope:       strings.TrimSpace(r.Scope),
		Enable:      r.Enable,
		UpdateTime:  r.UpdatedAt.UnixMilli(),
	}, nil
}

func NacosAIUpsertMcp(namespace, serverName, description, specJSON, bizTags, labelsJSON, scope string, enable *bool) error {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(serverName)
	if name == "" {
		return errors.New("serverName 必填")
	}
	if strings.TrimSpace(specJSON) == "" {
		return errors.New("spec 必填（JSON 字符串）")
	}
	if !json.Valid([]byte(specJSON)) {
		return errors.New("spec 不是合法 JSON")
	}
	sc := strings.TrimSpace(scope)
	if sc == "" {
		sc = "PUBLIC"
	}
	payload := []byte(specJSON)
	var row model.NacosAIMcpServer
	err := model.DB.Where("namespace_id = ? AND server_name = ?", ns, name).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		en := true
		if enable != nil {
			en = *enable
		}
		lb := labelsJSON
		if lb == "" {
			lb = "{}"
		}
		return model.DB.Transaction(func(tx *gorm.DB) error {
			if NacosRegistryPayloadBackend() == "db" {
				return tx.Create(&model.NacosAIMcpServer{
					NamespaceId:       ns,
					ServerName:        name,
					Description:       description,
					SpecJSON:          string(payload),
					SpecStorageKind:   "db",
					SpecStorageRef:    "",
					BizTags:           bizTags,
					LabelsJSON:        lb,
					Scope:             sc,
					Enable:            en,
				}).Error
			}
			rec := &model.NacosAIMcpServer{
				NamespaceId:       ns,
				ServerName:        name,
				Description:       description,
				SpecJSON:          "",
				SpecStorageKind:   "",
				SpecStorageRef:    "",
				BizTags:           bizTags,
				LabelsJSON:        lb,
				Scope:             sc,
				Enable:            en,
			}
			if err := tx.Create(rec).Error; err != nil {
				return err
			}
			kind, ref, _, werr := NacosWriteRegistryPayload(ns, "mcp", rec.Id, payload)
			if werr != nil {
				return werr
			}
			return tx.Model(rec).Updates(map[string]interface{}{
				"spec_storage_kind": kind,
				"spec_storage_ref":  ref,
				"spec_json":         "",
			}).Error
		})
	}
	if err != nil {
		return err
	}
	oldKind, oldRef := row.SpecStorageKind, row.SpecStorageRef
	return model.DB.Transaction(func(tx *gorm.DB) error {
		kind, ref, dbInline, werr := NacosWriteRegistryPayload(ns, "mcp", row.Id, payload)
		if werr != nil {
			return werr
		}
		sj := ""
		if kind == "db" {
			sj = dbInline
		}
		up := map[string]interface{}{
			"description":       description,
			"spec_json":         sj,
			"spec_storage_kind": kind,
			"spec_storage_ref":  ref,
			"biz_tags":          bizTags,
			"scope":             sc,
		}
		if labelsJSON != "" {
			up["labels_json"] = labelsJSON
		}
		if enable != nil {
			up["enable"] = *enable
		}
		if err := tx.Model(&row).Updates(up).Error; err != nil {
			_ = NacosRemoveRegistryPayload(kind, ref)
			return err
		}
		oldExt := strings.EqualFold(oldKind, "local") || strings.EqualFold(oldKind, "s3")
		if oldExt && (!strings.EqualFold(oldKind, kind) || oldRef != ref) {
			_ = NacosRemoveRegistryPayload(oldKind, oldRef)
		}
		return nil
	})
}

func NacosAIDeleteMcp(namespace, serverName string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(serverName)
	if name == "" {
		return errors.New("serverName 必填")
	}
	var row model.NacosAIMcpServer
	err := model.DB.Where("namespace_id = ? AND server_name = ?", ns, name).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	_ = NacosRemoveRegistryPayload(row.SpecStorageKind, row.SpecStorageRef)
	return model.DB.Where("namespace_id = ? AND server_name = ?", ns, name).Delete(&model.NacosAIMcpServer{}).Error
}

// --- A2A ---

type NacosA2AListData struct {
	TotalCount     int            `json:"totalCount"`
	PageNumber     int            `json:"pageNumber"`
	PagesAvailable int            `json:"pagesAvailable"`
	PageItems      []NacosA2AItem `json:"pageItems"`
}

type NacosA2AItem struct {
	NamespaceId string            `json:"namespaceId"`
	AgentName   string            `json:"agentName"`
	Description string            `json:"description,omitempty"`
	Card        json.RawMessage   `json:"card,omitempty"`
	BizTags     string            `json:"bizTags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Scope       string            `json:"scope"`
	Enable      bool              `json:"enable"`
	UpdateTime  int64             `json:"updateTime"`
}

func NacosAIListA2A(namespace, nameFilter string, pageNo, pageSize int) (*NacosA2AListData, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	q := model.DB.Model(&model.NacosAIA2AAgent{}).Where("namespace_id = ?", ns)
	if strings.TrimSpace(nameFilter) != "" {
		q = q.Where("agent_name LIKE ?", "%"+strings.TrimSpace(nameFilter)+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var rows []model.NacosAIA2AAgent
	offset := (pageNo - 1) * pageSize
	if err := q.Order("updated_at desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		pages = 0
	}
	items := make([]NacosA2AItem, 0, len(rows))
	for _, r := range rows {
		var card json.RawMessage
		if b, err := nacosA2ACardBytes(&r); err == nil && len(b) > 0 {
			card = json.RawMessage(b)
		}
		items = append(items, NacosA2AItem{
			NamespaceId: r.NamespaceId,
			AgentName:   r.AgentName,
			Description: r.Description,
			Card:        card,
			BizTags:     r.BizTags,
			Labels:      parseArtifactLabelsJSON(r.LabelsJSON),
			Scope:       strings.TrimSpace(r.Scope),
			Enable:      r.Enable,
			UpdateTime:  r.UpdatedAt.UnixMilli(),
		})
	}
	return &NacosA2AListData{
		TotalCount:     int(total),
		PageNumber:     pageNo,
		PagesAvailable: pages,
		PageItems:      items,
	}, nil
}

func NacosAIDescribeA2A(namespace, agentName string) (*NacosA2AItem, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(agentName)
	if name == "" {
		return nil, errors.New("agentName 必填")
	}
	var r model.NacosAIA2AAgent
	if err := model.DB.Where("namespace_id = ? AND agent_name = ?", ns, name).First(&r).Error; err != nil {
		return nil, err
	}
	var card json.RawMessage
	if b, err := nacosA2ACardBytes(&r); err == nil && len(b) > 0 {
		card = json.RawMessage(b)
	}
	return &NacosA2AItem{
		NamespaceId: r.NamespaceId,
		AgentName:   r.AgentName,
		Description: r.Description,
		Card:        card,
		BizTags:     r.BizTags,
		Labels:      parseArtifactLabelsJSON(r.LabelsJSON),
		Scope:       strings.TrimSpace(r.Scope),
		Enable:      r.Enable,
		UpdateTime:  r.UpdatedAt.UnixMilli(),
	}, nil
}

func NacosAIUpsertA2A(namespace, agentName, description, cardJSON, bizTags, labelsJSON, scope string, enable *bool) error {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(agentName)
	if name == "" {
		return errors.New("agentName 必填")
	}
	if strings.TrimSpace(cardJSON) == "" {
		return errors.New("card 必填（JSON 字符串）")
	}
	if !json.Valid([]byte(cardJSON)) {
		return errors.New("card 不是合法 JSON")
	}
	sc := strings.TrimSpace(scope)
	if sc == "" {
		sc = "PUBLIC"
	}
	payload := []byte(cardJSON)
	var row model.NacosAIA2AAgent
	err := model.DB.Where("namespace_id = ? AND agent_name = ?", ns, name).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		en := true
		if enable != nil {
			en = *enable
		}
		lb := labelsJSON
		if lb == "" {
			lb = "{}"
		}
		return model.DB.Transaction(func(tx *gorm.DB) error {
			if NacosRegistryPayloadBackend() == "db" {
				return tx.Create(&model.NacosAIA2AAgent{
					NamespaceId:       ns,
					AgentName:         name,
					Description:       description,
					CardJSON:          string(payload),
					CardStorageKind:   "db",
					CardStorageRef:    "",
					BizTags:           bizTags,
					LabelsJSON:        lb,
					Scope:             sc,
					Enable:            en,
				}).Error
			}
			rec := &model.NacosAIA2AAgent{
				NamespaceId:       ns,
				AgentName:         name,
				Description:       description,
				CardJSON:          "",
				CardStorageKind:   "",
				CardStorageRef:    "",
				BizTags:           bizTags,
				LabelsJSON:        lb,
				Scope:             sc,
				Enable:            en,
			}
			if err := tx.Create(rec).Error; err != nil {
				return err
			}
			kind, ref, _, werr := NacosWriteRegistryPayload(ns, "a2a", rec.Id, payload)
			if werr != nil {
				return werr
			}
			return tx.Model(rec).Updates(map[string]interface{}{
				"card_storage_kind": kind,
				"card_storage_ref":  ref,
				"card_json":         "",
			}).Error
		})
	}
	if err != nil {
		return err
	}
	oldKind, oldRef := row.CardStorageKind, row.CardStorageRef
	return model.DB.Transaction(func(tx *gorm.DB) error {
		kind, ref, dbInline, werr := NacosWriteRegistryPayload(ns, "a2a", row.Id, payload)
		if werr != nil {
			return werr
		}
		cj := ""
		if kind == "db" {
			cj = dbInline
		}
		up := map[string]interface{}{
			"description":       description,
			"card_json":         cj,
			"card_storage_kind": kind,
			"card_storage_ref":  ref,
			"biz_tags":          bizTags,
			"scope":             sc,
		}
		if labelsJSON != "" {
			up["labels_json"] = labelsJSON
		}
		if enable != nil {
			up["enable"] = *enable
		}
		if err := tx.Model(&row).Updates(up).Error; err != nil {
			_ = NacosRemoveRegistryPayload(kind, ref)
			return err
		}
		oldExt := strings.EqualFold(oldKind, "local") || strings.EqualFold(oldKind, "s3")
		if oldExt && (!strings.EqualFold(oldKind, kind) || oldRef != ref) {
			_ = NacosRemoveRegistryPayload(oldKind, oldRef)
		}
		return nil
	})
}

func NacosAIDeleteA2A(namespace, agentName string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(agentName)
	if name == "" {
		return errors.New("agentName 必填")
	}
	var row model.NacosAIA2AAgent
	err := model.DB.Where("namespace_id = ? AND agent_name = ?", ns, name).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	_ = NacosRemoveRegistryPayload(row.CardStorageKind, row.CardStorageRef)
	return model.DB.Where("namespace_id = ? AND agent_name = ?", ns, name).Delete(&model.NacosAIA2AAgent{}).Error
}

// --- Prompt ---

type NacosPromptListItem struct {
	NamespaceId      string            `json:"namespaceId"`
	PromptKey        string            `json:"promptKey"`
	Description      string            `json:"description,omitempty"`
	BizTags          string            `json:"bizTags,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	Scope            string            `json:"scope"`
	Enable           bool              `json:"enable"`
	EditingVersion   string            `json:"editingVersion,omitempty"`
	ReviewingVersion string            `json:"reviewingVersion,omitempty"`
	OnlineCnt        int               `json:"onlineCnt"`
	UpdateTime       int64             `json:"updateTime"`
}

type NacosPromptVersionSummary struct {
	Version    string `json:"version"`
	Status     string `json:"status"`
	CreateTime *int64 `json:"createTime,omitempty"`
	UpdateTime *int64 `json:"updateTime,omitempty"`
}

type NacosPromptDetail struct {
	NacosPromptListItem
	Versions []NacosPromptVersionSummary `json:"versions,omitempty"`
}

type NacosPromptListData struct {
	TotalCount     int                   `json:"totalCount"`
	PageNumber     int                   `json:"pageNumber"`
	PagesAvailable int                   `json:"pagesAvailable"`
	PageItems      []NacosPromptListItem `json:"pageItems"`
}

func findPrompt(ns, key string) (*model.NacosAIPrompt, error) {
	var p model.NacosAIPrompt
	err := model.DB.Where("namespace_id = ? AND prompt_key = ?", ns, key).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func listPromptVersions(promptID int64) ([]model.NacosAIPromptVersion, error) {
	var vs []model.NacosAIPromptVersion
	err := model.DB.Where("prompt_id = ?", promptID).Order("created_at desc").Find(&vs).Error
	return vs, err
}

func NacosAIListPrompts(namespace, keyFilter string, pageNo, pageSize int) (*NacosPromptListData, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	q := model.DB.Model(&model.NacosAIPrompt{}).Where("namespace_id = ?", ns)
	if strings.TrimSpace(keyFilter) != "" {
		q = q.Where("prompt_key LIKE ?", "%"+strings.TrimSpace(keyFilter)+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var rows []model.NacosAIPrompt
	offset := (pageNo - 1) * pageSize
	if err := q.Order("updated_at desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		pages = 0
	}
	items := make([]NacosPromptListItem, 0, len(rows))
	for _, p := range rows {
		vs, _ := listPromptVersions(p.Id)
		on := 0
		for _, v := range vs {
			if v.Status == model.NacosAIVersionOnline {
				on++
			}
		}
		items = append(items, NacosPromptListItem{
			NamespaceId:      p.NamespaceId,
			PromptKey:        p.PromptKey,
			Description:      p.Description,
			BizTags:          p.BizTags,
			Labels:           parseArtifactLabelsJSON(p.LabelsJSON),
			Scope:            strings.TrimSpace(p.Scope),
			Enable:           p.Enable,
			EditingVersion:   pickLatestVersionByStatusPrompt(vs, model.NacosAIVersionEditing),
			ReviewingVersion: pickLatestVersionByStatusPrompt(vs, model.NacosAIVersionReviewing),
			OnlineCnt:        on,
			UpdateTime:       p.UpdatedAt.UnixMilli(),
		})
	}
	return &NacosPromptListData{
		TotalCount:     int(total),
		PageNumber:     pageNo,
		PagesAvailable: pages,
		PageItems:      items,
	}, nil
}

func pickLatestVersionByStatusPrompt(vs []model.NacosAIPromptVersion, st string) string {
	for _, v := range vs {
		if v.Status == st {
			return v.Version
		}
	}
	return ""
}

func findPromptVersion(vs []model.NacosAIPromptVersion, ver string) *model.NacosAIPromptVersion {
	for i := range vs {
		if vs[i].Version == ver {
			return &vs[i]
		}
	}
	return nil
}

// NacosAIUpdatePromptLabels 合并或替换 Prompt 头的 labels（与 Skill/AgentSpec 语义一致）。
func NacosAIUpdatePromptLabels(namespace, promptKey string, labels map[string]string, replace bool) error {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	if key == "" {
		return errors.New("promptKey 必填")
	}
	p, err := findPrompt(ns, key)
	if err != nil {
		return err
	}
	var merged map[string]string
	if replace {
		merged = map[string]string{}
	} else {
		merged = parseArtifactLabelsJSON(p.LabelsJSON)
	}
	for k, v := range labels {
		kk := strings.TrimSpace(k)
		if kk == "" {
			continue
		}
		merged[kk] = strings.TrimSpace(v)
	}
	return model.DB.Model(p).Update("labels_json", marshalArtifactLabels(merged)).Error
}

func NacosAIDescribePrompt(namespace, promptKey string) (*NacosPromptDetail, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	if key == "" {
		return nil, errors.New("promptKey 必填")
	}
	p, err := findPrompt(ns, key)
	if err != nil {
		return nil, err
	}
	vs, err := listPromptVersions(p.Id)
	if err != nil {
		return nil, err
	}
	on := 0
	for _, v := range vs {
		if v.Status == model.NacosAIVersionOnline {
			on++
		}
	}
	base := NacosPromptListItem{
		NamespaceId:      p.NamespaceId,
		PromptKey:        p.PromptKey,
		Description:      p.Description,
		BizTags:          p.BizTags,
		Labels:           parseArtifactLabelsJSON(p.LabelsJSON),
		Scope:            strings.TrimSpace(p.Scope),
		Enable:           p.Enable,
		EditingVersion:   pickLatestVersionByStatusPrompt(vs, model.NacosAIVersionEditing),
		ReviewingVersion: pickLatestVersionByStatusPrompt(vs, model.NacosAIVersionReviewing),
		OnlineCnt:        on,
		UpdateTime:       p.UpdatedAt.UnixMilli(),
	}
	sums := make([]NacosPromptVersionSummary, 0, len(vs))
	for _, v := range vs {
		ct := v.CreatedAt.UnixMilli()
		ut := v.UpdatedAt.UnixMilli()
		sums = append(sums, NacosPromptVersionSummary{
			Version:    v.Version,
			Status:     v.Status,
			CreateTime: ptrI64(ct),
			UpdateTime: ptrI64(ut),
		})
	}
	return &NacosPromptDetail{NacosPromptListItem: base, Versions: sums}, nil
}

func NacosAIUpsertPromptHeader(namespace, promptKey, description, bizTags, labelsJSON, scope string, enable *bool) error {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	if key == "" {
		return errors.New("promptKey 必填")
	}
	sc := strings.TrimSpace(scope)
	if sc == "" {
		sc = "PUBLIC"
	}
	var p model.NacosAIPrompt
	err := model.DB.Where("namespace_id = ? AND prompt_key = ?", ns, key).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		en := true
		if enable != nil {
			en = *enable
		}
		lb := labelsJSON
		if lb == "" {
			lb = "{}"
		}
		return model.DB.Create(&model.NacosAIPrompt{
			NamespaceId: ns,
			PromptKey:   key,
			Description: description,
			BizTags:     bizTags,
			LabelsJSON:  lb,
			Scope:       sc,
			Enable:      en,
		}).Error
	}
	if err != nil {
		return err
	}
	up := map[string]interface{}{
		"description": description,
		"biz_tags":    bizTags,
		"scope":       sc,
	}
	if labelsJSON != "" {
		up["labels_json"] = labelsJSON
	}
	if enable != nil {
		up["enable"] = *enable
	}
	return model.DB.Model(&p).Updates(up).Error
}

func NacosAIPromptAddVersion(namespace, promptKey, contentJSON string) (version string, err error) {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	if key == "" {
		return "", errors.New("promptKey 必填")
	}
	if strings.TrimSpace(contentJSON) == "" {
		return "", errors.New("content 必填")
	}
	if !json.Valid([]byte(contentJSON)) {
		return "", errors.New("content 不是合法 JSON")
	}
	p, err := findPrompt(ns, key)
	if err != nil {
		return "", err
	}
	ver := fmt.Sprintf("v%d", time.Now().UnixMilli())
	payload := []byte(contentJSON)
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if NacosRegistryPayloadBackend() == "db" {
			nv := &model.NacosAIPromptVersion{
				PromptId:             p.Id,
				Version:              ver,
				Status:               model.NacosAIVersionEditing,
				ContentJSON:          contentJSON,
				ContentStorageKind:   "db",
				ContentStorageRef:    "",
			}
			return tx.Create(nv).Error
		}
		nv := &model.NacosAIPromptVersion{
			PromptId:             p.Id,
			Version:              ver,
			Status:               model.NacosAIVersionEditing,
			ContentJSON:          "",
			ContentStorageKind:   "",
			ContentStorageRef:    "",
		}
		if err := tx.Create(nv).Error; err != nil {
			return err
		}
		kind, ref, _, werr := NacosWriteRegistryPayload(ns, "prompt", nv.Id, payload)
		if werr != nil {
			return werr
		}
		return tx.Model(nv).Updates(map[string]interface{}{
			"content_storage_kind": kind,
			"content_storage_ref":    ref,
			"content_json":           "",
		}).Error
	})
	return ver, err
}

// NacosAIPromptVersionRawContent 读取任意状态 Prompt 版本正文（JSON 字节）。
func NacosAIPromptVersionRawContent(namespace, promptKey, version string) ([]byte, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	ver := strings.TrimSpace(version)
	if key == "" || ver == "" {
		return nil, errors.New("promptKey 与 version 必填")
	}
	p, err := findPrompt(ns, key)
	if err != nil {
		return nil, err
	}
	vs, err := listPromptVersions(p.Id)
	if err != nil {
		return nil, err
	}
	pv := findPromptVersion(vs, ver)
	if pv == nil {
		return nil, fmt.Errorf("版本 %q 不存在", ver)
	}
	return nacosPromptVersionContentBytes(pv)
}

// NacosAIPromptUpsertEditingContent 写入 editing 版本正文；若无 editing 版本则新建一条。
func NacosAIPromptUpsertEditingContent(namespace, promptKey, contentJSON string) (version string, err error) {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	if key == "" {
		return "", errors.New("promptKey 必填")
	}
	if strings.TrimSpace(contentJSON) == "" {
		return "", errors.New("content 必填")
	}
	if !json.Valid([]byte(contentJSON)) {
		return "", errors.New("content 不是合法 JSON")
	}
	p, err := findPrompt(ns, key)
	if err != nil {
		return "", err
	}
	vs, err := listPromptVersions(p.Id)
	if err != nil {
		return "", err
	}
	var target *model.NacosAIPromptVersion
	for i := range vs {
		if vs[i].Status == model.NacosAIVersionEditing {
			target = &vs[i]
			break
		}
	}
	if target == nil {
		return NacosAIPromptAddVersion(ns, key, contentJSON)
	}
	payload := []byte(contentJSON)
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if NacosRegistryPayloadBackend() == "db" {
			return tx.Model(target).Updates(map[string]interface{}{
				"content_json":           contentJSON,
				"content_storage_kind":   "db",
				"content_storage_ref":    "",
			}).Error
		}
		_ = NacosRemoveRegistryPayload(target.ContentStorageKind, target.ContentStorageRef)
		kind, ref, _, werr := NacosWriteRegistryPayload(ns, "prompt", target.Id, payload)
		if werr != nil {
			return werr
		}
		return tx.Model(target).Updates(map[string]interface{}{
			"content_json":           "",
			"content_storage_kind":   kind,
			"content_storage_ref":    ref,
		}).Error
	})
	return target.Version, err
}

// NacosAIPromptDeleteEditingVersions 删除所有 editing 状态的版本。
func NacosAIPromptDeleteEditingVersions(namespace, promptKey string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	if key == "" {
		return errors.New("promptKey 必填")
	}
	p, err := findPrompt(ns, key)
	if err != nil {
		return err
	}
	var vs []model.NacosAIPromptVersion
	if err := model.DB.Where("prompt_id = ? AND status = ?", p.Id, model.NacosAIVersionEditing).Find(&vs).Error; err != nil {
		return err
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		for i := range vs {
			_ = NacosRemoveRegistryPayload(vs[i].ContentStorageKind, vs[i].ContentStorageRef)
			if err := tx.Delete(&model.NacosAIPromptVersion{}, vs[i].Id).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// NacosAIPromptVersionListPage 分页返回版本列表（供 console /versions）。
func NacosAIPromptVersionListPage(namespace, promptKey string, pageNo, pageSize int) (total int64, items []NacosPromptVersionSummary, err error) {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	if key == "" {
		return 0, nil, errors.New("promptKey 必填")
	}
	p, err := findPrompt(ns, key)
	if err != nil {
		return 0, nil, err
	}
	q := model.DB.Model(&model.NacosAIPromptVersion{}).Where("prompt_id = ?", p.Id)
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var rows []model.NacosAIPromptVersion
	offset := (pageNo - 1) * pageSize
	if err := q.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	out := make([]NacosPromptVersionSummary, 0, len(rows))
	for _, v := range rows {
		ct := v.CreatedAt.UnixMilli()
		ut := v.UpdatedAt.UnixMilli()
		out = append(out, NacosPromptVersionSummary{
			Version:    v.Version,
			Status:     v.Status,
			CreateTime: ptrI64(ct),
			UpdateTime: ptrI64(ut),
		})
	}
	return total, out, nil
}

func NacosAIPromptSubmit(namespace, promptKey, version string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	p, err := findPrompt(ns, key)
	if err != nil {
		return err
	}
	vs, err := listPromptVersions(p.Id)
	if err != nil {
		return err
	}
	var target *model.NacosAIPromptVersion
	if strings.TrimSpace(version) != "" {
		target = findPromptVersion(vs, version)
	} else {
		for i := range vs {
			if vs[i].Status == model.NacosAIVersionEditing {
				target = &vs[i]
				break
			}
		}
	}
	if target == nil {
		return errors.New("没有可提交的草稿版本")
	}
	if target.Status != model.NacosAIVersionEditing {
		return fmt.Errorf("版本 %s 状态不是 editing", target.Version)
	}
	return model.DB.Model(target).Updates(map[string]interface{}{
		"status": model.NacosAIVersionReviewing,
	}).Error
}

func NacosAIPromptPublish(namespace, promptKey, version string, updateLatest bool, force bool) error {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	p, err := findPrompt(ns, key)
	if err != nil {
		return err
	}
	vs, err := listPromptVersions(p.Id)
	if err != nil {
		return err
	}
	target := findPromptVersion(vs, version)
	if target == nil {
		return fmt.Errorf("版本 %q 不存在", version)
	}
	if !force && target.Status != model.NacosAIVersionReviewing {
		return fmt.Errorf("版本需先 submit 进入 reviewing")
	}
	if force && target.Status != model.NacosAIVersionReviewing && target.Status != model.NacosAIVersionEditing {
		return fmt.Errorf("force 发布仅允许 editing 或 reviewing")
	}
	if err := model.DB.Model(target).Updates(map[string]interface{}{
		"status": model.NacosAIVersionOnline,
	}).Error; err != nil {
		return err
	}
	if updateLatest {
		labels := parseArtifactLabelsJSON(p.LabelsJSON)
		labels["latest"] = version
		p.LabelsJSON = marshalArtifactLabels(labels)
		return model.DB.Model(p).Update("labels_json", p.LabelsJSON).Error
	}
	return nil
}

func NacosAIPromptVersionSetOffline(namespace, promptKey, version string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	p, err := findPrompt(ns, key)
	if err != nil {
		return err
	}
	vs, err := listPromptVersions(p.Id)
	if err != nil {
		return err
	}
	pv := findPromptVersion(vs, strings.TrimSpace(version))
	if pv == nil {
		return fmt.Errorf("版本 %q 不存在", version)
	}
	if pv.Status != model.NacosAIVersionOnline {
		return fmt.Errorf("仅 online 版本可下线，当前为 %s", pv.Status)
	}
	return model.DB.Model(pv).Update("status", model.NacosAIVersionOffline).Error
}

// NacosAIPromptVersionEnsureOnline 将 Prompt 版本置为 online（含从 offline 恢复或 force 发布草稿）。
func NacosAIPromptVersionEnsureOnline(namespace, promptKey, version string, updateLatest bool) error {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	ver := strings.TrimSpace(version)
	if key == "" || ver == "" {
		return errors.New("promptKey 与 version 必填")
	}
	p, err := findPrompt(ns, key)
	if err != nil {
		return err
	}
	vs, err := listPromptVersions(p.Id)
	if err != nil {
		return err
	}
	target := findPromptVersion(vs, ver)
	if target == nil {
		return fmt.Errorf("版本 %q 不存在", version)
	}
	switch target.Status {
	case model.NacosAIVersionOnline:
		if updateLatest {
			labels := parseArtifactLabelsJSON(p.LabelsJSON)
			labels["latest"] = ver
			p.LabelsJSON = marshalArtifactLabels(labels)
			return model.DB.Model(p).Update("labels_json", p.LabelsJSON).Error
		}
		return nil
	case model.NacosAIVersionOffline:
		if err := model.DB.Model(target).Update("status", model.NacosAIVersionOnline).Error; err != nil {
			return err
		}
		if updateLatest {
			labels := parseArtifactLabelsJSON(p.LabelsJSON)
			labels["latest"] = ver
			p.LabelsJSON = marshalArtifactLabels(labels)
			return model.DB.Model(p).Update("labels_json", p.LabelsJSON).Error
		}
		return nil
	case model.NacosAIVersionEditing, model.NacosAIVersionReviewing:
		return NacosAIPromptPublish(ns, key, ver, updateLatest, true)
	default:
		return fmt.Errorf("版本状态 %s 不支持上线", target.Status)
	}
}

func NacosAIPromptGetContent(namespace, promptKey, label, version string) (json.RawMessage, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	p, err := findPrompt(ns, key)
	if err != nil {
		return nil, err
	}
	vs, err := listPromptVersions(p.Id)
	if err != nil {
		return nil, err
	}
	var pv *model.NacosAIPromptVersion
	if strings.TrimSpace(label) != "" {
		labels := parseArtifactLabelsJSON(p.LabelsJSON)
		if v, ok := labels[label]; ok {
			pv = findPromptVersion(vs, v)
		}
		if pv == nil {
			return nil, fmt.Errorf("label %q 未找到", label)
		}
	} else if strings.TrimSpace(version) != "" {
		pv = findPromptVersion(vs, version)
		if pv == nil {
			return nil, fmt.Errorf("版本 %q 不存在", version)
		}
	} else {
		fake := &model.NacosAIArtifact{LabelsJSON: p.LabelsJSON}
		pv2, err2 := ResolveVersionForGet(fake, toArtifactVersionsForResolve(vs), "", "")
		if err2 != nil {
			return nil, err2
		}
		pv = findPromptVersion(vs, pv2.Version)
	}
	if pv == nil || pv.Status != model.NacosAIVersionOnline {
		return nil, errors.New("仅已发布(online)版本可读取")
	}
	raw, err := nacosPromptVersionContentBytes(pv)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("无可用 content")
	}
	return json.RawMessage(raw), nil
}

func toArtifactVersionsForResolve(vs []model.NacosAIPromptVersion) []model.NacosAIArtifactVersion {
	out := make([]model.NacosAIArtifactVersion, 0, len(vs))
	for _, v := range vs {
		out = append(out, model.NacosAIArtifactVersion{
			Version: v.Version,
			Status:  v.Status,
		})
	}
	return out
}

func NacosAIDeletePrompt(namespace, promptKey string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	key := strings.TrimSpace(promptKey)
	if key == "" {
		return errors.New("promptKey 必填")
	}
	p, err := findPrompt(ns, key)
	if err != nil {
		return err
	}
	vs, err := listPromptVersions(p.Id)
	if err != nil {
		return err
	}
	for i := range vs {
		_ = NacosRemoveRegistryPayload(vs[i].ContentStorageKind, vs[i].ContentStorageRef)
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("prompt_id = ?", p.Id).Delete(&model.NacosAIPromptVersion{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.NacosAIPrompt{}, p.Id).Error
	})
}

// --- Pipeline ---

type NacosPipelineRunItem struct {
	Id              int64           `json:"id"`
	NamespaceId     string          `json:"namespaceId"`
	JobType         string          `json:"jobType"`
	ResourceKind    string          `json:"resourceKind,omitempty"`
	ResourceName    string          `json:"resourceName,omitempty"`
	ResourceVersion string          `json:"resourceVersion,omitempty"`
	Status          string          `json:"status"`
	Message         string          `json:"message,omitempty"`
	Detail          json.RawMessage `json:"detail,omitempty"`
	CreatedAt       int64           `json:"createdAt"`
	UpdatedAt       int64           `json:"updatedAt"`
}

type NacosPipelineListData struct {
	TotalCount     int                    `json:"totalCount"`
	PageNumber     int                    `json:"pageNumber"`
	PagesAvailable int                    `json:"pagesAvailable"`
	PageItems      []NacosPipelineRunItem `json:"pageItems"`
}

func NacosAIListPipelineRuns(namespace string, pageNo, pageSize int) (*NacosPipelineListData, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	q := model.DB.Model(&model.NacosAIPipelineRun{}).Where("namespace_id = ?", ns)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var rows []model.NacosAIPipelineRun
	offset := (pageNo - 1) * pageSize
	if err := q.Order("id desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		pages = 0
	}
	items := make([]NacosPipelineRunItem, 0, len(rows))
	for _, r := range rows {
		var det json.RawMessage
		if strings.TrimSpace(r.DetailJSON) != "" {
			det = json.RawMessage(r.DetailJSON)
		}
		items = append(items, NacosPipelineRunItem{
			Id:              r.Id,
			NamespaceId:     r.NamespaceId,
			JobType:         r.JobType,
			ResourceKind:    r.ResourceKind,
			ResourceName:    r.ResourceName,
			ResourceVersion: r.ResourceVersion,
			Status:          r.Status,
			Message:         r.Message,
			Detail:          det,
			CreatedAt:       r.CreatedAt.UnixMilli(),
			UpdatedAt:       r.UpdatedAt.UnixMilli(),
		})
	}
	return &NacosPipelineListData{
		TotalCount:     int(total),
		PageNumber:     pageNo,
		PagesAvailable: pages,
		PageItems:      items,
	}, nil
}

func NacosAIDescribePipelineRun(id int64) (*NacosPipelineRunItem, error) {
	var r model.NacosAIPipelineRun
	if err := model.DB.First(&r, id).Error; err != nil {
		return nil, err
	}
	var det json.RawMessage
	if strings.TrimSpace(r.DetailJSON) != "" {
		det = json.RawMessage(r.DetailJSON)
	}
	return &NacosPipelineRunItem{
		Id:              r.Id,
		NamespaceId:     r.NamespaceId,
		JobType:         r.JobType,
		ResourceKind:    r.ResourceKind,
		ResourceName:    r.ResourceName,
		ResourceVersion: r.ResourceVersion,
		Status:          r.Status,
		Message:         r.Message,
		Detail:          det,
		CreatedAt:       r.CreatedAt.UnixMilli(),
		UpdatedAt:       r.UpdatedAt.UnixMilli(),
	}, nil
}

// --- Client reads (MCP / A2A) ---

// NacosAIMcpClientGet 客户端拉取已启用的 MCP 描述（JSON）。
func NacosAIMcpClientGet(namespace, serverName string) (json.RawMessage, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(serverName)
	if name == "" {
		return nil, errors.New("serverName 必填")
	}
	var r model.NacosAIMcpServer
	if err := model.DB.Where("namespace_id = ? AND server_name = ?", ns, name).First(&r).Error; err != nil {
		return nil, err
	}
	if !r.Enable {
		return nil, errors.New("资源已禁用")
	}
	b, err := nacosMcpSpecBytes(&r)
	if err != nil || len(strings.TrimSpace(string(b))) == 0 {
		return nil, errors.New("无可用 spec")
	}
	return json.RawMessage(b), nil
}

// NacosAIA2AClientGet 客户端拉取已启用的 Agent 卡片（JSON）。
func NacosAIA2AClientGet(namespace, agentName string) (json.RawMessage, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(agentName)
	if name == "" {
		return nil, errors.New("agentName 必填")
	}
	var r model.NacosAIA2AAgent
	if err := model.DB.Where("namespace_id = ? AND agent_name = ?", ns, name).First(&r).Error; err != nil {
		return nil, err
	}
	if !r.Enable {
		return nil, errors.New("资源已禁用")
	}
	b, err := nacosA2ACardBytes(&r)
	if err != nil || len(strings.TrimSpace(string(b))) == 0 {
		return nil, errors.New("无可用 card")
	}
	return json.RawMessage(b), nil
}

// NacosAIRunRegistryScan 写入一条 registry_scan 汇总记录。
func NacosAIRunRegistryScan(namespace string) (*model.NacosAIPipelineRun, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	run := &model.NacosAIPipelineRun{
		NamespaceId:  ns,
		JobType:      "registry_scan",
		ResourceKind: "*",
		Status:       "running",
		Message:      "scanning",
	}
	if err := model.DB.Create(run).Error; err != nil {
		return nil, err
	}
	var nSkill, nSpec, nMcp, nA2a, nPrompt int64
	_ = model.DB.Model(&model.NacosAIArtifact{}).Where("namespace_id = ? AND kind = ?", ns, model.NacosAIKindSkill).Count(&nSkill).Error
	_ = model.DB.Model(&model.NacosAIArtifact{}).Where("namespace_id = ? AND kind = ?", ns, model.NacosAIKindAgentSpec).Count(&nSpec).Error
	_ = model.DB.Model(&model.NacosAIMcpServer{}).Where("namespace_id = ?", ns).Count(&nMcp).Error
	_ = model.DB.Model(&model.NacosAIA2AAgent{}).Where("namespace_id = ?", ns).Count(&nA2a).Error
	_ = model.DB.Model(&model.NacosAIPrompt{}).Where("namespace_id = ?", ns).Count(&nPrompt).Error
	detail := map[string]int64{
		"skills": nSkill, "agentspecs": nSpec, "mcpServers": nMcp, "a2aAgents": nA2a, "prompts": nPrompt,
	}
	b, _ := json.Marshal(detail)
	run.Status = "success"
	run.Message = "ok"
	run.DetailJSON = string(b)
	if err := model.DB.Model(run).Updates(map[string]interface{}{
		"status": run.Status, "message": run.Message, "detail_json": run.DetailJSON,
	}).Error; err != nil {
		return nil, err
	}
	return run, nil
}
