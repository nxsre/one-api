package model

import (
	"time"
)

// NacosCsConfig 兼容 nacos-cli 的配置中心（namespace + group + dataId 唯一）。
type NacosCsConfig struct {
	Id          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId string    `json:"namespaceId" gorm:"size:128;uniqueIndex:uix_nacos_cs;not null;default:''"`
	GroupName   string    `json:"groupName" gorm:"size:256;uniqueIndex:uix_nacos_cs;column:group_name;not null"`
	DataId      string    `json:"dataId" gorm:"size:256;uniqueIndex:uix_nacos_cs;not null"`
	Content           string    `json:"content" gorm:"type:longtext"`
	Type              string    `json:"type" gorm:"size:32"`
	EncryptedDataKey  string    `json:"encryptedDataKey,omitempty" gorm:"column:encrypted_data_key;type:text"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (NacosCsConfig) TableName() string {
	return "nacos_cs_configs"
}
