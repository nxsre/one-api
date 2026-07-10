package model

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common"
)

// TokenModelAllowEntry is one token allowlist entry. ChannelID > 0 means route only via that channel.
type TokenModelAllowEntry struct {
	ChannelID int
	Model     string
	Raw       string
}

// ParseTokenModelAllowEntry parses "model" or "#<channelId>:<model>".
func ParseTokenModelAllowEntry(s string) TokenModelAllowEntry {
	s = strings.TrimSpace(s)
	if s == "" {
		return TokenModelAllowEntry{}
	}
	if strings.HasPrefix(s, "#") {
		rest := strings.TrimPrefix(s, "#")
		if i := strings.IndexByte(rest, ':'); i > 0 {
			cidStr := strings.TrimSpace(rest[:i])
			model := strings.TrimSpace(rest[i+1:])
			if cid, err := strconv.Atoi(cidStr); err == nil && cid > 0 && model != "" {
				return TokenModelAllowEntry{ChannelID: cid, Model: model, Raw: s}
			}
		}
	}
	return TokenModelAllowEntry{Model: s, Raw: s}
}

// ParseTokenModelAllowlist splits a comma-separated allowlist into entries.
func ParseTokenModelAllowlist(allowlistCSV string) []TokenModelAllowEntry {
	parts := strings.Split(allowlistCSV, ",")
	out := make([]TokenModelAllowEntry, 0, len(parts))
	for _, part := range parts {
		ent := ParseTokenModelAllowEntry(part)
		if ent.Model != "" {
			out = append(out, ent)
		}
	}
	return out
}

// TokenAllowlistRoutingChannelIDs returns channel IDs allowed for requestModel when the token
// allowlist uses channel-scoped entries only. nil means no extra restriction from the token.
func TokenAllowlistRoutingChannelIDs(allowlistCSV, requestModel string, mappings []map[string]string) map[int]struct{} {
	req := strings.TrimSpace(requestModel)
	if req == "" || strings.TrimSpace(allowlistCSV) == "" {
		return nil
	}
	entries := ParseTokenModelAllowlist(allowlistCSV)
	allowed := make(map[int]struct{})
	anyChannel := false
	matched := false
	for _, ent := range entries {
		if !AllowanceMatchRequestModel(req, ent.Model, mappings) {
			continue
		}
		matched = true
		if ent.ChannelID > 0 {
			allowed[ent.ChannelID] = struct{}{}
		} else {
			anyChannel = true
		}
	}
	if !matched || anyChannel {
		return nil
	}
	return allowed
}

// IntersectChannelIDSets intersects two channel ID sets. nil on either side means "no restriction".
func IntersectChannelIDSets(a, b map[int]struct{}) map[int]struct{} {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	out := make(map[int]struct{})
	for id := range a {
		if _, ok := b[id]; ok {
			out[id] = struct{}{}
		}
	}
	return out
}

// ClientFacingModelName 返回客户端应使用的模型名：若 channel 的 model_mapping 将某请求名映射到 channelModel（上游/渠道模型名），则返回该请求名；否则返回 channelModel。
func ClientFacingModelName(channelModel string, mapping map[string]string) string {
	channelModel = strings.TrimSpace(channelModel)
	if channelModel == "" || len(mapping) == 0 {
		return channelModel
	}
	for req, upstream := range mapping {
		if strings.TrimSpace(upstream) == channelModel {
			if k := strings.TrimSpace(req); k != "" {
				return k
			}
		}
	}
	return channelModel
}

// AllowanceMatchRequestModel 判断 requestModel 是否被 allowedEntry 授权。
// 规则：请求模型名与 allowed 精确相等；或存在 model_mapping 且 mapping[requestModel]==allowed（allowed 为上游名）。
func AllowanceMatchRequestModel(requestModel, allowedEntry string, mappings []map[string]string) bool {
	req := strings.TrimSpace(requestModel)
	allow := strings.TrimSpace(allowedEntry)
	if req == "" || allow == "" {
		return false
	}
	if req == allow {
		return true
	}
	for _, m := range mappings {
		if m == nil {
			continue
		}
		if strings.TrimSpace(m[req]) == allow {
			return true
		}
	}
	return false
}

// IsRequestModelInAllowlist 检查 requestModel 是否在逗号分隔的 allowlist 中（结合用户分组下各渠道的 model_mapping）。
func IsRequestModelInAllowlist(ctx context.Context, userID int, requestModel, allowlistCSV string) bool {
	if strings.TrimSpace(requestModel) == "" || strings.TrimSpace(allowlistCSV) == "" {
		return false
	}
	mappings, _ := cacheGetRelayModelMappings(ctx, userID)
	for _, ent := range ParseTokenModelAllowlist(allowlistCSV) {
		if AllowanceMatchRequestModel(requestModel, ent.Model, mappings) {
			return true
		}
	}
	return false
}

// CollectGroupModelMappingsForUser 返回用户分组下各渠道的 model_mapping 列表。
func CollectGroupModelMappingsForUser(ctx context.Context, userID int) ([]map[string]string, error) {
	return cacheGetRelayModelMappings(ctx, userID)
}

// IsRequestModelInAllowlistSlice 与 IsRequestModelInAllowlist 相同，allowlist 为切片。
func IsRequestModelInAllowlistSlice(ctx context.Context, userID int, requestModel string, allowlist []string) bool {
	if strings.TrimSpace(requestModel) == "" || len(allowlist) == 0 {
		return false
	}
	mappings, _ := cacheGetRelayModelMappings(ctx, userID)
	for _, part := range allowlist {
		ent := ParseTokenModelAllowEntry(part)
		if AllowanceMatchRequestModel(requestModel, ent.Model, mappings) {
			return true
		}
	}
	return false
}

var (
	relayModelMappingsCache   = make(map[string]relayModelMappingsEntry)
	relayModelMappingsCacheMu sync.RWMutex
)

type relayModelMappingsEntry struct {
	mappings  []map[string]string
	expiresAt time.Time
}

func relayModelMappingsCacheKey(group string, userTenantID int) string {
	return fmt.Sprintf("%s|%d", strings.TrimSpace(group), userTenantID)
}

func cacheGetRelayModelMappings(_ context.Context, userID int) ([]map[string]string, error) {
	group, err := CacheGetUserGroup(userID)
	if err != nil {
		return nil, err
	}
	group = strings.TrimSpace(group)
	if group == "" {
		group = "default"
	}
	tenantID := GetUserTenantIDNumeric(userID)
	key := relayModelMappingsCacheKey(group, tenantID)

	relayModelMappingsCacheMu.RLock()
	if ent, ok := relayModelMappingsCache[key]; ok && time.Now().Before(ent.expiresAt) {
		m := ent.mappings
		relayModelMappingsCacheMu.RUnlock()
		return m, nil
	}
	relayModelMappingsCacheMu.RUnlock()

	mappings, err := collectGroupModelMappings(group, tenantID)
	if err != nil {
		return nil, err
	}

	ttl := time.Duration(GroupModelsCacheSeconds) * time.Second
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	relayModelMappingsCacheMu.Lock()
	relayModelMappingsCache[key] = relayModelMappingsEntry{mappings: mappings, expiresAt: time.Now().Add(ttl)}
	relayModelMappingsCacheMu.Unlock()
	return mappings, nil
}

// collectGroupModelMappings 汇总用户分组下启用渠道的 model_mapping（请求名 -> 上游名）。
func collectGroupModelMappings(group string, userTenantID int) ([]map[string]string, error) {
	channels, err := queryEnabledChannelsWithMappingInGroup(group, userTenantID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]string, 0, len(channels))
	for _, ch := range channels {
		if m := ch.GetModelMapping(); len(m) > 0 {
			out = append(out, m)
		}
	}
	return out, nil
}

func queryEnabledChannelsWithMappingInGroup(group string, userTenantID int) ([]*Channel, error) {
	groupCol := "abilities.`group`"
	trueVal := "1"
	if common.UsingPostgreSQL {
		groupCol = `abilities."group"`
		trueVal = "true"
	}
	var channels []*Channel
	tx := DB.Table("channels").
		Select("DISTINCT channels.*").
		Joins("INNER JOIN abilities ON abilities.channel_id = channels.id").
		Where(groupCol+" = ? AND abilities.enabled = "+trueVal+" AND channels.status = ?", group, ChannelStatusEnabled).
		Where("channels.model_mapping IS NOT NULL AND channels.model_mapping <> '' AND channels.model_mapping <> '{}'")
	if userTenantID == 0 {
		tx = tx.Where("channels.tenant_id IS NULL")
	} else {
		tx = tx.Where("(channels.tenant_id IS NULL OR channels.tenant_id = ?)", userTenantID)
	}
	err := tx.Find(&channels).Error
	return channels, err
}

// ApplyClientFacingModelNames 将明细行中的模型名改为客户端请求侧名称（有重定向时）。
func ApplyClientFacingModelNames(rows []AvailableModelChannelRow) ([]AvailableModelChannelRow, error) {
	if len(rows) == 0 {
		return rows, nil
	}
	ids := make([]int, 0, len(rows))
	seen := make(map[int]struct{})
	for _, r := range rows {
		if r.ChannelID <= 0 {
			continue
		}
		if _, ok := seen[r.ChannelID]; ok {
			continue
		}
		seen[r.ChannelID] = struct{}{}
		ids = append(ids, r.ChannelID)
	}
	if len(ids) == 0 {
		return rows, nil
	}
	var channels []*Channel
	if err := DB.Where("id IN ?", ids).Find(&channels).Error; err != nil {
		return nil, err
	}
	byID := make(map[int]*Channel, len(channels))
	for _, ch := range channels {
		byID[ch.Id] = ch
	}
	out := make([]AvailableModelChannelRow, len(rows))
	for i, r := range rows {
		out[i] = r
		if ch := byID[r.ChannelID]; ch != nil {
			out[i].Model = ClientFacingModelName(r.Model, ch.GetModelMapping())
		}
	}
	return out, nil
}
