package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/model"
)

type testChannelModelsPreviewReq struct {
	previewFetchUpstreamModelsReq
	ChannelId    int      `json:"channel_id"`
	Models       []string `json:"models"`
	ModelMapping string   `json:"model_mapping"`
	Concurrency  int      `json:"concurrency"`
}

type channelModelTestResult struct {
	Model     string  `json:"model"`
	Success   bool    `json:"success"`
	Message   string  `json:"message"`
	Time      float64 `json:"time"`
	TestedAt  int64   `json:"tested_at,omitempty"`
	ElapsedMs int64   `json:"elapsed_ms,omitempty"`
}

// PreviewTestChannelModels POST /api/channel/test_models_preview
// 使用编辑页表单配置逐个测试模型（按模型类型选择 chat/embeddings/images 等协议）；支持单模型或批量。
func PreviewTestChannelModels(c *gin.Context) {
	var req testChannelModelsPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求: " + err.Error()})
		return
	}
	verifySaved, err := channelModelTestVerifySaved(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ch, err := BuildChannelForModelTest(req, verifySaved)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	models := normalizePreviewTestModels(req.Models)
	if len(models) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "模型列表为空"})
		return
	}
	ch.Models = strings.Join(models, ",")
	if mm := strings.TrimSpace(req.ModelMapping); mm != "" && mm != "{}" {
		ch.ModelMapping = &mm
	}

	testBaseURL := model.NormalizeChannelTestBaseURL(req.BaseURL, req.Type)
	ctx := c.Request.Context()
	results := make([]channelModelTestResult, 0, len(models))
	for _, modelName := range models {
		tik := time.Now()
		msg, testErr, _ := testChannelByModel(ctx, ch, modelName)
		elapsed := time.Since(tik)
		elapsedMs := elapsed.Milliseconds()
		row := channelModelTestResult{
			Model:     modelName,
			Time:      elapsed.Seconds(),
			ElapsedMs: elapsedMs,
		}
		if testErr != nil {
			row.Success = false
			row.Message = testErr.Error()
		} else {
			row.Success = true
			row.Message = msg
		}
		if req.ChannelId > 0 {
			testedAt := time.Now().Unix()
			row.TestedAt = testedAt
			_ = model.UpsertChannelModelTestResult(
				req.ChannelId,
				testBaseURL,
				modelName,
				row.Success,
				row.Message,
				elapsedMs,
			)
		}
		results = append(results, row)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"results":  results,
			"base_url": testBaseURL,
		},
	})
}

func normalizePreviewTestModels(models []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(models))
	for _, raw := range models {
		s := strings.TrimSpace(raw)
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

// GetChannelModelTestResults GET /api/channel/:id/model_test_results?base_url=
func GetChannelModelTestResults(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的渠道 id"})
		return
	}
	ch, err := model.GetChannelById(id, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	channelType := ch.Type
	if t, err := strconv.Atoi(strings.TrimSpace(c.Query("channel_type"))); err == nil && t > 0 {
		channelType = t
	}
	baseURL := model.NormalizeChannelTestBaseURL(c.Query("base_url"), channelType)
	rows, err := model.ListChannelModelTestResults(id, baseURL)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	out := make([]channelModelTestResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, channelModelTestResult{
			Model:     row.ModelId,
			Success:   row.Success,
			Message:   row.Message,
			Time:      float64(row.ElapsedMs) / 1000.0,
			TestedAt:  row.TestedAt,
			ElapsedMs: row.ElapsedMs,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"base_url": baseURL,
			"results":  out,
		},
	})
}

// TenantGetChannelModelTestResults GET /api/tenant_console/meta/channel/:id/model_test_results
func TenantGetChannelModelTestResults(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_channels")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的渠道 id"})
		return
	}
	ch, err := model.GetChannelById(id, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if ch.TenantID == nil || *ch.TenantID != tid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权访问该渠道"})
		return
	}
	GetChannelModelTestResults(c)
}
