package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
)

func TestNacosFeatureGateDisabledReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.OptionMapRWMutex.Lock()
	if config.OptionMap == nil {
		config.OptionMap = make(map[string]string)
	}
	config.OptionMap["NacosEnabled"] = "false"
	config.NacosEnabled = false
	config.OptionMapRWMutex.Unlock()
	defer func() {
		config.OptionMapRWMutex.Lock()
		config.OptionMap["NacosEnabled"] = "true"
		config.NacosEnabled = true
		config.OptionMapRWMutex.Unlock()
	}()

	r := gin.New()
	r.Use(NacosFeatureGate())
	r.GET("/api/nacos/namespaces", func(c *gin.Context) { c.Status(200) })
	r.GET("/api/status", func(c *gin.Context) { c.Status(200) })

	for _, path := range []string{
		"/api/nacos/namespaces",
		"/nacos/v3/console/server/state",
		"/nacos-ui/v3/console/server/state",
		"/api/user/nacos-console-token",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: want 404, got %d", path, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/status: want 200, got %d", rec.Code)
	}
}

func TestIsNacosBlockedPath(t *testing.T) {
	if !IsNacosBlockedPath("/api/nacos/skills") {
		t.Fatal("expected blocked")
	}
	if IsNacosBlockedPath("/api/status") {
		t.Fatal("status should not be blocked")
	}
	if IsNacosBlockedPath("/nacos/skills") {
		t.Fatal("SPA path is not API blocked at middleware layer")
	}
}
