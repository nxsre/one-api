package model

import "time"

// NacosCsConfigHistory 配置变更历史（每次发布/回滚一条快照）。
type NacosCsConfigHistory struct {
	Id            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ConfigId      int64     `json:"configId" gorm:"column:config_id;index;not null"`
	NamespaceId   string    `json:"namespaceId" gorm:"size:128;index;not null;default:''"`
	GroupName     string    `json:"groupName" gorm:"size:256;column:group_name;not null"`
	DataId        string    `json:"dataId" gorm:"size:256;column:data_id;not null"`
	Content          string    `json:"content" gorm:"type:longtext"`
	Type             string    `json:"type" gorm:"size:32"`
	EncryptedDataKey string    `json:"encryptedDataKey,omitempty" gorm:"column:encrypted_data_key;type:text"`
	Action           string    `json:"action" gorm:"size:32;not null;default:'publish'"` // publish | rollback
	OperatorId    int       `json:"operatorId" gorm:"column:operator_id;default:0"`
	OperatorName  string    `json:"operatorName" gorm:"column:operator_name;size:128"`
	Remark        string    `json:"remark,omitempty" gorm:"size:512"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (NacosCsConfigHistory) TableName() string {
	return "nacos_cs_config_histories"
}
