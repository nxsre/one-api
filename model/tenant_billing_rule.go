package model

import "github.com/songquanpeng/one-api/common/helper"

const (
	BillingRuleTypeDiscountRatio = 1 // 折扣率
	BillingRuleTypePricePer1k    = 2 // 千次调用单价
)

// TenantBillingRule 租户时效计费规则
type TenantBillingRule struct {
	Id         int     `json:"id" gorm:"primaryKey"`
	TenantId   int     `json:"tenant_id" gorm:"index"`
	ChannelId  int     `json:"channel_id" gorm:"index;default:0"` // 0表示全局规则
	RuleType   int     `json:"rule_type" gorm:"type:int;not null"` // 1=折扣率，2=千次调用单价
	Value      float64 `json:"value" gorm:"type:double;not null"`
	StartTime  int64   `json:"start_time" gorm:"bigint;not null"` // 生效开始时间戳
	EndTime    int64   `json:"end_time" gorm:"bigint;not null"`   // 生效结束时间戳
	Status     int     `json:"status" gorm:"type:int;default:1"` // 1: 启用, 2: 禁用
	CreatedAt  int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt  int64   `json:"updated_at" gorm:"bigint"`
}

func InsertTenantBillingRule(rule *TenantBillingRule) error {
	rule.CreatedAt = helper.GetTimestamp()
	rule.UpdatedAt = rule.CreatedAt
	return DB.Create(rule).Error
}

func GetTenantBillingRuleById(id int) (*TenantBillingRule, error) {
	var rule TenantBillingRule
	err := DB.First(&rule, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func GetTenantBillingRules(tenantId int) ([]*TenantBillingRule, error) {
	var rules []*TenantBillingRule
	err := DB.Where("tenant_id = ?", tenantId).Order("id desc").Find(&rules).Error
	return rules, err
}

func GetActiveTenantBillingRules(tenantId int, ruleType int, now int64) ([]*TenantBillingRule, error) {
	var rules []*TenantBillingRule
	err := DB.Where("tenant_id = ? AND rule_type = ? AND status = 1 AND start_time <= ? AND end_time >= ?", tenantId, ruleType, now, now).Find(&rules).Error
	return rules, err
}

func UpdateTenantBillingRule(rule *TenantBillingRule) error {
	rule.UpdatedAt = helper.GetTimestamp()
	return DB.Save(rule).Error
}

func DeleteTenantBillingRule(id int) error {
	return DB.Delete(&TenantBillingRule{}, "id = ?", id).Error
}