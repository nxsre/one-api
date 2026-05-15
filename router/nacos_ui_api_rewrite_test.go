package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNacosUIV3ConsoleStateRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetNacosRegistryRouter(r)

	req := httptest.NewRequest(http.MethodGet, "/nacos-ui/v3/console/server/state", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
