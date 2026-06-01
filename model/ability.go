package model

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/utils"
)

type Ability struct {
	Group     string `json:"group" gorm:"type:varchar(64);primaryKey;autoIncrement:false"`
	Model     string `json:"model" gorm:"primaryKey;autoIncrement:false"`
	ChannelId int    `json:"channel_id" gorm:"primaryKey;autoIncrement:false;index"`
	Enabled   bool   `json:"enabled"`
	Priority  *int64 `json:"priority" gorm:"bigint;default:0;index"`
}

// QueryEnabledChannelsForGroupModel 返回分组下某模型可用渠道（按优先级降序、channel_id 升序，与内存缓存一致）。
// userTenantID==0 仅匹配平台渠道（tenant_id IS NULL）。
// userTenantID>0 为租户子账号：可选用本租户私有渠道 + 平台渠道（tenant_id IS NULL），均需 abilities 命中 group/model。
func QueryEnabledChannelsForGroupModel(group, model string, userTenantID int) ([]*Channel, error) {
	groupCol := "abilities.`group`"
	trueVal := "1"
	if common.UsingPostgreSQL {
		groupCol = `abilities."group"`
		trueVal = "true"
	}
	var channels []*Channel
	tx := DB.Table("channels").
		Select("channels.*").
		Joins("INNER JOIN abilities ON abilities.channel_id = channels.id").
		Where(groupCol+" = ? AND abilities.model = ? AND abilities.enabled = "+trueVal+" AND channels.status = ?", group, model, ChannelStatusEnabled)
	if userTenantID == 0 {
		tx = tx.Where("channels.tenant_id IS NULL")
	} else {
		tx = tx.Where("(channels.tenant_id IS NULL OR channels.tenant_id = ?)", userTenantID)
	}
	err := tx.Order("abilities.priority DESC, abilities.channel_id ASC").Find(&channels).Error
	return channels, err
}

func (channel *Channel) AddAbilities() error {
	models_ := strings.Split(channel.Models, ",")
	models_ = utils.DeDuplication(models_)
	groups_ := strings.Split(channel.Group, ",")
	abilities := make([]Ability, 0, len(models_))
	for _, model := range models_ {
		for _, group := range groups_ {
			ability := Ability{
				Group:     group,
				Model:     model,
				ChannelId: channel.Id,
				Enabled:   channel.Status == ChannelStatusEnabled,
				Priority:  channel.Priority,
			}
			abilities = append(abilities, ability)
		}
	}
	err := DB.Create(&abilities).Error
	if err == nil {
		PublishChannelsChanged()
	}
	return err
}

func (channel *Channel) DeleteAbilities() error {
	err := DB.Where("channel_id = ?", channel.Id).Delete(&Ability{}).Error
	if err == nil {
		PublishChannelsChanged()
	}
	return err
}

// UpdateAbilities updates abilities of this channel.
// Make sure the channel is completed before calling this function.
func (channel *Channel) UpdateAbilities() error {
	// A quick and dirty way to update abilities
	// First delete all abilities of this channel
	err := channel.DeleteAbilities()
	if err != nil {
		return err
	}
	// Then add new abilities
	err = channel.AddAbilities()
	if err != nil {
		return err
	}
	return nil
}

func UpdateAbilityStatus(channelId int, status bool) error {
	err := DB.Model(&Ability{}).Where("channel_id = ?", channelId).Select("enabled").Update("enabled", status).Error
	if err == nil {
		PublishChannelsChanged()
	}
	return err
}

func GetGroupModels(ctx context.Context, group string) ([]string, error) {
	groupCol := "`group`"
	trueVal := "1"
	if common.UsingPostgreSQL {
		groupCol = `"group"`
		trueVal = "true"
	}
	var models []string
	err := DB.Model(&Ability{}).Distinct("model").Where(groupCol+" = ? and enabled = "+trueVal, group).Pluck("model", &models).Error
	if err != nil {
		return nil, err
	}
	sort.Strings(models)
	return models, err
}

// GetDistinctModelsForGroupTenantChannelIDs 在给定分组下，针对渠道 ID 列表（可为租户私有 ID 或与分组匹配的平台渠道 ID）求 abilities 中已启用模型名的并集。
func GetDistinctModelsForGroupTenantChannelIDs(group string, tenantID int, channelIDs []int) ([]string, error) {
	if tenantID <= 0 || len(channelIDs) == 0 {
		return nil, nil
	}
	groupCol := "abilities.`group`"
	trueVal := "1"
	if common.UsingPostgreSQL {
		groupCol = `abilities."group"`
		trueVal = "true"
	}
	var models []string
	err := DB.Table("abilities").
		Distinct("abilities.model").
		Joins("INNER JOIN channels ON channels.id = abilities.channel_id").
		Where(groupCol+" = ? AND abilities.enabled = "+trueVal+" AND channels.status = ?", group, ChannelStatusEnabled).
		Where("(channels.tenant_id IS NULL OR channels.tenant_id = ?)", tenantID).
		Where("abilities.channel_id IN ?", channelIDs).
		Pluck("abilities.model", &models).Error
	if err != nil {
		return nil, err
	}
	sort.Strings(models)
	return models, nil
}

var fixAbilitiesMu sync.Mutex

// FixAbilities 清空 abilities 后按各渠道 models/group 重建，并刷新内存缓存。
func FixAbilities() (success int, fails int, err error) {
	if !fixAbilitiesMu.TryLock() {
		return 0, 0, errors.New("已有修复任务在执行中，请稍后再试")
	}
	defer fixAbilitiesMu.Unlock()
	if err = DB.Exec("DELETE FROM abilities").Error; err != nil {
		return 0, 0, err
	}
	var channels []*Channel
	if err = DB.Find(&channels).Error; err != nil {
		return 0, 0, err
	}
	for _, ch := range channels {
		if e := ch.AddAbilities(); e != nil {
			logger.SysError("FixAbilities AddAbilities channel " + strconv.Itoa(ch.Id) + ": " + e.Error())
			fails++
		} else {
			success++
		}
	}
	InitChannelCache()
	return success, fails, nil
}

// ListDistinctEnabledModels 返回 abilities 中已启用条目的去重模型名。
func ListDistinctEnabledModels() ([]string, error) {
	trueVal := "1"
	if common.UsingPostgreSQL {
		trueVal = "true"
	}
	var models []string
	err := DB.Model(&Ability{}).Distinct("model").Where("enabled = "+trueVal).Pluck("model", &models).Error
	if err != nil {
		return nil, err
	}
	sort.Strings(models)
	return models, nil
}
