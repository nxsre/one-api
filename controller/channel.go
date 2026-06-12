package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

func GetAllChannels(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	pageSize := config.ItemsPerPage
	if ps := c.Query("page_size"); ps != "" {
		if v, e := strconv.Atoi(ps); e == nil && v > 0 && v <= 500 {
			pageSize = v
		}
	}
	idSort, _ := strconv.ParseBool(c.Query("id_sort"))
	typeF := -1
	if t := strings.TrimSpace(c.Query("type")); t != "" {
		typeF, _ = strconv.Atoi(t)
	}
	statusF := -1
	switch strings.ToLower(strings.TrimSpace(c.Query("status"))) {
	case "enabled", "1":
		statusF = model.ChannelStatusEnabled
	case "disabled", "0":
		statusF = 0
	}
	opt := model.ChannelPageOptions{
		Offset:               p * pageSize,
		Limit:                pageSize,
		Scope:                "limited",
		TypeFilter:           typeF,
		StatusFilter:         statusF,
		SortBy:               c.Query("sort_by"),
		SortOrder:            c.Query("sort_order"),
		IdSort:               idSort,
		PlatformChannelsOnly: true,
	}
	if tq := strings.TrimSpace(c.Query("tenant_id")); tq != "" {
		if tid, e := strconv.Atoi(tq); e == nil && tid > 0 {
			opt.TenantIDEq = &tid
			opt.PlatformChannelsOnly = false
		}
	}
	channels, _, err := model.GetChannelsPage(opt)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channels,
	})
	return
}

func SearchChannels(c *gin.Context) {
	keyword := c.Query("keyword")
	group := c.Query("group")
	modelKw := c.Query("model")
	var channels []*model.Channel
	var err error
	if strings.TrimSpace(group) != "" || strings.TrimSpace(modelKw) != "" {
		channels, err = model.SearchChannelsFiltered(keyword, group, modelKw)
	} else {
		channels, err = model.SearchChannels(keyword)
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channels,
	})
	return
}

// maskChannelKey 把渠道密钥脱敏为预览串（保留首尾各 4 位，中间固定 ****）。
// 仅用于 GetChannel 这类详情展示；完整密钥须经 2FA 门禁的 GetChannelKey 获取。
func maskChannelKey(key string) string {
	r := []rune(key)
	if len(r) == 0 {
		return ""
	}
	if len(r) <= 8 {
		return "****"
	}
	return string(r[:4]) + "****" + string(r[len(r)-4:])
}

func GetChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	// 取含密钥的渠道，但仅返回脱敏后的预览串（默认脱敏显示）。
	// root 与平台管理员可经 AdminAuth 访问本接口；查看完整密钥须走 2FA 门禁的 POST /:id/key。
	channel, err := model.GetChannelById(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	channel.Key = maskChannelKey(channel.Key)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channel,
	})
	return
}

// GetChannelKey 返回渠道完整密钥；需先通过 POST /api/verify 完成二次验证（SecureVerificationRequired）。
func GetChannelKey(c *gin.Context) {
	userId := c.GetInt("id")
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "渠道 ID 格式错误"})
		return
	}
	channel, err := model.GetChannelById(channelId, true)
	if err != nil || channel == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "渠道不存在"})
		return
	}
	model.RecordLog(c.Request.Context(), userId, model.LogTypeSystem, fmt.Sprintf("查看渠道密钥 (渠道ID: %d)", channelId))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"key": channel.Key,
		},
	})
}

// effectiveChannelTypeBaseURL 合并已有渠道与 PATCH，用于校验 Type / Base URL（更新时 PATCH 可能省略字段）。
func effectiveChannelTypeBaseURL(existing *model.Channel, patch *model.Channel) (int, string) {
	t := patch.Type
	if existing != nil && t == 0 {
		t = existing.Type
	}
	var base string
	if patch.BaseURL != nil {
		base = patch.GetBaseURL()
	} else if existing != nil {
		base = existing.GetBaseURL()
	} else {
		base = patch.GetBaseURL()
	}
	return t, base
}

func AddChannel(c *gin.Context) {
	channel := model.Channel{}
	err := c.ShouldBindJSON(&channel)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	channel.CreatedTime = helper.GetTimestamp()
	keys := strings.Split(channel.Key, "\n")
	channels := make([]model.Channel, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		localChannel := channel
		localChannel.Key = key
		if err := channeltype.ValidateChannelBaseURL(localChannel.Type, localChannel.GetBaseURL()); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		channels = append(channels, localChannel)
	}
	err = model.BatchInsertChannels(channels)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteChannel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	channel := model.Channel{Id: id}
	err := channel.Delete()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteDisabledChannel(c *gin.Context) {
	rows, err := model.DeleteDisabledChannel()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
	return
}

func UpdateChannel(c *gin.Context) {
	channel := model.Channel{}
	err := c.ShouldBindJSON(&channel)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	existing, err := model.GetChannelById(channel.Id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	t, base := effectiveChannelTypeBaseURL(existing, &channel)
	if err := channeltype.ValidateChannelBaseURL(t, base); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = channel.Update()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channel,
	})
	return
}

// ListEnabledChannelModels GET /api/channel/models_enabled
func ListEnabledChannelModels(c *gin.Context) {
	models, err := model.ListDistinctEnabledModels()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": models})
}

// FixChannelsAbilities POST /api/channel/fix
func FixChannelsAbilities(c *gin.Context) {
	s, f, err := model.FixAbilities()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"success": s,
			"fails":   f,
		},
	})
}

type channelBatchDeleteRequest struct {
	Ids []int `json:"ids"`
}

// DeleteChannelBatch POST /api/channel/batch
func DeleteChannelBatch(c *gin.Context) {
	var req channelBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	deleted := 0
	for _, id := range req.Ids {
		if id <= 0 {
			continue
		}
		ch := model.Channel{Id: id}
		if err := ch.Delete(); err == nil {
			deleted++
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": deleted})
}

// CopyChannel POST /api/channel/copy/:id
func CopyChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的渠道 Id"})
		return
	}
	src, err := model.GetChannelById(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	suffix := c.DefaultQuery("suffix", " copy")
	dst := *src
	dst.Id = 0
	dst.Name = src.Name + suffix
	dst.CreatedTime = helper.GetTimestamp()
	dst.TestTime = 0
	dst.ResponseTime = 0
	dst.UsedQuota = 0
	if c.DefaultQuery("reset_balance", "true") == "true" {
		dst.Balance = 0
		dst.BalanceUpdatedTime = 0
	}
	if err := channeltype.ValidateChannelBaseURL(dst.Type, dst.GetBaseURL()); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := dst.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	out, _ := model.GetChannelById(dst.Id, false)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": out})
}
