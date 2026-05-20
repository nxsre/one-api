package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/service"
)

// nacosEffectiveUser Web 会话（ctxkey.Id）或 Nacos access-token 上下文中的用户。
func nacosEffectiveUser(c *gin.Context) *model.User {
	if v, ok := c.Get(service.NacosCtxUserKey()); ok {
		if u, ok := v.(*model.User); ok && u != nil {
			return u
		}
	}
	uid := c.GetInt(ctxkey.Id)
	if uid <= 0 {
		return nil
	}
	u, err := model.GetUserById(uid, false)
	if err != nil {
		return nil
	}
	return u
}

// nacosCallerNamespaceTenantScope 判定当前请求是否应按「租户命名空间」隔离（本子账号/租户管理员，或平台管理员代管且请求头带租户）。
func nacosCallerNamespaceTenantScope(c *gin.Context) (tenantID int, scoped bool) {
	u := nacosEffectiveUser(c)
	if u == nil {
		return 0, false
	}
	if u.Role >= model.RoleAdminUser && u.TenantID == nil {
		if model.IsPlatformConsoleOperator(u.Id) {
			if tid, err := GetTenantConsoleTenantID(c, ""); err == nil && tid > 0 {
				return tid, true
			}
		}
		return 0, false
	}
	if u.TenantID == nil {
		return 0, false
	}
	return *u.TenantID, true
}

func nacosTenantLegacyFallbackNS(c *gin.Context, tenantID int) string {
	t, err := model.GetTenantByID(tenantID)
	if err != nil || t == nil {
		return "public"
	}
	leg := service.NacosTenantLegacyNamespaceID(t)
	if leg == "" {
		return "public"
	}
	return leg
}

func nacosFirstQueryNamespace(c *gin.Context, keys ...string) string {
	for _, k := range keys {
		v := strings.TrimSpace(c.Query(k))
		if v != "" {
			return v
		}
		v = strings.TrimSpace(c.PostForm(k))
		if v != "" {
			return v
		}
	}
	return ""
}

// nacosResolveNamespaceConsole /api/nacos：租户仅能使用自有或平台开放的命名空间；平台管理员不限。
func nacosResolveNamespaceConsole(c *gin.Context, fallback string) (string, bool) {
	raw := nacosFirstQueryNamespace(c, "namespace", "namespaceId")
	if raw == "" {
		raw = fallback
	}
	ns := service.NormalizeNacosNamespaceID(raw)
	tid, scoped := nacosCallerNamespaceTenantScope(c)
	if !scoped {
		if ns == "" {
			ns = service.NormalizeNacosNamespaceID(fallback)
		}
		return ns, true
	}
	if ns == "" {
		ns = service.NormalizeNacosNamespaceID(nacosTenantLegacyFallbackNS(c, tid))
	}
	if err := service.NacosAssertTenantMayUseNamespace(tid, ns); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return "", false
	}
	return ns, true
}

// nacosResolveNamespaceV3 /nacos/v1|v3 注册表 API。
func nacosResolveNamespaceV3(c *gin.Context, fallback string) (string, bool) {
	raw := nacosParam(c, "namespaceId")
	if raw == "" {
		raw = nacosParam(c, "namespace")
	}
	if raw == "" {
		raw = fallback
	}
	ns := service.NormalizeNacosNamespaceID(raw)
	tid, scoped := nacosCallerNamespaceTenantScope(c)
	if !scoped {
		return ns, true
	}
	if ns == "" {
		ns = service.NormalizeNacosNamespaceID(nacosTenantLegacyFallbackNS(c, tid))
	}
	if err := service.NacosAssertTenantMayUseNamespace(tid, ns); err != nil {
		nacosV3Err(c, 40300, err.Error())
		return "", false
	}
	return ns, true
}

// nacosAssertTenantNamespaceWeb /api/nacos JSON：校验租户是否可使用某 namespace（已归一化或非空即可传入）。
func nacosAssertTenantNamespaceWeb(c *gin.Context, namespaceID string) bool {
	ns := service.NormalizeNacosNamespaceID(namespaceID)
	if ns == "" {
		ns = "public"
	}
	tid, scoped := nacosCallerNamespaceTenantScope(c)
	if !scoped {
		return true
	}
	if err := service.NacosAssertTenantMayUseNamespace(tid, ns); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return false
	}
	return true
}
