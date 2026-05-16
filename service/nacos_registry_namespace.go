package service

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/songquanpeng/one-api/model"
)

var nacosNamespaceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,128}$`)

// NacosEnsureDefaultRegistryNamespace 保证存在 public 登记项。
func NacosEnsureDefaultRegistryNamespace() error {
	var n int64
	if err := model.DB.Model(&model.NacosRegistryNamespace{}).Where("namespace_id = ?", "public").Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return model.DB.Create(&model.NacosRegistryNamespace{
		NamespaceId: "public",
		Remark:      "default",
	}).Error
}

// NacosListRegistryNamespaceRows 仅登记表的命名空间（与 Nacos 原生「命名空间列表」一致）。
func NacosListRegistryNamespaceRows() ([]model.NacosRegistryNamespace, error) {
	if err := NacosEnsureDefaultRegistryNamespace(); err != nil {
		return nil, err
	}
	var rows []model.NacosRegistryNamespace
	err := model.DB.Order("namespace_id asc").Find(&rows).Error
	return rows, err
}

// NacosListNamespaceOptions 仅返回已登记命名空间 id，按 keyword 模糊过滤（与 Nacos 原生一致，不再合并配置/制品里出现但未登记的 id）。
func NacosListNamespaceOptions(keyword string) ([]string, error) {
	rows, err := NacosListRegistryNamespaceRows()
	if err != nil {
		return nil, err
	}
	kw := strings.ToLower(strings.TrimSpace(keyword))
	list := make([]string, 0, len(rows))
	for _, r := range rows {
		id := r.NamespaceId
		if kw != "" && !strings.Contains(strings.ToLower(id), kw) {
			continue
		}
		list = append(list, id)
	}
	sort.Strings(list)
	return list, nil
}

// NacosConsoleNamespaceItem 与 console-ui-next Namespace 对齐。
type NacosConsoleNamespaceItem struct {
	Namespace         string `json:"namespace"`
	NamespaceShowName string `json:"namespaceShowName"`
	NamespaceDesc     string `json:"namespaceDesc,omitempty"`
	Quota             int    `json:"quota"`
	ConfigCount       int    `json:"configCount"`
	Type              int    `json:"type"`
}

// nacosConsoleNamespaceType 与 console-ui-next 一致：0=public，非 0 为自定义。
func nacosConsoleNamespaceType(namespaceID string) int {
	if strings.EqualFold(strings.TrimSpace(namespaceID), "public") {
		return 0
	}
	return 1
}

// NacosConsoleNamespaceItemFromRegistryRow 由登记表行构造列表项（含配置条数统计）。
func NacosConsoleNamespaceItemFromRegistryRow(row model.NacosRegistryNamespace) NacosConsoleNamespaceItem {
	id := row.NamespaceId
	var cnt int64
	_ = model.DB.Model(&model.NacosCsConfig{}).Where("namespace_id = ?", id).Count(&cnt).Error
	return NacosConsoleNamespaceItem{
		Namespace:         id,
		NamespaceShowName: id,
		NamespaceDesc:     row.Remark,
		Quota:             128,
		ConfigCount:       int(cnt),
		Type:              nacosConsoleNamespaceType(id),
	}
}

// NacosGetConsoleNamespaceItem 单条详情（与 GET /v3/console/core/namespace 对齐）；未在登记表中登记则返回 gorm.ErrRecordNotFound。
func NacosGetConsoleNamespaceItem(namespaceID string) (NacosConsoleNamespaceItem, error) {
	ns := NormalizeNacosNamespaceID(namespaceID)
	if ns == "" {
		return NacosConsoleNamespaceItem{}, errors.New("namespaceId 必填")
	}
	var row model.NacosRegistryNamespace
	if err := model.DB.Where("namespace_id = ?", ns).First(&row).Error; err != nil {
		return NacosConsoleNamespaceItem{}, err
	}
	return NacosConsoleNamespaceItemFromRegistryRow(row), nil
}

// NacosListConsoleNamespaces 供原生控制台命名空间列表：仅已登记命名空间（对齐 Nacos 原生，与删除/编辑语义一致）。
func NacosListConsoleNamespaces() ([]NacosConsoleNamespaceItem, error) {
	rows, err := NacosListRegistryNamespaceRows()
	if err != nil {
		return nil, err
	}
	out := make([]NacosConsoleNamespaceItem, 0, len(rows))
	for i := range rows {
		out = append(out, NacosConsoleNamespaceItemFromRegistryRow(rows[i]))
	}
	return out, nil
}

// nacosNamespaceDeleteBlockReason 若命名空间下仍有配置、AI 资源或服务发现数据，返回非空原因（对齐 Nacos「非空不可删」）。
func nacosNamespaceDeleteBlockReason(ns string) (string, error) {
	type nsCheck struct {
		label string
		count func() (int64, error)
	}
	svc := model.NacosConsoleDiscoveryService{}
	inst := model.NacosConsoleDiscoveryInstance{}
	checks := []nsCheck{
		{"配置", func() (int64, error) {
			var n int64
			err := model.DB.Model(&model.NacosCsConfig{}).Where("namespace_id = ?", ns).Count(&n).Error
			return n, err
		}},
		{"配置历史", func() (int64, error) {
			var n int64
			err := model.DB.Model(&model.NacosCsConfigHistory{}).Where("namespace_id = ?", ns).Count(&n).Error
			return n, err
		}},
		{"灰度配置", func() (int64, error) {
			var n int64
			err := model.DB.Model(&model.NacosCsConfigBeta{}).Where("namespace_id = ?", ns).Count(&n).Error
			return n, err
		}},
		{"配置监听", func() (int64, error) {
			var n int64
			err := model.DB.Model(&model.NacosCsConfigListener{}).Where("namespace_id = ?", ns).Count(&n).Error
			return n, err
		}},
		{"AI 制品（Skill/AgentSpec 等）", func() (int64, error) {
			var n int64
			err := model.DB.Model(&model.NacosAIArtifact{}).Where("namespace_id = ?", ns).Count(&n).Error
			return n, err
		}},
		{"MCP 服务", func() (int64, error) {
			var n int64
			err := model.DB.Model(&model.NacosAIMcpServer{}).Where("namespace_id = ?", ns).Count(&n).Error
			return n, err
		}},
		{"A2A Agent", func() (int64, error) {
			var n int64
			err := model.DB.Model(&model.NacosAIA2AAgent{}).Where("namespace_id = ?", ns).Count(&n).Error
			return n, err
		}},
		{"Prompt", func() (int64, error) {
			var n int64
			err := model.DB.Model(&model.NacosAIPrompt{}).Where("namespace_id = ?", ns).Count(&n).Error
			return n, err
		}},
		{"服务发现", func() (int64, error) {
			var n int64
			err := model.DB.Model(&model.NacosConsoleDiscoveryService{}).Where("namespace_id = ?", ns).Count(&n).Error
			return n, err
		}},
		{"服务实例", func() (int64, error) {
			var n int64
			err := model.DB.Table(inst.TableName()+" AS i").
				Joins("JOIN "+svc.TableName()+" AS s ON s.id = i.service_id").
				Where("s.namespace_id = ?", ns).
				Count(&n).Error
			return n, err
		}},
		{"订阅者", func() (int64, error) {
			var n int64
			err := model.DB.Model(&model.NacosConsoleSubscriber{}).Where("namespace_id = ?", ns).Count(&n).Error
			return n, err
		}},
	}
	for _, c := range checks {
		n, err := c.count()
		if err != nil {
			return "", err
		}
		if n > 0 {
			return fmt.Sprintf("命名空间下仍存在%s，无法删除（请先清理后再删）", c.label), nil
		}
	}
	return "", nil
}

// NacosUpdateRegistryNamespaceRemark 更新登记表备注（namespace 已存在时）。
func NacosUpdateRegistryNamespaceRemark(namespaceID, remark string) error {
	ns := NormalizeNacosNamespaceID(namespaceID)
	if ns == "" {
		return errors.New("namespace 不能为空")
	}
	var row model.NacosRegistryNamespace
	if err := model.DB.Where("namespace_id = ?", ns).First(&row).Error; err != nil {
		return err
	}
	return model.DB.Model(&row).Update("remark", strings.TrimSpace(remark)).Error
}

// NacosDeleteRegistryNamespaceByNamespaceId 按 namespace_id 删除登记（非 public）；若仍有配置或业务数据则拒绝删除（对齐 Nacos 原生）。
func NacosDeleteRegistryNamespaceByNamespaceId(namespaceID string) error {
	ns := NormalizeNacosNamespaceID(namespaceID)
	if strings.EqualFold(ns, "public") {
		return errors.New("不能删除 public 命名空间")
	}
	reason, err := nacosNamespaceDeleteBlockReason(ns)
	if err != nil {
		return err
	}
	if reason != "" {
		return errors.New(reason)
	}
	return model.DB.Where("namespace_id = ?", ns).Delete(&model.NacosRegistryNamespace{}).Error
}

// NacosCreateRegistryNamespace 登记新 namespace（不创建远端数据，仅元数据）。
func NacosCreateRegistryNamespace(namespaceID, remark string) error {
	ns := NormalizeNacosNamespaceID(namespaceID)
	if ns == "" {
		return errors.New("namespace 不能为空")
	}
	if !nacosNamespaceIDPattern.MatchString(ns) {
		return fmt.Errorf("namespace 仅允许字母数字及 ._-，长度 1~128，得到 %q", ns)
	}
	if err := NacosEnsureDefaultRegistryNamespace(); err != nil {
		return err
	}
	var n int64
	if err := model.DB.Model(&model.NacosRegistryNamespace{}).Where("namespace_id = ?", ns).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("namespace %q 已登记", ns)
	}
	return model.DB.Create(&model.NacosRegistryNamespace{
		NamespaceId: ns,
		Remark:      strings.TrimSpace(remark),
	}).Error
}

// NacosDeleteRegistryNamespace 按登记表主键删除；不可删除 public。
func NacosDeleteRegistryNamespace(id int64) error {
	var row model.NacosRegistryNamespace
	if err := model.DB.First(&row, id).Error; err != nil {
		return err
	}
	if strings.EqualFold(row.NamespaceId, "public") {
		return errors.New("不能删除 public 命名空间")
	}
	reason, err := nacosNamespaceDeleteBlockReason(row.NamespaceId)
	if err != nil {
		return err
	}
	if reason != "" {
		return errors.New(reason)
	}
	return model.DB.Delete(&model.NacosRegistryNamespace{}, id).Error
}
