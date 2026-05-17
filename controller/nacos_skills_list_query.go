package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/service"
)

// applySkillListOwnerQuery 非 Root 管理员不可按他人用户名筛选（与 console-ui-next 行为一致）。
func applySkillListOwnerQuery(c *gin.Context, owner string) string {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return ""
	}
	if rv, ok := c.Get("role"); ok {
		if r, ok := rv.(int); ok {
			if r < model.RoleRootUser {
				if uv, ok := c.Get("username"); ok {
					if self, ok := uv.(string); ok && owner != self {
						return ""
					}
				}
			}
			return owner
		}
	}
	if v, ok := c.Get(service.NacosCtxUserKey()); ok {
		if u, ok := v.(*model.User); ok && u != nil {
			if u.Role < model.RoleRootUser && owner != u.Username {
				return ""
			}
			return owner
		}
	}
	return owner
}

// parseNacosSkillListFilter 解析 skills 列表查询（pageNo/page、pageSize/size、skillName、search、orderBy、scope、bizTag、owner）。
func parseNacosSkillListFilter(c *gin.Context) (service.NacosAIListSkillsFilter, int, int) {
	pageNo := 1
	if s := strings.TrimSpace(c.Query("pageNo")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			pageNo = n
		}
	} else if s := strings.TrimSpace(c.Query("page")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			pageNo = n
		}
	}
	pageSize := 10
	if s := strings.TrimSpace(c.Query("pageSize")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			pageSize = n
		}
	} else if s := strings.TrimSpace(c.Query("size")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			pageSize = n
		}
	}
	var filter service.NacosAIListSkillsFilter
	filter.SkillName = strings.TrimSpace(c.Query("skillName"))
	if filter.SkillName == "" {
		filter.SkillName = strings.TrimSpace(c.Query("name"))
	}
	sm := strings.TrimSpace(c.Query("search"))
	filter.SearchBlur = strings.EqualFold(sm, "blur")
	filter.OrderBy = strings.TrimSpace(c.Query("orderBy"))
	filter.Scope = strings.TrimSpace(c.Query("scope"))
	filter.BizTag = strings.TrimSpace(c.Query("bizTag"))
	filter.OwnerUsername = applySkillListOwnerQuery(c, c.Query("owner"))
	return filter, pageNo, pageSize
}
