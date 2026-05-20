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

func nacosV3OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data})
}

// nacosV3Err 返回与 nacosV3OK 相同的 JSON 形状，但使用非 2xx HTTP 状态码。
// 嵌入的 console-ui（jQuery）在业务失败时若仍用 HTTP 200，多数回调只读 data、不读 code/message，用户看不到提示；
// 非 2xx 会进入 request() 里 $.ajax 的 fail 链并 toastError(responseJSON.message)。
func nacosV3Err(c *gin.Context, code int, msg string) {
	status := http.StatusBadRequest
	switch code {
	case 10001:
		status = http.StatusUnauthorized
	case 40300:
		status = http.StatusForbidden
	case 20004, 22001:
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"code": code, "message": msg, "data": nil})
}

// nacosParam 读取 URL query 或 POST 表单字段（含 application/x-www-form-urlencoded 与 multipart 中的文本字段）。
// console-ui-next 的 axios 将 JSON 式 body 序列化为 urlencoded，不会出现在 Query 里。
func nacosParam(c *gin.Context, key string) string {
	if v := strings.TrimSpace(c.Query(key)); v != "" {
		return v
	}
	return strings.TrimSpace(c.PostForm(key))
}

func nacosDefaultParam(c *gin.Context, key, def string) string {
	if v := nacosParam(c, key); v != "" {
		return v
	}
	return def
}

// nacosPublishForceFlag 是否按强制发版处理。console-ui-next 的 force-publish 请求通常不带 forcePublish 字段，此时应对齐官方行为默认为 true。
func nacosPublishForceFlag(c *gin.Context) bool {
	explicit := nacosParam(c, "forcePublish")
	if strings.HasSuffix(c.Request.URL.Path, "/force-publish") {
		if explicit == "" {
			return true
		}
		return explicit == "true" || explicit == "1"
	}
	return explicit == "true" || explicit == "1"
}

// nacosNamespaceOK 解析并校验命名空间（租户隔离）；失败时已写入 v3 风格错误响应。
func nacosNamespaceOK(c *gin.Context, fallback string) (string, bool) {
	return nacosResolveNamespaceV3(c, fallback)
}

// --- Skills admin ---

func NacosSkillList(c *gin.Context) {
	filter, pageNo, pageSize := parseNacosSkillListFilter(c)
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	data, err := service.NacosAIListSkills(ns, filter, pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	nacosV3OK(c, data)
}

func NacosSkillDescribe(c *gin.Context) {
	skillName := strings.TrimSpace(c.Query("skillName"))
	if skillName == "" {
		nacosV3Err(c, 10000, "skillName 必填")
		return
	}
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	data, err := service.NacosAIDescribeSkill(ns, skillName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "skill 不存在")
			return
		}
		nacosV3Err(c, 30000, err.Error())
		return
	}
	nacosV3OK(c, data)
}

// NacosSkillDelete DELETE …/skills?namespaceId=&skillName= 删除 Skill 及全部版本（与官方 console-ui-next 一致）。
func NacosSkillDelete(c *gin.Context) {
	skillName := strings.TrimSpace(nacosParam(c, "skillName"))
	if skillName == "" {
		nacosV3Err(c, 10000, "skillName 必填")
		return
	}
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	if err := service.NacosAIDeleteSkill(ns, skillName); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "skill 不存在")
			return
		}
		nacosV3Err(c, 30000, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosSkillUpload(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		nacosV3Err(c, 10000, "缺少 multipart 字段 file")
		return
	}
	if f.Size > config.NacosRegistryMaxUploadBytes {
		nacosV3Err(c, 20002, "上传文件过大")
		return
	}
	src, err := f.Open()
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	defer src.Close()
	zipData, err := io.ReadAll(src)
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	if int64(len(zipData)) > config.NacosRegistryMaxUploadBytes {
		nacosV3Err(c, 20002, "上传文件过大")
		return
	}
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	if err := service.NacosAIUploadSkill(ns, zipData, 0, f.Filename); err != nil {
		nacosV3Err(c, 20002, err.Error())
		return
	}
	c.Status(http.StatusOK)
}

func NacosSkillSubmit(c *gin.Context) {
	skillName := strings.TrimSpace(nacosParam(c, "skillName"))
	if skillName == "" {
		nacosV3Err(c, 10000, "skillName 必填")
		return
	}
	version := strings.TrimSpace(nacosParam(c, "version"))
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	if err := service.NacosAISubmit(ns, model.NacosAIKindSkill, skillName, version); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "skill 不存在")
			return
		}
		nacosV3Err(c, 20002, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosSkillPublish(c *gin.Context) {
	skillName := strings.TrimSpace(nacosParam(c, "skillName"))
	version := strings.TrimSpace(nacosParam(c, "version"))
	if skillName == "" || version == "" {
		nacosV3Err(c, 10000, "skillName 与 version 必填")
		return
	}
	updateLatest := nacosDefaultParam(c, "updateLatestLabel", "true") == "true"
	force := nacosPublishForceFlag(c)
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	if err := service.NacosAIPublish(ns, model.NacosAIKindSkill, skillName, version, updateLatest, force); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "skill 不存在")
			return
		}
		nacosV3Err(c, 20002, err.Error())
		return
	}
	nacosV3OK(c, true)
}

// --- Skills client ---

func NacosSkillClientGet(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name 必填", "data": nil})
		return
	}
	label := c.Query("label")
	version := c.Query("version")
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	zipData, err := service.NacosAIGetSkillZIP(ns, name, label, version)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error(), "data": nil})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	c.Data(http.StatusOK, "application/zip", zipData)
}

// --- AgentSpec admin ---

func NacosAgentSpecList(c *gin.Context) {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	name := c.Query("agentSpecName")
	search := c.Query("search")
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	data, err := service.NacosAIListAgentSpecs(ns, name, search, pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	nacosV3OK(c, data)
}

func NacosAgentSpecDescribe(c *gin.Context) {
	name := strings.TrimSpace(c.Query("agentSpecName"))
	if name == "" {
		nacosV3Err(c, 10000, "agentSpecName 必填")
		return
	}
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	data, err := service.NacosAIDescribeAgentSpec(ns, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "agentspec 不存在")
			return
		}
		nacosV3Err(c, 30000, err.Error())
		return
	}
	nacosV3OK(c, data)
}

func NacosAgentSpecUpload(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		nacosV3Err(c, 10000, "缺少 multipart 字段 file")
		return
	}
	if f.Size > config.NacosRegistryMaxUploadBytes {
		nacosV3Err(c, 20002, "上传文件过大")
		return
	}
	src, err := f.Open()
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	defer src.Close()
	zipData, err := io.ReadAll(src)
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	overwrite := nacosDefaultParam(c, "overwrite", "false") == "true"
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	if err := service.NacosAIUploadAgentSpec(ns, zipData, 0, overwrite, f.Filename); err != nil {
		nacosV3Err(c, 20002, err.Error())
		return
	}
	c.Status(http.StatusOK)
}

func NacosAgentSpecSubmit(c *gin.Context) {
	name := strings.TrimSpace(nacosParam(c, "agentSpecName"))
	if name == "" {
		nacosV3Err(c, 10000, "agentSpecName 必填")
		return
	}
	version := strings.TrimSpace(nacosParam(c, "version"))
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	if err := service.NacosAISubmit(ns, model.NacosAIKindAgentSpec, name, version); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "agentspec 不存在")
			return
		}
		nacosV3Err(c, 20002, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosAgentSpecPublish(c *gin.Context) {
	name := strings.TrimSpace(nacosParam(c, "agentSpecName"))
	version := strings.TrimSpace(nacosParam(c, "version"))
	if name == "" || version == "" {
		nacosV3Err(c, 10000, "agentSpecName 与 version 必填")
		return
	}
	updateLatest := nacosDefaultParam(c, "updateLatestLabel", "true") == "true"
	force := nacosPublishForceFlag(c)
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	if err := service.NacosAIPublish(ns, model.NacosAIKindAgentSpec, name, version, updateLatest, force); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "agentspec 不存在")
			return
		}
		nacosV3Err(c, 20002, err.Error())
		return
	}
	nacosV3OK(c, true)
}

// --- AgentSpec client ---

func NacosAgentSpecClientGet(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		nacosV3Err(c, 10000, "name 必填")
		return
	}
	label := c.Query("label")
	version := c.Query("version")
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	payload, err := service.NacosAIGetAgentSpecJSON(ns, name, label, version)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, err.Error())
			return
		}
		nacosV3Err(c, 20002, err.Error())
		return
	}
	nacosV3OK(c, payload)
}

// NacosSkillDownloadAdmin GET .../skills/download?namespaceId=&skillName=&version= （任意状态 ZIP）
func NacosSkillDownloadAdmin(c *gin.Context) {
	skillName := strings.TrimSpace(c.Query("skillName"))
	version := strings.TrimSpace(c.Query("version"))
	if skillName == "" || version == "" {
		nacosV3Err(c, 10000, "skillName 与 version 必填")
		return
	}
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	zipData, err := service.NacosAIGetSkillZIPAdmin(ns, skillName, version)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "skill 不存在")
			return
		}
		nacosV3Err(c, 20002, err.Error())
		return
	}
	c.Data(http.StatusOK, "application/zip", zipData)
}

// NacosAgentSpecDownloadAdmin GET .../agentspecs/download?namespaceId=&agentSpecName=&version= （任意状态 ZIP）
func NacosAgentSpecDownloadAdmin(c *gin.Context) {
	name := strings.TrimSpace(c.Query("agentSpecName"))
	version := strings.TrimSpace(c.Query("version"))
	if name == "" || version == "" {
		nacosV3Err(c, 10000, "agentSpecName 与 version 必填")
		return
	}
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	zipData, err := service.NacosAIGetAgentSpecZIPAdmin(ns, name, version)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "agentspec 不存在")
			return
		}
		nacosV3Err(c, 20002, err.Error())
		return
	}
	c.Data(http.StatusOK, "application/zip", zipData)
}

type nacosArtifactLabelsBody struct {
	Name    string            `json:"name"`
	Labels  map[string]string `json:"labels"`
	Replace bool              `json:"replace"`
}

// NacosSkillLabelsUpdate POST .../skills/labels
func NacosSkillLabelsUpdate(c *gin.Context) {
	var body nacosArtifactLabelsBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 20002, err.Error())
		return
	}
	if body.Labels == nil {
		body.Labels = map[string]string{}
	}
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	if err := service.NacosAIUpdateArtifactLabels(ns, model.NacosAIKindSkill, body.Name, body.Labels, body.Replace); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "skill 不存在")
			return
		}
		nacosV3Err(c, 20002, err.Error())
		return
	}
	nacosV3OK(c, true)
}

// NacosAgentSpecLabelsUpdate POST .../agentspecs/labels
func NacosAgentSpecLabelsUpdate(c *gin.Context) {
	var body nacosArtifactLabelsBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 20002, err.Error())
		return
	}
	if body.Labels == nil {
		body.Labels = map[string]string{}
	}
	ns, ok := nacosNamespaceOK(c, "public")
	if !ok {
		return
	}
	if err := service.NacosAIUpdateArtifactLabels(ns, model.NacosAIKindAgentSpec, body.Name, body.Labels, body.Replace); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "agentspec 不存在")
			return
		}
		nacosV3Err(c, 20002, err.Error())
		return
	}
	nacosV3OK(c, true)
}
