package model

import "time"

// NacosAIMcpServer MCP 服务元数据（SpecJSON 存完整 JSON，对齐 nacos-cli / OpenAPI 形态）。
type NacosAIMcpServer struct {
	Id          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId string    `json:"namespaceId" gorm:"size:128;uniqueIndex:uix_nacos_mcp_ns_name;not null;default:''"`
	ServerName  string    `json:"serverName" gorm:"size:256;uniqueIndex:uix_nacos_mcp_ns_name;not null"`
	Description string `json:"description" gorm:"type:text"`
	SpecJSON    string `json:"-" gorm:"column:spec_json;type:text"`
	// SpecStorageKind 与 Skill ZIP 一致：db | local | s3；非 db 时 SpecJSON 通常为空，内容见 SpecStorageRef。
	SpecStorageKind string `json:"-" gorm:"column:spec_storage_kind;size:16;default:''"`
	SpecStorageRef   string `json:"-" gorm:"column:spec_storage_ref;size:1024"`
	BizTags          string `json:"bizTags" gorm:"size:512"`
	LabelsJSON  string    `json:"-" gorm:"column:labels_json;type:text"`
	Scope       string    `json:"scope" gorm:"size:32;default:'PUBLIC';not null"`
	Enable      bool      `json:"enable" gorm:"default:true;not null"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (NacosAIMcpServer) TableName() string { return "nacos_ai_mcp_servers" }

// NacosAIA2AAgent A2A Agent 卡片（CardJSON 存完整 agent card）。
type NacosAIA2AAgent struct {
	Id          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId string    `json:"namespaceId" gorm:"size:128;uniqueIndex:uix_nacos_a2a_ns_name;not null;default:''"`
	AgentName   string    `json:"agentName" gorm:"size:256;uniqueIndex:uix_nacos_a2a_ns_name;not null"`
	Description string `json:"description" gorm:"type:text"`
	CardJSON    string `json:"-" gorm:"column:card_json;type:text"`
	CardStorageKind string `json:"-" gorm:"column:card_storage_kind;size:16;default:''"`
	CardStorageRef   string `json:"-" gorm:"column:card_storage_ref;size:1024"`
	BizTags          string `json:"bizTags" gorm:"size:512"`
	LabelsJSON  string    `json:"-" gorm:"column:labels_json;type:text"`
	Scope       string    `json:"scope" gorm:"size:32;default:'PUBLIC';not null"`
	Enable      bool      `json:"enable" gorm:"default:true;not null"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (NacosAIA2AAgent) TableName() string { return "nacos_ai_a2a_agents" }

// NacosAIPrompt Prompt 资源头（多版本见 NacosAIPromptVersion）。
type NacosAIPrompt struct {
	Id          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId string    `json:"namespaceId" gorm:"size:128;uniqueIndex:uix_nacos_prompt_ns_key;not null;default:''"`
	PromptKey   string    `json:"promptKey" gorm:"size:256;uniqueIndex:uix_nacos_prompt_ns_key;not null"`
	Description string    `json:"description" gorm:"type:text"`
	BizTags     string    `json:"bizTags" gorm:"size:512"`
	LabelsJSON  string    `json:"-" gorm:"column:labels_json;type:text"`
	Scope       string    `json:"scope" gorm:"size:32;default:'PUBLIC';not null"`
	Enable      bool      `json:"enable" gorm:"default:true;not null"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (NacosAIPrompt) TableName() string { return "nacos_ai_prompts" }

// NacosAIPromptVersion Prompt 版本（content_json 为模板/变量等 JSON）。
type NacosAIPromptVersion struct {
	Id          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	PromptId    int64     `json:"promptId" gorm:"column:prompt_id;uniqueIndex:uix_nacos_prompt_ver;not null"`
	Version     string    `json:"version" gorm:"size:64;uniqueIndex:uix_nacos_prompt_ver;not null"`
	Status               string    `json:"status" gorm:"size:32;not null;default:'editing'"` // editing | reviewing | online | offline
	ContentJSON          string    `json:"-" gorm:"column:content_json;type:text"`
	ContentStorageKind   string    `json:"-" gorm:"column:content_storage_kind;size:16;default:''"`
	ContentStorageRef    string    `json:"-" gorm:"column:content_storage_ref;size:1024"`
	CreatedAt            time.Time `json:"createTime"`
	UpdatedAt            time.Time `json:"updateTime"`
}

func (NacosAIPromptVersion) TableName() string { return "nacos_ai_prompt_versions" }

// NacosAIPipelineRun 流水线/校验任务记录。
type NacosAIPipelineRun struct {
	Id               int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId      string    `json:"namespaceId" gorm:"size:128;index;not null;default:''"`
	JobType          string    `json:"jobType" gorm:"size:64;not null"` // registry_scan | manual
	ResourceKind     string    `json:"resourceKind" gorm:"size:32"`
	ResourceName     string    `json:"resourceName" gorm:"size:256"`
	ResourceVersion  string    `json:"resourceVersion" gorm:"size:64"`
	Status           string    `json:"status" gorm:"size:32;not null"` // pending | running | success | failed
	Message          string    `json:"message" gorm:"type:text"`
	DetailJSON       string    `json:"-" gorm:"column:detail_json;type:text"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (NacosAIPipelineRun) TableName() string { return "nacos_ai_pipeline_runs" }
