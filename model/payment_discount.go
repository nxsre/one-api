package model

import (
	"strings"

	"github.com/songquanpeng/one-api/common/helper"
)

// 支付折扣规则匹配维度。
const (
	DiscountMatchUser  = "user"  // 账号白名单：按 username 精确匹配
	DiscountMatchGroup = "group" // 按用户分组 user.group
	DiscountMatchTag   = "tag"   // 按用户标签 user.tags 之一
)

// PaymentDiscountRule 支付折扣规则：对命中的用户给定实付折扣（0,1]。
// 命中优先级：账号白名单(user) 优先；否则在分组/标签命中的规则里取折扣最低(最优)。无命中回落全局折扣。
type PaymentDiscountRule struct {
	Id          int     `json:"id" gorm:"primaryKey;autoIncrement"`
	MatchType   string  `json:"match_type" gorm:"size:16;not null;uniqueIndex:idx_pdr_match"`  // user|group|tag
	MatchValue  string  `json:"match_value" gorm:"size:191;not null;uniqueIndex:idx_pdr_match"` // username / group / tag
	Discount    float64 `json:"discount" gorm:"not null;default:1"`
	Enabled     bool    `json:"enabled" gorm:"not null;default:true"`
	UpdatedTime int64   `json:"updated_time" gorm:"bigint"`
}

func validDiscount(d float64) bool { return d > 0 && d <= 1 }

func ListPaymentDiscountRules() ([]PaymentDiscountRule, error) {
	var rows []PaymentDiscountRule
	err := DB.Order("match_type asc, match_value asc").Find(&rows).Error
	return rows, err
}

// SetPaymentDiscountRule upsert 一条折扣规则（按 match_type+match_value 唯一）。
func SetPaymentDiscountRule(matchType, matchValue string, discount float64, enabled bool) error {
	var row PaymentDiscountRule
	DB.FirstOrCreate(&row, PaymentDiscountRule{MatchType: matchType, MatchValue: matchValue})
	row.Discount = discount
	row.Enabled = enabled
	row.UpdatedTime = helper.GetTimestamp()
	return DB.Save(&row).Error
}

func DeletePaymentDiscountRule(matchType, matchValue string) error {
	return DB.Where("match_type = ? AND match_value = ?", matchType, matchValue).
		Delete(&PaymentDiscountRule{}).Error
}

// ResolveUserDiscount 计算某用户的实付折扣：账号白名单优先，其余取最优；均无命中返回 fallback。
func ResolveUserDiscount(userId int, fallback float64) float64 {
	u, err := GetUserById(userId, false)
	if err != nil {
		return fallback
	}
	rules, err := ListPaymentDiscountRules()
	if err != nil {
		return fallback
	}

	// 1) 账号白名单（user）优先：命中即用（多条取最低）。
	userBest := -1.0
	for _, r := range rules {
		if !r.Enabled || !validDiscount(r.Discount) {
			continue
		}
		if r.MatchType == DiscountMatchUser && r.MatchValue == u.Username {
			if userBest < 0 || r.Discount < userBest {
				userBest = r.Discount
			}
		}
	}
	if userBest > 0 {
		return userBest
	}

	// 2) 分组 / 标签命中：取折扣最低（最优）。
	tagSet := make(map[string]struct{}, len(u.Tags))
	for _, t := range u.Tags {
		tagSet[strings.TrimSpace(t)] = struct{}{}
	}
	best := -1.0
	for _, r := range rules {
		if !r.Enabled || !validDiscount(r.Discount) {
			continue
		}
		hit := false
		switch r.MatchType {
		case DiscountMatchGroup:
			hit = u.Group == r.MatchValue
		case DiscountMatchTag:
			_, hit = tagSet[r.MatchValue]
		}
		if hit && (best < 0 || r.Discount < best) {
			best = r.Discount
		}
	}
	if best > 0 {
		return best
	}
	return fallback
}
