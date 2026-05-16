package model

import (
	"strings"

	"gorm.io/gorm"
)

// NacosConsoleRoleLabel 将 one-api 用户角色映射为控制台展示的「角色」名。
func NacosConsoleRoleLabel(role int) string {
	if role >= RoleRootUser {
		return "ROOT"
	}
	if role >= RoleAdminUser {
		return "ADMIN"
	}
	return "USER"
}

// NacosConsoleRoleRow 供 GET /v3/auth/role/list 的 pageItems。
type NacosConsoleRoleRow struct {
	Role     string `json:"role"`
	Username string `json:"username"`
}

// nacosLikePattern 去掉 LIKE 通配符注入，仅保留字面匹配片段。
func nacosLikePattern(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "%", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// NacosConsoleRoleListPage 从用户表派生「角色-用户」行，支持 search=blur|accurate 与 role、username 过滤。
func NacosConsoleRoleListPage(pageNo, pageSize int, roleKw, usernameKw, mode string) (total int64, rows []NacosConsoleRoleRow, err error) {
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 500 {
		pageSize = 500
	}
	accurate := strings.EqualFold(strings.TrimSpace(mode), "accurate")
	roleCase := "(CASE WHEN role >= ? THEN 'ROOT' WHEN role >= ? THEN 'ADMIN' ELSE 'USER' END)"

	q := DB.Model(&User{}).Where("status != ?", UserStatusDeleted)
	if ukw := strings.TrimSpace(usernameKw); ukw != "" {
		if accurate {
			q = q.Where("username = ?", ukw)
		} else if p := nacosLikePattern(ukw); p != "" {
			q = q.Where("username LIKE ?", "%"+p+"%")
		}
	}
	if rkw := strings.TrimSpace(roleKw); rkw != "" {
		if accurate {
			q = q.Where(roleCase+" = ?", RoleRootUser, RoleAdminUser, rkw)
		} else if p := nacosLikePattern(rkw); p != "" {
			q = q.Where(roleCase+" LIKE ?", RoleRootUser, RoleAdminUser, "%"+p+"%")
		}
	}

	if err = q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var users []User
	err = q.Select("username", "role").Order("id desc").Offset((pageNo - 1) * pageSize).Limit(pageSize).Find(&users).Error
	if err != nil {
		return 0, nil, err
	}
	rows = make([]NacosConsoleRoleRow, 0, len(users))
	for i := range users {
		rows = append(rows, NacosConsoleRoleRow{
			Role:     NacosConsoleRoleLabel(users[i].Role),
			Username: users[i].Username,
		})
	}
	return total, rows, nil
}

// NacosConsoleUserPage 供嵌入 Nacos 控制台 GET /v3/auth/user/list。
// usernameKw 为搜索框关键词；mode 为前端传入的 search 参数：blur=模糊，accurate=仅用户名精确匹配。
func NacosConsoleUserPage(pageNo, pageSize int, usernameKw, mode string) (total int64, users []*User, err error) {
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 500 {
		pageSize = 500
	}
	accurate := strings.EqualFold(strings.TrimSpace(mode), "accurate")

	q := DB.Model(&User{}).Where("status != ?", UserStatusDeleted)
	kw := strings.TrimSpace(usernameKw)
	if kw != "" {
		if accurate {
			q = q.Where("username = ?", kw)
		} else if p := nacosLikePattern(kw); p != "" {
			like := "%" + p + "%"
			q = q.Where("username LIKE ? OR email LIKE ? OR display_name LIKE ? OR uid LIKE ?", like, like, like, like)
		}
	}

	if err = q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return 0, nil, err
	}
	users = []*User{}
	err = q.Omit("password", "S3SecretKey", "access_token").Order("id desc").
		Offset((pageNo - 1) * pageSize).Limit(pageSize).Find(&users).Error
	if err != nil {
		return 0, nil, err
	}
	return total, users, nil
}
