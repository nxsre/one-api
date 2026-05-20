package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/model"
)

var errChannelModelTestJobRunning = errors.New("channel model test job already running")

type channelModelTestJob struct {
	ScopeKey     string
	ChannelID    int
	BaseURL      string
	Total        int
	Completed    int
	Concurrency  int
	CurrentModel string
	Running      bool
	Results      []channelModelTestResult
	LastError    string
	StartedAt    int64
	FinishedAt   int64
	mu           sync.RWMutex
}

type channelModelTestJobManager struct {
	mu   sync.Mutex
	jobs map[string]*channelModelTestJob
}

var channelModelTestJobs channelModelTestJobManager

func init() {
	channelModelTestJobs.jobs = make(map[string]*channelModelTestJob)
}

func (m *channelModelTestJobManager) tryStart(scopeKey string, total, concurrency, channelID int, baseURL string) (*channelModelTestJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.jobs[scopeKey]; ok && existing.Running {
		return existing, errChannelModelTestJobRunning
	}
	job := &channelModelTestJob{
		ScopeKey:    scopeKey,
		ChannelID:   channelID,
		BaseURL:     baseURL,
		Total:       total,
		Concurrency: concurrency,
		Running:     true,
		StartedAt:   time.Now().Unix(),
		Results:     make([]channelModelTestResult, 0, total),
	}
	m.jobs[scopeKey] = job
	return job, nil
}

func (m *channelModelTestJobManager) get(scopeKey string) *channelModelTestJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.jobs[scopeKey]
}

func (m *channelModelTestJobManager) finish(scopeKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job, ok := m.jobs[scopeKey]; ok {
		job.mu.Lock()
		job.Running = false
		job.FinishedAt = time.Now().Unix()
		job.CurrentModel = ""
		job.mu.Unlock()
	}
}

func (j *channelModelTestJob) snapshot() map[string]interface{} {
	j.mu.RLock()
	defer j.mu.RUnlock()
	results := make([]channelModelTestResult, len(j.Results))
	copy(results, j.Results)
	return map[string]interface{}{
		"scope_key":     j.ScopeKey,
		"channel_id":    j.ChannelID,
		"base_url":      j.BaseURL,
		"total":         j.Total,
		"completed":     j.Completed,
		"concurrency":   j.Concurrency,
		"current_model": j.CurrentModel,
		"running":       j.Running,
		"results":       results,
		"last_error":    j.LastError,
		"started_at":    j.StartedAt,
		"finished_at":   j.FinishedAt,
	}
}

func (j *channelModelTestJob) setCurrentModels(names []string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.CurrentModel = strings.Join(names, ", ")
}

func (j *channelModelTestJob) addResult(row channelModelTestResult) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Results = append(j.Results, row)
	j.Completed = len(j.Results)
}

func (j *channelModelTestJob) setError(msg string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.LastError = msg
}

func normalizeModelTestConcurrency(n int) int {
	if n <= 0 {
		return 3
	}
	if n > 10 {
		return 10
	}
	return n
}

func runChannelModelTestJob(ch *model.Channel, req testChannelModelsPreviewReq, job *channelModelTestJob, models []string) {
	defer channelModelTestJobs.finish(job.ScopeKey)

	ctx := context.Background()
	testBaseURL := model.NormalizeChannelTestBaseURL(req.BaseURL, req.Type)
	ch.Models = strings.Join(models, ",")
	if mm := strings.TrimSpace(req.ModelMapping); mm != "" && mm != "{}" {
		ch.ModelMapping = &mm
	}

	concurrency := normalizeModelTestConcurrency(req.Concurrency)
	if concurrency > len(models) {
		concurrency = len(models)
	}

	workCh := make(chan string, len(models))
	for _, modelName := range models {
		workCh <- modelName
	}
	close(workCh)

	var activeMu sync.Mutex
	active := make(map[string]struct{}, concurrency)
	refreshActive := func() {
		names := make([]string, 0, len(active))
		for name := range active {
			names = append(names, name)
		}
		sort.Strings(names)
		job.setCurrentModels(names)
	}

	testOne := func(modelName string) {
		tik := time.Now()
		msg, testErr, _ := testChannelByModel(ctx, ch, modelName)
		elapsedMs := time.Since(tik).Milliseconds()
		row := channelModelTestResult{
			Model:     modelName,
			Time:      float64(elapsedMs) / 1000.0,
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
			row.TestedAt = time.Now().Unix()
			_ = model.UpsertChannelModelTestResult(
				req.ChannelId,
				testBaseURL,
				modelName,
				row.Success,
				row.Message,
				elapsedMs,
			)
		}
		job.addResult(row)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for modelName := range workCh {
				activeMu.Lock()
				active[modelName] = struct{}{}
				activeMu.Unlock()
				refreshActive()
				testOne(modelName)
				activeMu.Lock()
				delete(active, modelName)
				activeMu.Unlock()
				refreshActive()
			}
		}()
	}
	wg.Wait()
}

func channelModelTestScopeFromReq(channelID int, baseURL string, channelType int) (scopeKey string, normalizedBase string) {
	normalizedBase = model.NormalizeChannelTestBaseURL(baseURL, channelType)
	return model.ChannelTestScopeKey(channelID, normalizedBase), normalizedBase
}

func channelModelTestJobStatusPayload(scopeKey string) (map[string]interface{}, bool) {
	job := channelModelTestJobs.get(scopeKey)
	if job == nil {
		return nil, false
	}
	return job.snapshot(), true
}

func channelModelTestJobBusyMessage(scopeKey string) string {
	job := channelModelTestJobs.get(scopeKey)
	if job == nil || !job.Running {
		return ""
	}
	job.mu.RLock()
	defer job.mu.RUnlock()
	return fmt.Sprintf("已有测试任务运行中（%d/%d）", job.Completed, job.Total)
}

func channelModelTestVerifySaved(c *gin.Context) (func(*model.Channel) error, error) {
	verifySaved := func(*model.Channel) error { return nil }
	if strings.Contains(c.Request.URL.Path, "/tenant_console/") {
		tid, err := GetTenantConsoleTenantID(c, "manage_channels")
		if err != nil {
			return nil, err
		}
		verifySaved = func(ch *model.Channel) error {
			if ch.TenantID == nil || *ch.TenantID != tid {
				return fmt.Errorf("无权访问该渠道")
			}
			return nil
		}
	}
	return verifySaved, nil
}

// StartChannelModelTestJob POST /api/channel/test_models/jobs
func StartChannelModelTestJob(c *gin.Context) {
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
	scopeKey, testBaseURL := channelModelTestScopeFromReq(req.ChannelId, req.BaseURL, req.Type)
	concurrency := normalizeModelTestConcurrency(req.Concurrency)
	job, err := channelModelTestJobs.tryStart(scopeKey, len(models), concurrency, req.ChannelId, testBaseURL)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": channelModelTestJobBusyMessage(scopeKey),
			"data":    job.snapshot(),
		})
		return
	}
	go runChannelModelTestJob(ch, req, job, models)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    job.snapshot(),
	})
}

// GetChannelModelTestJobStatus GET /api/channel/test_models/jobs/status
func GetChannelModelTestJobStatus(c *gin.Context) {
	channelID, _ := strconv.Atoi(strings.TrimSpace(c.Query("channel_id")))
	channelType, _ := strconv.Atoi(strings.TrimSpace(c.Query("channel_type")))
	baseURL := strings.TrimSpace(c.Query("base_url"))
	scopeKey, normalizedBase := channelModelTestScopeFromReq(channelID, baseURL, channelType)
	payload, ok := channelModelTestJobStatusPayload(scopeKey)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"scope_key":     scopeKey,
				"channel_id":    channelID,
				"base_url":      normalizedBase,
				"total":         0,
				"completed":     0,
				"current_model": "",
				"running":       false,
				"results":       []channelModelTestResult{},
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": payload})
}

func TenantStartChannelModelTestJob(c *gin.Context) {
	StartChannelModelTestJob(c)
}

func TenantGetChannelModelTestJobStatus(c *gin.Context) {
	GetChannelModelTestJobStatus(c)
}
