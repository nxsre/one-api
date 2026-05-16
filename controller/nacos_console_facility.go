package controller

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/service"
)

// --- 服务发现（/v3/console/ns/*）---

func NacosConsoleDiscoveryServiceList(c *gin.Context) {
	ns := nacosNamespace(c)
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	total, items, err := service.NacosDiscoveryListServices(ns, c.Query("serviceNameParam"), c.Query("groupNameParam"), pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, gin.H{"totalCount": total, "pageItems": items})
}

func NacosConsoleDiscoveryServiceGet(c *gin.Context) {
	ns := nacosNamespace(c)
	detail, err := service.NacosDiscoveryGetServiceDetail(ns, c.Query("groupName"), c.Query("serviceName"))
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, detail)
}

func NacosConsoleDiscoveryServicePost(c *gin.Context) {
	var body struct {
		NamespaceId        string            `json:"namespaceId"`
		ServiceName        string            `json:"serviceName"`
		GroupName          string            `json:"groupName"`
		Ephemeral          *bool             `json:"ephemeral"`
		ProtectThreshold   float64           `json:"protectThreshold"`
		Metadata           string            `json:"metadata"`
		Selector           string            `json:"selector"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	selT, selE := service.ParseDiscoverySelector(body.Selector)
	if err := service.NacosDiscoveryCreateService(service.DiscoveryServiceForm{
		NamespaceId:        body.NamespaceId,
		ServiceName:        body.ServiceName,
		GroupName:          body.GroupName,
		Ephemeral:          body.Ephemeral,
		ProtectThreshold:   body.ProtectThreshold,
		Metadata:           service.ParseDiscoveryMetadataString(body.Metadata),
		SelectorType:       selT,
		SelectorExpression: selE,
	}); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	service.NacosDiscoveryEnsureDemoSubscriber(service.NormalizeNacosNamespaceID(body.NamespaceId), strings.TrimSpace(body.GroupName), strings.TrimSpace(body.ServiceName))
	nacosV3OK(c, "ok")
}

func NacosConsoleDiscoveryServicePut(c *gin.Context) {
	var body struct {
		NamespaceId        string            `json:"namespaceId"`
		ServiceName        string            `json:"serviceName"`
		GroupName          string            `json:"groupName"`
		Ephemeral          *bool             `json:"ephemeral"`
		ProtectThreshold   float64           `json:"protectThreshold"`
		Metadata           string            `json:"metadata"`
		Selector           string            `json:"selector"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	selT, selE := service.ParseDiscoverySelector(body.Selector)
	if err := service.NacosDiscoveryUpdateService(service.DiscoveryServiceForm{
		NamespaceId:        body.NamespaceId,
		ServiceName:        body.ServiceName,
		GroupName:          body.GroupName,
		Ephemeral:          body.Ephemeral,
		ProtectThreshold:   body.ProtectThreshold,
		Metadata:           service.ParseDiscoveryMetadataString(body.Metadata),
		SelectorType:       selT,
		SelectorExpression: selE,
	}); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleDiscoveryServiceDelete(c *gin.Context) {
	ns := nacosNamespace(c)
	if err := service.NacosDiscoveryDeleteService(ns, c.Query("groupName"), c.Query("serviceName")); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleDiscoverySelectorTypes(c *gin.Context) {
	nacosV3OK(c, service.NacosDiscoverySelectorTypes())
}

func NacosConsoleDiscoveryClusterPut(c *gin.Context) {
	var body service.DiscoveryClusterUpdate
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	if err := service.NacosDiscoveryUpdateCluster(body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleDiscoveryInstanceList(c *gin.Context) {
	ns := nacosNamespace(c)
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	total, items, err := service.NacosDiscoveryListInstances(ns, c.Query("groupName"), c.Query("serviceName"), c.Query("clusterName"), pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, gin.H{"totalCount": total, "pageItems": items})
}

func NacosConsoleDiscoveryInstancePut(c *gin.Context) {
	var body service.DiscoveryInstanceForm
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	if err := service.NacosDiscoveryUpsertInstance(body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleDiscoveryInstanceDelete(c *gin.Context) {
	ns := nacosNamespace(c)
	port, _ := strconv.Atoi(c.Query("port"))
	ephem := service.ParseBoolQuery(c.Query("ephemeral"), true)
	if err := service.NacosDiscoveryDeleteInstance(ns, c.Query("groupName"), c.Query("serviceName"), c.Query("clusterName"), c.Query("ip"), port, ephem); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

func NacosConsoleDiscoverySubscribersList(c *gin.Context) {
	ns := nacosNamespace(c)
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	total, items, err := service.NacosDiscoveryListSubscribers(ns, c.Query("serviceName"), c.Query("groupName"), pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, gin.H{"totalCount": total, "pageItems": items})
}

// --- CS 扩展：监听 / 灰度 / 导入 / 克隆 ---

func NacosCsConfigListenerConsole(c *gin.Context) {
	ns := nacosNamespace(c)
	m := service.NacosCsListenerMap(ns, c.Query("dataId"), c.Query("groupName"), c.Query("ip"))
	qt := "config"
	if strings.Contains(c.Request.URL.Path, "/listener/ip") {
		qt = "ip"
	}
	nacosV3OK(c, gin.H{"listenersStatus": m, "queryType": qt})
}

func NacosCsConfigBetaGetConsole(c *gin.Context) {
	ns := nacosNamespace(c)
	b, err := service.NacosCsBetaGet(ns, c.Query("dataId"), c.Query("groupName"))
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, b)
}

func NacosCsConfigBetaDeleteConsole(c *gin.Context) {
	ns := nacosNamespace(c)
	if err := service.NacosCsBetaDelete(ns, c.Query("dataId"), c.Query("groupName")); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosCsConfigImportConsole(c *gin.Context) {
	ns := nacosNamespace(c)
	policy := c.Query("policy")
	fh, err := c.FormFile("file")
	if err != nil {
		nacosV3Err(c, 400, "缺少上传文件 file")
		return
	}
	n, err := service.NacosCsImportFromMultipart(ns, policy, fh)
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, gin.H{"created": n})
}

func NacosCsConfigCloneConsole(c *gin.Context) {
	ns := nacosNamespace(c)
	dst := c.Query("targetNamespaceId")
	policy := c.Query("policy")
	var items []service.NacosCsCloneItem
	if err := json.NewDecoder(c.Request.Body).Decode(&items); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	cloned, skipped, err := service.NacosCsCloneConfigs(ns, dst, policy, items)
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, gin.H{"cloned": cloned, "skipped": skipped})
}

// --- 插件 / 集群 ---

func NacosConsolePluginListConsole(c *gin.Context) {
	rows, err := service.NacosConsolePluginList(c.Query("pluginType"))
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"pluginId":           r.PluginId,
			"pluginName":         r.PluginName,
			"pluginType":         r.PluginType,
			"enabled":            r.Enabled,
			"critical":           r.Critical,
			"configurable":       r.Configurable,
			"exclusive":          r.Exclusive,
			"availableNodeCount": r.AvailableNodeCount,
			"totalNodeCount":     r.TotalNodeCount,
		})
	}
	nacosV3OK(c, out)
}

func NacosConsolePluginStatusPut(c *gin.Context) {
	en := service.ParseBoolQuery(c.Query("enabled"), false)
	if err := service.NacosConsolePluginSetStatus(c.Query("pluginType"), c.Query("pluginName"), en); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func NacosConsoleClusterNodesList(c *gin.Context) {
	rows, err := service.NacosConsoleClusterList(c.Query("keyword"))
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"ip":      r.Ip,
			"port":    r.Port,
			"state":   r.State,
			"address": r.Address,
		})
	}
	nacosV3OK(c, out)
}

func NacosConsoleClusterServerLeave(c *gin.Context) {
	var addrs []string
	if err := json.NewDecoder(c.Request.Body).Decode(&addrs); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	if err := service.NacosConsoleClusterLeave(addrs); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, "ok")
}

// --- MCP 工具列表 ---

func NacosConsoleMcpImportToolsList(c *gin.Context) {
	tools, err := service.NacosMcpImportToolsList(c.Query("transportType"), c.Query("baseUrl"), c.Query("endpoint"), c.Query("authToken"))
	if err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	out := make([]gin.H, 0, len(tools))
	for _, t := range tools {
		name, _ := t["name"].(string)
		desc, _ := t["description"].(string)
		h := gin.H{"name": name, "description": desc}
		if s, ok := t["inputSchema"].(map[string]interface{}); ok {
			h["inputSchema"] = s
		}
		out = append(out, h)
	}
	nacosV3OK(c, out)
}
