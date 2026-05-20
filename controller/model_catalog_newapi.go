package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/model"
)

const (
	newAPIModelsDefaultBaseURL = "https://www.anyfast.ai"
	newAPIModelsPath           = "/api/models"
	newAPIModelsSource         = "new_api_models"
)

func newAPIModelsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = newAPIModelsDefaultBaseURL
	}
	if strings.HasSuffix(strings.ToLower(base), "/api/models") {
		return base
	}
	return base + newAPIModelsPath
}

func newAPIAuthorizationHeader(accessToken string) string {
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return ""
	}
	lower := strings.ToLower(token)
	if strings.HasPrefix(lower, "bearer ") {
		return token
	}
	return token
}

func syncFromNewAPIModels(ctx context.Context, req *modelCatalogSyncRequest) ([]model.ModelCatalog, error) {
	accessToken := strings.TrimSpace(req.APIKey)
	if accessToken == "" {
		return nil, fmt.Errorf("需要 access token（控制台个人设置中生成，非 sk- 密钥）")
	}
	if req.NewAPIUserID <= 0 {
		return nil, fmt.Errorf("需要 new_api_user_id（New-Api-User，正整数）")
	}

	fullURL := newAPIModelsURL(req.BaseURL)
	if err := common.ValidateOutboundURL(fullURL); err != nil {
		return nil, fmt.Errorf("URL 未通过安全校验: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", newAPIAuthorizationHeader(accessToken))
	httpReq.Header.Set("New-Api-User", strconv.Itoa(req.NewAPIUserID))
	httpReq.Header.Set("User-Agent", "one-api/new-api-models-sync")
	httpReq.Header.Set("Accept", "application/json")

	hc := client.NewOutboundHTTPClient(120 * time.Second)
	resp, err := hc.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncCatalogErr(string(body), 480))
	}

	return rowsFromNewAPIModelsPayload(body)
}

func rowsFromNewAPIModelsPayload(body []byte) ([]model.ModelCatalog, error) {
	var payload struct {
		Success bool `json:"success"`
		Message string `json:"message"`
		Data    any    `json:"data"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	if !payload.Success {
		msg := strings.TrimSpace(payload.Message)
		if msg == "" {
			msg = "upstream returned success=false"
		}
		return nil, fmt.Errorf("%s", msg)
	}

	byID := make(map[string]*model.ModelCatalog)
	channelTypesByModel := make(map[string]map[string]struct{})

	addModel := func(modelID, channelType string) {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			return
		}
		if channelTypesByModel[modelID] == nil {
			channelTypesByModel[modelID] = make(map[string]struct{})
		}
		if channelType != "" {
			channelTypesByModel[modelID][channelType] = struct{}{}
		}
		if _, ok := byID[modelID]; ok {
			return
		}
		ownedBy := "new-api"
		if channelType != "" {
			ownedBy = fmt.Sprintf("new-api · type %s", channelType)
		}
		byID[modelID] = &model.ModelCatalog{
			ModelId:         modelID,
			ModelName:       modelID,
			OwnedBy:         ownedBy,
			ProviderKey:     channelType,
			ProviderDisplay: ownedBy,
			Enabled:         true,
			Source:          newAPIModelsSource,
		}
	}

	switch data := payload.Data.(type) {
	case map[string]any:
		keys := make([]string, 0, len(data))
		for k := range data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, chType := range keys {
			for _, modelID := range extractNewAPIModelNames(data[chType]) {
				addModel(modelID, chType)
			}
		}
	case []any:
		for _, item := range data {
			for _, modelID := range extractNewAPIModelNames(item) {
				addModel(modelID, "")
			}
		}
	default:
		return nil, fmt.Errorf("unrecognized /api/models data format")
	}

	if len(byID) == 0 {
		return nil, fmt.Errorf("no models parsed from /api/models response")
	}

	for modelID, row := range byID {
		types := channelTypesByModel[modelID]
		if len(types) <= 1 {
			continue
		}
		typeList := make([]string, 0, len(types))
		for t := range types {
			typeList = append(typeList, t)
		}
		sort.Strings(typeList)
		row.Tags = "channel_types:" + strings.Join(typeList, ",")
		row.Notes = fmt.Sprintf("出现在 %d 个渠道类型: %s", len(typeList), strings.Join(typeList, ", "))
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	rows := make([]model.ModelCatalog, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, *byID[id])
	}
	return rows, nil
}

func extractNewAPIModelNames(raw any) []string {
	switch v := raw.(type) {
	case string:
		if s := strings.TrimSpace(v); s != "" {
			return []string{s}
		}
	case []any:
		var out []string
		for _, item := range v {
			out = append(out, extractNewAPIModelNames(item)...)
		}
		return out
	case []string:
		var out []string
		for _, s := range v {
			if t := strings.TrimSpace(s); t != "" {
				out = append(out, t)
			}
		}
		return out
	case map[string]any:
		if name, ok := v["model_name"].(string); ok && strings.TrimSpace(name) != "" {
			return []string{strings.TrimSpace(name)}
		}
		if id, ok := v["id"].(string); ok && strings.TrimSpace(id) != "" {
			return []string{strings.TrimSpace(id)}
		}
	}
	return nil
}
