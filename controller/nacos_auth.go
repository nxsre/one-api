package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/service"
)

// NacosUserLoginDisabledV3 关闭 v3 表单登录：控制台嵌入 One API 时应使用站点 /login 会话，再由 /api/user/nacos-console-token 注入 token。
func NacosUserLoginDisabledV3(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"message": "已关闭控制台表单登录，请先在 One API 登录后访问 /nacos-ui/",
	})
}

// NacosUserLogin POST /nacos/v1|v3/auth/user/login 及 /nacos/legacy/v1|v3/... 同路径，表单 username/password。
// v1 与 legacy 下的 v1 保持 {code,data} 以兼容 nacos-cli；v3 与 legacy 下的 v3 返回扁平 JSON 以兼容 console-ui-next。
func NacosUserLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	if username == "" || password == "" {
		if strings.Contains(c.FullPath(), "/v1/") {
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": "用户名或密码为空", "data": nil})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"message": "用户名或密码为空"})
		return
	}
	u := model.User{Username: username, Password: password}
	if err := u.ValidateAndFill(); err != nil {
		if strings.Contains(c.FullPath(), "/v1/") {
			c.JSON(http.StatusOK, gin.H{"code": 401, "message": err.Error(), "data": nil})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	if u.TenantID != nil {
		if strings.Contains(c.FullPath(), "/v1/") {
			c.JSON(http.StatusOK, gin.H{
				"code":    401,
				"message": "租户账号请使用站点「租户登录」或主站会话，勿使用控制台表单登录。",
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{
			"message": "租户账号请使用站点「租户登录」或先在 One API 主站以租户方式登录后再访问控制台。",
		})
		return
	}
	if strings.Contains(c.FullPath(), "/v1/") {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"accessToken": u.AccessToken,
				"tokenTtl":    86400 * 7,
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"accessToken": u.AccessToken,
		"username":    u.Username,
		"globalAdmin": u.Role >= model.RoleAdminUser,
	})
}

func nacosAuthRequireAdmin(c *gin.Context) (*model.User, bool) {
	raw, ok := c.Get(service.NacosCtxUserKey())
	if !ok {
		nacosV3Err(c, 10001, "未登录")
		return nil, false
	}
	caller := raw.(*model.User)
	if caller.Role < model.RoleAdminUser {
		nacosV3Err(c, 10001, "需要管理员权限")
		return nil, false
	}
	return caller, true
}

// NacosAuthRoleList GET /nacos/v3/auth/role/list：由用户表派生 ROOT/ADMIN/USER 与 username；search=blur|accurate 控制模糊/精确。
func NacosAuthRoleList(c *gin.Context) {
	if _, ok := nacosAuthRequireAdmin(c); !ok {
		return
	}
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	mode := strings.TrimSpace(c.DefaultQuery("search", "blur"))
	roleKw := c.Query("role")
	usernameKw := c.Query("username")
	total, list, err := model.NacosConsoleRoleListPage(pageNo, pageSize, roleKw, usernameKw, mode)
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	items := make([]gin.H, 0, len(list))
	for i := range list {
		items = append(items, gin.H{"role": list[i].Role, "username": list[i].Username})
	}
	nacosV3OK(c, gin.H{
		"totalCount":     total,
		"pageNumber":     pageNo,
		"pagesAvailable": pages,
		"pageItems":      items,
	})
}

// NacosAuthPermissionList GET /nacos/v3/auth/permission/list：静态权限目录；role 查询参数为关键词；search=blur|accurate。
func NacosAuthPermissionList(c *gin.Context) {
	if _, ok := nacosAuthRequireAdmin(c); !ok {
		return
	}
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	mode := strings.TrimSpace(c.DefaultQuery("search", "blur"))
	keyword := strings.TrimSpace(c.Query("role"))
	total, list, err := service.NacosConsolePermissionListPage(pageNo, pageSize, keyword, mode)
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	items := make([]gin.H, 0, len(list))
	for i := range list {
		items = append(items, gin.H{"role": list[i].Role, "resource": list[i].Resource, "action": list[i].Action})
	}
	nacosV3OK(c, gin.H{
		"totalCount":     total,
		"pageNumber":     pageNo,
		"pagesAvailable": pages,
		"pageItems":      items,
	})
}

// NacosAuthUserList GET /nacos/v3/auth/user/list，分页对接 one-api 用户表（需管理员 token）。
// 与 console-ui-next 一致：username 为关键词，search=blur|accurate 为搜索模式。
func NacosAuthUserList(c *gin.Context) {
	if _, ok := nacosAuthRequireAdmin(c); !ok {
		return
	}
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	username := strings.TrimSpace(c.Query("username"))
	mode := strings.TrimSpace(c.DefaultQuery("search", "blur"))
	total, users, err := model.NacosConsoleUserPage(pageNo, pageSize, username, mode)
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	items := make([]gin.H, 0, len(users))
	for _, u := range users {
		items = append(items, gin.H{"username": u.Username})
	}
	nacosV3OK(c, gin.H{
		"totalCount":     total,
		"pageNumber":     pageNo,
		"pagesAvailable": pages,
		"pageItems":      items,
	})
}
