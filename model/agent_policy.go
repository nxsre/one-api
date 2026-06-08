package model

import (
	"github.com/songquanpeng/one-api/common/agentpolicy"
	"github.com/songquanpeng/one-api/common/logger"
)

// GetUserAgentPolicy 读取用户级 agent 客户端策略（仅取所需列）；无则返回 nil。
func GetUserAgentPolicy(userId int) *agentpolicy.Policy {
	if userId <= 0 {
		return nil
	}
	var u User
	if err := DB.Model(&User{}).Select("agent_client_policy").Where("id = ?", userId).First(&u).Error; err != nil {
		return nil
	}
	if u.AgentClientPolicy.IsZero() {
		return nil
	}
	return u.AgentClientPolicy
}

// GetTenantAgentPolicy 读取租户级 agent 客户端策略；无则返回 nil。
func GetTenantAgentPolicy(tenantId int) *agentpolicy.Policy {
	if tenantId <= 0 {
		return nil
	}
	var t Tenant
	if err := DB.Model(&Tenant{}).Select("agent_client_policy").Where("id = ?", tenantId).First(&t).Error; err != nil {
		return nil
	}
	if t.AgentClientPolicy.IsZero() {
		return nil
	}
	return t.AgentClientPolicy
}

// SetTenantAgentPolicy 更新租户级 agent 客户端策略。
func SetTenantAgentPolicy(id int, p *agentpolicy.Policy) error {
	t := Tenant{Id: id, AgentClientPolicy: p}
	if err := DB.Model(&Tenant{}).Where("id = ?", id).Select("agent_client_policy").Updates(&t).Error; err != nil {
		return err
	}
	if p != nil && !p.IsZero() {
		agentpolicy.SetEnabled(true)
	}
	return nil
}

// InitAgentPolicyEnabled 在启动时探测是否存在任意非空策略（全局/令牌/用户/租户），
// 据此设置进程级总开关，使未启用本功能的部署在中继热路径上零额外开销。
func InitAgentPolicyEnabled() {
	if !agentpolicy.Global().IsZero() {
		agentpolicy.SetEnabled(true)
		return
	}
	notEmpty := "agent_client_policy IS NOT NULL AND agent_client_policy <> '' AND agent_client_policy <> '{}' AND agent_client_policy <> 'null'"
	var n int64
	for _, q := range []*struct{ tbl any }{{&Token{}}, {&User{}}, {&Tenant{}}} {
		if err := DB.Model(q.tbl).Where(notEmpty).Limit(1).Count(&n).Error; err != nil {
			logger.SysError("agent policy enabled probe failed: " + err.Error())
			continue
		}
		if n > 0 {
			agentpolicy.SetEnabled(true)
			return
		}
	}
}
