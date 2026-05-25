package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/model"
)

func truncCatalogErr(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func joinOpenAIModelsListURL(base string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	if b == "" {
		return "https://api.openai.com/v1/models"
	}
	if strings.HasSuffix(b, "/models") {
		return b
	}
	// Gemini 官方 OpenAI 兼容根地址 …/v1beta/openai
	if strings.HasSuffix(b, "/openai") {
		return b + "/models"
	}
	if strings.HasSuffix(b, "/v1") {
		return b + "/models"
	}
	return b + "/v1/models"
}

func joinAnthropicModelsListURL(base string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	if b == "" {
		b = "https://api.anthropic.com"
	}
	if strings.HasSuffix(b, "/models") {
		return b
	}
	if strings.HasSuffix(b, "/v1") {
		return b + "/models"
	}
	return b + "/v1/models"
}

// fetchOpenAIStyleModelList 解析 OpenAI 兼容 GET /v1/models（含 OpenRouter、new-api 代理）。
func fetchOpenAIStyleModelList(ctx context.Context, listURL, authBearer string, ownedByLabel, sourceTag string) ([]model.ModelCatalog, error) {
	authBearer = strings.TrimSpace(authBearer)
	if authBearer == "" {
		return nil, fmt.Errorf("密钥为空")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", bearerAuthorizationValue(authBearer))
	client := client.NewOutboundHTTPClient(120 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncCatalogErr(string(raw), 480))
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	var rows []model.ModelCatalog
	for _, it := range out.Data {
		id := strings.TrimSpace(it.ID)
		if id == "" {
			continue
		}
		rows = append(rows, model.ModelCatalog{
			ModelId: id,
			OwnedBy: ownedByLabel,
			Enabled: true,
			Source:  sourceTag,
		})
	}
	return rows, nil
}

func fetchAnthropicModelList(ctx context.Context, baseURL, apiKey string, opts anthropicModelListFetchOpts) ([]model.ModelCatalog, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("密钥为空")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinAnthropicModelsListURL(baseURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	if opts.useBearer {
		req.Header.Set("Authorization", bearerAuthorizationValue(apiKey))
	}
	client := client.NewOutboundHTTPClient(120 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncCatalogErr(string(raw), 480))
	}
	return parseAnthropicModelListPayload(raw)
}

type anthropicModelListFetchOpts struct {
	useBearer bool
}

func parseAnthropicModelListPayload(raw []byte) ([]model.ModelCatalog, error) {
	var out struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	var rows []model.ModelCatalog
	for _, it := range out.Data {
		id := strings.TrimSpace(it.ID)
		if id == "" {
			continue
		}
		ob := "anthropic"
		if strings.TrimSpace(it.DisplayName) != "" {
			ob = "anthropic · " + strings.TrimSpace(it.DisplayName)
		}
		rows = append(rows, model.ModelCatalog{
			ModelId: id,
			OwnedBy: ob,
			Enabled: true,
			Source:  "sync_anthropic",
		})
	}
	return rows, nil
}

func fetchGeminiModelList(ctx context.Context, baseURL, apiKey, apiVersion string, opts geminiModelListFetchOpts) ([]model.ModelCatalog, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("密钥为空")
	}
	queryKey := apiKey
	if !opts.useQueryKey {
		queryKey = ""
	}
	u := buildGeminiModelsListURL(baseURL, apiVersion, queryKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if opts.useGoogHeader {
		req.Header.Set("x-goog-api-key", apiKey)
	}
	if opts.useBearer {
		req.Header.Set("Authorization", bearerAuthorizationValue(apiKey))
	}
	client := client.NewOutboundHTTPClient(120 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncCatalogErr(string(raw), 480))
	}
	return parseGeminiModelListPayload(raw)
}

type geminiModelListFetchOpts struct {
	useQueryKey  bool
	useGoogHeader bool
	useBearer    bool
}

func bearerAuthorizationValue(apiKey string) string {
	token := strings.TrimSpace(apiKey)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token
	}
	return "Bearer " + token
}

func parseGeminiModelListPayload(raw []byte) ([]model.ModelCatalog, error) {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	arr, _ := root["models"].([]any)
	var rows []model.ModelCatalog
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		id := strings.TrimPrefix(name, "models/")
		if id == "" {
			continue
		}
		rows = append(rows, model.ModelCatalog{
			ModelId: id,
			OwnedBy: "google-gemini",
			Enabled: true,
			Source:  "sync_gemini",
		})
	}
	return rows, nil
}

// buildGeminiModelsListURL 构造 Generative Language ListModels 地址；baseURL 为空时使用 Google 官方根地址。
func buildGeminiModelsListURL(baseURL, apiVersion, apiKey string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "https://generativelanguage.googleapis.com"
	}
	version := strings.TrimSpace(apiVersion)
	if version == "" {
		version = "v1beta"
	}
	u := fmt.Sprintf("%s/%s/models", base, version)
	if k := strings.TrimSpace(apiKey); k != "" {
		u += "?key=" + url.QueryEscape(k)
	}
	return u
}

// modelsDevAPIURL 默认官方 api.json；base_url 可传镜像完整 URL 或仅 origin（自动补 /api.json）。
func modelsDevAPIURL(baseFromReq string) string {
	s := strings.TrimSpace(baseFromReq)
	if s == "" {
		return "https://models.dev/api.json"
	}
	s = strings.TrimRight(s, "/")
	if strings.HasSuffix(strings.ToLower(s), "/api.json") {
		return s
	}
	return s + "/api.json"
}

// fetchModelsDevCatalog 拉取 https://models.dev/api.json（匿名，无需密钥）。
// 文档：https://models.dev — 结构为 { [providerKey]: { id, name, models: { [key]: { id, name, ... } } } }
func fetchModelsDevCatalog(ctx context.Context, listURL string) ([]model.ModelCatalog, error) {
	return fetchProviderTreeCatalog(ctx, listURL, "sync_models_dev", "models.dev")
}

// fetchBasellmCatalog 拉取 BaseLLM LLM Metadata（结构与 models.dev 相同）。
func fetchBasellmCatalog(ctx context.Context, listURL string) ([]model.ModelCatalog, error) {
	return fetchProviderTreeCatalog(ctx, listURL, "basellm", "basellm")
}

func fetchProviderTreeCatalog(ctx context.Context, listURL, sourceTag, catalogLabel string) ([]model.ModelCatalog, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	client := client.NewOutboundHTTPClient(180 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncCatalogErr(string(raw), 480))
	}
	return rowsFromProviderTreePayload(raw, sourceTag, catalogLabel)
}

func rowsFromProviderTreePayload(raw []byte, sourceTag, catalogLabel string) ([]model.ModelCatalog, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	type devModel struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Family      string `json:"family"`
		Attachment  bool   `json:"attachment"`
		Reasoning   bool   `json:"reasoning"`
		ToolCall    bool   `json:"tool_call"`
		Temperature bool   `json:"temperature"`
		Knowledge   string `json:"knowledge"`
		ReleaseDate string `json:"release_date"`
		LastUpdated string `json:"last_updated"`
		Modalities  struct {
			Input  []string `json:"input"`
			Output []string `json:"output"`
		} `json:"modalities"`
		OpenWeights bool `json:"open_weights"`
		Limit       struct {
			Context int `json:"context"`
			Output  int `json:"output"`
		} `json:"limit"`
		Cost struct {
			Input      float64 `json:"input"`
			Output     float64 `json:"output"`
			CacheRead  float64 `json:"cache_read"`
			CacheWrite float64 `json:"cache_write"`
		} `json:"cost"`
	}
	type devProv struct {
		ID     string                     `json:"id"`
		Name   string                     `json:"name"`
		Npm    string                     `json:"npm"`
		API    string                     `json:"api"`
		Doc    string                     `json:"doc"`
		Models map[string]json.RawMessage `json:"models"`
	}

	provKeys := make([]string, 0, len(root))
	for k := range root {
		provKeys = append(provKeys, k)
	}
	sort.Strings(provKeys)

	// 同一 model id 在多 provider 下可能出现：按 provider 键名字典序保留首次
	byModelID := make(map[string]model.ModelCatalog)

	for _, pkey := range provKeys {
		var sp devProv
		if err := json.Unmarshal(root[pkey], &sp); err != nil {
			continue
		}
		if len(sp.Models) == 0 {
			continue
		}
		provLabel := strings.TrimSpace(sp.Name)
		if provLabel == "" {
			provLabel = strings.TrimSpace(sp.ID)
		}
		if provLabel == "" {
			provLabel = pkey
		}
		provID := strings.TrimSpace(sp.ID)
		if provID == "" {
			provID = pkey
		}
		mkeys := make([]string, 0, len(sp.Models))
		for mk := range sp.Models {
			mkeys = append(mkeys, mk)
		}
		sort.Strings(mkeys)
		for _, mk := range mkeys {
			var m devModel
			_ = json.Unmarshal(sp.Models[mk], &m)
			id := strings.TrimSpace(m.ID)
			if id == "" {
				id = strings.TrimSpace(mk)
			}
			if id == "" {
				continue
			}
			if _, dup := byModelID[id]; dup {
				continue
			}
			displayName := strings.TrimSpace(m.Name)
			if displayName == "" {
				displayName = id
			}
			ob := catalogLabel + " · " + provLabel + " · " + displayName
			inMod := strings.Join(m.Modalities.Input, ", ")
			outMod := strings.Join(m.Modalities.Output, ", ")
			provDisplay := provLabel
			if provDisplay == "" {
				provDisplay = provID
			}
			byModelID[id] = model.ModelCatalog{
				ModelId:         id,
				OwnedBy:         ob,
				Enabled:         true,
				Source:          sourceTag,
				ModelName:       displayName,
				ProviderKey:     pkey,
				ProviderDisplay: provDisplay,
				Family:          strings.TrimSpace(m.Family),
				NpmPackage:      strings.TrimSpace(sp.Npm),
				APIBase:         strings.TrimSpace(sp.API),
				DocURL:          strings.TrimSpace(sp.Doc),
				ModalitiesIn:    inMod,
				ModalitiesOut:   outMod,
				ContextLimit:    m.Limit.Context,
				OutputLimit:     m.Limit.Output,
				CostInput:       m.Cost.Input,
				CostOutput:      m.Cost.Output,
				CostCacheRead:   m.Cost.CacheRead,
				CostCacheWrite:  m.Cost.CacheWrite,
				Reasoning:       m.Reasoning,
				ToolCall:        m.ToolCall,
				TemperatureOK:   m.Temperature,
				AttachmentOK:    m.Attachment,
				OpenWeights:     m.OpenWeights,
				KnowledgeCutoff: strings.TrimSpace(m.Knowledge),
				ReleaseDate:     strings.TrimSpace(m.ReleaseDate),
				LastUpdatedDev:  strings.TrimSpace(m.LastUpdated),
			}
		}
	}

	ids := make([]string, 0, len(byModelID))
	for id := range byModelID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]model.ModelCatalog, 0, len(ids))
	for _, id := range ids {
		out = append(out, byModelID[id])
	}
	return out, nil
}

type modelCatalogSyncRequest struct {
	Source       string `json:"source"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	ChannelID    int    `json:"channel_id"`
	NewAPIUserID int    `json:"new_api_user_id"`
}

func buildRowsForModelCatalogSync(ctx context.Context, req *modelCatalogSyncRequest) ([]model.ModelCatalog, error) {
	if req == nil {
		return nil, fmt.Errorf("empty request")
	}
	switch strings.ToLower(strings.TrimSpace(req.Source)) {
	case "openai_compatible", "openai", "compatible":
		key := strings.TrimSpace(req.APIKey)
		if key == "" {
			return nil, fmt.Errorf("需要 api_key")
		}
		u := joinOpenAIModelsListURL(req.BaseURL)
		return fetchOpenAIStyleModelList(ctx, u, key, "openai-compatible", "sync_openai_compatible")
	case "openrouter":
		key := strings.TrimSpace(req.APIKey)
		if key == "" {
			return nil, fmt.Errorf("需要 api_key")
		}
		return fetchOpenAIStyleModelList(ctx, "https://openrouter.ai/api/v1/models", key, "openrouter", "sync_openrouter")
	case "anthropic", "claude":
		key := strings.TrimSpace(req.APIKey)
		if key == "" {
			return nil, fmt.Errorf("需要 api_key")
		}
		return fetchAnthropicModelList(ctx, req.BaseURL, key, anthropicModelListFetchOpts{})
	case "gemini", "google":
		key := strings.TrimSpace(req.APIKey)
		if key == "" {
			return nil, fmt.Errorf("需要 api_key")
		}
		return fetchGeminiModelList(ctx, "", key, "v1beta", geminiModelListFetchOpts{
			useQueryKey: true, useGoogHeader: true,
		})
	case "models_dev", "models.dev", "modelsdev":
		u := modelsDevAPIURL(req.BaseURL)
		return fetchModelsDevCatalog(ctx, u)
	case "basellm", "base_llm", "llm-metadata", "llm_metadata":
		u := basellmAPIURL(req.BaseURL)
		return fetchBasellmCatalog(ctx, u)
	case "aliapi":
		return syncFromAliapi(ctx, req)
	case "anyfast":
		return syncFromAnyfast(ctx, req)
	case "new_api", "newapi", "new_api_models":
		return syncFromNewAPIModels(ctx, req)
	case "channel":
		if req.ChannelID <= 0 {
			return nil, fmt.Errorf("需要 channel_id")
		}
		ch, err := model.GetChannelById(req.ChannelID, true)
		if err != nil {
			return nil, err
		}
		key := strings.TrimSpace(ch.Key)
		if key == "" {
			return nil, fmt.Errorf("渠道密钥为空")
		}
		u := joinOpenAIModelsListURL(ch.GetBaseURL())
		label := strings.TrimSpace(ch.Name)
		if label == "" {
			label = fmt.Sprintf("channel #%d", ch.Id)
		}
		srcTag := fmt.Sprintf("sync_channel_%d", ch.Id)
		return fetchOpenAIStyleModelList(ctx, u, key, label, srcTag)
	default:
		return nil, fmt.Errorf("未知 source: %s", req.Source)
	}
}

// PostModelCatalogSync POST /api/model_catalog/sync
func PostModelCatalogSync(c *gin.Context) {
	var req modelCatalogSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	rows, err := buildRowsForModelCatalogSync(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if len(rows) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "上游未返回任何模型"})
		return
	}
	added, updated, err := model.BatchUpsertModelCatalogForSync(rows)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"total_upstream": len(rows),
			"added":          added,
			"updated":        updated,
		},
	})
}

// FetchUpstreamOpenAIStyleModelIDs GET OpenAI 兼容 /v1/models，仅返回模型 id（供渠道「从上游拉取」）。
func FetchUpstreamOpenAIStyleModelIDs(ctx context.Context, listURL, bearer string) ([]string, error) {
	rows, err := fetchOpenAIStyleModelList(ctx, listURL, bearer, "upstream", "channel_fetch_models")
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		if s := strings.TrimSpace(r.ModelId); s != "" {
			ids = append(ids, s)
		}
	}
	return ids, nil
}

// FetchUpstreamAnthropicModelIDs 拉取 Anthropic Messages 模型 id；baseURL 为空时使用 https://api.anthropic.com。
func FetchUpstreamAnthropicModelIDs(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	rows, err := fetchAnthropicModelList(ctx, baseURL, apiKey, anthropicModelListFetchOpts{})
	if err != nil {
		return nil, err
	}
	return modelCatalogRowsToIDs(rows), nil
}

// FetchUpstreamAnthropicCompatibleProxyModelIDs 第三方 Anthropic 兼容代理（new-api 等）：ListModels 同时带 x-api-key 与 Bearer，失败时回退 OpenAI /v1/models。
func FetchUpstreamAnthropicCompatibleProxyModelIDs(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	rows, err := fetchAnthropicModelList(ctx, baseURL, apiKey, anthropicModelListFetchOpts{useBearer: true})
	if err == nil && len(rows) > 0 {
		return modelCatalogRowsToIDs(rows), nil
	}
	if strings.TrimSpace(baseURL) != "" {
		if ids, err2 := FetchUpstreamOpenAIStyleModelIDs(ctx, joinOpenAIModelsListURL(baseURL), apiKey); err2 == nil && len(ids) > 0 {
			return ids, nil
		}
	}
	if err != nil {
		return nil, err
	}
	return modelCatalogRowsToIDs(rows), nil
}

// FetchUpstreamGeminiModelIDs 拉取 Generative Language 模型 id；baseURL 为空时使用 Google 官方根地址。
func FetchUpstreamGeminiModelIDs(ctx context.Context, baseURL, apiKey, apiVersion string) ([]string, error) {
	rows, err := fetchGeminiModelList(ctx, baseURL, apiKey, apiVersion, geminiModelListFetchOpts{
		useQueryKey: true, useGoogHeader: true,
	})
	if err != nil {
		return nil, err
	}
	return modelCatalogRowsToIDs(rows), nil
}

// FetchUpstreamGeminiNativeProxyModelIDs 第三方 Gemini 原生代理（如 Anyfast/new-api）：ListModels 需 Bearer，且可能忽略 ?key=。
func FetchUpstreamGeminiNativeProxyModelIDs(ctx context.Context, baseURL, apiKey, apiVersion string) ([]string, error) {
	opts := geminiModelListFetchOpts{useGoogHeader: true, useBearer: true}
	rows, err := fetchGeminiModelList(ctx, baseURL, apiKey, apiVersion, opts)
	if err == nil && len(rows) > 0 {
		return modelCatalogRowsToIDs(rows), nil
	}
	if strings.TrimSpace(baseURL) != "" {
		if ids, err2 := FetchUpstreamOpenAIStyleModelIDs(ctx, joinOpenAIModelsListURL(baseURL), apiKey); err2 == nil && len(ids) > 0 {
			return ids, nil
		}
	}
	if err != nil {
		return nil, err
	}
	return modelCatalogRowsToIDs(rows), nil
}

func modelCatalogRowsToIDs(rows []model.ModelCatalog) []string {
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		if s := strings.TrimSpace(r.ModelId); s != "" {
			ids = append(ids, s)
		}
	}
	return ids
}
