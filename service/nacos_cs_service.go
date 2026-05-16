package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/songquanpeng/one-api/model"
	"gorm.io/gorm"
)

type NacosCsOperator struct {
	UserID   int
	Username string
}

type NacosCsConfigItem struct {
	Id         string `json:"id,omitempty"`
	DataID     string `json:"dataId"`
	Group      string `json:"group"`
	GroupName  string `json:"groupName"`
	Content    string `json:"content"`
	// EncryptedDataKey 与 Nacos/SDK 一致：客户端拉取密文模式时与 content（密文）一并返回；明文模式省略。
	EncryptedDataKey string `json:"encryptedDataKey,omitempty"`
	Type       string `json:"type"`
	Md5        string `json:"md5,omitempty"`
	AppName    string `json:"appName,omitempty"`
	ConfigTags string `json:"configTags,omitempty"`
	Desc       string `json:"desc,omitempty"`
	CreateTime string `json:"createTime,omitempty"`
	ModifyTime string `json:"modifyTime,omitempty"`
	UpdateTime int64  `json:"updateTime,omitempty"`
}

type NacosCsConfigListData struct {
	TotalCount     int                 `json:"totalCount"`
	PageNumber     int                 `json:"pageNumber"`
	PagesAvailable int                 `json:"pagesAvailable"`
	PageItems      []NacosCsConfigItem `json:"pageItems"`
}

type NacosCsHistoryItem struct {
	Id           int64  `json:"id"`
	ConfigId     int64  `json:"configId"`
	DataID       string `json:"dataId"`
	GroupName    string `json:"groupName"`
	Content      string `json:"content"`
	Type         string `json:"type"`
	Action       string `json:"action"`
	OperatorId   int    `json:"operatorId"`
	OperatorName string `json:"operatorName"`
	Remark       string `json:"remark,omitempty"`
	CreatedAt    int64  `json:"createdAt"`
}

type NacosCsHistoryListData struct {
	TotalCount     int                  `json:"totalCount"`
	PageNumber     int                  `json:"pageNumber"`
	PagesAvailable int                  `json:"pagesAvailable"`
	PageItems      []NacosCsHistoryItem `json:"pageItems"`
}

func nacosCsAppendHistory(tx *gorm.DB, cfg *model.NacosCsConfig, action, remark string, op *NacosCsOperator) error {
	h := &model.NacosCsConfigHistory{
		ConfigId:         cfg.Id,
		NamespaceId:      cfg.NamespaceId,
		GroupName:        cfg.GroupName,
		DataId:           cfg.DataId,
		Content:          cfg.Content,
		Type:             cfg.Type,
		EncryptedDataKey: cfg.EncryptedDataKey,
		Action:           action,
		Remark:        remark,
		OperatorId:    0,
		OperatorName:  "",
	}
	if op != nil {
		h.OperatorId = op.UserID
		if strings.TrimSpace(op.Username) != "" {
			h.OperatorName = strings.TrimSpace(op.Username)
		}
	}
	if h.Action == "" {
		h.Action = "publish"
	}
	return tx.Create(h).Error
}

func NacosCsList(namespace, dataID, groupName, search string, pageNo, pageSize int) (*NacosCsConfigListData, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	q := model.DB.Model(&model.NacosCsConfig{}).Where("namespace_id = ?", ns)
	if strings.TrimSpace(dataID) != "" {
		if search == "blur" || strings.Contains(dataID, "*") {
			q = q.Where("data_id LIKE ?", strings.ReplaceAll(dataID, "*", "%"))
		} else {
			q = q.Where("data_id = ?", dataID)
		}
	}
	if strings.TrimSpace(groupName) != "" {
		if search == "blur" || strings.Contains(groupName, "*") {
			q = q.Where("group_name LIKE ?", strings.ReplaceAll(groupName, "*", "%"))
		} else {
			q = q.Where("group_name = ?", groupName)
		}
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
	var rows []model.NacosCsConfig
	offset := (pageNo - 1) * pageSize
	if err := q.Order("updated_at desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		pages = 0
	}
	items := make([]NacosCsConfigItem, 0, len(rows))
	for _, r := range rows {
		it, err := nacosCsConfigItemFromModel(r, false)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return &NacosCsConfigListData{
		TotalCount:     int(total),
		PageNumber:     pageNo,
		PagesAvailable: pages,
		PageItems:      items,
	}, nil
}

func NacosCsGet(namespace, dataID, groupName string) (*NacosCsConfigItem, error) {
	return nacosCsGetDeliver(namespace, dataID, groupName, false)
}

// NacosCsGetForClient 客户端拉取；deliverCiphertext=true 时对已加密落库的配置返回密文 + encryptedDataKey（与 SDK 从 Server 获取的形态一致）。
func NacosCsGetForClient(namespace, dataID, groupName string, deliverCiphertext bool) (*NacosCsConfigItem, error) {
	return nacosCsGetDeliver(namespace, dataID, groupName, deliverCiphertext)
}

func nacosCsGetDeliver(namespace, dataID, groupName string, deliverCiphertext bool) (*NacosCsConfigItem, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	var r model.NacosCsConfig
	err := model.DB.Where("namespace_id = ? AND data_id = ? AND group_name = ?", ns, dataID, groupName).First(&r).Error
	if err != nil {
		return nil, err
	}
	item, err := nacosCsConfigItemFromModel(r, deliverCiphertext)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func nacosCsMD5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func nacosCsConfigItemFromModel(r model.NacosCsConfig, deliverCiphertext bool) (NacosCsConfigItem, error) {
	encField := strings.TrimSpace(r.EncryptedDataKey)
	storedCipher := encField != "" && encField == nacosCsEncEnvelope
	if deliverCiphertext && storedCipher {
		ct := r.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")
		mt := r.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")
		return NacosCsConfigItem{
			Id:               strconv.FormatInt(r.Id, 10),
			DataID:           r.DataId,
			Group:            r.GroupName,
			GroupName:        r.GroupName,
			Content:          r.Content,
			EncryptedDataKey: encField,
			Type:             r.Type,
			Md5:              nacosCsMD5Hex(r.Content),
			AppName:          "",
			ConfigTags:       "",
			Desc:             "",
			CreateTime:       ct,
			ModifyTime:       mt,
			UpdateTime:       r.UpdatedAt.UnixMilli(),
		}, nil
	}
	plain, err := NacosCsDecryptStored(r.Content, r.EncryptedDataKey)
	if err != nil {
		return NacosCsConfigItem{}, err
	}
	ct := r.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")
	mt := r.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")
	return NacosCsConfigItem{
		Id:         strconv.FormatInt(r.Id, 10),
		DataID:     r.DataId,
		Group:      r.GroupName,
		GroupName:  r.GroupName,
		Content:    plain,
		Type:       r.Type,
		Md5:        nacosCsMD5Hex(plain),
		AppName:    "",
		ConfigTags: "",
		Desc:       "",
		CreateTime: ct,
		ModifyTime: mt,
		UpdateTime: r.UpdatedAt.UnixMilli(),
	}, nil
}

// NacosCsPublish 兼容 nacos-cli（不记录操作者）；历史仍会写入。
func NacosCsPublish(namespace, dataID, groupName, content, typ string) (bool, error) {
	return NacosCsPublishWithOperator(namespace, dataID, groupName, content, typ, nil)
}

func NacosCsPublishWithOperator(namespace, dataID, groupName, content, typ string, op *NacosCsOperator) (bool, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	dataID = strings.TrimSpace(dataID)
	groupName = strings.TrimSpace(groupName)
	if dataID == "" || groupName == "" {
		return false, errors.New("dataId 与 groupName 必填")
	}
	storeContent, storeEnc, err := NacosCsPreparePublishPayload(dataID, content)
	if err != nil {
		return false, err
	}
	var created bool
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		var r model.NacosCsConfig
		err := tx.Where("namespace_id = ? AND data_id = ? AND group_name = ?", ns, dataID, groupName).First(&r).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r = model.NacosCsConfig{
				NamespaceId:      ns,
				GroupName:        groupName,
				DataId:           dataID,
				Content:          storeContent,
				Type:             typ,
				EncryptedDataKey: storeEnc,
			}
			if err := tx.Create(&r).Error; err != nil {
				return err
			}
			created = true
			return nacosCsAppendHistory(tx, &r, "publish", "", op)
		}
		r.Content = storeContent
		r.EncryptedDataKey = storeEnc
		if typ != "" {
			r.Type = typ
		}
		if err := tx.Save(&r).Error; err != nil {
			return err
		}
		return nacosCsAppendHistory(tx, &r, "publish", "", op)
	})
	if err != nil {
		return false, err
	}
	return created, nil
}

// NacosCsUpsertRaw 写入已持久化的 content / encrypted_data_key（克隆等场景，不二次加密）。
func NacosCsUpsertRaw(namespace, dataID, groupName, content, typ, encryptedDataKey string, op *NacosCsOperator) (bool, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	dataID = strings.TrimSpace(dataID)
	groupName = strings.TrimSpace(groupName)
	if dataID == "" || groupName == "" {
		return false, errors.New("dataId 与 groupName 必填")
	}
	var created bool
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var r model.NacosCsConfig
		err := tx.Where("namespace_id = ? AND data_id = ? AND group_name = ?", ns, dataID, groupName).First(&r).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r = model.NacosCsConfig{
				NamespaceId:      ns,
				GroupName:        groupName,
				DataId:           dataID,
				Content:          content,
				Type:             typ,
				EncryptedDataKey: encryptedDataKey,
			}
			if err := tx.Create(&r).Error; err != nil {
				return err
			}
			created = true
			return nacosCsAppendHistory(tx, &r, "publish", "", op)
		}
		r.Content = content
		r.EncryptedDataKey = encryptedDataKey
		if typ != "" {
			r.Type = typ
		}
		if err := tx.Save(&r).Error; err != nil {
			return err
		}
		return nacosCsAppendHistory(tx, &r, "publish", "", op)
	})
	return created, err
}

func NacosCsHistoryList(namespace, dataID, groupName string, pageNo, pageSize int) (*NacosCsHistoryListData, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	dataID = strings.TrimSpace(dataID)
	groupName = strings.TrimSpace(groupName)
	if dataID == "" || groupName == "" {
		return nil, errors.New("dataId 与 groupName 必填")
	}
	var cfg model.NacosCsConfig
	if err := model.DB.Where("namespace_id = ? AND data_id = ? AND group_name = ?", ns, dataID, groupName).First(&cfg).Error; err != nil {
		return nil, err
	}
	q := model.DB.Model(&model.NacosCsConfigHistory{}).Where("config_id = ?", cfg.Id)
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
	var rows []model.NacosCsConfigHistory
	offset := (pageNo - 1) * pageSize
	if err := q.Order("id desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		pages = 0
	}
	items := make([]NacosCsHistoryItem, 0, len(rows))
	for _, h := range rows {
		plain, err := NacosCsDecryptStored(h.Content, h.EncryptedDataKey)
		if err != nil {
			return nil, err
		}
		items = append(items, NacosCsHistoryItem{
			Id:           h.Id,
			ConfigId:     h.ConfigId,
			DataID:       h.DataId,
			GroupName:    h.GroupName,
			Content:      plain,
			Type:         h.Type,
			Action:       h.Action,
			OperatorId:   h.OperatorId,
			OperatorName: h.OperatorName,
			Remark:       h.Remark,
			CreatedAt:    h.CreatedAt.UnixMilli(),
		})
	}
	return &NacosCsHistoryListData{
		TotalCount:     int(total),
		PageNumber:     pageNo,
		PagesAvailable: pages,
		PageItems:      items,
	}, nil
}

// NacosCsRollback 将配置内容恢复为指定历史快照，并追加一条 rollback 历史。
func NacosCsRollback(namespace, dataID, groupName string, historyId int64, op *NacosCsOperator) error {
	if historyId <= 0 {
		return errors.New("historyId 无效")
	}
	ns := NormalizeNacosNamespaceID(namespace)
	dataID = strings.TrimSpace(dataID)
	groupName = strings.TrimSpace(groupName)
	if dataID == "" || groupName == "" {
		return errors.New("dataId 与 groupName 必填")
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var cfg model.NacosCsConfig
		if err := tx.Where("namespace_id = ? AND data_id = ? AND group_name = ?", ns, dataID, groupName).First(&cfg).Error; err != nil {
			return err
		}
		var hist model.NacosCsConfigHistory
		if err := tx.First(&hist, historyId).Error; err != nil {
			return err
		}
		if hist.ConfigId != cfg.Id || hist.NamespaceId != ns {
			return fmt.Errorf("历史记录与当前配置不匹配")
		}
		cfg.Content = hist.Content
		cfg.EncryptedDataKey = hist.EncryptedDataKey
		if strings.TrimSpace(hist.Type) != "" {
			cfg.Type = hist.Type
		}
		if err := tx.Save(&cfg).Error; err != nil {
			return err
		}
		remark := fmt.Sprintf("rollback from history id=%d", historyId)
		return nacosCsAppendHistory(tx, &cfg, "rollback", remark, op)
	})
}

func NacosCsDelete(namespace, dataID, groupName string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	dataID = strings.TrimSpace(dataID)
	groupName = strings.TrimSpace(groupName)
	if dataID == "" || groupName == "" {
		return errors.New("dataId 与 groupName 必填")
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var cfg model.NacosCsConfig
		if err := tx.Where("namespace_id = ? AND data_id = ? AND group_name = ?", ns, dataID, groupName).First(&cfg).Error; err != nil {
			return err
		}
		if err := tx.Where("config_id = ?", cfg.Id).Delete(&model.NacosCsConfigHistory{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.NacosCsConfig{}, cfg.Id).Error
	})
}

// NacosCsBatchDeleteByIDs 按主键批量删除（仅匹配 namespaceId）。
func NacosCsBatchDeleteByIDs(namespace string, ids []int64) (deleted int, err error) {
	ns := NormalizeNacosNamespaceID(namespace)
	if len(ids) == 0 {
		return 0, errors.New("ids 不能为空")
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			if id <= 0 {
				continue
			}
			var cfg model.NacosCsConfig
			if err := tx.Where("id = ? AND namespace_id = ?", id, ns).First(&cfg).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return err
			}
			if err := tx.Where("config_id = ?", cfg.Id).Delete(&model.NacosCsConfigHistory{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&model.NacosCsConfig{}, cfg.Id).Error; err != nil {
				return err
			}
			deleted++
		}
		return nil
	})
	return deleted, err
}

// NacosCsExportPayload 控制台 export2 下载用 JSON 数组。
func NacosCsExportPayload(namespace string, ids []int64, dataID, groupName string) ([]byte, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	q := model.DB.Model(&model.NacosCsConfig{}).Where("namespace_id = ?", ns)
	if len(ids) > 0 {
		q = q.Where("id IN ?", ids)
	} else {
		if strings.TrimSpace(dataID) != "" {
			q = q.Where("data_id = ?", strings.TrimSpace(dataID))
		}
		if strings.TrimSpace(groupName) != "" {
			q = q.Where("group_name = ?", strings.TrimSpace(groupName))
		}
	}
	var rows []model.NacosCsConfig
	if err := q.Order("data_id asc, group_name asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		item, err := nacosCsConfigItemFromModel(r, false)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"namespaceId": ns,
			"dataId":      item.DataID,
			"groupName":   item.GroupName,
			"content":     item.Content,
			"type":        item.Type,
			"md5":         item.Md5,
		})
	}
	return json.MarshalIndent(out, "", "  ")
}

// NacosCsHistoryGetByID 取单条历史（校验 namespace）。
func NacosCsHistoryGetByID(namespace string, historyID int64) (*model.NacosCsConfigHistory, error) {
	if historyID <= 0 {
		return nil, errors.New("nid 无效")
	}
	ns := NormalizeNacosNamespaceID(namespace)
	var h model.NacosCsConfigHistory
	if err := model.DB.First(&h, historyID).Error; err != nil {
		return nil, err
	}
	if h.NamespaceId != ns {
		return nil, fmt.Errorf("历史记录与 namespace 不匹配")
	}
	return &h, nil
}

// NacosCsHistoryPreviousByID 返回同一 config 下 id 小于 currentID 的最近一条历史。
func NacosCsHistoryPreviousByID(namespace string, currentID int64, dataID, groupName string) (*model.NacosCsConfigHistory, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	dataID = strings.TrimSpace(dataID)
	groupName = strings.TrimSpace(groupName)
	if currentID <= 0 || dataID == "" || groupName == "" {
		return nil, errors.New("id、dataId、groupName 必填")
	}
	var cur model.NacosCsConfigHistory
	if err := model.DB.First(&cur, currentID).Error; err != nil {
		return nil, err
	}
	if cur.NamespaceId != ns || cur.DataId != dataID || cur.GroupName != groupName {
		return nil, fmt.Errorf("历史记录与参数不匹配")
	}
	var prev model.NacosCsConfigHistory
	err := model.DB.Where("config_id = ? AND id < ?", cur.ConfigId, currentID).
		Order("id desc").First(&prev).Error
	if err != nil {
		return nil, err
	}
	return &prev, nil
}
