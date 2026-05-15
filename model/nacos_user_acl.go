package model

import (
	"time"
)

// NacosUserACL 按用户细粒度 Nacos 兼容权限（与 Nacos Admin/Open API 的 READ/WRITE 语义对齐）。
// RulesJSON 为扁平 map，例如 {"admin:ai:skills:read":true,"admin:ai:skills:write":false}。
// Root 用户忽略此表；管理员无记录时默认全开；普通用户无记录时默认全无（仍受匿名读开关影响）。
type NacosUserACL struct {
	Id         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId     int       `json:"user_id" gorm:"column:user_id;uniqueIndex;not null"`
	RulesJSON  string    `json:"rules" gorm:"column:rules_json;type:text"`
	UpdatedBy  int       `json:"updated_by" gorm:"column:updated_by;default:0"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (NacosUserACL) TableName() string {
	return "nacos_user_acls"
}
