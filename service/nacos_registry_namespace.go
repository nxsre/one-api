package service

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/songquanpeng/one-api/model"
	"gorm.io/gorm"
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
	Namespace           string `json:"namespace"`
	NamespaceShowName   string `json:"namespaceShowName"`
	NamespaceDesc       string `json:"namespaceDesc,omitempty"`
	Quota               int    `json:"quota"`
	ConfigCount         int    `json:"configCount"`
	Type                int    `json:"type"`
	OwnerTenantID       *int   `json:"ownerTenantId,omitempty"`
	OwnerTenantName     string `json:"ownerTenantName,omitempty"` // 登记表 tenant_id 对应租户名称（平台列表展示）
	ExposeToTenants     bool   `json:"exposeToTenants"`
	LegacyPrimaryHint   bool   `json:"legacyPrimaryHint,omitempty"` // 由 tenant-{slug} 隐式空间生成、未在登记表落库时
}

func nacosTenantNamesByIDs(ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	var tenants []model.Tenant
	if err := model.DB.Select("id", "name").Where("id IN ?", ids).Find(&tenants).Error; err != nil {
		return nil, err
	}
	out := make(map[int]string, len(tenants))
	for i := range tenants {
		out[tenants[i].Id] = tenants[i].Name
	}
	return out, nil
}

func nacosFillTenantNamesOnItems(items []NacosConsoleNamespaceItem) {
	idSet := make(map[int]struct{})
	for i := range items {
		if items[i].OwnerTenantID != nil {
			idSet[*items[i].OwnerTenantID] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return
	}
	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	names, err := nacosTenantNamesByIDs(ids)
	if err != nil {
		return
	}
	for i := range items {
		if items[i].OwnerTenantID != nil {
			if n, ok := names[*items[i].OwnerTenantID]; ok {
				items[i].OwnerTenantName = n
			}
		}
	}
}

func nacosFillTenantNameItem(item NacosConsoleNamespaceItem) NacosConsoleNamespaceItem {
	if item.OwnerTenantID == nil {
		return item
	}
	m, err := nacosTenantNamesByIDs([]int{*item.OwnerTenantID})
	if err != nil {
		return item
	}
	if n, ok := m[*item.OwnerTenantID]; ok {
		item.OwnerTenantName = n
	}
	return item
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
		OwnerTenantID:     row.TenantID,
		ExposeToTenants:   row.ExposeToTenants && row.TenantID == nil,
	}
}

// NacosTenantLegacyNamespaceID 租户默认命名空间 id（与历史逻辑 tenant-{slug} 一致）。
func NacosTenantLegacyNamespaceID(t *model.Tenant) string {
	if t == nil {
		return ""
	}
	slug := strings.TrimSpace(t.Slug)
	if slug == "" {
		return ""
	}
	return "tenant-" + slug
}

// NacosConsoleNamespaceItemSynthetic 无登记表行时构造列表项（如隐式 legacy 空间）。
func NacosConsoleNamespaceItemSynthetic(namespaceID, desc string, owner *int, legacyHint bool) NacosConsoleNamespaceItem {
	id := NormalizeNacosNamespaceID(namespaceID)
	var cnt int64
	_ = model.DB.Model(&model.NacosCsConfig{}).Where("namespace_id = ?", id).Count(&cnt).Error
	return NacosConsoleNamespaceItem{
		Namespace:         id,
		NamespaceShowName: id,
		NamespaceDesc:     desc,
		Quota:             128,
		ConfigCount:       int(cnt),
		Type:              nacosConsoleNamespaceType(id),
		OwnerTenantID:     owner,
		ExposeToTenants:   false,
		LegacyPrimaryHint: legacyHint,
	}
}

// NacosTenantMayUseNamespace 租户是否可使用该命名空间（自有、平台开放、或隐式 tenant-{slug}）。
func NacosTenantMayUseNamespace(tenantID int, namespaceID string) (bool, error) {
	if tenantID <= 0 {
		return false, errors.New("租户 ID 无效")
	}
	ns := NormalizeNacosNamespaceID(namespaceID)
	if ns == "" {
		return false, nil
	}
	t, err := model.GetTenantByID(tenantID)
	if err != nil {
		return false, err
	}
	if t == nil {
		return false, errors.New("租户不存在")
	}
	if ns == NacosTenantLegacyNamespaceID(t) {
		return true, nil
	}
	var row model.NacosRegistryNamespace
	err = model.DB.Where("namespace_id = ?", ns).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if row.TenantID != nil && *row.TenantID == tenantID {
		return true, nil
	}
	if row.TenantID == nil && row.ExposeToTenants {
		return true, nil
	}
	return false, nil
}

// NacosAssertTenantOwnsRegistryNS 租户删除/独占更新前：须为登记表中 tenant_id 指向本租户的命名空间。
func NacosAssertTenantOwnsRegistryNS(tenantID int, namespaceID string) error {
	ns := NormalizeNacosNamespaceID(namespaceID)
	if ns == "" {
		return errors.New("namespace 无效")
	}
	t, err := model.GetTenantByID(tenantID)
	if err != nil {
		return err
	}
	if t != nil && ns == NacosTenantLegacyNamespaceID(t) {
		return errors.New("默认租户空间未登记时无法从命名空间列表删除；请先在名称空间管理中创建登记后再操作")
	}
	var row model.NacosRegistryNamespace
	if err := model.DB.Where("namespace_id = ?", ns).First(&row).Error; err != nil {
		return err
	}
	if row.TenantID == nil || *row.TenantID != tenantID {
		return errors.New("只能操作本租户创建的命名空间登记")
	}
	return nil
}

// NacosListRegistryNamespaceRowsForTenant 租户可见的登记表行（自有 + 平台勾选开放的）。
func NacosListRegistryNamespaceRowsForTenant(tenantID int) ([]model.NacosRegistryNamespace, error) {
	if err := NacosEnsureDefaultRegistryNamespace(); err != nil {
		return nil, err
	}
	var rows []model.NacosRegistryNamespace
	err := model.DB.Where("tenant_id = ? OR (tenant_id IS NULL AND expose_to_tenants = ?)", tenantID, true).
		Order("namespace_id asc").Find(&rows).Error
	return rows, err
}

// NacosListNamespaceOptionsForTenant keyword 过滤后与 NacosListNamespaceOptions 语义一致。
func NacosListNamespaceOptionsForTenant(tenantID int, keyword string) ([]string, error) {
	rows, err := NacosListRegistryNamespaceRowsForTenant(tenantID)
	if err != nil {
		return nil, err
	}
	t, err := model.GetTenantByID(tenantID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("租户不存在")
	}
	kw := strings.ToLower(strings.TrimSpace(keyword))
	list := make([]string, 0, len(rows)+1)
	leg := NacosTenantLegacyNamespaceID(t)
	if leg != "" && (kw == "" || strings.Contains(strings.ToLower(leg), kw)) {
		list = append(list, leg)
	}
	for _, r := range rows {
		id := r.NamespaceId
		if kw != "" && !strings.Contains(strings.ToLower(id), kw) {
			continue
		}
		if leg != "" && id == leg {
			continue
		}
		list = append(list, id)
	}
	sort.Strings(list)

	// 严格交叉校验：只返回 NacosTenantMayUseNamespace 也认可的命名空间，
	// 防止因逻辑漂移或数据不一致导致 options 接口返回 assert 会拒绝的选项。
	filtered := make([]string, 0, len(list))
	for _, id := range list {
		if ok, _ := NacosTenantMayUseNamespace(tenantID, id); ok {
			filtered = append(filtered, id)
		}
	}
	return filtered, nil
}

// NacosListConsoleNamespacesForTenant 租户控制台命名空间列表（含隐式 legacy）。
func NacosListConsoleNamespacesForTenant(tenantID int) ([]NacosConsoleNamespaceItem, error) {
	t, err := model.GetTenantByID(tenantID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("租户不存在")
	}
	rows, err := NacosListRegistryNamespaceRowsForTenant(tenantID)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	out := make([]NacosConsoleNamespaceItem, 0, len(rows)+1)
	leg := NacosTenantLegacyNamespaceID(t)
	if leg != "" {
		out = append(out, NacosConsoleNamespaceItemSynthetic(leg, "租户默认空间（与租户 slug 对应）", &tenantID, true))
		seen[leg] = struct{}{}
	}
	for i := range rows {
		id := rows[i].NamespaceId
		if _, ok := seen[id]; ok {
			continue
		}
		item := NacosConsoleNamespaceItemFromRegistryRow(rows[i])
		seen[id] = struct{}{}
		out = append(out, item)
	}
	nacosFillTenantNamesOnItems(out)
	return out, nil
}

// NacosGetConsoleNamespaceItem 单条详情（与 GET /v3/console/core/namespace 对齐）；未在登记表中登记则返回 gorm.ErrRecordNotFound。
// 与列表接口一致：先保证 public 默认登记存在（避免仅打开详情、未先拉列表时出现 record not found）。
func NacosGetConsoleNamespaceItem(namespaceID string) (NacosConsoleNamespaceItem, error) {
	ns := NormalizeNacosNamespaceID(namespaceID)
	if ns == "" {
		return NacosConsoleNamespaceItem{}, errors.New("namespaceId 必填")
	}
	if err := NacosEnsureDefaultRegistryNamespace(); err != nil {
		return NacosConsoleNamespaceItem{}, err
	}
	var row model.NacosRegistryNamespace
	if err := model.DB.Where("namespace_id = ?", ns).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) && strings.EqualFold(ns, "public") {
			return NacosConsoleNamespaceItemSynthetic("public", "default", nil, false), nil
		}
		return NacosConsoleNamespaceItem{}, err
	}
	return nacosFillTenantNameItem(NacosConsoleNamespaceItemFromRegistryRow(row)), nil
}

// NacosGetConsoleNamespaceItemScoped 租户上下文下读取详情；允许隐式 tenant-{slug} 无登记表行。
func NacosGetConsoleNamespaceItemScoped(tenantID int, tenantScoped bool, namespaceID string) (NacosConsoleNamespaceItem, error) {
	ns := NormalizeNacosNamespaceID(namespaceID)
	if ns == "" {
		return NacosConsoleNamespaceItem{}, errors.New("namespaceId 必填")
	}
	if tenantScoped {
		if err := NacosAssertTenantMayUseNamespace(tenantID, ns); err != nil {
			return NacosConsoleNamespaceItem{}, err
		}
	}
	item, err := NacosGetConsoleNamespaceItem(ns)
	if err == nil {
		return item, nil
	}
	if !tenantScoped || !errors.Is(err, gorm.ErrRecordNotFound) {
		return NacosConsoleNamespaceItem{}, err
	}
	t, e2 := model.GetTenantByID(tenantID)
	if e2 != nil || t == nil {
		return NacosConsoleNamespaceItem{}, err
	}
	if ns == NacosTenantLegacyNamespaceID(t) {
		return nacosFillTenantNameItem(NacosConsoleNamespaceItemSynthetic(ns, "租户默认空间（与租户 slug 对应）", &tenantID, true)), nil
	}
	return NacosConsoleNamespaceItem{}, err
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
	nacosFillTenantNamesOnItems(out)
	return out, nil
}

// NacosAssertTenantMayUseNamespace 与 NacosTenantMayUseNamespace 相同，不允许时返回错误（供 API 直接提示）。
func NacosAssertTenantMayUseNamespace(tenantID int, namespaceID string) error {
	ok, err := NacosTenantMayUseNamespace(tenantID, namespaceID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("无权访问命名空间 %q（仅限本租户自有、平台已对本租户开放的命名空间，或默认的 tenant-{slug}）", NormalizeNacosNamespaceID(namespaceID))
	}
	return nil
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

// NacosUpdateRegistryNamespacePlatformFields 仅平台命名空间：更新备注与/或「对租户开放」。
func NacosUpdateRegistryNamespacePlatformFields(namespaceID string, remark *string, expose *bool) error {
	ns := NormalizeNacosNamespaceID(namespaceID)
	if ns == "" {
		return errors.New("namespace 不能为空")
	}
	var row model.NacosRegistryNamespace
	if err := model.DB.Where("namespace_id = ?", ns).First(&row).Error; err != nil {
		return err
	}
	if row.TenantID != nil {
		return errors.New("仅平台创建的命名空间可配置「对租户开放」")
	}
	updates := map[string]interface{}{}
	if remark != nil {
		updates["remark"] = strings.TrimSpace(*remark)
	}
	if expose != nil {
		updates["expose_to_tenants"] = *expose
	}
	if len(updates) == 0 {
		return nil
	}
	return model.DB.Model(&row).Updates(updates).Error
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

// NacosCreateRegistryNamespace 登记新 namespace（平台创建，默认不对租户开放）。
func NacosCreateRegistryNamespace(namespaceID, remark string) error {
	return NacosCreateRegistryNamespaceExt(namespaceID, remark, nil, false)
}

// NacosCreateRegistryNamespaceExt ownerTenantID 非空表示租户创建；平台创建时可设 exposeToTenants（对全部租户开放）。
func NacosCreateRegistryNamespaceExt(namespaceID, remark string, ownerTenantID *int, exposeToTenants bool) error {
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
	expose := exposeToTenants && ownerTenantID == nil
	return model.DB.Create(&model.NacosRegistryNamespace{
		NamespaceId:      ns,
		Remark:           strings.TrimSpace(remark),
		TenantID:         ownerTenantID,
		ExposeToTenants:  expose,
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
