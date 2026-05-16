package model

import (
	"time"
)

// Nacos AI 本地仓库：与 nacos-cli 的 skill / agentspec 命令兼容。

const (
	NacosAIKindSkill      = "skill"
	NacosAIKindAgentSpec = "agentspec"
)

const (
	NacosAIVersionEditing    = "editing"
	NacosAIVersionReviewing  = "reviewing"
	NacosAIVersionOnline     = "online"
	NacosAIVersionOffline    = "offline"
)

// NacosAIArtifact 一条 Skill 或 AgentSpec 资源（按 namespace + kind + name 唯一）。
type NacosAIArtifact struct {
	Id             int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceId    string    `json:"namespaceId" gorm:"size:128;uniqueIndex:uix_nacos_ai_art_ns_kind_name;not null;default:''"`
	Kind           string    `json:"kind" gorm:"size:32;uniqueIndex:uix_nacos_ai_art_ns_kind_name;not null"` // skill | agentspec
	Name           string    `json:"name" gorm:"size:256;uniqueIndex:uix_nacos_ai_art_ns_kind_name;not null"`
	Description    string    `json:"description" gorm:"type:text"`
	BizTags        string    `json:"bizTags" gorm:"size:512"`
	LabelsJSON     string    `json:"-" gorm:"column:labels_json;type:text"` // JSON map label->version
	Scope          string    `json:"scope" gorm:"size:32;default:'PUBLIC';not null"` // PUBLIC | PRIVATE
	Enable         bool      `json:"enable" gorm:"default:true;not null"`
	OwnerUserId    int       `json:"ownerUserId" gorm:"column:owner_user_id;default:0"`
	DownloadCount  int64     `json:"downloadCount" gorm:"column:download_count;default:0"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (NacosAIArtifact) TableName() string {
	return "nacos_ai_artifacts"
}

// NacosAIArtifactVersion 单个版本（含上传 ZIP 全量）。
type NacosAIArtifactVersion struct {
	Id            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ArtifactId    int64     `json:"artifactId" gorm:"column:artifact_id;uniqueIndex:uix_nacos_ai_ver_art_ver;not null"`
	Version       string    `json:"version" gorm:"size:64;uniqueIndex:uix_nacos_ai_ver_art_ver;not null"`
	Status        string    `json:"status" gorm:"size:32;not null;default:'editing'"` // editing | reviewing | online
	Author        string    `json:"author" gorm:"size:128"`
	CommitMsg       string    `json:"commitMsg" gorm:"column:commit_msg;size:512"`
	ZipBytes        []byte    `json:"-" gorm:"column:zip_bytes;type:blob"` // db 模式或迁移过渡期
	ZipStorageKind  string    `json:"-" gorm:"column:zip_storage_kind;size:16;default:''"` // db|local|s3
	ZipStorageRef   string    `json:"-" gorm:"column:zip_storage_ref;size:1024"`           // 本地路径或 S3 对象 key
	DownloadCount   int64     `json:"downloadCount" gorm:"column:download_count;default:0"`
	CreatedAt     time.Time `json:"createTime"`
	UpdatedAt     time.Time `json:"updateTime"`
}

func (NacosAIArtifactVersion) TableName() string {
	return "nacos_ai_artifact_versions"
}
