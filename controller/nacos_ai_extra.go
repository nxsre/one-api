package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/service"
	"gorm.io/gorm"
)

// --- MCP v3 ---

func NacosMcpList(c *gin.Context) {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	name := c.Query("serverName")
	data, err := service.NacosAIListMcp(nacosNamespace(c), name, pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, data)
}

func NacosMcpDescribe(c *gin.Context) {
	name := strings.TrimSpace(c.Query("serverName"))
	if name == "" {
		nacosV3Err(c, 400, "serverName 必填")
		return
	}
	data, err := service.NacosAIDescribeMcp(nacosNamespace(c), name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "MCP 不存在")
			return
		}
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, data)
}

type nacosMcpUpsertBody struct {
	ServerName  string          `json:"serverName"`
	Description string          `json:"description"`
	Spec        json.RawMessage `json:"spec"`
	BizTags     string          `json:"bizTags"`
	Labels      map[string]string `json:"labels"`
	Scope       string          `json:"scope"`
	Enable      *bool           `json:"enable"`
}

func NacosMcpUpsert(c *gin.Context) {
	var body nacosMcpUpsertBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	lb, _ := json.Marshal(body.Labels)
	if err := service.NacosAIUpsertMcp(nacosNamespace(c), body.ServerName, body.Description, string(body.Spec), body.BizTags, string(lb), body.Scope, body.Enable); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosMcpDelete(c *gin.Context) {
	name := strings.TrimSpace(c.Query("serverName"))
	if name == "" {
		nacosV3Err(c, 400, "serverName 必填")
		return
	}
	if err := service.NacosAIDeleteMcp(nacosNamespace(c), name); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosMcpClientGet(c *gin.Context) {
	name := strings.TrimSpace(c.Query("serverName"))
	if name == "" {
		nacosV3Err(c, 400, "serverName 必填")
		return
	}
	data, err := service.NacosAIMcpClientGet(nacosNamespace(c), name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "MCP 不存在")
			return
		}
		nacosV3Err(c, 400, err.Error())
		return
	}
	var v any
	_ = json.Unmarshal(data, &v)
	nacosV3OK(c, v)
}

// --- A2A v3 ---

func NacosA2AList(c *gin.Context) {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	name := c.Query("agentName")
	data, err := service.NacosAIListA2A(nacosNamespace(c), name, pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, data)
}

func NacosA2ADescribe(c *gin.Context) {
	name := strings.TrimSpace(c.Query("agentName"))
	if name == "" {
		nacosV3Err(c, 400, "agentName 必填")
		return
	}
	data, err := service.NacosAIDescribeA2A(nacosNamespace(c), name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "Agent 不存在")
			return
		}
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, data)
}

type nacosA2AUpsertBody struct {
	AgentName   string          `json:"agentName"`
	Description string          `json:"description"`
	Card        json.RawMessage `json:"card"`
	BizTags     string          `json:"bizTags"`
	Labels      map[string]string `json:"labels"`
	Scope       string          `json:"scope"`
	Enable      *bool           `json:"enable"`
}

func NacosA2AUpsert(c *gin.Context) {
	var body nacosA2AUpsertBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	lb, _ := json.Marshal(body.Labels)
	if err := service.NacosAIUpsertA2A(nacosNamespace(c), body.AgentName, body.Description, string(body.Card), body.BizTags, string(lb), body.Scope, body.Enable); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosA2ADelete(c *gin.Context) {
	name := strings.TrimSpace(c.Query("agentName"))
	if name == "" {
		nacosV3Err(c, 400, "agentName 必填")
		return
	}
	if err := service.NacosAIDeleteA2A(nacosNamespace(c), name); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosA2AClientGet(c *gin.Context) {
	name := strings.TrimSpace(c.Query("agentName"))
	if name == "" {
		nacosV3Err(c, 400, "agentName 必填")
		return
	}
	data, err := service.NacosAIA2AClientGet(nacosNamespace(c), name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "Agent 不存在")
			return
		}
		nacosV3Err(c, 400, err.Error())
		return
	}
	var v any
	_ = json.Unmarshal(data, &v)
	nacosV3OK(c, v)
}

// --- Prompt v3 ---

func NacosPromptList(c *gin.Context) {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	key := c.Query("promptKey")
	data, err := service.NacosAIListPrompts(nacosNamespace(c), key, pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, data)
}

func NacosPromptDescribe(c *gin.Context) {
	key := strings.TrimSpace(c.Query("promptKey"))
	if key == "" {
		nacosV3Err(c, 400, "promptKey 必填")
		return
	}
	data, err := service.NacosAIDescribePrompt(nacosNamespace(c), key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "Prompt 不存在")
			return
		}
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, data)
}

type nacosPromptLabelsBody struct {
	PromptKey string            `json:"promptKey"`
	Labels      map[string]string `json:"labels"`
	Replace     bool              `json:"replace"`
}

// NacosPromptLabelsUpdate POST .../prompt/labels
func NacosPromptLabelsUpdate(c *gin.Context) {
	var body nacosPromptLabelsBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	if strings.TrimSpace(body.PromptKey) == "" {
		nacosV3Err(c, 400, "promptKey 必填")
		return
	}
	if body.Labels == nil {
		body.Labels = map[string]string{}
	}
	if err := service.NacosAIUpdatePromptLabels(nacosNamespace(c), body.PromptKey, body.Labels, body.Replace); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "Prompt 不存在")
			return
		}
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

type nacosPromptHeaderBody struct {
	PromptKey   string            `json:"promptKey"`
	Description string            `json:"description"`
	BizTags     string            `json:"bizTags"`
	Labels      map[string]string `json:"labels"`
	Scope       string            `json:"scope"`
	Enable      *bool             `json:"enable"`
}

func NacosPromptUpsertHeader(c *gin.Context) {
	var body nacosPromptHeaderBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	lb, _ := json.Marshal(body.Labels)
	if err := service.NacosAIUpsertPromptHeader(nacosNamespace(c), body.PromptKey, body.Description, body.BizTags, string(lb), body.Scope, body.Enable); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

type nacosPromptVersionBody struct {
	PromptKey   string          `json:"promptKey"`
	Content     json.RawMessage `json:"content"`
}

func NacosPromptAddVersion(c *gin.Context) {
	var body nacosPromptVersionBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	ver, err := service.NacosAIPromptAddVersion(nacosNamespace(c), body.PromptKey, string(body.Content))
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, gin.H{"version": ver})
}

func NacosPromptSubmit(c *gin.Context) {
	key := strings.TrimSpace(c.Query("promptKey"))
	if key == "" {
		nacosV3Err(c, 400, "promptKey 必填")
		return
	}
	ver := c.Query("version")
	if err := service.NacosAIPromptSubmit(nacosNamespace(c), key, ver); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosPromptPublish(c *gin.Context) {
	key := strings.TrimSpace(c.Query("promptKey"))
	ver := strings.TrimSpace(c.Query("version"))
	if key == "" || ver == "" {
		nacosV3Err(c, 400, "promptKey 与 version 必填")
		return
	}
	updateLatest := c.DefaultQuery("updateLatestLabel", "true") == "true"
	force := c.DefaultQuery("forcePublish", "false") == "true"
	if err := service.NacosAIPromptPublish(nacosNamespace(c), key, ver, updateLatest, force); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosPromptDelete(c *gin.Context) {
	key := strings.TrimSpace(c.Query("promptKey"))
	if key == "" {
		nacosV3Err(c, 400, "promptKey 必填")
		return
	}
	if err := service.NacosAIDeletePrompt(nacosNamespace(c), key); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosPromptClientGet(c *gin.Context) {
	key := strings.TrimSpace(c.Query("promptKey"))
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "promptKey 必填", "data": nil})
		return
	}
	label := c.Query("label")
	version := c.Query("version")
	data, err := service.NacosAIPromptGetContent(nacosNamespace(c), key, label, version)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	var v any
	_ = json.Unmarshal(data, &v)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": v})
}

// --- Pipeline v3 ---

func NacosPipelineList(c *gin.Context) {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	data, err := service.NacosAIListPipelineRuns(nacosNamespace(c), pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, data)
}

func NacosPipelineDescribe(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Query("id")), 10, 64)
	if err != nil || id <= 0 {
		nacosV3Err(c, 400, "无效 id")
		return
	}
	data, err := service.NacosAIDescribePipelineRun(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "记录不存在")
			return
		}
		nacosV3Err(c, 500, err.Error())
		return
	}
	ns := nacosNamespace(c)
	if data.NamespaceId != ns {
		nacosV3Err(c, 403, "namespace 不匹配")
		return
	}
	nacosV3OK(c, data)
}

func NacosPipelineRunScan(c *gin.Context) {
	run, err := service.NacosAIRunRegistryScan(nacosNamespace(c))
	if err != nil {
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, gin.H{"id": run.Id, "status": run.Status, "jobType": run.JobType})
}
