package controller

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	dbmodel "github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/routing"
)

// GetRoutingModelCatalog 静态适配器内置模型名 ∪ 全库渠道「已开通模型」字段（去重、排序），供路由运维下拉使用。
func GetRoutingModelCatalog(c *gin.Context) {
	set := make(map[string]struct{})
	for _, m := range mergedOpenAIModelsBestEffort() {
		id := strings.TrimSpace(m.Id)
		if id != "" {
			set[id] = struct{}{}
		}
	}
	channels, err := dbmodel.GetAllChannels(0, 0, "all")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		for _, part := range strings.Split(ch.Models, ",") {
			id := strings.TrimSpace(part)
			if id != "" {
				set[id] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

// GetRoutingMetricsDay Redis 按日聚合指标（展示用）。
func GetRoutingMetricsDay(c *gin.Context) {
	day := strings.TrimSpace(c.Query("day"))
	if day == "" {
		day = time.Now().UTC().Format("20060102")
	}
	data, err := routing.ScanChannelMetricsDay(day)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// PostRoutingManualWeight 写入 Redis 权重倍率（跨实例）。
func PostRoutingManualWeight(c *gin.Context) {
	var body struct {
		ChannelID  int     `json:"channel_id"`
		Multiplier float64 `json:"multiplier"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ChannelID <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid channel_id"})
		return
	}
	if err := routing.SetManualWeightMultiplier(body.ChannelID, body.Multiplier); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetRoutingChannelPreview 返回某分组+解析后模型下的候选渠道及有效权重（只做观测，不落库）。
func GetRoutingChannelPreview(c *gin.Context) {
	group := strings.TrimSpace(c.Query("group"))
	modelName := strings.TrimSpace(c.Query("model"))
	if group == "" || modelName == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "group 与 model 必填"})
		return
	}
	chs, err := dbmodel.LoadSortedChannelsForGroupModel(group, modelName, 0)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	type row struct {
		ID               int     `json:"id"`
		Name             string  `json:"name"`
		Provider         string  `json:"provider"`
		BaseWeight       uint    `json:"base_weight"`
		ManualMultiplier float64 `json:"manual_multiplier"`
		AutoMultiplier   float64 `json:"auto_multiplier"`
		AdaptiveEnabled  bool    `json:"adaptive_enabled"`
		EffectiveWeight  uint    `json:"effective_weight"`
		Priority         int64   `json:"priority"`
		CircuitOpen      bool    `json:"circuit_open"`
	}
	parsedPol := routing.CurrentRoutingPolicy()
	out := make([]row, 0, len(chs))
	for _, ch := range chs {
		mul := routing.GetManualWeightMultiplier(ch.Id)
		auto := routing.GetAutoWeightMultiplier(ch.Id)
		eff := ch.GetWeight()
		v := float64(eff) * mul
		adaptiveOn := ch.RoutingAdaptiveEffective()
		if parsedPol.AutoAdaptiveEnabled && adaptiveOn {
			v *= auto
		}
		if v < 1 {
			v = 1
		}
		out = append(out, row{
			ID:               ch.Id,
			Name:             ch.Name,
			Provider:         ch.RoutingProviderTag(),
			BaseWeight:       ch.GetWeight(),
			ManualMultiplier: mul,
			AutoMultiplier:   auto,
			AdaptiveEnabled:  adaptiveOn,
			EffectiveWeight:  uint(v + 0.999),
			Priority:         ch.GetPriority(),
			CircuitOpen:      routing.IsCircuitOpen(ch.Id),
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"candidate_count": len(out),
		"channels":        out,
	}})
}

// GetRoutingCircuitStates 批量查询渠道熔断 JSON。
func GetRoutingCircuitStates(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("channel_ids"))
	if raw == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "channel_ids 必填，逗号分隔"})
		return
	}
	parts := strings.Split(raw, ",")
	states := map[int]string{}
	for _, p := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || id <= 0 {
			continue
		}
		states[id] = routing.ReadCircuitRaw(id)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": states})
}

// GetRoutingPolicyBundle 返回四个路由相关 Option 原文（仅管理员）。
func GetRoutingPolicyBundle(c *gin.Context) {
	config.OptionMapRWMutex.RLock()
	defer config.OptionMapRWMutex.RUnlock()
	keys := []string{"RoutingPolicy", "RelayRetryPolicy", "ModelAliasPolicy", "ModelRateLimitPolicy", "RelayProtocolBridgeEnabled"}
	data := map[string]string{}
	for _, k := range keys {
		if config.OptionMap != nil {
			data[k] = config.OptionMap[k]
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// PostRoutingValidateAliasPolicy 校验 ModelAliasPolicy JSON（含分组覆盖环检测）。
func PostRoutingValidateAliasPolicy(c *gin.Context) {
	var body struct {
		Raw string `json:"model_alias_policy_json"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Raw) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid body"})
		return
	}
	if err := routing.ValidateModelAliasPolicyRaw(body.Raw); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetRoutingFuseEvents 最近熔断事件列表。
func GetRoutingFuseEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	ev, err := routing.ListRecentFuseEvents(limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": ev})
}

// GetRoutingTimeseries 渠道 + 模型分钟桶曲线。
func GetRoutingTimeseries(c *gin.Context) {
	idStr := strings.TrimSpace(c.Query("channel_id"))
	id, err := strconv.Atoi(idStr)
	model := strings.TrimSpace(c.Query("model"))
	hours, _ := strconv.Atoi(c.Query("hours"))
	if err != nil || id <= 0 || model == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "channel_id 与 model 必填"})
		return
	}
	pts, err := routing.QueryTimeseries(id, model, hours)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": pts})
}

// PostRoutingResetAutoWeight 清除某渠道自适应 Redis 倍率。
func PostRoutingResetAutoWeight(c *gin.Context) {
	var body struct {
		ChannelID int `json:"channel_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ChannelID <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid channel_id"})
		return
	}
	if err := routing.SetAutoWeightMultiplier(body.ChannelID, 0); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
