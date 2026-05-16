package service

import (
	"encoding/json"
	"strings"

	"github.com/songquanpeng/one-api/model"
)

// Nacos 兼容权限键（对齐 Nacos Admin/Open + CONFIG 的 READ/WRITE 语义）。
const (
	PermAdminAISkillsRead        = "admin:ai:skills:read"
	PermAdminAISkillsWrite       = "admin:ai:skills:write"
	PermAdminAIAgentSpecsRead    = "admin:ai:agentspecs:read"
	PermAdminAIAgentSpecsWrite   = "admin:ai:agentspecs:write"
	PermClientAISkillsRead       = "client:ai:skills:read"
	PermClientAIAgentSpecsRead   = "client:ai:agentspecs:read"
	PermAdminAIMcpRead           = "admin:ai:mcp:read"
	PermAdminAIMcpWrite         = "admin:ai:mcp:write"
	PermAdminAIA2ARead          = "admin:ai:a2a:read"
	PermAdminAIA2AWrite         = "admin:ai:a2a:write"
	PermAdminAIPromptRead       = "admin:ai:prompt:read"
	PermAdminAIPromptWrite      = "admin:ai:prompt:write"
	PermAdminAIPipelineRead     = "admin:ai:pipeline:read"
	PermAdminAIPipelineWrite    = "admin:ai:pipeline:write"
	PermClientAIMcpRead         = "client:ai:mcp:read"
	PermClientAIA2ARead         = "client:ai:a2a:read"
	PermClientAIPromptRead      = "client:ai:prompt:read"
	PermAdminCsConfigRead        = "admin:cs:config:read"
	PermAdminCsConfigWrite       = "admin:cs:config:write"
	PermClientCsConfigRead       = "client:cs:config:read"
)

// NacosAllPermissionKeys 供管理页展示全部可配置项。
func NacosAllPermissionKeys() []string {
	return []string{
		PermAdminAISkillsRead,
		PermAdminAISkillsWrite,
		PermAdminAIAgentSpecsRead,
		PermAdminAIAgentSpecsWrite,
		PermClientAISkillsRead,
		PermClientAIAgentSpecsRead,
		PermAdminAIMcpRead,
		PermAdminAIMcpWrite,
		PermAdminAIA2ARead,
		PermAdminAIA2AWrite,
		PermAdminAIPromptRead,
		PermAdminAIPromptWrite,
		PermAdminAIPipelineRead,
		PermAdminAIPipelineWrite,
		PermClientAIMcpRead,
		PermClientAIA2ARead,
		PermClientAIPromptRead,
		PermAdminCsConfigRead,
		PermAdminCsConfigWrite,
		PermClientCsConfigRead,
	}
}

// NacosPermissionItem 权限项（含中文说明，供管理页展示）。
type NacosPermissionItem struct {
	Key     string `json:"key"`
	LabelZh string `json:"label_zh"`
	LabelEn string `json:"label_en"`
}

// NacosPermissionCatalog 与 NacosAllPermissionKeys 顺序一致，每项带中英文描述。
func NacosPermissionCatalog() []NacosPermissionItem {
	labels := map[string][2]string{
		PermAdminAISkillsRead:      {"管理端 Skills 列表/详情/下载（只读）", "Admin: list/describe/download Skills (read)"},
		PermAdminAISkillsWrite:     {"管理端 Skills 上传/提交/发布（写）", "Admin: upload/submit/publish Skills (write)"},
		PermAdminAIAgentSpecsRead:  {"管理端 AgentSpec 列表/详情（只读）", "Admin: list/describe AgentSpecs (read)"},
		PermAdminAIAgentSpecsWrite: {"管理端 AgentSpec 上传/提交/发布（写）", "Admin: upload/submit/publish AgentSpecs (write)"},
		PermClientAISkillsRead:     {"客户端按标签/版本获取已发布 Skill ZIP（只读）", "Client: get published Skill ZIP by label/version (read)"},
		PermClientAIAgentSpecsRead: {"客户端获取已发布 AgentSpec 内容（只读）", "Client: get published AgentSpec payload (read)"},
		PermAdminAIMcpRead:         {"管理端 MCP 列表/详情（只读）", "Admin: list/describe MCP servers (read)"},
		PermAdminAIMcpWrite:        {"管理端 MCP 创建/更新/删除（写）", "Admin: upsert/delete MCP servers (write)"},
		PermAdminAIA2ARead:         {"管理端 A2A Agent 列表/详情（只读）", "Admin: list/describe A2A agents (read)"},
		PermAdminAIA2AWrite:        {"管理端 A2A Agent 创建/更新/删除（写）", "Admin: upsert/delete A2A agents (write)"},
		PermAdminAIPromptRead:      {"管理端 Prompt 列表/详情/版本（只读）", "Admin: list/describe prompts (read)"},
		PermAdminAIPromptWrite:     {"管理端 Prompt 头/版本/提交/发布（写）", "Admin: upsert prompt versions (write)"},
		PermAdminAIPipelineRead:    {"管理端流水线运行记录查询（只读）", "Admin: list pipeline runs (read)"},
		PermAdminAIPipelineWrite:   {"管理端触发流水线/校验任务（写）", "Admin: trigger pipeline scan (write)"},
		PermClientAIMcpRead:        {"客户端获取已启用 MCP 描述 JSON（只读）", "Client: get enabled MCP spec (read)"},
		PermClientAIA2ARead:        {"客户端获取已启用 A2A 卡片 JSON（只读）", "Client: get enabled A2A card (read)"},
		PermClientAIPromptRead:     {"客户端获取已发布 Prompt 内容 JSON（只读）", "Client: get published prompt JSON (read)"},
		PermAdminCsConfigRead:      {"管理端配置中心：列举/查询配置（只读）", "Admin config center: list/get config (read)"},
		PermAdminCsConfigWrite:     {"管理端配置中心：发布/变更配置（写）", "Admin config center: publish/update config (write)"},
		PermClientCsConfigRead:     {"客户端配置中心：按 dataId/group 拉取配置（只读）", "Client config center: get config by dataId/group (read)"},
	}
	out := make([]NacosPermissionItem, 0, len(labels))
	for _, k := range NacosAllPermissionKeys() {
		pair := labels[k]
		out = append(out, NacosPermissionItem{Key: k, LabelZh: pair[0], LabelEn: pair[1]})
	}
	return out
}

func nacosACLRulesMap(jsonStr string) map[string]bool {
	out := map[string]bool{}
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" || jsonStr == "{}" {
		return out
	}
	_ = json.Unmarshal([]byte(jsonStr), &out)
	return out
}

// NacosCheckPerm 校验是否允许某权限。user 不可为 nil（调用方保证）。
func NacosCheckPerm(user *model.User, perm string) bool {
	if user == nil {
		return false
	}
	if user.Status != model.UserStatusEnabled {
		return false
	}
	if user.Role >= model.RoleRootUser {
		return true
	}
	if user.Role >= model.RoleAdminUser {
		var acl model.NacosUserACL
		err := model.DB.Where("user_id = ?", user.Id).First(&acl).Error
		if err != nil || strings.TrimSpace(acl.RulesJSON) == "" || acl.RulesJSON == "{}" {
			return true
		}
		return nacosACLRulesMap(acl.RulesJSON)[perm]
	}
	var acl model.NacosUserACL
	if err := model.DB.Where("user_id = ?", user.Id).First(&acl).Error; err != nil {
		return false
	}
	return nacosACLRulesMap(acl.RulesJSON)[perm]
}
