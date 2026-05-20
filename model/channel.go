package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"gorm.io/gorm"
)

const (
	ChannelStatusUnknown          = 0
	ChannelStatusEnabled          = 1 // don't use 0, 0 is the default value!
	ChannelStatusManuallyDisabled = 2 // also don't use 0
	ChannelStatusAutoDisabled     = 3
)

type Channel struct {
	Id                 int     `json:"id"`
	Type               int     `json:"type" gorm:"default:0"`
	Key                string  `json:"key" gorm:"type:text"`
	// TenantID 非空表示租户专属渠道；NULL 表示平台级渠道（租户子账号可选用与其 Group 字段匹配的平台渠道）。
	TenantID           *int    `json:"tenant_id,omitempty" gorm:"column:tenant_id;index"`
	OpenAIOrganization *string `json:"openai_organization" gorm:"type:varchar(128)"`
	TestModel          *string `json:"test_model" gorm:"type:varchar(255)"`
	Status             int     `json:"status" gorm:"default:1"`
	Name               string  `json:"name" gorm:"index"`
	Weight             *uint   `json:"weight" gorm:"default:0"`
	CreatedTime        int64   `json:"created_time" gorm:"bigint"`
	TestTime           int64   `json:"test_time" gorm:"bigint"`
	ResponseTime       int     `json:"response_time"` // in milliseconds
	BaseURL            *string `json:"base_url" gorm:"column:base_url;default:''"`
	Balance            float64 `json:"balance"` // in USD
	BalanceUpdatedTime int64   `json:"balance_updated_time" gorm:"bigint"`
	Models             string  `json:"models"`
	Group              string  `json:"group" gorm:"type:varchar(64);default:'default'"`
	UsedQuota          int64   `json:"used_quota" gorm:"bigint;default:0"`
	ModelMapping       *string `json:"model_mapping" gorm:"type:text"`
	StatusCodeMapping  *string `json:"status_code_mapping" gorm:"type:varchar(1024)"`
	Priority           *int64  `json:"priority" gorm:"bigint;default:0"`
	AutoBan            *int    `json:"auto_ban" gorm:"default:1"`
	Remark             *string `json:"remark" gorm:"type:varchar(255)"`
	Tag                *string `json:"tag" gorm:"type:varchar(64);index"`
	ParamOverride      *string `json:"param_override" gorm:"type:text"`
	HeaderOverride     *string `json:"header_override" gorm:"type:text"`
	OtherInfo          string  `json:"other_info" gorm:"type:text"`
	Setting *string `json:"setting" gorm:"type:text"` // 渠道额外设置，JSON 文本
	// OtherSettings 其它扩展配置（无需索引），如 Azure API 版本、OpenRouter 等附加参数，JSON 文本；库列名为 settings。
	OtherSettings string `json:"settings" gorm:"column:settings;type:text"`
	Config             string  `json:"config"`
	SystemPrompt       *string `json:"system_prompt" gorm:"type:text"`
	ApiCallCount       int64   `json:"api_call_count" gorm:"bigint;default:0"` // 记录 API 调用总次数
}

type ChannelConfig struct {
	Region            string `json:"region,omitempty"`
	SK                string `json:"sk,omitempty"`
	AK                string `json:"ak,omitempty"`
	UserID            string `json:"user_id,omitempty"`
	APIVersion        string `json:"api_version,omitempty"`
	LibraryID         string `json:"library_id,omitempty"`
	Plugin            string `json:"plugin,omitempty"`
	VertexAIProjectID string `json:"vertex_ai_project_id,omitempty"`
	VertexAIADC       string `json:"vertex_ai_adc,omitempty"`
	// RoutingProvider Direction（上游集群/供应商）分组标签，用于双层打分与探测流量。
	RoutingProvider string `json:"routing_provider,omitempty"`
	// RoutingSkipAdaptive 为 true 时该渠道不参与自动降权/恢复倍率（仍保留手工倍率与熔断）。
	RoutingSkipAdaptive bool `json:"routing_skip_adaptive,omitempty"`
	// DeepResearchMode 深知调研模式：general | academic | user_files；空则表示不传（上游默认 general）。
	DeepResearchMode string `json:"deep_research_mode,omitempty"`
}

func GetAllChannels(startIdx int, num int, scope string) ([]*Channel, error) {
	channels, _, err := GetChannelsPage(ChannelPageOptions{
		Offset:       startIdx,
		Limit:        num,
		Scope:        scope,
		TypeFilter:   -1,
		StatusFilter: -1,
		SortBy:       "",
		SortOrder:    "",
		IdSort:       false,
	})
	return channels, err
}

// ChannelPageOptions 渠道列表筛选与排序选项。
type ChannelPageOptions struct {
	Offset       int
	Limit        int
	Scope        string // "all" | "disabled" | 其他为分页 Omit key
	TypeFilter   int    // -1 忽略；>=0 按类型过滤
	StatusFilter int    // -1 忽略；1=仅启用；0=非启用（含手动/自动禁用）
	SortBy       string // id,name,priority,balance,response_time,test_time
	SortOrder    string // asc|desc
	IdSort       bool   // true 且 SortBy 为空时按 id 降序（与 air 前端 id_sort 一致）
	// TenantIDEq 非 nil 时只返回该租户下的渠道。
	TenantIDEq *int
	// PlatformChannelsOnly 为 true 时只返回 tenant_id 为 NULL 的平台渠道（平台控制台列表）。
	PlatformChannelsOnly bool
}

var channelSortColumns = map[string]string{
	"id":            "id",
	"name":          "name",
	"priority":      "priority",
	"balance":       "balance",
	"response_time": "response_time",
	"test_time":     "test_time",
}

// GetChannelsPage 分页查询渠道；Scope=all 返回全部（含密钥）；Scope=disabled 仅禁用态。
func GetChannelsPage(opt ChannelPageOptions) ([]*Channel, int64, error) {
	var channels []*Channel
	var total int64
	switch opt.Scope {
	case "all":
		err := DB.Order("id desc").Find(&channels).Error
		return channels, int64(len(channels)), err
	case "disabled":
		tx := DB.Model(&Channel{}).Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled)
		_ = tx.Count(&total).Error
		err := tx.Order("id desc").Limit(opt.Limit).Offset(opt.Offset).Find(&channels).Error
		return channels, total, err
	default:
		q := DB.Model(&Channel{}).Omit("key")
		if opt.TenantIDEq != nil {
			q = q.Where("tenant_id = ?", *opt.TenantIDEq)
		} else if opt.PlatformChannelsOnly {
			q = q.Where("tenant_id IS NULL")
		}
		if opt.TypeFilter >= 0 {
			q = q.Where("type = ?", opt.TypeFilter)
		}
		if opt.StatusFilter == ChannelStatusEnabled {
			q = q.Where("status = ?", ChannelStatusEnabled)
		} else if opt.StatusFilter == 0 {
			q = q.Where("status != ?", ChannelStatusEnabled)
		}
		if err := q.Count(&total).Error; err != nil {
			return nil, 0, err
		}
		orderCol := "id"
		desc := true
		if col, ok := channelSortColumns[strings.ToLower(strings.TrimSpace(opt.SortBy))]; ok {
			orderCol = col
		}
		if opt.IdSort && strings.TrimSpace(opt.SortBy) == "" {
			orderCol = "id"
			desc = true
		}
		if strings.ToLower(strings.TrimSpace(opt.SortOrder)) == "asc" {
			desc = false
		}
		dir := "DESC"
		if !desc {
			dir = "ASC"
		}
		err := q.Order(orderCol + " " + dir).Limit(opt.Limit).Offset(opt.Offset).Find(&channels).Error
		return channels, total, err
	}
}

func SearchChannels(keyword string) (channels []*Channel, err error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []*Channel{}, nil
	}
	pattern := "%" + keyword + "%"
	idMatch := helper.String2Int(keyword)
	q := DB.Omit("key")
	if idMatch > 0 {
		err = q.Where("id = ? OR name LIKE ? OR COALESCE(remark,'') LIKE ? OR COALESCE(tag,'') LIKE ?", idMatch, pattern, pattern, pattern).Find(&channels).Error
	} else {
		err = q.Where("name LIKE ? OR COALESCE(remark,'') LIKE ? OR COALESCE(tag,'') LIKE ?", pattern, pattern, pattern).Find(&channels).Error
	}
	return channels, err
}

// SearchChannelsFiltered 支持按分组、模型关键字附加过滤（供 air 等前端使用）。
func SearchChannelsFiltered(keyword, groupFilter, modelFilter string) (channels []*Channel, err error) {
	keyword = strings.TrimSpace(keyword)
	groupFilter = strings.TrimSpace(groupFilter)
	modelFilter = strings.TrimSpace(modelFilter)
	if keyword == "" && groupFilter == "" && modelFilter == "" {
		return []*Channel{}, nil
	}
	q := DB.Omit("key")
	if keyword != "" {
		pattern := "%" + keyword + "%"
		idMatch := helper.String2Int(keyword)
		if idMatch > 0 {
			q = q.Where("id = ? OR name LIKE ? OR COALESCE(remark,'') LIKE ? OR COALESCE(tag,'') LIKE ?", idMatch, pattern, pattern, pattern)
		} else {
			q = q.Where("name LIKE ? OR COALESCE(remark,'') LIKE ? OR COALESCE(tag,'') LIKE ?", pattern, pattern, pattern)
		}
	}
	if groupFilter != "" {
		q = q.Where("group LIKE ?", "%"+groupFilter+"%")
	}
	if modelFilter != "" {
		q = q.Where("models LIKE ?", "%"+modelFilter+"%")
	}
	err = q.Order("id desc").Find(&channels).Error
	return channels, err
}

// SearchTenantChannels 在单个租户下按关键字 / 分组 / 模型过滤搜索渠道。
func SearchTenantChannels(tenantID int, keyword, groupFilter, modelFilter string) (channels []*Channel, err error) {
	if tenantID <= 0 {
		return nil, errors.New("tenant id 无效")
	}
	keyword = strings.TrimSpace(keyword)
	groupFilter = strings.TrimSpace(groupFilter)
	modelFilter = strings.TrimSpace(modelFilter)
	if keyword == "" && groupFilter == "" && modelFilter == "" {
		return []*Channel{}, nil
	}
	q := DB.Omit("key").Where("tenant_id = ?", tenantID)
	if keyword != "" {
		pattern := "%" + keyword + "%"
		idMatch := helper.String2Int(keyword)
		if idMatch > 0 {
			q = q.Where("id = ? OR name LIKE ? OR COALESCE(remark,'') LIKE ? OR COALESCE(tag,'') LIKE ?", idMatch, pattern, pattern, pattern)
		} else {
			q = q.Where("name LIKE ? OR COALESCE(remark,'') LIKE ? OR COALESCE(tag,'') LIKE ?", pattern, pattern, pattern)
		}
	}
	if groupFilter != "" {
		q = q.Where("group LIKE ?", "%"+groupFilter+"%")
	}
	if modelFilter != "" {
		q = q.Where("models LIKE ?", "%"+modelFilter+"%")
	}
	err = q.Order("id desc").Find(&channels).Error
	return channels, err
}

func GetChannelById(id int, selectAll bool) (*Channel, error) {
	channel := Channel{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(&channel, "id = ?", id).Error
	} else {
		err = DB.Omit("key").First(&channel, "id = ?", id).Error
	}
	return &channel, err
}

// GetChannelsByIds 按 ID 批量查询渠道（含密钥，供倍率同步等管理端功能使用）。
func GetChannelsByIds(ids []int) ([]*Channel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var channels []*Channel
	err := DB.Where("id IN ?", ids).Find(&channels).Error
	return channels, err
}

func BatchInsertChannels(channels []Channel) error {
	var err error
	err = DB.Create(&channels).Error
	if err != nil {
		return err
	}
	for _, channel_ := range channels {
		err = channel_.AddAbilities()
		if err != nil {
			return err
		}
	}
	return nil
}

func (channel *Channel) GetPriority() int64 {
	if channel.Priority == nil {
		return 0
	}
	return *channel.Priority
}

// GetWeight 返回渠道权重；nil 或 0 视为 1，用于加权路由。
func (channel *Channel) GetWeight() uint {
	if channel.Weight == nil || *channel.Weight == 0 {
		return 1
	}
	return *channel.Weight
}

func (channel *Channel) GetBaseURL() string {
	if channel.BaseURL == nil {
		return ""
	}
	return *channel.BaseURL
}

func (channel *Channel) GetModelMapping() map[string]string {
	if channel.ModelMapping == nil || *channel.ModelMapping == "" || *channel.ModelMapping == "{}" {
		return nil
	}
	modelMapping := make(map[string]string)
	err := json.Unmarshal([]byte(*channel.ModelMapping), &modelMapping)
	if err != nil {
		logger.SysError(fmt.Sprintf("failed to unmarshal model mapping for channel %d, error: %s", channel.Id, err.Error()))
		return nil
	}
	return modelMapping
}

func (channel *Channel) Insert() error {
	var err error
	err = DB.Create(channel).Error
	if err != nil {
		return err
	}
	err = channel.AddAbilities()
	return err
}

func (channel *Channel) Update() error {
	var err error
	err = DB.Model(channel).Updates(channel).Error
	if err != nil {
		return err
	}
	DB.Model(channel).First(channel, "id = ?", channel.Id)
	err = channel.UpdateAbilities()
	return err
}

func (channel *Channel) UpdateResponseTime(responseTime int64) {
	err := DB.Model(channel).Select("response_time", "test_time").Updates(Channel{
		TestTime:     helper.GetTimestamp(),
		ResponseTime: int(responseTime),
	}).Error
	if err != nil {
		logger.SysError("failed to update response time: " + err.Error())
	}
}

func (channel *Channel) UpdateBalance(balance float64) {
	err := DB.Model(channel).Select("balance_updated_time", "balance").Updates(Channel{
		BalanceUpdatedTime: helper.GetTimestamp(),
		Balance:            balance,
	}).Error
	if err != nil {
		logger.SysError("failed to update balance: " + err.Error())
	}
}

func (channel *Channel) Delete() error {
	var err error
	err = DB.Delete(channel).Error
	if err != nil {
		return err
	}
	_ = DeleteChannelModelTestResultsByChannel(channel.Id)
	err = channel.DeleteAbilities()
	return err
}

func (channel *Channel) LoadConfig() (ChannelConfig, error) {
	var cfg ChannelConfig
	if channel.Config == "" {
		return cfg, nil
	}
	err := json.Unmarshal([]byte(channel.Config), &cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

// AutoBanEnabled nil 或非 0 表示允许自动禁用渠道。
func (channel *Channel) AutoBanEnabled() bool {
	if channel.AutoBan == nil {
		return true
	}
	return *channel.AutoBan != 0
}

// ParamOverrideMap 解析 param_override JSON（简单 K-V / 嵌套对象合并用）。
func (channel *Channel) ParamOverrideMap() map[string]interface{} {
	if channel.ParamOverride == nil || strings.TrimSpace(*channel.ParamOverride) == "" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(*channel.ParamOverride), &m); err != nil {
		return nil
	}
	return m
}

// HeaderOverrideMap 解析 header_override（仅支持 string 值）。
func (channel *Channel) HeaderOverrideMap() map[string]string {
	if channel.HeaderOverride == nil || strings.TrimSpace(*channel.HeaderOverride) == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(*channel.HeaderOverride), &m); err != nil {
		return nil
	}
	return m
}

// RawStatusCodeMapping 返回非空的 status_code_mapping 原文。
func (channel *Channel) RawStatusCodeMapping() string {
	if channel.StatusCodeMapping == nil {
		return ""
	}
	return strings.TrimSpace(*channel.StatusCodeMapping)
}

// RoutingProviderTag Direction 分组标识（默认 default）。
func (channel *Channel) RoutingProviderTag() string {
	cfg, err := channel.LoadConfig()
	if err != nil {
		return "default"
	}
	s := strings.TrimSpace(cfg.RoutingProvider)
	if s == "" {
		return "default"
	}
	return s
}

// RoutingAdaptiveEffective 是否参与自适应权重（与 RoutingSkipAdaptive 相反）。
func (channel *Channel) RoutingAdaptiveEffective() bool {
	cfg, err := channel.LoadConfig()
	if err != nil {
		return true
	}
	return !cfg.RoutingSkipAdaptive
}

// PickAmapWebServiceKeyFromEnabledChannel 返回首个启用且密钥非空的高德渠道 Key；
// 供 POST /amap 高德转发在未配置 AMAP_KEY / AmapWebServiceSecret 时使用（与后台「高德」渠道密钥一致）。
func PickAmapWebServiceKeyFromEnabledChannel() string {
	var ch Channel
	err := DB.Where("type = ? AND status = ?", channeltype.AmapPOI, ChannelStatusEnabled).
		Order("id asc").
		First(&ch).Error
	if err != nil {
		return ""
	}
	k := strings.TrimSpace(ch.Key)
	if k == "" {
		return ""
	}
	return k
}

// ChannelAppliesToGroup 判断渠道配置的 group 列表（逗号分隔）是否包含指定分组。
func ChannelAppliesToGroup(ch *Channel, userGroup string) bool {
	userGroup = strings.TrimSpace(userGroup)
	if userGroup == "" || ch == nil {
		return false
	}
	for _, g := range strings.Split(ch.Group, ",") {
		if strings.TrimSpace(g) == userGroup {
			return true
		}
	}
	return false
}

// ValidateChannelIDsForTenantSubUser 校验渠道均已启用；每条须为本租户私有渠道，或与 subUserGroup 匹配的平台渠道（tenant_id 为空且 group 命中）。
func ValidateChannelIDsForTenantSubUser(tenantID int, subUserGroup string, channelIDs []int) error {
	if tenantID <= 0 || len(channelIDs) == 0 {
		return nil
	}
	subUserGroup = strings.TrimSpace(subUserGroup)
	if subUserGroup == "" {
		subUserGroup = "default"
	}
	uniq := make(map[int]struct{}, len(channelIDs))
	for _, id := range channelIDs {
		if id <= 0 {
			return errors.New("渠道 ID 无效")
		}
		if _, dup := uniq[id]; dup {
			continue
		}
		uniq[id] = struct{}{}
	}
	var channels []Channel
	err := DB.Where("id IN ? AND status = ?", channelIDs, ChannelStatusEnabled).Find(&channels).Error
	if err != nil {
		return err
	}
	if len(channels) != len(uniq) {
		return errors.New("渠道无效或未启用")
	}
	for i := range channels {
		ch := &channels[i]
		if ch.TenantID != nil && *ch.TenantID == tenantID {
			continue
		}
		if ch.TenantID == nil && ChannelAppliesToGroup(ch, subUserGroup) {
			continue
		}
		return fmt.Errorf("渠道 #%d 不可分配给该子账号（须为本租户私有或与当前分组匹配的平台渠道）", ch.Id)
	}
	return nil
}

// ListChannelsForTenantSubUserACL 子账号渠道白名单编辑：本租户启用渠道 + 平台启用渠道（channel.group 命中 subUserGroup）。
func ListChannelsForTenantSubUserACL(tenantID int, subUserGroup string) ([]*Channel, error) {
	if tenantID <= 0 {
		return nil, errors.New("租户 ID 无效")
	}
	subUserGroup = strings.TrimSpace(subUserGroup)
	if subUserGroup == "" {
		subUserGroup = "default"
	}
	var tenantCh []*Channel
	err := DB.Omit("key").Where("tenant_id = ? AND status = ?", tenantID, ChannelStatusEnabled).
		Order("id ASC").Find(&tenantCh).Error
	if err != nil {
		return nil, err
	}
	var platform []*Channel
	err = DB.Omit("key").Where("tenant_id IS NULL AND status = ?", ChannelStatusEnabled).
		Order("id ASC").Find(&platform).Error
	if err != nil {
		return nil, err
	}
	out := make([]*Channel, 0, len(tenantCh)+len(platform))
	out = append(out, tenantCh...)
	for _, ch := range platform {
		if ChannelAppliesToGroup(ch, subUserGroup) {
			out = append(out, ch)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return out, nil
}

func UpdateChannelStatusById(id int, status int) {
	err := UpdateAbilityStatus(id, status == ChannelStatusEnabled)
	if err != nil {
		logger.SysError("failed to update ability status: " + err.Error())
	}
	err = DB.Model(&Channel{}).Where("id = ?", id).Update("status", status).Error
	if err != nil {
		logger.SysError("failed to update channel status: " + err.Error())
	}
}

func UpdateChannelUsedQuota(id int, quota int64) {
	if config.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeChannelUsedQuota, id, quota)
		return
	}
	updateChannelUsedQuota(id, quota)
}

func updateChannelUsedQuota(id int, quota int64) {
	err := DB.Model(&Channel{}).Where("id = ?", id).Update("used_quota", gorm.Expr("used_quota + ?", quota)).Error
	if err != nil {
		logger.SysError("failed to update channel used quota: " + err.Error())
	}
}

func DeleteChannelByStatus(status int64) (int64, error) {
	result := DB.Where("status = ?", status).Delete(&Channel{})
	return result.RowsAffected, result.Error
}

func DeleteDisabledChannel() (int64, error) {
	result := DB.Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled).Delete(&Channel{})
	return result.RowsAffected, result.Error
}
