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
func QueryEnabledChannelsForGroupModel(group, model string) ([]*Channel, error) {
	groupCol := "abilities.`group`"
	trueVal := "1"
	if common.UsingPostgreSQL {
		groupCol = `abilities."group"`
		trueVal = "true"
	}
	var channels []*Channel
	err := DB.Table("channels").
		Select("channels.*").
		Joins("INNER JOIN abilities ON abilities.channel_id = channels.id").
		Where(groupCol+" = ? AND abilities.model = ? AND abilities.enabled = "+trueVal+" AND channels.status = ?", group, model, ChannelStatusEnabled).
		Order("abilities.priority DESC, abilities.channel_id ASC").
		Find(&channels).Error
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
	return DB.Create(&abilities).Error
}

func (channel *Channel) DeleteAbilities() error {
	return DB.Where("channel_id = ?", channel.Id).Delete(&Ability{}).Error
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
	return DB.Model(&Ability{}).Where("channel_id = ?", channelId).Select("enabled").Update("enabled", status).Error
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
