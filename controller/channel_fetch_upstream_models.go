package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

type previewFetchUpstreamModelsReq struct {
	Type    int    `json:"type"`
	BaseURL string `json:"base_url"`
	Key     string `json:"key"`
	Config  string `json:"config"` // 可选，JSON 文本（与渠道 config 一致，用于 AK/SK 组合密钥等）
}

// PreviewFetchUpstreamChannelModels POST /api/channel/fetch_upstream_models_preview
// 使用表单中的 Base URL + 密钥请求上游模型列表（新建或未保存修改时使用）；管理员接口。
func PreviewFetchUpstreamChannelModels(c *gin.Context) {
	var req previewFetchUpstreamModelsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求: " + err.Error()})
		return
	}
	key := strings.TrimSpace(req.Key)
	cfgJSON := strings.TrimSpace(req.Config)
	var cfg model.ChannelConfig
	if cfgJSON != "" {
		_ = json.Unmarshal([]byte(cfgJSON), &cfg)
	}
	if key == "" && cfg.AK != "" && cfg.SK != "" && cfg.Region != "" {
		key = cfg.AK + "|" + cfg.SK + "|" + cfg.Region
	}
	if key == "" && cfg.Region != "" && cfg.VertexAIProjectID != "" && cfg.VertexAIADC != "" {
		key = cfg.Region + "|" + cfg.VertexAIProjectID + "|" + cfg.VertexAIADC
	}

	baseTrim := strings.TrimSpace(req.BaseURL)
	baseTrim = strings.TrimRight(baseTrim, "/")
	var basePtr *string
	if baseTrim != "" {
		basePtr = &baseTrim
	}

	ch := &model.Channel{
		Type:    req.Type,
		Key:     key,
		BaseURL: basePtr,
		Config:  cfgJSON,
	}
	apiKey := channelFirstAPIKey(ch)
	if apiKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "密钥为空"})
		return
	}

	ids, err := fetchUpstreamModelIDsForChannel(c.Request.Context(), ch, apiKey)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取上游模型列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": ids})
}

// FetchUpstreamChannelModels GET /api/channel/fetch_models/:id
// 按渠道类型请求上游列出模型（与 new-api 行为对齐：OpenAI 兼容 /v1/models、Anthropic、Gemini、阿里 compatible-mode、Ollama /api/tags）。
func FetchUpstreamChannelModels(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的渠道 id"})
		return
	}
	ch, err := model.GetChannelById(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	key := channelFirstAPIKey(ch)
	if key == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "渠道密钥为空"})
		return
	}
	ids, err := fetchUpstreamModelIDsForChannel(c.Request.Context(), ch, key)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取上游模型列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": ids})
}

func channelFirstAPIKey(ch *model.Channel) string {
	key := strings.TrimSpace(ch.Key)
	if key == "" {
		return ""
	}
	if idx := strings.IndexAny(key, "\n\r"); idx >= 0 {
		key = strings.TrimSpace(key[:idx])
	}
	return key
}

func resolveChannelListBase(ch *model.Channel) string {
	base := strings.TrimSpace(ch.GetBaseURL())
	if base != "" {
		return strings.TrimRight(base, "/")
	}
	if ch.Type > 0 && ch.Type < channeltype.Dummy {
		return strings.TrimRight(strings.TrimSpace(channeltype.DefaultBaseURL(ch.Type)), "/")
	}
	return ""
}

func fetchUpstreamModelIDsForChannel(ctx context.Context, ch *model.Channel, key string) ([]string, error) {
	base := resolveChannelListBase(ch)

	switch ch.Type {
	case channeltype.Anthropic:
		return FetchUpstreamAnthropicModelIDs(ctx, key)
	case channeltype.Gemini:
		return FetchUpstreamGeminiModelIDs(ctx, key)
	case channeltype.Ali:
		if base == "" {
			return nil, fmt.Errorf("无法解析 Base URL（请在渠道中填写或依赖内置默认）")
		}
		u := base + "/compatible-mode/v1/models"
		return FetchUpstreamOpenAIStyleModelIDs(ctx, u, key)
	case channeltype.Ollama:
		return fetchOllamaUpstreamModelIDs(base, key)
	default:
		u := joinOpenAIModelsListURL(base)
		return FetchUpstreamOpenAIStyleModelIDs(ctx, u, key)
	}
}

func fetchOllamaUpstreamModelIDs(base, key string) ([]string, error) {
	if base == "" {
		return nil, fmt.Errorf("Ollama Base URL 为空")
	}
	url := strings.TrimRight(base, "/") + "/api/tags"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(key) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(key))
	}
	client := &http.Client{Timeout: 60 * time.Second}
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
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse ollama tags: %w", err)
	}
	ids := make([]string, 0, len(out.Models))
	for _, m := range out.Models {
		if n := strings.TrimSpace(m.Name); n != "" {
			ids = append(ids, n)
		}
	}
	return normalizeUpstreamModelIDs(ids), nil
}

func normalizeUpstreamModelIDs(ids []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		s := strings.TrimSpace(id)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
