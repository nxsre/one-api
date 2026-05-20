package service

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/model"
	"gorm.io/gorm"
)

// --- CS 灰度 beta ---

func NacosCsBetaGet(namespace, dataID, group string) (map[string]interface{}, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	var b model.NacosCsConfigBeta
	err := model.DB.Where("namespace_id = ? AND data_id = ? AND group_name = ?", ns, dataID, group).First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]interface{}{
			"dataId": dataID, "groupName": group, "content": "", "type": "text",
			"appName": "", "desc": "", "configTags": "", "grayRule": "", "md5": "",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"dataId": b.DataId, "groupName": b.GroupName, "content": b.Content, "type": b.Type,
		"appName": b.AppName, "desc": b.Desc, "configTags": b.ConfigTags, "grayRule": b.GrayRule, "md5": b.Md5,
	}, nil
}

func NacosCsBetaUpsert(namespace, dataID, group, betaIps, content, typ, appName, desc, tags, gray string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	dataID = strings.TrimSpace(dataID)
	group = strings.TrimSpace(group)
	if dataID == "" || group == "" {
		return errors.New("dataId 与 groupName 必填")
	}
	h := md5.Sum([]byte(content))
	md := hex.EncodeToString(h[:])
	var row model.NacosCsConfigBeta
	err := model.DB.Where("namespace_id = ? AND data_id = ? AND group_name = ?", ns, dataID, group).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.DB.Create(&model.NacosCsConfigBeta{
			NamespaceId: ns, DataId: dataID, GroupName: group, BetaIps: betaIps,
			Content: content, Type: typ, AppName: appName, Desc: desc, ConfigTags: tags, GrayRule: gray, Md5: md,
		}).Error
	}
	if err != nil {
		return err
	}
	return model.DB.Model(&row).Updates(map[string]interface{}{
		"beta_ips": betaIps, "content": content, "type": typ, "app_name": appName,
		"desc": desc, "config_tags": tags, "gray_rule": gray, "md5": md,
	}).Error
}

func NacosCsBetaDelete(namespace, dataID, group string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	return model.DB.Where("namespace_id = ? AND data_id = ? AND group_name = ?", ns, dataID, group).Delete(&model.NacosCsConfigBeta{}).Error
}

// --- 监听（占位 + 持久化）---

func NacosCsListenerMap(namespace, dataID, group, queryIP string) map[string]string {
	ns := NormalizeNacosNamespaceID(namespace)
	dataID = strings.TrimSpace(dataID)
	group = strings.TrimSpace(group)
	q := model.DB.Model(&model.NacosCsConfigListener{}).Where("namespace_id = ?", ns)
	if dataID != "" && group != "" {
		q = q.Where("data_id = ? AND group_name = ?", dataID, group)
	}
	if strings.TrimSpace(queryIP) != "" {
		q = q.Where("ip = ?", strings.TrimSpace(queryIP))
	}
	var rows []model.NacosCsConfigListener
	_ = q.Find(&rows).Error
	if len(rows) == 0 && dataID != "" && group != "" {
		// 演示数据：首次查询时写入，便于控制台非空
		_ = model.DB.Create(&model.NacosCsConfigListener{
			NamespaceId: ns, DataId: dataID, GroupName: group,
			ClientId: "demo-" + fmt.Sprintf("%d", time.Now().UnixNano()), Ip: "127.0.0.1", AppName: "console",
			QueryType: "config", Status: "UP",
		}).Error
		_ = q.Find(&rows).Error
	}
	out := map[string]string{}
	for _, r := range rows {
		key := r.ClientId
		if key == "" {
			key = fmt.Sprintf("c-%d", r.Id)
		}
		out[key] = r.Status
	}
	return out
}

// --- 克隆 ---

type NacosCsCloneItem struct {
	CfgID  string `json:"cfgId"`
	DataID string `json:"dataId"`
	Group  string `json:"group"`
}

func NacosCsCloneConfigs(srcNs, dstNs, policy string, items []NacosCsCloneItem) (cloned, skipped int, err error) {
	srcNs = NormalizeNacosNamespaceID(srcNs)
	dstNs = NormalizeNacosNamespaceID(dstNs)
	policy = strings.ToUpper(strings.TrimSpace(policy))
	if policy == "" {
		policy = "ABORT"
	}
	for _, it := range items {
		id, _ := parseInt64(it.CfgID)
		var src model.NacosCsConfig
		q := model.DB.Where("namespace_id = ?", srcNs)
		if id > 0 {
			q = q.Where("id = ?", id)
		} else if strings.TrimSpace(it.DataID) != "" && strings.TrimSpace(it.Group) != "" {
			q = q.Where("data_id = ? AND group_name = ?", strings.TrimSpace(it.DataID), strings.TrimSpace(it.Group))
		} else {
			continue
		}
		if err := q.First(&src).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return cloned, skipped, err
		}
		var exist int64
		_ = model.DB.Model(&model.NacosCsConfig{}).Where("namespace_id = ? AND data_id = ? AND group_name = ?", dstNs, src.DataId, src.GroupName).Count(&exist).Error
		if exist > 0 {
			if policy == "SKIP" {
				skipped++
				continue
			}
			if policy == "ABORT" {
				return cloned, skipped, fmt.Errorf("目标命名空间已存在配置 %s@@%s", src.GroupName, src.DataId)
			}
		}
		if _, err := NacosCsUpsertRaw(dstNs, src.DataId, src.GroupName, src.Content, src.Type, src.EncryptedDataKey, nil); err != nil {
			return cloned, skipped, err
		}
		cloned++
	}
	return cloned, skipped, nil
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

// --- 导入（ZIP 或 JSON 数组）---

func NacosCsImportFromMultipart(ns string, policy string, file *multipart.FileHeader) (created int, err error) {
	ns = NormalizeNacosNamespaceID(ns)
	policy = strings.ToUpper(strings.TrimSpace(policy))
	if policy == "" {
		policy = "ABORT"
	}
	src, err := file.Open()
	if err != nil {
		return 0, err
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}
	ct := file.Header.Get("Content-Type")
	if strings.Contains(ct, "zip") || strings.HasSuffix(strings.ToLower(file.Filename), ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return 0, err
		}
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				continue
			}
			b, _ := io.ReadAll(rc)
			_ = rc.Close()
			if json.Valid(b) {
				n, e := nacosCsImportJSONArray(ns, policy, b)
				if e != nil {
					return created, e
				}
				created += n
			}
		}
		return created, nil
	}
	if json.Valid(data) {
		n, err := nacosCsImportJSONArray(ns, policy, data)
		return n, err
	}
	return 0, errors.New("仅支持 JSON 数组或含 JSON 的 ZIP")
}

func nacosCsImportJSONArray(ns, policy string, raw []byte) (int, error) {
	var arr []map[string]interface{}
	if err := json.Unmarshal(raw, &arr); err != nil {
		return 0, err
	}
	created := 0
	for _, it := range arr {
		dataID, _ := it["dataId"].(string)
		group, _ := it["groupName"].(string)
		if g, ok := it["group"].(string); ok && group == "" {
			group = g
		}
		content, _ := it["content"].(string)
		typ, _ := it["type"].(string)
		if strings.TrimSpace(dataID) == "" || strings.TrimSpace(group) == "" {
			continue
		}
		var exist int64
		_ = model.DB.Model(&model.NacosCsConfig{}).Where("namespace_id = ? AND data_id = ? AND group_name = ?", ns, dataID, group).Count(&exist).Error
		if exist > 0 {
			if policy == "SKIP" {
				continue
			}
			if policy == "ABORT" {
				return created, fmt.Errorf("已存在 %s@@%s", group, dataID)
			}
		}
		if _, err := NacosCsPublish(ns, dataID, group, content, typ); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

// --- 插件 ---

func NacosConsolePluginList(pluginType string) ([]model.NacosConsolePlugin, error) {
	var rows []model.NacosConsolePlugin
	q := model.DB.Model(&model.NacosConsolePlugin{})
	if strings.TrimSpace(pluginType) != "" {
		q = q.Where("plugin_type = ?", strings.TrimSpace(pluginType))
	}
	if err := q.Order("plugin_id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func NacosConsolePluginSetStatus(pluginType, pluginName string, enabled bool) error {
	q := model.DB.Model(&model.NacosConsolePlugin{})
	if strings.TrimSpace(pluginType) != "" {
		q = q.Where("plugin_type = ?", pluginType)
	}
	if strings.TrimSpace(pluginName) != "" {
		q = q.Where("plugin_name = ?", pluginName)
	}
	return q.Update("enabled", enabled).Error
}

// --- 集群节点 ---

func NacosConsoleClusterList(keyword string) ([]model.NacosConsoleClusterNode, error) {
	var rows []model.NacosConsoleClusterNode
	q := model.DB.Order("address asc")
	if kw := strings.TrimSpace(strings.ToLower(keyword)); kw != "" {
		q = q.Where("LOWER(ip) LIKE ?", "%"+kw+"%")
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func NacosConsoleClusterLeave(addresses []string) error {
	for _, a := range addresses {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if err := model.DB.Where("address = ?", a).Delete(&model.NacosConsoleClusterNode{}).Error; err != nil {
			return err
		}
	}
	return nil
}

// --- MCP 从远端拉工具列表（JSON-RPC tools/list）---

func NacosMcpImportToolsList(transportType, baseURL, endpoint, authToken string) ([]map[string]interface{}, error) {
	u := strings.TrimSpace(baseURL)
	if u == "" {
		return nil, errors.New("baseUrl 必填")
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "http://" + u
	}
	path := strings.TrimSpace(endpoint)
	if path == "" {
		path = "/mcp"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	full := strings.TrimRight(u, "/") + path
	body := map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]interface{}{},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, full, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(authToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(authToken))
	}
	cli := client.NewOutboundHTTPClient(15 * time.Second)
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("MCP HTTP %d: %s", resp.StatusCode, string(rb))
	}
	var wrap struct {
		Result struct {
			Tools []map[string]interface{} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rb, &wrap); err != nil {
		return nil, fmt.Errorf("解析 MCP 响应失败: %w", err)
	}
	out := wrap.Result.Tools
	if out == nil {
		out = []map[string]interface{}{}
	}
	_ = transportType
	return out, nil
}
