package controller

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/service"
	"gorm.io/gorm"
)

// --- CS 控制台路径 ---

func NacosCsConfigDeleteConsole(c *gin.Context) {
	dataID := strings.TrimSpace(c.Query("dataId"))
	group := strings.TrimSpace(c.Query("groupName"))
	if dataID == "" || group == "" {
		nacosV3Err(c, 400, "dataId 与 groupName 必填")
		return
	}
	if err := service.NacosCsDelete(service.NormalizeNacosNamespaceID(c.Query("namespaceId")), dataID, group); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "config 不存在")
			return
		}
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosCsConfigBatchDeleteConsole(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("ids"))
	ns := service.NormalizeNacosNamespaceID(c.Query("namespaceId"))
	if raw == "" || ns == "" {
		nacosV3Err(c, 400, "ids 与 namespaceId 必填")
		return
	}
	var ids []int64
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil || id <= 0 {
			nacosV3Err(c, 400, "ids 格式无效")
			return
		}
		ids = append(ids, id)
	}
	n, err := service.NacosCsBatchDeleteByIDs(ns, ids)
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, n > 0)
}

func NacosCsConfigExport2Console(c *gin.Context) {
	ns := service.NormalizeNacosNamespaceID(c.Query("namespaceId"))
	if ns == "" {
		nacosV3Err(c, 400, "namespaceId 必填")
		return
	}
	var ids []int64
	if s := strings.TrimSpace(c.Query("ids")); s != "" {
		for _, p := range strings.Split(s, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.ParseInt(p, 10, 64)
			if err == nil && id > 0 {
				ids = append(ids, id)
			}
		}
	}
	body, err := service.NacosCsExportPayload(ns, ids, c.Query("dataId"), c.Query("groupName"))
	if err != nil {
		nacosV3Err(c, 500, err.Error())
		return
	}
	fn := fmt.Sprintf("nacos-config-export-%s.json", ns)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fn))
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}

func nacosCsHistoryToConsole(h *model.NacosCsConfigHistory) (gin.H, error) {
	plain, err := service.NacosCsDecryptStored(h.Content, h.EncryptedDataKey)
	if err != nil {
		return nil, err
	}
	md := md5.Sum([]byte(plain))
	return gin.H{
		"id":           fmt.Sprintf("%d", h.Id),
		"nid":          fmt.Sprintf("%d", h.Id),
		"dataId":       h.DataId,
		"groupName":    h.GroupName,
		"content":      plain,
		"md5":          hex.EncodeToString(md[:]),
		"type":         h.Type,
		"appName":      "",
		"srcUser":      h.OperatorName,
		"srcIp":        "",
		"opType":       h.Action,
		"publishType":  "formal",
		"extInfo":      h.Remark,
		"createdTime":  h.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		"modifyTime":   h.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		"configTags":   "",
	}, nil
}

func NacosCsConfigHistoryDetailConsole(c *gin.Context) {
	nid := strings.TrimSpace(c.Query("nid"))
	if nid == "" {
		nacosV3Err(c, 400, "nid 必填")
		return
	}
	hid, err := strconv.ParseInt(nid, 10, 64)
	if err != nil || hid <= 0 {
		nacosV3Err(c, 400, "nid 无效")
		return
	}
	h, err := service.NacosCsHistoryGetByID(service.NormalizeNacosNamespaceID(c.Query("namespaceId")), hid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "历史不存在")
			return
		}
		nacosV3Err(c, 500, err.Error())
		return
	}
	body, err := nacosCsHistoryToConsole(h)
	if err != nil {
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, body)
}

func NacosCsConfigHistoryPreviousConsole(c *gin.Context) {
	curID, err := strconv.ParseInt(strings.TrimSpace(c.Query("id")), 10, 64)
	if err != nil || curID <= 0 {
		nacosV3Err(c, 400, "id 无效")
		return
	}
	dataID := strings.TrimSpace(c.Query("dataId"))
	group := strings.TrimSpace(c.Query("groupName"))
	h, err := service.NacosCsHistoryPreviousByID(service.NormalizeNacosNamespaceID(c.Query("namespaceId")), curID, dataID, group)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "无更早历史")
			return
		}
		nacosV3Err(c, 400, err.Error())
		return
	}
	body, err := nacosCsHistoryToConsole(h)
	if err != nil {
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, body)
}

// --- 命名空间 ---

func NacosConsoleNamespaceList(c *gin.Context) {
	data, err := service.NacosListConsoleNamespaces()
	if err != nil {
		nacosV3Err(c, 500, err.Error())
		return
	}
	nacosV3OK(c, data)
}

func NacosConsoleNamespaceGet(c *gin.Context) {
	ns := service.NormalizeNacosNamespaceID(c.Query("namespaceId"))
	if ns == "" {
		nacosV3Err(c, 400, "namespaceId 必填")
		return
	}
	item, err := service.NacosGetConsoleNamespaceItem(ns)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "命名空间不存在")
			return
		}
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, item)
}

func NacosConsoleNamespaceMutateStub(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodPost:
		var body struct {
			CustomNamespaceId string `json:"customNamespaceId"`
			NamespaceName     string `json:"namespaceName"`
			NamespaceDesc     string `json:"namespaceDesc"`
		}
		if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		remark := strings.TrimSpace(body.NamespaceDesc)
		if remark == "" {
			remark = strings.TrimSpace(body.NamespaceName)
		}
		if err := service.NacosCreateRegistryNamespace(body.CustomNamespaceId, remark); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, true)
	case http.MethodPut:
		var body struct {
			Namespace         string `json:"namespace"`
			NamespaceShowName string `json:"namespaceShowName"`
			NamespaceDesc     string `json:"namespaceDesc"`
		}
		if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		ns := strings.TrimSpace(body.Namespace)
		if ns == "" {
			nacosV3Err(c, 400, "namespace 必填")
			return
		}
		remark := strings.TrimSpace(body.NamespaceDesc)
		if remark == "" {
			remark = strings.TrimSpace(body.NamespaceShowName)
		}
		if err := service.NacosUpdateRegistryNamespaceRemark(ns, remark); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, true)
	default:
		nacosV3Err(c, 405, "不支持的方法")
	}
}

func NacosConsoleNamespaceDeleteStub(c *gin.Context) {
	ns := strings.TrimSpace(c.Query("namespaceId"))
	if ns == "" {
		nacosV3Err(c, 400, "namespaceId 必填")
		return
	}
	if err := service.NacosDeleteRegistryNamespaceByNamespaceId(ns); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

// --- 占位：服务发现 / 插件 / 集群 ---

func NacosConsoleStubEmptyPage(c *gin.Context) {
	nacosV3OK(c, gin.H{"totalCount": 0, "pageItems": []any{}})
}

func NacosConsoleStubEmptyString(c *gin.Context) {
	nacosV3OK(c, "")
}

func NacosConsoleStubOKTrue(c *gin.Context) {
	nacosV3OK(c, true)
}

func NacosConsoleStubStringArray(c *gin.Context) {
	nacosV3OK(c, []string{})
}

func NacosConsoleStubEmptyArray(c *gin.Context) {
	nacosV3OK(c, []any{})
}

func NacosConsoleStubEmptyListener(c *gin.Context) {
	qt := "config"
	if strings.Contains(c.Request.URL.Path, "/listener/ip") {
		qt = "ip"
	}
	nacosV3OK(c, gin.H{"listenersStatus": gin.H{}, "queryType": qt})
}

func NacosConsoleStubBetaEmpty(c *gin.Context) {
	nacosV3OK(c, gin.H{
		"dataId":     c.Query("dataId"),
		"groupName":  c.Query("groupName"),
		"content":    "",
		"type":       "text",
		"appName":    "",
		"desc":       "",
		"configTags": "",
		"grayRule":   "",
		"md5":        "",
	})
}

func NacosConsoleStubPluginList(c *gin.Context) {
	nacosV3OK(c, []gin.H{})
}

func NacosConsoleStubClusterNodes(c *gin.Context) {
	nacosV3OK(c, []gin.H{{
		"ip": "127.0.0.1", "port": 8848, "state": "UP",
		"address": "127.0.0.1:8848",
	}})
}

// --- AI：Skill / AgentSpec 版本与元数据 ---

func NacosSkillConsoleGetVersion(c *gin.Context) {
	doc, err := service.NacosAIDocumentSkillVersion(nacosNamespace(c), strings.TrimSpace(c.Query("skillName")), strings.TrimSpace(c.Query("version")))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "skill 不存在")
			return
		}
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, doc)
}

func NacosAgentSpecConsoleGetVersion(c *gin.Context) {
	doc, err := service.NacosAIDocumentAgentSpecVersion(nacosNamespace(c), strings.TrimSpace(c.Query("agentSpecName")), strings.TrimSpace(c.Query("version")))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "agentspec 不存在")
			return
		}
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, doc)
}

func buildSkillZip(skillName, skillMd string) ([]byte, error) {
	name := strings.TrimSpace(skillName)
	md := strings.TrimSpace(skillMd)
	if name == "" || md == "" {
		return nil, errors.New("skillName 与 skillCard 必填")
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	path := name + "/SKILL.md"
	f, err := w.Create(path)
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(f, md); err != nil {
		_ = w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type skillCardPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SkillMd     string `json:"skillMd"`
}

func parseSkillCardJSON(raw string) (*skillCardPayload, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("skillCard 不能为空")
	}
	var p skillCardPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func NacosConsoleSkillDraftStub(c *gin.Context) {
	ns := nacosNamespace(c)
	switch c.Request.Method {
	case http.MethodPost:
		var body struct {
			NamespaceId    string `json:"namespaceId" form:"namespaceId"`
			SkillName        string `json:"skillName" form:"skillName"`
			SkillCard        string `json:"skillCard" form:"skillCard"`
			BasedOnVersion   string `json:"basedOnVersion" form:"basedOnVersion"`
			TargetVersion    string `json:"targetVersion" form:"targetVersion"`
			CommitMsg        string `json:"commitMsg" form:"commitMsg"`
		}
		if err := c.ShouldBind(&body); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		name := strings.TrimSpace(body.SkillName)
		if strings.TrimSpace(body.SkillCard) == "" {
			if name == "" || strings.TrimSpace(body.BasedOnVersion) == "" {
				nacosV3Err(c, 400, "基于版本创建草稿需要 skillName 与 basedOnVersion，或提供 skillCard")
				return
			}
			nv, err := service.NacosAIArtifactCreateDraftFromVersion(ns, model.NacosAIKindSkill, name, body.BasedOnVersion, body.TargetVersion, body.CommitMsg)
			if err != nil {
				nacosV3Err(c, 400, err.Error())
				return
			}
			nacosV3OK(c, nv)
			return
		}
		card, err := parseSkillCardJSON(body.SkillCard)
		if err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		if name == "" {
			name = strings.TrimSpace(card.Name)
		}
		if name == "" {
			nacosV3Err(c, 400, "skillName 必填")
			return
		}
		md := strings.TrimSpace(card.SkillMd)
		if md == "" {
			nacosV3Err(c, 400, "skillCard.skillMd 必填")
			return
		}
		zipB, err := buildSkillZip(name, md)
		if err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		if err := service.NacosAIUploadSkill(ns, zipB, 0, name+".zip"); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		if card.Description != "" {
			_ = service.NacosAIUpdateSkillMetadata(ns, name, &card.Description, nil, nil, nil)
		}
		d, _ := service.NacosAIDescribeSkill(ns, name)
		if d != nil && strings.TrimSpace(d.EditingVersion) != "" {
			nacosV3OK(c, d.EditingVersion)
			return
		}
		nacosV3OK(c, "ok")
	case http.MethodPut:
		var body struct {
			NamespaceId string `json:"namespaceId" form:"namespaceId"`
			SkillCard   string `json:"skillCard" form:"skillCard"`
			CommitMsg   string `json:"commitMsg" form:"commitMsg"`
			SkillName   string `json:"skillName" form:"skillName"`
		}
		if err := c.ShouldBind(&body); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		card, err := parseSkillCardJSON(body.SkillCard)
		if err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		name := strings.TrimSpace(body.SkillName)
		if name == "" {
			name = strings.TrimSpace(card.Name)
		}
		if name == "" {
			nacosV3Err(c, 400, "skillName 必填（查询参数或 skillCard.name）")
			return
		}
		md := strings.TrimSpace(card.SkillMd)
		if md == "" {
			nacosV3Err(c, 400, "skillCard.skillMd 必填")
			return
		}
		zipB, err := buildSkillZip(name, md)
		if err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		if err := service.NacosAIUploadSkill(ns, zipB, 0, name+".zip"); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		if card.Description != "" {
			_ = service.NacosAIUpdateSkillMetadata(ns, name, &card.Description, nil, nil, nil)
		}
		nacosV3OK(c, "ok")
	case http.MethodDelete:
		name := strings.TrimSpace(c.Query("skillName"))
		if name == "" {
			nacosV3Err(c, 400, "skillName 必填")
			return
		}
		if err := service.NacosAIDeleteEditingArtifactVersions(ns, model.NacosAIKindSkill, name); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, true)
	default:
		nacosV3Err(c, 405, "不支持的方法")
	}
}

func NacosConsoleSkillOnlineOfflineStub(c *gin.Context) {
	ns := nacosNamespace(c)
	name := strings.TrimSpace(c.PostForm("skillName"))
	if name == "" {
		nacosV3Err(c, 400, "skillName 必填")
		return
	}
	ver := strings.TrimSpace(c.PostForm("version"))
	updateLatest := c.DefaultPostForm("updateLatestLabel", "true") == "true"
	if strings.Contains(c.Request.URL.Path, "/offline") {
		if ver != "" {
			if err := service.NacosAIArtifactVersionSetOffline(ns, model.NacosAIKindSkill, name, ver); err != nil {
				nacosV3Err(c, 400, err.Error())
				return
			}
			nacosV3OK(c, "ok")
			return
		}
		en := false
		if err := service.NacosAIUpdateSkillMetadata(ns, name, nil, nil, &en, nil); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, "ok")
		return
	}
	if ver != "" {
		if err := service.NacosAIArtifactVersionEnsureOnline(ns, model.NacosAIKindSkill, name, ver, updateLatest); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, "ok")
		return
	}
	en := true
	if err := service.NacosAIUpdateSkillMetadata(ns, name, nil, nil, &en, nil); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleSkillBizTagsUpdate(c *gin.Context) {
	ns := nacosNamespace(c)
	name := strings.TrimSpace(c.PostForm("skillName"))
	if name == "" {
		nacosV3Err(c, 400, "skillName 必填")
		return
	}
	bt := c.PostForm("bizTags")
	if err := service.NacosAIUpdateSkillMetadata(ns, name, nil, &bt, nil, nil); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleSkillScopeUpdate(c *gin.Context) {
	ns := nacosNamespace(c)
	name := strings.TrimSpace(c.PostForm("skillName"))
	sc := strings.TrimSpace(c.PostForm("scope"))
	if name == "" || sc == "" {
		nacosV3Err(c, 400, "skillName 与 scope 必填")
		return
	}
	if err := service.NacosAIUpdateSkillMetadata(ns, name, nil, nil, nil, &sc); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosSkillLabelsUpdateConsole(c *gin.Context) {
	ns := nacosNamespace(c)
	name := strings.TrimSpace(c.PostForm("skillName"))
	raw := strings.TrimSpace(c.PostForm("labels"))
	if name == "" || raw == "" {
		nacosV3Err(c, 400, "skillName 与 labels 必填")
		return
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		nacosV3Err(c, 400, "labels 须为 JSON 对象")
		return
	}
	if err := service.NacosAIUpdateArtifactLabels(ns, model.NacosAIKindSkill, name, labels, false); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleAgentSpecDraftStub(c *gin.Context) {
	ns := nacosNamespace(c)
	switch c.Request.Method {
	case http.MethodPost:
		var body struct {
			NamespaceId    string `json:"namespaceId" form:"namespaceId"`
			AgentSpecName  string `json:"agentSpecName" form:"agentSpecName"`
			BasedOnVersion string `json:"basedOnVersion" form:"basedOnVersion"`
			TargetVersion  string `json:"targetVersion" form:"targetVersion"`
			AgentSpecCard  string `json:"agentSpecCard" form:"agentSpecCard"`
		}
		if err := c.ShouldBind(&body); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		name := strings.TrimSpace(body.AgentSpecName)
		if strings.TrimSpace(body.AgentSpecCard) == "" {
			if name == "" || strings.TrimSpace(body.BasedOnVersion) == "" {
				nacosV3Err(c, 400, "基于版本创建草稿需要 agentSpecName 与 basedOnVersion，或提供 agentSpecCard")
				return
			}
			nv, err := service.NacosAIArtifactCreateDraftFromVersion(ns, model.NacosAIKindAgentSpec, name, body.BasedOnVersion, body.TargetVersion, "")
			if err != nil {
				nacosV3Err(c, 400, err.Error())
				return
			}
			nacosV3OK(c, nv)
			return
		}
		var probe struct {
			Name        string                     `json:"name"`
			Description string                     `json:"description"`
			Content     string                     `json:"content"`
			Resource    map[string]json.RawMessage `json:"resource"`
		}
		if err := json.Unmarshal([]byte(body.AgentSpecCard), &probe); err != nil {
			nacosV3Err(c, 400, "agentSpecCard 须为合法 JSON")
			return
		}
		if name == "" {
			name = strings.TrimSpace(probe.Name)
		}
		if name == "" {
			nacosV3Err(c, 400, "agentSpecName 或 agentSpecCard.name 必填")
			return
		}
		man := strings.TrimSpace(probe.Content)
		if man == "" {
			desc, _ := json.Marshal(strings.TrimSpace(probe.Description))
			man = fmt.Sprintf(`{"name":%q,"description":%s}`, name, string(desc))
		}
		if probe.Resource == nil {
			probe.Resource = map[string]json.RawMessage{}
		}
		zipB, err := service.BuildAgentSpecZipFromEditorCard(name, man, probe.Resource)
		if err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		if err := service.NacosAIUploadAgentSpec(ns, zipB, 0, true, name+".zip"); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		if probe.Description != "" {
			d := probe.Description
			_ = service.NacosAIUpdateAgentSpecMetadata(ns, name, &d, nil, nil, nil)
		}
		d, _ := service.NacosAIDescribeAgentSpec(ns, name)
		if d != nil && d.EditingVersion != nil && strings.TrimSpace(*d.EditingVersion) != "" {
			nacosV3OK(c, *d.EditingVersion)
			return
		}
		nacosV3OK(c, "ok")
	case http.MethodPut:
		var body struct {
			NamespaceId   string `json:"namespaceId" form:"namespaceId"`
			AgentSpecCard string `json:"agentSpecCard" form:"agentSpecCard"`
			AgentSpecName string `json:"agentSpecName" form:"agentSpecName"`
		}
		if err := c.ShouldBind(&body); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		name := strings.TrimSpace(body.AgentSpecName)
		var probe struct {
			Name        string                     `json:"name"`
			Description string                     `json:"description"`
			Content     string                     `json:"content"`
			Resource    map[string]json.RawMessage `json:"resource"`
		}
		if err := json.Unmarshal([]byte(body.AgentSpecCard), &probe); err != nil {
			nacosV3Err(c, 400, "agentSpecCard 须为合法 JSON")
			return
		}
		if name == "" {
			name = strings.TrimSpace(probe.Name)
		}
		if name == "" {
			nacosV3Err(c, 400, "agentSpecName 必填（查询参数或 agentSpecCard.name）")
			return
		}
		man := strings.TrimSpace(probe.Content)
		if man == "" {
			desc, _ := json.Marshal(strings.TrimSpace(probe.Description))
			man = fmt.Sprintf(`{"name":%q,"description":%s}`, name, string(desc))
		}
		if probe.Resource == nil {
			probe.Resource = map[string]json.RawMessage{}
		}
		zipB, err := service.BuildAgentSpecZipFromEditorCard(name, man, probe.Resource)
		if err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		if err := service.NacosAIUploadAgentSpec(ns, zipB, 0, true, name+".zip"); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		if probe.Description != "" {
			d := probe.Description
			_ = service.NacosAIUpdateAgentSpecMetadata(ns, name, &d, nil, nil, nil)
		}
		nacosV3OK(c, "ok")
	case http.MethodDelete:
		name := strings.TrimSpace(c.Query("agentSpecName"))
		if name == "" {
			nacosV3Err(c, 400, "agentSpecName 必填")
			return
		}
		if err := service.NacosAIDeleteEditingArtifactVersions(ns, model.NacosAIKindAgentSpec, name); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, true)
	default:
		nacosV3Err(c, 405, "不支持的方法")
	}
}

func NacosConsoleAgentSpecOnlineOfflineStub(c *gin.Context) {
	ns := nacosNamespace(c)
	name := strings.TrimSpace(c.PostForm("agentSpecName"))
	if name == "" {
		nacosV3Err(c, 400, "agentSpecName 必填")
		return
	}
	ver := strings.TrimSpace(c.PostForm("version"))
	updateLatest := c.DefaultPostForm("updateLatestLabel", "true") == "true"
	if strings.Contains(c.Request.URL.Path, "/offline") {
		if ver != "" {
			if err := service.NacosAIArtifactVersionSetOffline(ns, model.NacosAIKindAgentSpec, name, ver); err != nil {
				nacosV3Err(c, 400, err.Error())
				return
			}
			nacosV3OK(c, "ok")
			return
		}
		en := false
		if err := service.NacosAIUpdateAgentSpecMetadata(ns, name, nil, nil, &en, nil); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, "ok")
		return
	}
	if ver != "" {
		if err := service.NacosAIArtifactVersionEnsureOnline(ns, model.NacosAIKindAgentSpec, name, ver, updateLatest); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, "ok")
		return
	}
	en := true
	if err := service.NacosAIUpdateAgentSpecMetadata(ns, name, nil, nil, &en, nil); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleAgentSpecBizTagsUpdate(c *gin.Context) {
	ns := nacosNamespace(c)
	name := strings.TrimSpace(c.PostForm("agentSpecName"))
	bt := c.PostForm("bizTags")
	if name == "" {
		nacosV3Err(c, 400, "agentSpecName 必填")
		return
	}
	if err := service.NacosAIUpdateAgentSpecMetadata(ns, name, nil, &bt, nil, nil); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleAgentSpecScopeUpdate(c *gin.Context) {
	ns := nacosNamespace(c)
	name := strings.TrimSpace(c.PostForm("agentSpecName"))
	sc := strings.TrimSpace(c.PostForm("scope"))
	if name == "" || sc == "" {
		nacosV3Err(c, 400, "agentSpecName 与 scope 必填")
		return
	}
	if err := service.NacosAIUpdateAgentSpecMetadata(ns, name, nil, nil, nil, &sc); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosAgentSpecLabelsUpdateConsole(c *gin.Context) {
	ns := nacosNamespace(c)
	name := strings.TrimSpace(c.PostForm("agentSpecName"))
	raw := strings.TrimSpace(c.PostForm("labels"))
	if name == "" || raw == "" {
		nacosV3Err(c, 400, "agentSpecName 与 labels 必填")
		return
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		nacosV3Err(c, 400, "labels 须为 JSON 对象")
		return
	}
	if err := service.NacosAIUpdateArtifactLabels(ns, model.NacosAIKindAgentSpec, name, labels, false); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleMcpImportValidateStub(c *gin.Context) {
	nacosV3OK(c, gin.H{"servers": []gin.H{}})
}

func NacosA2AVersionListConsole(c *gin.Context) {
	ns := nacosNamespace(c)
	name := strings.TrimSpace(c.Query("agentName"))
	if name == "" {
		nacosV3Err(c, 400, "agentName 必填")
		return
	}
	var r model.NacosAIA2AAgent
	if err := model.DB.Where("namespace_id = ? AND agent_name = ?", service.NormalizeNacosNamespaceID(ns), name).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "Agent 不存在")
			return
		}
		nacosV3Err(c, 500, err.Error())
		return
	}
	ver := "v1"
	item, err := service.NacosAIDescribeA2A(ns, name)
	if err == nil && len(item.Card) > 0 {
		var m map[string]any
		if json.Unmarshal(item.Card, &m) == nil {
			if v, ok := m["version"].(string); ok && strings.TrimSpace(v) != "" {
				ver = strings.TrimSpace(v)
			}
		}
	}
	nacosV3OK(c, []gin.H{{
		"version":   ver,
		"latest":    true,
		"createdAt": r.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}})
}

func mapPromptVersionStatus(st string) string {
	switch st {
	case model.NacosAIVersionEditing:
		return "draft"
	case model.NacosAIVersionReviewing:
		return "reviewing"
	case model.NacosAIVersionOnline:
		return "online"
	case model.NacosAIVersionOffline:
		return "offline"
	default:
		return st
	}
}

func promptContentMD5Hex(b []byte) string {
	h := md5.Sum(b)
	return hex.EncodeToString(h[:])
}

func parseBizTagsLoose(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		return arr
	}
	out := strings.Split(raw, ",")
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	return out
}

func parsePromptVersionPayload(raw []byte) (template string, variables []gin.H) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil || root == nil {
		return string(raw), nil
	}
	if t, ok := root["template"]; ok {
		_ = json.Unmarshal(t, &template)
	} else {
		template = string(raw)
	}
	if v, ok := root["variables"]; ok {
		var arr []map[string]string
		if json.Unmarshal(v, &arr) == nil {
			for _, x := range arr {
				variables = append(variables, gin.H{
					"name":         x["name"],
					"defaultValue": x["defaultValue"],
					"description":  x["description"],
				})
			}
		}
	}
	return template, variables
}

func NacosPromptConsoleGovernance(c *gin.Context) {
	key := strings.TrimSpace(c.Query("promptKey"))
	if key == "" {
		nacosV3Err(c, 400, "promptKey 必填")
		return
	}
	d, err := service.NacosAIDescribePrompt(nacosNamespace(c), key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 404, "prompt 不存在")
			return
		}
		nacosV3Err(c, 500, err.Error())
		return
	}
	labels := d.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	latest := labels["latest"]
	bizArr := parseBizTagsLoose(d.BizTags)
	verKeys := make([]string, 0, len(d.Versions))
	details := make([]gin.H, 0, len(d.Versions))
	for _, v := range d.Versions {
		verKeys = append(verKeys, v.Version)
		gm := int64(0)
		if v.UpdateTime != nil {
			gm = *v.UpdateTime
		} else if v.CreateTime != nil {
			gm = *v.CreateTime
		}
		details = append(details, gin.H{
			"promptKey":           d.PromptKey,
			"version":             v.Version,
			"status":              mapPromptVersionStatus(v.Status),
			"commitMsg":           "",
			"srcUser":             "",
			"gmtModified":         gm,
			"publishPipelineInfo": nil,
			"downloadCount":       nil,
		})
	}
	nacosV3OK(c, gin.H{
		"schemaVersion":      1,
		"promptKey":          d.PromptKey,
		"description":        d.Description,
		"bizTags":            bizArr,
		"bizTagsStr":         d.BizTags,
		"latestVersion":      latest,
		"gmtModified":        d.UpdateTime,
		"editingVersion":     nullStrPtr(d.EditingVersion),
		"reviewingVersion":   nullStrPtr(d.ReviewingVersion),
		"onlineCnt":          d.OnlineCnt,
		"labels":             labels,
		"downloadCount":      nil,
		"versions":           verKeys,
		"versionDetails":     details,
	})
}

func nullStrPtr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func NacosPromptConsoleVersionDetail(c *gin.Context) {
	ns := nacosNamespace(c)
	key := strings.TrimSpace(c.Query("promptKey"))
	ver := strings.TrimSpace(c.Query("version"))
	if key == "" || ver == "" {
		nacosV3Err(c, 400, "promptKey 与 version 必填")
		return
	}
	raw, err := service.NacosAIPromptVersionRawContent(ns, key, ver)
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	d, err := service.NacosAIDescribePrompt(ns, key)
	if err != nil {
		nacosV3Err(c, 500, err.Error())
		return
	}
	var st string
	var gmt int64
	for _, v := range d.Versions {
		if v.Version == ver {
			st = v.Status
			if v.UpdateTime != nil {
				gmt = *v.UpdateTime
			} else if v.CreateTime != nil {
				gmt = *v.CreateTime
			}
			break
		}
	}
	tpl, vars := parsePromptVersionPayload(raw)
	nacosV3OK(c, gin.H{
		"promptKey":           key,
		"version":             ver,
		"status":              mapPromptVersionStatus(st),
		"commitMsg":           "",
		"srcUser":             "",
		"gmtModified":         gmt,
		"publishPipelineInfo": nil,
		"downloadCount":       nil,
		"template":            tpl,
		"md5":                 promptContentMD5Hex(raw),
		"variables":           vars,
	})
}

func NacosPromptConsoleVersionsPage(c *gin.Context) {
	ns := nacosNamespace(c)
	key := strings.TrimSpace(c.Query("promptKey"))
	if key == "" {
		nacosV3Err(c, 400, "promptKey 必填")
		return
	}
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	total, rows, err := service.NacosAIPromptVersionListPage(ns, key, pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		pages = 0
	}
	items := make([]gin.H, 0, len(rows))
	for _, v := range rows {
		gm := int64(0)
		if v.UpdateTime != nil {
			gm = *v.UpdateTime
		} else if v.CreateTime != nil {
			gm = *v.CreateTime
		}
		items = append(items, gin.H{
			"promptKey":           key,
			"version":             v.Version,
			"status":              mapPromptVersionStatus(v.Status),
			"commitMsg":           "",
			"srcUser":             "",
			"gmtModified":         gm,
			"publishPipelineInfo": nil,
			"downloadCount":       nil,
		})
	}
	nacosV3OK(c, gin.H{
		"pageNo":         pageNo,
		"pageSize":       pageSize,
		"totalCount":     total,
		"pagesAvailable": pages,
		"pageItems":      items,
	})
}

func NacosPromptConsoleVersionDownload(c *gin.Context) {
	ns := nacosNamespace(c)
	key := strings.TrimSpace(c.Query("promptKey"))
	ver := strings.TrimSpace(c.Query("version"))
	if key == "" || ver == "" {
		nacosV3Err(c, 400, "promptKey 与 version 必填")
		return
	}
	raw, err := service.NacosAIPromptVersionRawContent(ns, key, ver)
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	tpl, _ := parsePromptVersionPayload(raw)
	fn := fmt.Sprintf("%s-%s.md", key, ver)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fn))
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(tpl))
}

func NacosPromptConsoleDraftStub(c *gin.Context) {
	ns := nacosNamespace(c)
	key := strings.TrimSpace(c.PostForm("promptKey"))
	if key == "" {
		nacosV3Err(c, 400, "promptKey 必填")
		return
	}
	switch c.Request.Method {
	case http.MethodPost:
		desc := c.PostForm("description")
		biz := c.PostForm("bizTags")
		if err := service.NacosAIUpsertPromptHeader(ns, key, desc, biz, "", "PUBLIC", nil); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		tpl := c.PostForm("template")
		if strings.TrimSpace(tpl) == "" {
			nacosV3Err(c, 400, "template 必填")
			return
		}
		vars := strings.TrimSpace(c.PostForm("variables"))
		obj := map[string]any{"template": tpl}
		if vars != "" {
			if !json.Valid([]byte(vars)) {
				nacosV3Err(c, 400, "variables 须为合法 JSON")
				return
			}
			obj["variables"] = json.RawMessage(vars)
		}
		b, err := json.Marshal(obj)
		if err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		ver, err := service.NacosAIPromptAddVersion(ns, key, string(b))
		if err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, ver)
	case http.MethodPut:
		tpl := c.PostForm("template")
		if strings.TrimSpace(tpl) == "" {
			nacosV3Err(c, 400, "template 必填")
			return
		}
		vars := strings.TrimSpace(c.PostForm("variables"))
		obj := map[string]any{"template": tpl}
		if vars != "" {
			if !json.Valid([]byte(vars)) {
				nacosV3Err(c, 400, "variables 须为合法 JSON")
				return
			}
			obj["variables"] = json.RawMessage(vars)
		}
		b, err := json.Marshal(obj)
		if err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		_, err = service.NacosAIPromptUpsertEditingContent(ns, key, string(b))
		if err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, true)
	case http.MethodDelete:
		if err := service.NacosAIPromptDeleteEditingVersions(ns, key); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, true)
	default:
		nacosV3Err(c, 405, "不支持的方法")
	}
}

func NacosPromptConsoleOnlineOfflineStub(c *gin.Context) {
	ns := nacosNamespace(c)
	key := strings.TrimSpace(c.PostForm("promptKey"))
	ver := strings.TrimSpace(c.PostForm("version"))
	if key == "" || ver == "" {
		nacosV3Err(c, 400, "promptKey 与 version 必填")
		return
	}
	updateLatest := c.DefaultPostForm("updateLatestLabel", "true") == "true"
	if strings.Contains(c.Request.URL.Path, "/offline") {
		if err := service.NacosAIPromptVersionSetOffline(ns, key, ver); err != nil {
			nacosV3Err(c, 400, err.Error())
			return
		}
		nacosV3OK(c, true)
		return
	}
	if err := service.NacosAIPromptVersionEnsureOnline(ns, key, ver, updateLatest); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosPromptLabelsUpdateConsole(c *gin.Context) {
	ns := nacosNamespace(c)
	key := strings.TrimSpace(c.PostForm("promptKey"))
	raw := strings.TrimSpace(c.PostForm("labels"))
	if key == "" || raw == "" {
		nacosV3Err(c, 400, "promptKey 与 labels 必填")
		return
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		nacosV3Err(c, 400, "labels 须为 JSON 对象")
		return
	}
	if err := service.NacosAIUpdatePromptLabels(ns, key, labels, false); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosPromptConsoleDescriptionUpdate(c *gin.Context) {
	ns := nacosNamespace(c)
	key := strings.TrimSpace(c.PostForm("promptKey"))
	desc := c.PostForm("description")
	if key == "" {
		nacosV3Err(c, 400, "promptKey 必填")
		return
	}
	if err := service.NacosAIUpsertPromptHeader(ns, key, desc, "", "", "", nil); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosPromptConsoleBizTagsUpdate(c *gin.Context) {
	ns := nacosNamespace(c)
	key := strings.TrimSpace(c.PostForm("promptKey"))
	biz := c.PostForm("bizTags")
	if key == "" {
		nacosV3Err(c, 400, "promptKey 必填")
		return
	}
	if err := service.NacosAIUpsertPromptHeader(ns, key, "", biz, "", "", nil); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}
