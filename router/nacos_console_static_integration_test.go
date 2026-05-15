package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestNacosUIV3WithStaticMiddlewareNoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dist := fstest.MapFS{
		"index.html": {Data: []byte(`<!doctype html><html><body>next</body></html>`)},
		"js/main.js": {Data: []byte("// next")},
		"legacy/index.html": {Data: []byte(`<!doctype html><html><body>legacy</body></html>`)},
	}

	r := gin.New()
	SetNacosRegistryRouter(r)
	r.Use(nacosUILegacyIndexMiddleware(dist))
	r.Use(nacosUIStaticFilesMiddleware(dist))

	req := httptest.NewRequest(http.MethodGet, "/nacos-ui/v3/console/server/state", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("api status = %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/nacos-ui/js/main.js", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("static status = %d", rec2.Code)
	}
}
