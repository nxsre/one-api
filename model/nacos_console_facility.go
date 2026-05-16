package model

import "time"

// --- 嵌入控制台：服务发现（简化版 Nacos Naming，持久化到 DB）---

type NacosConsoleDiscoveryService struct {
	Id                 int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId        string    `json:"namespaceId" gorm:"size:128;uniqueIndex:uix_nc_disc_svc;not null;default:''"`
	GroupName          string    `json:"groupName" gorm:"size:256;uniqueIndex:uix_nc_disc_svc;not null;default:'DEFAULT_GROUP'"`
	ServiceName        string    `json:"serviceName" gorm:"size:256;uniqueIndex:uix_nc_disc_svc;not null"`
	Ephemeral          bool      `json:"ephemeral" gorm:"not null;default:true"`
	ProtectThreshold   float64   `json:"protectThreshold" gorm:"not null;default:0"`
	MetadataJSON       string    `json:"-" gorm:"column:metadata_json;type:text"` // object JSON
	SelectorType       string    `json:"selectorType" gorm:"size:64;default:'none'"`
	SelectorExpression string    `json:"selectorExpression" gorm:"size:512"`
	ClusterProfilesJSON string   `json:"-" gorm:"column:cluster_profiles_json;type:text"` // clusterName -> health/profile JSON
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

func (NacosConsoleDiscoveryService) TableName() string { return "nacos_console_discovery_services" }

type NacosConsoleDiscoveryInstance struct {
	Id          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ServiceId   int64     `json:"serviceId" gorm:"column:service_id;index;not null"`
	ClusterName string    `json:"clusterName" gorm:"size:128;not null;default:'DEFAULT'"`
	Ip          string    `json:"ip" gorm:"size:64;not null"`
	Port        int       `json:"port" gorm:"not null"`
	Weight      int       `json:"weight" gorm:"not null;default:100"`
	Healthy     bool      `json:"healthy" gorm:"not null;default:true"`
	Enabled     bool      `json:"enabled" gorm:"not null;default:true"`
	Ephemeral   bool      `json:"ephemeral" gorm:"not null;default:true"`
	MetadataJSON string   `json:"-" gorm:"column:metadata_json;type:text"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (NacosConsoleDiscoveryInstance) TableName() string { return "nacos_console_discovery_instances" }

type NacosConsoleSubscriber struct {
	Id              int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId     string    `json:"namespaceId" gorm:"size:128;index;not null"`
	GroupName       string    `json:"groupName" gorm:"size:256;not null"`
	ServiceName     string    `json:"serviceName" gorm:"size:256;not null"`
	SubscriberName  string    `json:"subscriberName" gorm:"size:256;not null"`
	SubscribeCount  int       `json:"subscribeCount" gorm:"not null;default:1"`
	Clusters        string    `json:"clusters" gorm:"size:512"` // comma-separated
	CreatedAt       time.Time `json:"createdAt"`
}

func (NacosConsoleSubscriber) TableName() string { return "nacos_console_subscribers" }

// --- 配置灰度（beta）---

type NacosCsConfigBeta struct {
	Id          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId string  `json:"namespaceId" gorm:"size:128;uniqueIndex:uix_nc_cs_beta;not null"`
	DataId      string    `json:"dataId" gorm:"size:256;uniqueIndex:uix_nc_cs_beta;not null"`
	GroupName   string    `json:"groupName" gorm:"size:256;uniqueIndex:uix_nc_cs_beta;not null"`
	BetaIps     string    `json:"betaIps" gorm:"size:1024"` // comma-separated
	Content     string    `json:"content" gorm:"type:longtext"`
	Type        string    `json:"type" gorm:"size:32"`
	AppName     string    `json:"appName" gorm:"size:256"`
	Desc        string    `json:"desc" gorm:"size:512"`
	ConfigTags  string    `json:"configTags" gorm:"size:512"`
	GrayRule    string    `json:"grayRule" gorm:"size:512"`
	Md5         string    `json:"md5" gorm:"size:64"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (NacosCsConfigBeta) TableName() string { return "nacos_cs_config_betas" }

// --- 配置监听（长轮询客户端占位，便于控制台展示）---

type NacosCsConfigListener struct {
	Id          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId string    `json:"namespaceId" gorm:"size:128;index;not null"`
	DataId      string    `json:"dataId" gorm:"size:256;index;not null"`
	GroupName   string    `json:"groupName" gorm:"size:256;index;not null"`
	ClientId    string    `json:"clientId" gorm:"size:256;not null"`
	Ip          string    `json:"ip" gorm:"size:64"`
	AppName     string    `json:"appName" gorm:"size:256"`
	QueryType   string    `json:"queryType" gorm:"size:16;default:'config'"` // config | ip
	Status      string    `json:"status" gorm:"size:32;default:'UP'"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (NacosCsConfigListener) TableName() string { return "nacos_cs_config_listeners" }

// --- 插件市场（控制台开关）---

type NacosConsolePlugin struct {
	PluginId           string `json:"pluginId" gorm:"size:128;primaryKey"`
	PluginName         string `json:"pluginName" gorm:"size:256;not null"`
	PluginType         string `json:"pluginType" gorm:"size:64;not null"`
	Enabled            bool   `json:"enabled" gorm:"not null;default:false"`
	Critical           bool   `json:"critical" gorm:"not null;default:false"`
	Configurable       bool   `json:"configurable" gorm:"not null;default:true"`
	Exclusive          bool   `json:"exclusive" gorm:"not null;default:false"`
	AvailableNodeCount int    `json:"availableNodeCount" gorm:"not null;default:1"`
	TotalNodeCount     int    `json:"totalNodeCount" gorm:"not null;default:1"`
}

func (NacosConsolePlugin) TableName() string { return "nacos_console_plugins" }

// --- 集群成员（控制台展示）---

type NacosConsoleClusterNode struct {
	Address   string    `json:"address" gorm:"size:256;primaryKey"`
	Ip        string    `json:"ip" gorm:"size:64;not null"`
	Port      int       `json:"port" gorm:"not null"`
	State     string    `json:"state" gorm:"size:32;not null;default:'UP'"`
	ExtendJSON string   `json:"-" gorm:"column:extend_json;type:text"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (NacosConsoleClusterNode) TableName() string { return "nacos_console_cluster_nodes" }
