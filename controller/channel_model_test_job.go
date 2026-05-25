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
	Paused       bool
	CancelReq    bool
	Interrupted  bool
	Results      []channelModelTestResult
	LastError    string
	StartedAt    int64
	FinishedAt   int64
	mu           sync.RWMutex
	cond         *sync.Cond
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
	job.cond = sync.NewCond(&job.mu)
	m.jobs[scopeKey] = job
	return job, nil
}

func (m *channelModelTestJobManager) get(scopeKey string) *channelModelTestJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.jobs[scopeKey]
}

func (m *channelModelTestJobManager) finish(scopeKey string, interrupted bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job, ok := m.jobs[scopeKey]; ok {
		job.mu.Lock()
		job.Running = false
		job.Paused = false
		job.Interrupted = interrupted || job.CancelReq
		job.FinishedAt = time.Now().Unix()
		job.CurrentModel = ""
		if job.cond != nil {
			job.cond.Broadcast()
		}
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
		"paused":        j.Paused,
		"cancel_req":    j.CancelReq,
		"interrupted":   j.Interrupted,
		"state":         j.stateLocked(),
		"results":       results,
		"last_error":    j.LastError,
		"started_at":    j.StartedAt,
		"finished_at":   j.FinishedAt,
	}
}

func (j *channelModelTestJob) stateLocked() string {
	if j.Running {
		if j.CancelReq {
			return "cancelling"
		}
		if j.Paused {
			return "paused"
		}
		return "running"
	}
	if j.Interrupted || j.CancelReq {
		return "cancelled"
	}
	if j.FinishedAt > 0 {
		return "finished"
	}
	return "idle"
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

func (j *channelModelTestJob) pause() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.Running || j.CancelReq || j.Paused {
		return false
	}
	j.Paused = true
	return true
}

func (j *channelModelTestJob) resume() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.Running || j.CancelReq || !j.Paused {
		return false
	}
	j.Paused = false
	if j.cond != nil {
		j.cond.Broadcast()
	}
	return true
}

func (j *channelModelTestJob) cancel() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.Running || j.CancelReq {
		return false
	}
	j.CancelReq = true
	j.Paused = false
	j.CurrentModel = ""
	if j.cond != nil {
		j.cond.Broadcast()
	}
	return true
}

func (j *channelModelTestJob) waitForDispatch() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	for j.Running && j.Paused && !j.CancelReq {
		if j.cond != nil {
			j.cond.Wait()
		} else {
			break
		}
	}
	return j.Running && !j.CancelReq
}

func (j *channelModelTestJob) isInterrupted() bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.CancelReq
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
	defer channelModelTestJobs.finish(job.ScopeKey, job.isInterrupted())

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
		row, elapsedMs := runSingleChannelModelTest(ctx, ch, modelName)
		persistChannelModelTestResult(req.ChannelId, testBaseURL, &row, elapsedMs)
		job.addResult(row)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if !job.waitForDispatch() {
					return
				}
				modelName, ok := <-workCh
				if !ok {
					return
				}
				if !job.waitForDispatch() {
					return
				}
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

func controlChannelModelTestJob(scopeKey, action string) (map[string]interface{}, bool, error) {
	job := channelModelTestJobs.get(scopeKey)
	if job == nil {
		return nil, false, errors.New("测试任务不存在")
	}
	action = strings.TrimSpace(strings.ToLower(action))
	changed := false
	switch action {
	case "pause":
		changed = job.pause()
	case "resume":
		changed = job.resume()
	case "cancel":
		changed = job.cancel()
	default:
		return nil, false, errors.New("无效的任务操作")
	}
	return job.snapshot(), changed, nil
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
				"paused":        false,
				"cancel_req":    false,
				"interrupted":   false,
				"state":         "idle",
				"results":       []channelModelTestResult{},
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": payload})
}

type channelModelTestJobControlReq struct {
	ChannelID   int    `json:"channel_id"`
	BaseURL     string `json:"base_url"`
	ChannelType int    `json:"channel_type"`
	Action      string `json:"action"`
}

// ControlChannelModelTestJob POST /api/channel/test_models/jobs/control
func ControlChannelModelTestJob(c *gin.Context) {
	var req channelModelTestJobControlReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求: " + err.Error()})
		return
	}
	verifySaved, err := channelModelTestVerifySaved(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if req.ChannelID > 0 {
		ch, getErr := model.GetChannelById(req.ChannelID, true)
		if getErr != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": getErr.Error()})
			return
		}
		if verifyErr := verifySaved(ch); verifyErr != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": verifyErr.Error()})
			return
		}
	}
	scopeKey, _ := channelModelTestScopeFromReq(req.ChannelID, req.BaseURL, req.ChannelType)
	payload, changed, err := controlChannelModelTestJob(scopeKey, req.Action)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	msg := "操作已提交"
	if !changed {
		msg = "任务状态未变化"
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": msg, "data": payload})
}

func TenantStartChannelModelTestJob(c *gin.Context) {
	StartChannelModelTestJob(c)
}

func TenantGetChannelModelTestJobStatus(c *gin.Context) {
	GetChannelModelTestJobStatus(c)
}

func TenantControlChannelModelTestJob(c *gin.Context) {
	ControlChannelModelTestJob(c)
}
