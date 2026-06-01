package model

import (
	"sort"
	"strings"

	"github.com/songquanpeng/one-api/common/helper"
)

// 支付渠道授权作用域。
const (
	PaymentScopeUser   = "user"
	PaymentScopeTenant = "tenant"
)

// PaymentChannelAccess 记录「对某个用户 / 某个租户开放哪些支付渠道」。
// 默认策略：无任何授权 = 全部关闭。某用户的有效渠道 = 其用户级授权 ∪ 其所属租户级授权。
type PaymentChannelAccess struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ScopeType   string `json:"scope_type" gorm:"size:16;not null;uniqueIndex:idx_pca_scope"` // user | tenant
	ScopeId     int    `json:"scope_id" gorm:"not null;uniqueIndex:idx_pca_scope"`           // 用户 ID 或租户 ID
	Channels    string `json:"channels" gorm:"size:191;not null;default:''"`                 // CSV，如 "wxpay,alipay"
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

func splitChannels(csv string) []string {
	out := make([]string, 0, 4)
	seen := map[string]struct{}{}
	for _, p := range strings.Split(csv, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// GetPaymentChannelAccess 读取某作用域的授权渠道（不存在则返回空切片）。
func GetPaymentChannelAccess(scopeType string, scopeId int) ([]string, error) {
	var row PaymentChannelAccess
	err := DB.Where("scope_type = ? AND scope_id = ?", scopeType, scopeId).First(&row).Error
	if err != nil {
		return []string{}, nil // 含 record not found：视为未授权
	}
	return splitChannels(row.Channels), nil
}

// SetPaymentChannelAccess upsert 某作用域的授权渠道；channels 为空即清空（关闭全部）。
func SetPaymentChannelAccess(scopeType string, scopeId int, channels []string) error {
	value := strings.Join(splitChannels(strings.Join(channels, ",")), ",")
	var row PaymentChannelAccess
	DB.FirstOrCreate(&row, PaymentChannelAccess{ScopeType: scopeType, ScopeId: scopeId})
	row.Channels = value
	row.UpdatedTime = helper.GetTimestamp()
	return DB.Save(&row).Error
}

// ListPaymentChannelAccess 返回全部授权记录（后台管理列表用）。
func ListPaymentChannelAccess() ([]PaymentChannelAccess, error) {
	var rows []PaymentChannelAccess
	err := DB.Order("scope_type asc, scope_id asc").Find(&rows).Error
	return rows, err
}

// DeletePaymentChannelAccess 删除某作用域的授权。
func DeletePaymentChannelAccess(scopeType string, scopeId int) error {
	return DB.Where("scope_type = ? AND scope_id = ?", scopeType, scopeId).Delete(&PaymentChannelAccess{}).Error
}

// ResolveUserAllowedPaymentChannels 计算某用户「被授权」的支付渠道（默认全关）：
// 用户级授权 ∪ 所属租户级授权。返回结果未与「全局已启用渠道」求交，调用方需再过一遍全局开关。
func ResolveUserAllowedPaymentChannels(userId int) ([]string, error) {
	set := map[string]struct{}{}
	if uc, err := GetPaymentChannelAccess(PaymentScopeUser, userId); err == nil {
		for _, c := range uc {
			set[c] = struct{}{}
		}
	}
	if u, err := GetUserById(userId, false); err == nil && u.TenantID != nil {
		if tc, err := GetPaymentChannelAccess(PaymentScopeTenant, *u.TenantID); err == nil {
			for _, c := range tc {
				set[c] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out, nil
}
