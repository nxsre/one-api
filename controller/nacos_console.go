package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/nacosdist"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/service"
	"gorm.io/gorm"
)

// GetNacosRegistryInfo GET /api/nacos/registry/info
func GetNacosRegistryInfo(c *gin.Context) {
	s3ok := strings.TrimSpace(common.S3RemoteBucket) != ""
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":              true,
			"anonymous_read":       config.NacosRegistryAnonymousRead,
			"zip_storage":          service.NacosZipStorageBackend(),
			"payload_storage":      service.NacosRegistryPayloadBackend(),
			"zip_local_dir":        service.NacosRegistryEffectiveZipLocalDir(),
			"s3_remote_configured": s3ok,
			"permission_keys":      service.NacosAllPermissionKeys(),
			"permission_catalog":   service.NacosPermissionCatalog(),
			"max_upload_bytes":       config.NacosRegistryMaxUploadBytes,
			"native_console_serving": nacosdist.UIBundled(),
			"cs_cipher_aes_enabled":   true,
			"cs_encryption_configured": service.NacosCsEncryptionKeyConfigured(),
			"cs_encryption_rotation_configured": service.NacosCsEncryptionRotationConfigured(),
			"cs_client_get_return_ciphertext":   config.NacosCsClientGetReturnCiphertext,
		},
	})
}

// ListNacosSkillsAdmin GET /api/nacos/skills
func ListNacosSkillsAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	filter, page, size := parseNacosSkillListFilter(c)
	data, err := service.NacosAIListSkills(ns, filter, page, size)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// GetNacosSkillDetailAdmin GET /api/nacos/skills/detail?name=&namespace=
func GetNacosSkillDetailAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 必填"})
		return
	}
	data, err := service.NacosAIDescribeSkill(ns, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "skill 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// UploadNacosSkillAdmin POST /api/nacos/skills/upload?namespace= （multipart 字段 file）
func UploadNacosSkillAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
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
	if err := service.NacosAIUploadSkill(ns, zipData, ownerID, f.Filename); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type patchNacosSkillBody struct {
	Namespace   string  `json:"namespace"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	BizTags     *string `json:"bizTags"`
	Enable      *bool   `json:"enable"`
	Scope       *string `json:"scope"`
}

// UpdateNacosSkillMetadataAdmin PUT /api/nacos/skills/metadata
func UpdateNacosSkillMetadataAdmin(c *gin.Context) {
	var body patchNacosSkillBody
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
	if err := service.NacosAIUpdateSkillMetadata(ns, name, body.Description, body.BizTags, body.Enable, body.Scope); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "skill 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteNacosSkillAdmin DELETE /api/nacos/skills/item?name=&namespace=
func DeleteNacosSkillAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 必填"})
		return
	}
	if err := service.NacosAIDeleteSkill(ns, name); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "skill 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type nacosSkillSubmitBody struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

// SubmitNacosSkillAdmin POST /api/nacos/skills/submit
func SubmitNacosSkillAdmin(c *gin.Context) {
	var body nacosSkillSubmitBody
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
	if err := service.NacosAISubmit(ns, model.NacosAIKindSkill, name, strings.TrimSpace(body.Version)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "skill 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type nacosSkillPublishBody struct {
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	UpdateLatest bool   `json:"updateLatest"`
	ForcePublish bool   `json:"forcePublish"`
}

// PublishNacosSkillAdmin POST /api/nacos/skills/publish
func PublishNacosSkillAdmin(c *gin.Context) {
	var body nacosSkillPublishBody
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
	if err := service.NacosAIPublish(ns, model.NacosAIKindSkill, name, ver, body.UpdateLatest, body.ForcePublish); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "skill 不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type nacosSkillVersionOnlineBody struct {
	Namespace         string `json:"namespace"`
	Name              string `json:"name"`
	Version           string `json:"version"`
	UpdateLatestLabel *bool  `json:"updateLatestLabel"`
}

// NacosSkillVersionOnlineAdmin POST /api/nacos/skills/version/online
// 与原生控制台「版本恢复上线」一致：仅 offline → online；可选将 latest 指向该版本（等同带 version 的 online 接口）。
func NacosSkillVersionOnlineAdmin(c *gin.Context) {
	var body nacosSkillVersionOnlineBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := strings.TrimSpace(body.Namespace)
	if ns == "" {
		ns = "public"
	}
	name := strings.TrimSpace(body.Name)
	ver := strings.TrimSpace(body.Version)
	if name == "" || ver == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 与 version 必填"})
		return
	}
	updateLatest := true
	if body.UpdateLatestLabel != nil {
		updateLatest = *body.UpdateLatestLabel
	}
	if err := service.NacosAIArtifactVersionSetOnlineFromOffline(ns, model.NacosAIKindSkill, name, ver); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if updateLatest {
		if err := service.NacosAIArtifactVersionEnsureOnline(ns, model.NacosAIKindSkill, name, ver, true); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type nacosSkillVersionOfflineBody struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

// NacosSkillVersionOfflineAdmin POST /api/nacos/skills/version/offline
// 与原生控制台「版本下线」一致：仅 online → offline。
func NacosSkillVersionOfflineAdmin(c *gin.Context) {
	var body nacosSkillVersionOfflineBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := strings.TrimSpace(body.Namespace)
	if ns == "" {
		ns = "public"
	}
	name := strings.TrimSpace(body.Name)
	ver := strings.TrimSpace(body.Version)
	if name == "" || ver == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name 与 version 必填"})
		return
	}
	if err := service.NacosAIArtifactVersionSetOffline(ns, model.NacosAIKindSkill, name, ver); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListNacosNamespaceOptions GET /api/nacos/namespaces/options?q=
func ListNacosNamespaceOptions(c *gin.Context) {
	q := c.Query("q")
	items, err := service.NacosListNamespaceOptions(q)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"namespaces": items}})
}

// ListNacosRegistryNamespaces GET /api/nacos/namespaces
// 返回与 Nacos 原生控制台一致的字段（仅已登记命名空间）。
func ListNacosRegistryNamespaces(c *gin.Context) {
	data, err := service.NacosListConsoleNamespaces()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

type createNacosNsBody struct {
	NamespaceId       string `json:"namespace_id"`
	Remark              string `json:"remark"`
	CustomNamespaceId   string `json:"customNamespaceId"`
	NamespaceName       string `json:"namespaceName"`
	NamespaceDesc       string `json:"namespaceDesc"`
}

// CreateNacosRegistryNamespace POST /api/nacos/namespaces
// 支持旧字段 namespace_id/remark，或与原生 UI 一致的 customNamespaceId、namespaceName、namespaceDesc。
func CreateNacosRegistryNamespace(c *gin.Context) {
	var body createNacosNsBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	var nsID, remark string
	if strings.TrimSpace(body.NamespaceName) != "" {
		remark = strings.TrimSpace(body.NamespaceDesc)
		if remark == "" {
			remark = strings.TrimSpace(body.NamespaceName)
		}
		nsID = strings.TrimSpace(body.CustomNamespaceId)
		if nsID == "" {
			nsID = strings.TrimSpace(body.NamespaceName)
		}
	} else {
		nsID = strings.TrimSpace(body.NamespaceId)
		remark = strings.TrimSpace(body.Remark)
	}
	if nsID == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "命名空间名称或 ID 不能为空"})
		return
	}
	if err := service.NacosCreateRegistryNamespace(nsID, remark); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type updateNacosNsBody struct {
	Namespace         string `json:"namespace"`
	NamespaceShowName string `json:"namespaceShowName"`
	NamespaceDesc     string `json:"namespaceDesc"`
}

// UpdateNacosRegistryNamespace PUT /api/nacos/namespaces（与原生控制台编辑语义一致）
func UpdateNacosRegistryNamespace(c *gin.Context) {
	var body updateNacosNsBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := strings.TrimSpace(body.Namespace)
	if ns == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "namespace 必填"})
		return
	}
	remark := strings.TrimSpace(body.NamespaceDesc)
	if remark == "" {
		remark = strings.TrimSpace(body.NamespaceShowName)
	}
	if err := service.NacosUpdateRegistryNamespaceRemark(ns, remark); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetNacosNamespaceDetailAdmin GET /api/nacos/namespaces/detail?namespaceId=
func GetNacosNamespaceDetailAdmin(c *gin.Context) {
	ns := strings.TrimSpace(c.Query("namespaceId"))
	item, err := service.NacosGetConsoleNamespaceItem(ns)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

// DeleteNacosRegistryNamespace DELETE /api/nacos/namespaces/:namespaceId（与原生控制台按 namespaceId 删除一致）
func DeleteNacosRegistryNamespace(c *gin.Context) {
	nsID := strings.TrimSpace(c.Param("namespaceId"))
	if nsID == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的 namespaceId"})
		return
	}
	if err := service.NacosDeleteRegistryNamespaceByNamespaceId(nsID); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListNacosAgentSpecsAdmin GET /api/nacos/agentspecs
func ListNacosAgentSpecsAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	data, err := service.NacosAIListAgentSpecs(ns, "", "", page, size)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// GetNacosUserACL GET /api/nacos/users/:id/acl（:id 为 ULID）
func GetNacosUserACL(c *gin.Context) {
	pk, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效用户 id"})
		return
	}
	u, err := model.GetUserById(pk, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	acl, err := model.GetNacosUserACL(pk)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"user_id": u.Uid, "rules": map[string]bool{}}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	var rules map[string]bool
	_ = json.Unmarshal([]byte(acl.RulesJSON), &rules)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"user_id": u.Uid, "rules": rules, "updated_at": acl.UpdatedAt}})
}

type putNacosACLBody struct {
	Rules map[string]bool `json:"rules"`
}

// PutNacosUserACL PUT /api/nacos/users/:id/acl（:id 为 ULID）
func PutNacosUserACL(c *gin.Context) {
	pk, err := model.ParseUserRouteParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效用户 id"})
		return
	}
	var body putNacosACLBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	b, _ := json.Marshal(body.Rules)
	uid := 0
	if v, ok := c.Get("id"); ok {
		if ii, ok := v.(int); ok {
			uid = ii
		}
	}
	if err := model.UpsertNacosUserACL(pk, string(b), uid); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
