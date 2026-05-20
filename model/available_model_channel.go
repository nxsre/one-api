package model

import (
	"strings"

	"github.com/songquanpeng/one-api/common"
)

// AvailableModelChannelRow 令牌「模型范围」展示用：模型与可提供该模型的渠道。
type AvailableModelChannelRow struct {
	Model       string `json:"model"`
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
}

// QueryAvailableModelChannelRowsAdmin 平台管理员：全部启用中的平台渠道（tenant_id 为空）及其 abilities。
func QueryAvailableModelChannelRowsAdmin() ([]AvailableModelChannelRow, error) {
	trueVal := "1"
	if common.UsingPostgreSQL {
		trueVal = "true"
	}
	var rows []AvailableModelChannelRow
	err := DB.Table("abilities").
		Select("abilities.model AS model, channels.id AS channel_id, channels.name AS channel_name").
		Joins("INNER JOIN channels ON channels.id = abilities.channel_id").
		Where("abilities.enabled = "+trueVal+" AND channels.status = ? AND channels.tenant_id IS NULL", ChannelStatusEnabled).
		Order("abilities.model ASC, channels.id ASC").
		Scan(&rows).Error
	return rows, err
}

// QueryAvailableModelChannelRowsScoped 指定用户分组下的可用 (模型, 渠道)：
// userTenantID==0 仅平台渠道；>0 为平台渠道或该租户私有渠道（与选路逻辑一致）。
func QueryAvailableModelChannelRowsScoped(group string, userTenantID int) ([]AvailableModelChannelRow, error) {
	group = strings.TrimSpace(group)
	if group == "" {
		group = "default"
	}
	groupCol := "abilities.`group`"
	trueVal := "1"
	if common.UsingPostgreSQL {
		groupCol = `abilities."group"`
		trueVal = "true"
	}
	var rows []AvailableModelChannelRow
	tx := DB.Table("abilities").
		Select("abilities.model AS model, channels.id AS channel_id, channels.name AS channel_name").
		Joins("INNER JOIN channels ON channels.id = abilities.channel_id").
		Where(groupCol+" = ? AND abilities.enabled = "+trueVal+" AND channels.status = ?", group, ChannelStatusEnabled)
	if userTenantID == 0 {
		tx = tx.Where("channels.tenant_id IS NULL")
	} else {
		tx = tx.Where("(channels.tenant_id IS NULL OR channels.tenant_id = ?)", userTenantID)
	}
	err := tx.Order("abilities.model ASC, channels.id ASC").Scan(&rows).Error
	return rows, err
}

// FilterModelChannelRowsForTenantSubUser 按租户子账号的 allowed_models / allowed_channel_ids 过滤行。
func FilterModelChannelRowsForTenantSubUser(u *User, rows []AvailableModelChannelRow) []AvailableModelChannelRow {
	if u == nil || u.TenantID == nil || u.Role != RoleCommonUser {
		return rows
	}
	var chAllow map[int]struct{}
	for _, id := range u.AllowedChannelIDs {
		if id > 0 {
			if chAllow == nil {
				chAllow = make(map[int]struct{})
			}
			chAllow[id] = struct{}{}
		}
	}
	var modelAllow map[string]struct{}
	for _, m := range u.AllowedModels {
		t := strings.TrimSpace(m)
		if t == "" {
			continue
		}
		if modelAllow == nil {
			modelAllow = make(map[string]struct{})
		}
		modelAllow[t] = struct{}{}
	}
	if chAllow == nil && modelAllow == nil {
		return rows
	}
	out := make([]AvailableModelChannelRow, 0, len(rows))
	for _, r := range rows {
		if chAllow != nil {
			if _, ok := chAllow[r.ChannelID]; !ok {
				continue
			}
		}
		if modelAllow != nil {
			if _, ok := modelAllow[r.Model]; !ok {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}
