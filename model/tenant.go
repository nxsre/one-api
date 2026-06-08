package model

import (
	"strings"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/agentpolicy"
	"gorm.io/gorm"
)

const (
	TenantStatusUnknown = 0
	TenantStatusEnabled = 1
	TenantStatusLocked  = 2
)

// Tenant 多租户空间；用户与渠道通过 tenant_id 关联（NULL 表示平台全局）。
type Tenant struct {
	Id                int     `json:"id" gorm:"primaryKey"`
	Name              string  `json:"name" gorm:"size:128;not null"`
	Slug              string  `json:"slug" gorm:"size:64;uniqueIndex;not null"` // URL/对外标识
	Status            int     `json:"status" gorm:"type:int;default:1"`
	DiscountRatio     float64 `json:"discount_ratio" gorm:"type:double;default:1.0"`         // 租户全局折扣倍率
	PricePer1kApiCall float64 `json:"price_per_1k_api_call" gorm:"type:double;default:-1.0"` // 租户私有渠道默认千次调用单价（覆盖系统全局，-1表示未配置则用全局）
	CreatedTime       int64   `json:"created_time" gorm:"bigint"`
	Remark            string  `json:"remark" gorm:"type:varchar(512);default:''"`
	// AgentClientPolicy 租户级 agent 客户端策略（白名单 + 可选限流）；空表示放开所有类型。
	AgentClientPolicy *agentpolicy.Policy `json:"agent_client_policy,omitempty" gorm:"column:agent_client_policy;type:text;serializer:json"`
}

func normalizeTenantSlug(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func GetTenantByID(id int) (*Tenant, error) {
	var t Tenant
	err := DB.First(&t, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func GetTenantBySlug(slug string) (*Tenant, error) {
	slug = normalizeTenantSlug(slug)
	if slug == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var t Tenant
	err := DB.Where("slug = ?", slug).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (t *Tenant) BeforeCreate(tx *gorm.DB) (err error) {
	if t.Id == 0 {
		t.Id = common.GetNextId()
	}
	return nil
}

func InsertTenant(t *Tenant) error {
	t.Slug = normalizeTenantSlug(t.Slug)
	return DB.Create(t).Error
}

// ListTenantsForPlatform 平台侧下拉/代管用租户列表（按 id 升序）。
func ListTenantsForPlatform(limit int) ([]Tenant, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	var rows []Tenant
	err := DB.Order("id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}
