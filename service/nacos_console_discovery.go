package service

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/songquanpeng/one-api/model"
	"gorm.io/gorm"
)

type discoveryClusterProfile struct {
	CheckPort            int                    `json:"checkPort"`
	UseInstancePort4Check bool                  `json:"useInstancePort4Check"`
	HealthChecker        map[string]interface{} `json:"healthChecker"`
	Metadata             map[string]string      `json:"metadata"`
}

func parseMetadataMap(s string) map[string]string {
	s = strings.TrimSpace(s)
	if s == "" {
		return map[string]string{}
	}
	var m map[string]string
	_ = json.Unmarshal([]byte(s), &m)
	if m == nil {
		return map[string]string{}
	}
	return m
}

func marshalMetadataMap(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func parseClusterProfiles(s string) map[string]discoveryClusterProfile {
	out := map[string]discoveryClusterProfile{}
	s = strings.TrimSpace(s)
	if s == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func marshalClusterProfiles(m map[string]discoveryClusterProfile) string {
	if len(m) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// NacosDiscoveryListServices 分页列出服务摘要。
func NacosDiscoveryListServices(namespace, nameFilter, groupFilter string, pageNo, pageSize int) (total int64, items []map[string]interface{}, err error) {
	ns := NormalizeNacosNamespaceID(namespace)
	q := model.DB.Model(&model.NacosConsoleDiscoveryService{}).Where("namespace_id = ?", ns)
	if strings.TrimSpace(nameFilter) != "" {
		q = q.Where("service_name LIKE ?", "%"+strings.TrimSpace(nameFilter)+"%")
	}
	if strings.TrimSpace(groupFilter) != "" {
		q = q.Where("group_name LIKE ?", "%"+strings.TrimSpace(groupFilter)+"%")
	}
	if err = q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var rows []model.NacosConsoleDiscoveryService
	offset := (pageNo - 1) * pageSize
	if err = q.Order("service_name asc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	items = make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		var inst []model.NacosConsoleDiscoveryInstance
		_ = model.DB.Where("service_id = ?", r.Id).Find(&inst).Error
		clusters := map[string]struct{}{}
		ipCnt := 0
		healthy := 0
		for _, in := range inst {
			clusters[in.ClusterName] = struct{}{}
			ipCnt++
			if in.Healthy {
				healthy++
			}
		}
		items = append(items, map[string]interface{}{
			"name":                   r.ServiceName,
			"groupName":              r.GroupName,
			"clusterCount":           len(clusters),
			"ipCount":                ipCnt,
			"healthyInstanceCount":   healthy,
			"triggerFlag":            false,
		})
	}
	return total, items, nil
}

func findDiscoveryService(ns, group, name string) (*model.NacosConsoleDiscoveryService, error) {
	var r model.NacosConsoleDiscoveryService
	err := model.DB.Where("namespace_id = ? AND group_name = ? AND service_name = ?", ns, group, name).First(&r).Error
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// NacosDiscoveryGetServiceDetail 返回 console ServiceDetailInfo 结构（map）。
func NacosDiscoveryGetServiceDetail(namespace, group, name string) (map[string]interface{}, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	svc, err := findDiscoveryService(ns, group, name)
	if err != nil {
		return nil, err
	}
	var inst []model.NacosConsoleDiscoveryInstance
	if err := model.DB.Where("service_id = ?", svc.Id).Order("cluster_name asc, ip asc").Find(&inst).Error; err != nil {
		return nil, err
	}
	profiles := parseClusterProfiles(svc.ClusterProfilesJSON)
	byCluster := map[string][]model.NacosConsoleDiscoveryInstance{}
	for _, in := range inst {
		cn := strings.TrimSpace(in.ClusterName)
		if cn == "" {
			cn = "DEFAULT"
		}
		byCluster[cn] = append(byCluster[cn], in)
	}
	clusterMap := map[string]interface{}{}
	for cn, hosts := range byCluster {
		prof, ok := profiles[cn]
		if !ok {
			prof = discoveryClusterProfile{
				CheckPort:             0,
				UseInstancePort4Check: false,
				HealthChecker: map[string]interface{}{
					"type": "TCP",
				},
				Metadata: map[string]string{},
			}
		}
		hlist := make([]map[string]interface{}, 0, len(hosts))
		for _, h := range hosts {
			meta := parseMetadataMap(h.MetadataJSON)
			hlist = append(hlist, map[string]interface{}{
				"ip": h.Ip, "port": h.Port, "weight": h.Weight,
				"healthy": h.Healthy, "enabled": h.Enabled, "ephemeral": h.Ephemeral,
				"clusterName": cn, "serviceName": svc.ServiceName, "metadata": meta,
			})
		}
		hc := prof.HealthChecker
		if hc == nil {
			hc = map[string]interface{}{"type": "TCP"}
		}
		clusterMap[cn] = map[string]interface{}{
			"healthChecker":        hc,
			"metadata":             prof.Metadata,
			"hosts":                hlist,
			"healthyCheckPort":     prof.CheckPort,
			"useInstancePortForCheck": prof.UseInstancePort4Check,
		}
	}
	if len(clusterMap) == 0 {
		clusterMap["DEFAULT"] = map[string]interface{}{
			"healthChecker":        map[string]interface{}{"type": "TCP"},
			"metadata":             map[string]string{},
			"hosts":                []map[string]interface{}{},
			"healthyCheckPort":     0,
			"useInstancePortForCheck": false,
		}
	}
	return map[string]interface{}{
		"serviceName":      svc.ServiceName,
		"groupName":        svc.GroupName,
		"namespaceId":      svc.NamespaceId,
		"protectThreshold": svc.ProtectThreshold,
		"metadata":         parseMetadataMap(svc.MetadataJSON),
		"selector": map[string]interface{}{
			"type":       svc.SelectorType,
			"expression": svc.SelectorExpression,
		},
		"ephemeral":   svc.Ephemeral,
		"clusterMap":  clusterMap,
	}, nil
}

type DiscoveryServiceForm struct {
	NamespaceId        string
	ServiceName        string
	GroupName          string
	Ephemeral          *bool
	ProtectThreshold   float64
	Metadata           map[string]string
	SelectorType       string
	SelectorExpression string
}

func NacosDiscoveryCreateService(f DiscoveryServiceForm) error {
	ns := NormalizeNacosNamespaceID(f.NamespaceId)
	g := strings.TrimSpace(f.GroupName)
	if g == "" {
		g = "DEFAULT_GROUP"
	}
	name := strings.TrimSpace(f.ServiceName)
	if name == "" {
		return errors.New("serviceName 必填")
	}
	ephem := true
	if f.Ephemeral != nil {
		ephem = *f.Ephemeral
	}
	selT := strings.TrimSpace(f.SelectorType)
	if selT == "" {
		selT = "none"
	}
	row := model.NacosConsoleDiscoveryService{
		NamespaceId:         ns,
		GroupName:           g,
		ServiceName:         name,
		Ephemeral:           ephem,
		ProtectThreshold:    f.ProtectThreshold,
		MetadataJSON:        marshalMetadataMap(f.Metadata),
		SelectorType:        selT,
		SelectorExpression:  strings.TrimSpace(f.SelectorExpression),
		ClusterProfilesJSON: "{}",
	}
	return model.DB.Create(&row).Error
}

func NacosDiscoveryUpdateService(f DiscoveryServiceForm) error {
	ns := NormalizeNacosNamespaceID(f.NamespaceId)
	g := strings.TrimSpace(f.GroupName)
	if g == "" {
		g = "DEFAULT_GROUP"
	}
	svc, err := findDiscoveryService(ns, g, strings.TrimSpace(f.ServiceName))
	if err != nil {
		return err
	}
	up := map[string]interface{}{
		"protect_threshold":    f.ProtectThreshold,
		"metadata_json":      marshalMetadataMap(f.Metadata),
		"selector_type":      strings.TrimSpace(f.SelectorType),
		"selector_expression": strings.TrimSpace(f.SelectorExpression),
	}
	if f.Ephemeral != nil {
		up["ephemeral"] = *f.Ephemeral
	}
	return model.DB.Model(svc).Updates(up).Error
}

func NacosDiscoveryDeleteService(namespace, group, name string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	svc, err := findDiscoveryService(ns, group, name)
	if err != nil {
		return err
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("service_id = ?", svc.Id).Delete(&model.NacosConsoleDiscoveryInstance{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.NacosConsoleDiscoveryService{}, svc.Id).Error
	})
}

type DiscoveryInstanceForm struct {
	NamespaceId string
	ServiceName string
	GroupName   string
	ClusterName string
	Ip          string
	Port        int
	Weight      int
	Enabled     bool
	Ephemeral   bool
	Healthy     bool
	Metadata    map[string]string
}

func NacosDiscoveryUpsertInstance(f DiscoveryInstanceForm) error {
	ns := NormalizeNacosNamespaceID(f.NamespaceId)
	g := strings.TrimSpace(f.GroupName)
	if g == "" {
		g = "DEFAULT_GROUP"
	}
	svc, err := findDiscoveryService(ns, g, strings.TrimSpace(f.ServiceName))
	if err != nil {
		return err
	}
	cn := strings.TrimSpace(f.ClusterName)
	if cn == "" {
		cn = "DEFAULT"
	}
	ip := strings.TrimSpace(f.Ip)
	if ip == "" || f.Port <= 0 {
		return errors.New("ip 与 port 必填")
	}
	var row model.NacosConsoleDiscoveryInstance
	err = model.DB.Where("service_id = ? AND cluster_name = ? AND ip = ? AND port = ?", svc.Id, cn, ip, f.Port).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = model.NacosConsoleDiscoveryInstance{
			ServiceId: svc.Id, ClusterName: cn, Ip: ip, Port: f.Port,
			Weight: f.Weight, Healthy: f.Healthy, Enabled: f.Enabled, Ephemeral: f.Ephemeral,
			MetadataJSON: marshalMetadataMap(f.Metadata),
		}
		if row.Weight <= 0 {
			row.Weight = 100
		}
		return model.DB.Create(&row).Error
	}
	if err != nil {
		return err
	}
	return model.DB.Model(&row).Updates(map[string]interface{}{
		"weight": f.Weight, "healthy": f.Healthy, "enabled": f.Enabled, "ephemeral": f.Ephemeral,
		"metadata_json": marshalMetadataMap(f.Metadata),
	}).Error
}

func NacosDiscoveryDeleteInstance(namespace, group, svcName, cluster, ip string, port int, ephemeral bool) error {
	ns := NormalizeNacosNamespaceID(namespace)
	svc, err := findDiscoveryService(ns, group, svcName)
	if err != nil {
		return err
	}
	q := model.DB.Where("service_id = ? AND ip = ? AND port = ?", svc.Id, strings.TrimSpace(ip), port)
	if strings.TrimSpace(cluster) != "" {
		q = q.Where("cluster_name = ?", cluster)
	}
	return q.Delete(&model.NacosConsoleDiscoveryInstance{}).Error
}

func NacosDiscoveryListInstances(namespace, group, svcName, cluster string, pageNo, pageSize int) (total int64, pageItems []map[string]interface{}, err error) {
	ns := NormalizeNacosNamespaceID(namespace)
	svc, err := findDiscoveryService(ns, group, svcName)
	if err != nil {
		return 0, nil, err
	}
	q := model.DB.Model(&model.NacosConsoleDiscoveryInstance{}).Where("service_id = ?", svc.Id)
	if strings.TrimSpace(cluster) != "" {
		q = q.Where("cluster_name = ?", cluster)
	}
	if err = q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var rows []model.NacosConsoleDiscoveryInstance
	offset := (pageNo - 1) * pageSize
	if err = q.Order("id desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	pageItems = make([]map[string]interface{}, 0, len(rows))
	for _, h := range rows {
		cn := h.ClusterName
		if strings.TrimSpace(cn) == "" {
			cn = "DEFAULT"
		}
		pageItems = append(pageItems, map[string]interface{}{
			"ip": h.Ip, "port": h.Port, "weight": h.Weight,
			"healthy": h.Healthy, "enabled": h.Enabled, "ephemeral": h.Ephemeral,
			"clusterName": cn, "serviceName": svc.ServiceName,
			"metadata":    parseMetadataMap(h.MetadataJSON),
		})
	}
	return total, pageItems, nil
}

type DiscoveryClusterUpdate struct {
	NamespaceId           string
	ServiceName           string
	GroupName             string
	ClusterName           string
	CheckPort             int
	UseInstancePort4Check bool
	HealthChecker         string // JSON string
	Metadata              string // JSON string
}

func NacosDiscoveryUpdateCluster(u DiscoveryClusterUpdate) error {
	ns := NormalizeNacosNamespaceID(u.NamespaceId)
	g := strings.TrimSpace(u.GroupName)
	if g == "" {
		g = "DEFAULT_GROUP"
	}
	svc, err := findDiscoveryService(ns, g, strings.TrimSpace(u.ServiceName))
	if err != nil {
		return err
	}
	cn := strings.TrimSpace(u.ClusterName)
	if cn == "" {
		cn = "DEFAULT"
	}
	profiles := parseClusterProfiles(svc.ClusterProfilesJSON)
	p := profiles[cn]
	p.CheckPort = u.CheckPort
	p.UseInstancePort4Check = u.UseInstancePort4Check
	if strings.TrimSpace(u.HealthChecker) != "" {
		var hc map[string]interface{}
		if json.Unmarshal([]byte(u.HealthChecker), &hc) == nil {
			p.HealthChecker = hc
		}
	}
	if strings.TrimSpace(u.Metadata) != "" {
		var md map[string]string
		if json.Unmarshal([]byte(u.Metadata), &md) == nil {
			p.Metadata = md
		}
	}
	profiles[cn] = p
	return model.DB.Model(svc).Update("cluster_profiles_json", marshalClusterProfiles(profiles)).Error
}

func NacosDiscoveryListSubscribers(namespace, svcName, group string, pageNo, pageSize int) (total int64, pageItems []map[string]interface{}, err error) {
	ns := NormalizeNacosNamespaceID(namespace)
	q := model.DB.Model(&model.NacosConsoleSubscriber{}).Where("namespace_id = ?", ns)
	if strings.TrimSpace(svcName) != "" {
		q = q.Where("service_name LIKE ?", "%"+strings.TrimSpace(svcName)+"%")
	}
	if strings.TrimSpace(group) != "" {
		q = q.Where("group_name = ?", group)
	}
	if err = q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var rows []model.NacosConsoleSubscriber
	offset := (pageNo - 1) * pageSize
	if err = q.Order("id desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	pageItems = make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		pageItems = append(pageItems, map[string]interface{}{
			"subscriberName": r.SubscriberName,
			"groupName":      r.GroupName,
			"serviceName":    r.ServiceName,
			"namespaceId":    r.NamespaceId,
			"subscribeCount": r.SubscribeCount,
			"clusters":       r.Clusters,
		})
	}
	return total, pageItems, nil
}

// NacosDiscoveryEnsureDemoSubscriber 创建服务后可选调用，便于控制台非空展示。
func NacosDiscoveryEnsureDemoSubscriber(namespace, group, svcName string) {
	ns := NormalizeNacosNamespaceID(namespace)
	var n int64
	_ = model.DB.Model(&model.NacosConsoleSubscriber{}).
		Where("namespace_id = ? AND group_name = ? AND service_name = ?", ns, group, svcName).Count(&n).Error
	if n > 0 {
		return
	}
	_ = model.DB.Create(&model.NacosConsoleSubscriber{
		NamespaceId: ns, GroupName: group, ServiceName: svcName,
		SubscriberName: "console-demo-client", SubscribeCount: 1, Clusters: "DEFAULT",
	}).Error
}

func NacosDiscoverySelectorTypes() []string {
	return []string{"none", "label", "expression", "metadata", "tag"}
}

// ParseDiscoveryMetadataString 解析 ServiceFormData.metadata 字符串（JSON 或空）。
func ParseDiscoveryMetadataString(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}
	}
	var m map[string]string
	if json.Unmarshal([]byte(raw), &m) == nil && m != nil {
		return m
	}
	return map[string]string{"raw": raw}
}

// ParseDiscoverySelector 解析 selector 字符串为 type/expression。
func ParseDiscoverySelector(raw string) (typ, expr string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "none", ""
	}
	var obj struct {
		Type       string `json:"type"`
		Expression string `json:"expression"`
	}
	if json.Unmarshal([]byte(raw), &obj) == nil {
		if strings.TrimSpace(obj.Type) != "" {
			return obj.Type, obj.Expression
		}
	}
	return "none", raw
}

func ParseBoolQuery(s string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return def
	}
}

func ParseFloat(s string, def float64) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return f
}
