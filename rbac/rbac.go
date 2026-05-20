package rbac

import (
	_ "embed"
	"strconv"
	"sync"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"

	"github.com/songquanpeng/one-api/common/logger"
	modeldb "github.com/songquanpeng/one-api/model"
)

//go:embed model.conf
var modelConf string

const (
	// ObjPlatform 平台控制台（/api/channel、/api/user 管理端等）。
	ObjPlatform = "platform"
	// ObjTenantConsole 租户控制台（子账号、租户渠道、代管令牌等）。
	ObjTenantConsole = "tenant_console"
)

var (
	enforcerMu sync.RWMutex
	enforcer   *casbin.SyncedEnforcer
)

// Init 初始化 Casbin（内存 p 策略）；分组 g 需调用 SyncUser / SyncAllUsers 与 users 表对齐。
func Init() error {
	m, err := casbinmodel.NewModelFromString(modelConf)
	if err != nil {
		return err
	}
	e, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		return err
	}
	e.EnableAutoSave(false)
	_, _ = e.AddPolicy("root", "*", "*")
	_, _ = e.AddPolicy("super_admin", ObjPlatform, "*")
	_, _ = e.AddPolicy("super_admin", ObjTenantConsole, "*")
	_, _ = e.AddPolicy("platform_admin", ObjPlatform, "*")
	// 平台管理员可像租户管理员一样调用租户控制台 API（须配合请求头 X-Tenant-Console-Tenant-Id 指定租户）。
	_, _ = e.AddPolicy("platform_admin", ObjTenantConsole, "*")
	_, _ = e.AddPolicy("tenant_admin", ObjTenantConsole, "*")
	_, _ = e.AddPolicy("tenant_sub_admin", ObjTenantConsole, "*")

	enforcerMu.Lock()
	enforcer = e
	enforcerMu.Unlock()
	return nil
}

func ef() *casbin.SyncedEnforcer {
	enforcerMu.RLock()
	defer enforcerMu.RUnlock()
	return enforcer
}

// SyncUser 根据当前 users 表记录刷新单个主体的 Casbin 分组。
func SyncUser(userID int) error {
	e := ef()
	if e == nil || userID <= 0 {
		return nil
	}
	u, err := modeldb.GetUserById(userID, false)
	if err != nil {
		return err
	}
	sub := strconv.Itoa(userID)
	_, _ = e.RemoveFilteredGroupingPolicy(0, sub)
	role := casbinRoleFromUser(u)
	if role == "" {
		return nil
	}
	_, err = e.AddGroupingPolicy(sub, role)
	return err
}

// RemoveSubject 删除用户的所有分组（删除账号前调用）。
func RemoveSubject(userID int) {
	e := ef()
	if e == nil || userID <= 0 {
		return
	}
	sub := strconv.Itoa(userID)
	_, _ = e.RemoveFilteredGroupingPolicy(0, sub)
}

// SyncAllUsers 全量同步分组（启动或运维用）。
func SyncAllUsers() error {
	e := ef()
	db := modeldb.DB
	if e == nil || db == nil {
		return nil
	}
	var ids []int
	if err := db.Model(&modeldb.User{}).Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := SyncUser(id); err != nil {
			logger.SysErrorf("rbac SyncUser %d: %v", id, err)
		}
	}
	return nil
}

func casbinRoleFromUser(u *modeldb.User) string {
	if u == nil {
		return ""
	}
	switch {
	case u.Role >= modeldb.RoleRootUser:
		return "root"
	case u.Role >= modeldb.RoleSuperAdmin && u.TenantID == nil:
		return "super_admin"
	case u.Role == modeldb.RoleAdminUser && u.TenantID == nil:
		return "platform_admin"
	case u.Role == modeldb.RoleTenantAdmin && u.TenantID != nil:
		return "tenant_admin"
	case u.Role == modeldb.RoleCommonUser && u.TenantID != nil && len(u.TenantPermissions) > 0:
		return "tenant_sub_admin"
	default:
		return ""
	}
}

// Enforce sub 与分组一致：strconv.Itoa(uid)。
func Enforce(userID int, obj, act string) bool {
	e := ef()
	if e == nil {
		return false
	}
	sub := strconv.Itoa(userID)
	ok, err := e.Enforce(sub, obj, act)
	if err != nil {
		logger.SysError("casbin Enforce: " + err.Error())
		return false
	}
	return ok
}

// EnforceHTTPMethod 使用 HTTP Method 作为 act（GET/POST/...），策略侧为 * 放行。
func EnforceHTTPMethod(userID int, obj, method string) bool {
	if method == "" {
		method = "*"
	}
	return Enforce(userID, obj, method)
}
