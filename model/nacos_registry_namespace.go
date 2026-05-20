package model

import "time"

// NacosRegistryNamespace 控制台登记的 Nacos 仓库命名空间（用于下拉与固定 public 等）。
type NacosRegistryNamespace struct {
	Id              int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId     string    `json:"namespace_id" gorm:"size:128;uniqueIndex:uix_nacos_reg_ns_id;not null"`
	Remark          string    `json:"remark" gorm:"size:512"`
	TenantID        *int      `json:"tenant_id" gorm:"column:tenant_id;index"` // NULL=平台创建；非空=该租户拥有
	ExposeToTenants bool      `json:"expose_to_tenants" gorm:"column:expose_to_tenants;default:false"` // 仅平台命名空间：是否对租户开放（默认否）
	CreatedAt       time.Time `json:"created_at"`
}

func (NacosRegistryNamespace) TableName() string {
	return "nacos_registry_namespaces"
}
