package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/service"
	"gorm.io/gorm"
)

// DownloadNacosSkillZipAdmin GET /api/nacos/skills/download?namespace=&name=&version=
func DownloadNacosSkillZipAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	name := strings.TrimSpace(c.Query("name"))
	ver := strings.TrimSpace(c.Query("version"))
	if name == "" || ver == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 与 version 必填"})
		return
	}
	zipData, err := service.NacosAIGetSkillZIPAdmin(ns, name, ver)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "skill 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\""+name+".zip\"")
	c.Data(http.StatusOK, "application/zip", zipData)
}

// DownloadNacosAgentSpecZipAdmin GET /api/nacos/agentspecs/download?namespace=&name=&version=
func DownloadNacosAgentSpecZipAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	name := strings.TrimSpace(c.Query("name"))
	ver := strings.TrimSpace(c.Query("version"))
	if name == "" || ver == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 与 version 必填"})
		return
	}
	zipData, err := service.NacosAIGetAgentSpecZIPAdmin(ns, name, ver)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "agentspec 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\""+name+".zip\"")
	c.Data(http.StatusOK, "application/zip", zipData)
}

type nacosLabelsAdminBody struct {
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	Replace   bool              `json:"replace"`
}

// UpdateNacosSkillLabelsAdmin POST /api/nacos/skills/labels
func UpdateNacosSkillLabelsAdmin(c *gin.Context) {
	var body nacosLabelsAdminBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := body.Namespace
	if strings.TrimSpace(ns) == "" {
		ns = "public"
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 必填"})
		return
	}
	if body.Labels == nil {
		body.Labels = map[string]string{}
	}
	if err := service.NacosAIUpdateArtifactLabels(ns, model.NacosAIKindSkill, name, body.Labels, body.Replace); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "skill 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateNacosAgentSpecLabelsAdmin POST /api/nacos/agentspecs/labels
func UpdateNacosAgentSpecLabelsAdmin(c *gin.Context) {
	var body nacosLabelsAdminBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := body.Namespace
	if strings.TrimSpace(ns) == "" {
		ns = "public"
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 必填"})
		return
	}
	if body.Labels == nil {
		body.Labels = map[string]string{}
	}
	if err := service.NacosAIUpdateArtifactLabels(ns, model.NacosAIKindAgentSpec, name, body.Labels, body.Replace); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "agentspec 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetNacosAgentSpecDetailAdmin GET /api/nacos/agentspecs/detail
func GetNacosAgentSpecDetailAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 必填"})
		return
	}
	data, err := service.NacosAIDescribeAgentSpec(ns, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "agentspec 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

type patchNacosAgentSpecBody struct {
	Namespace   string  `json:"namespace"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	BizTags     *string `json:"bizTags"`
	Enable      *bool   `json:"enable"`
	Scope       *string `json:"scope"`
}

// UpdateNacosAgentSpecMetadataAdmin PUT /api/nacos/agentspecs/metadata
func UpdateNacosAgentSpecMetadataAdmin(c *gin.Context) {
	var body patchNacosAgentSpecBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := body.Namespace
	if strings.TrimSpace(ns) == "" {
		ns = "public"
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 必填"})
		return
	}
	if err := service.NacosAIUpdateAgentSpecMetadata(ns, name, body.Description, body.BizTags, body.Enable, body.Scope); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "agentspec 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteNacosAgentSpecAdmin DELETE /api/nacos/agentspecs/item?name=&namespace=
func DeleteNacosAgentSpecAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 必填"})
		return
	}
	if err := service.NacosAIDeleteAgentSpec(ns, name); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "agentspec 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UploadNacosAgentSpecAdmin POST /api/nacos/agentspecs/upload?namespace=&overwrite=
func UploadNacosAgentSpecAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	overwrite := c.DefaultQuery("overwrite", "false") == "true"
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "缺少 multipart 字段 file"})
		return
	}
	if f.Size > config.NacosRegistryMaxUploadBytes {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "上传文件过大"})
		return
	}
	src, err := f.Open()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer src.Close()
	zipData, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if int64(len(zipData)) > config.NacosRegistryMaxUploadBytes {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "上传文件过大"})
		return
	}
	ownerID := 0
	if v, ok := c.Get("id"); ok {
		if ii, ok := v.(int); ok {
			ownerID = ii
		}
	}
	if err := service.NacosAIUploadAgentSpec(ns, zipData, ownerID, overwrite, f.Filename); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type nacosAgentSpecSubmitBody struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

// SubmitNacosAgentSpecAdmin POST /api/nacos/agentspecs/submit
func SubmitNacosAgentSpecAdmin(c *gin.Context) {
	var body nacosAgentSpecSubmitBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := body.Namespace
	if strings.TrimSpace(ns) == "" {
		ns = "public"
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 必填"})
		return
	}
	if err := service.NacosAISubmit(ns, model.NacosAIKindAgentSpec, name, strings.TrimSpace(body.Version)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "agentspec 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type nacosAgentSpecPublishBody struct {
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	UpdateLatest bool   `json:"updateLatest"`
	ForcePublish bool   `json:"forcePublish"`
}

// PublishNacosAgentSpecAdmin POST /api/nacos/agentspecs/publish
func PublishNacosAgentSpecAdmin(c *gin.Context) {
	var body nacosAgentSpecPublishBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := body.Namespace
	if strings.TrimSpace(ns) == "" {
		ns = "public"
	}
	name := strings.TrimSpace(body.Name)
	ver := strings.TrimSpace(body.Version)
	if name == "" || ver == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 与 version 必填"})
		return
	}
	if err := service.NacosAIPublish(ns, model.NacosAIKindAgentSpec, name, ver, body.UpdateLatest, body.ForcePublish); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "agentspec 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// --- MCP / A2A / Prompt / Pipeline 控制台 ---

func ListNacosMcpAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	data, err := service.NacosAIListMcp(ns, "", page, size)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func GetNacosMcpDetailAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	name := strings.TrimSpace(c.Query("serverName"))
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "serverName 必填"})
		return
	}
	data, err := service.NacosAIDescribeMcp(ns, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func UpsertNacosMcpAdmin(c *gin.Context) {
	var body nacosMcpUpsertBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := c.DefaultQuery("namespace", "public")
	lb, _ := json.Marshal(body.Labels)
	if err := service.NacosAIUpsertMcp(ns, body.ServerName, body.Description, string(body.Spec), body.BizTags, string(lb), body.Scope, body.Enable); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteNacosMcpAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	name := strings.TrimSpace(c.Query("serverName"))
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "serverName 必填"})
		return
	}
	if err := service.NacosAIDeleteMcp(ns, name); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ListNacosA2AAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	data, err := service.NacosAIListA2A(ns, "", page, size)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func GetNacosA2ADetailAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	name := strings.TrimSpace(c.Query("agentName"))
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "agentName 必填"})
		return
	}
	data, err := service.NacosAIDescribeA2A(ns, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func UpsertNacosA2AAdmin(c *gin.Context) {
	var body nacosA2AUpsertBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := c.DefaultQuery("namespace", "public")
	lb, _ := json.Marshal(body.Labels)
	if err := service.NacosAIUpsertA2A(ns, body.AgentName, body.Description, string(body.Card), body.BizTags, string(lb), body.Scope, body.Enable); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteNacosA2AAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	name := strings.TrimSpace(c.Query("agentName"))
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "agentName 必填"})
		return
	}
	if err := service.NacosAIDeleteA2A(ns, name); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ListNacosPromptsAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	data, err := service.NacosAIListPrompts(ns, "", page, size)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func GetNacosPromptDetailAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	key := strings.TrimSpace(c.Query("promptKey"))
	if key == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "promptKey 必填"})
		return
	}
	data, err := service.NacosAIDescribePrompt(ns, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

type nacosPromptLabelsAdminBody struct {
	Namespace string            `json:"namespace"`
	PromptKey string            `json:"promptKey"`
	Labels    map[string]string `json:"labels"`
	Replace   bool              `json:"replace"`
}

// UpdateNacosPromptLabelsAdmin POST /api/nacos/prompts/labels
func UpdateNacosPromptLabelsAdmin(c *gin.Context) {
	var body nacosPromptLabelsAdminBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := body.Namespace
	if strings.TrimSpace(ns) == "" {
		ns = "public"
	}
	key := strings.TrimSpace(body.PromptKey)
	if key == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "promptKey 必填"})
		return
	}
	if body.Labels == nil {
		body.Labels = map[string]string{}
	}
	if err := service.NacosAIUpdatePromptLabels(ns, key, body.Labels, body.Replace); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func UpsertNacosPromptHeaderAdmin(c *gin.Context) {
	var body nacosPromptHeaderBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := c.DefaultQuery("namespace", "public")
	lb, _ := json.Marshal(body.Labels)
	if err := service.NacosAIUpsertPromptHeader(ns, body.PromptKey, body.Description, body.BizTags, string(lb), body.Scope, body.Enable); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AddNacosPromptVersionAdmin(c *gin.Context) {
	var body nacosPromptVersionBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := c.DefaultQuery("namespace", "public")
	ver, err := service.NacosAIPromptAddVersion(ns, body.PromptKey, string(body.Content))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"version": ver}})
}

type nacosPromptSubmitPubBody struct {
	Namespace    string `json:"namespace"`
	PromptKey    string `json:"promptKey"`
	Version      string `json:"version"`
	UpdateLatest bool   `json:"updateLatest"`
	ForcePublish bool   `json:"forcePublish"`
}

func SubmitNacosPromptAdmin(c *gin.Context) {
	var body struct {
		Namespace string `json:"namespace"`
		PromptKey string `json:"promptKey"`
		Version   string `json:"version"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := body.Namespace
	if strings.TrimSpace(ns) == "" {
		ns = "public"
	}
	if err := service.NacosAIPromptSubmit(ns, body.PromptKey, body.Version); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func PublishNacosPromptAdmin(c *gin.Context) {
	var body nacosPromptSubmitPubBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := body.Namespace
	if strings.TrimSpace(ns) == "" {
		ns = "public"
	}
	if strings.TrimSpace(body.PromptKey) == "" || strings.TrimSpace(body.Version) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "promptKey 与 version 必填"})
		return
	}
	if err := service.NacosAIPromptPublish(ns, body.PromptKey, body.Version, body.UpdateLatest, body.ForcePublish); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteNacosPromptAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	key := strings.TrimSpace(c.Query("promptKey"))
	if key == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "promptKey 必填"})
		return
	}
	if err := service.NacosAIDeletePrompt(ns, key); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ListNacosPipelinesAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	data, err := service.NacosAIListPipelineRuns(ns, page, size)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func GetNacosPipelineDetailAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	id, err := strconv.ParseInt(strings.TrimSpace(c.Query("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效 id"})
		return
	}
	data, err := service.NacosAIDescribePipelineRun(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if data.NamespaceId != ns {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "namespace 不匹配"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func RunNacosPipelineScanAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	run, err := service.NacosAIRunRegistryScan(ns)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": run.Id, "status": run.Status, "jobType": run.JobType}})
}
