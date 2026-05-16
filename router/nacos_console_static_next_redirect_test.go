package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestNacosUINextRedirectsToRoot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dist := fstest.MapFS{
		"legacy/index.html": {Data: []byte(`<html></html>`)},
		"index.html":        {Data: []byte(`<html>next</html>`)},
	}
	r := gin.New()
	r.Use(nacosUILegacyIndexMiddleware(dist))

	req := httptest.NewRequest(http.MethodGet, "/nacos-ui/next/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/nacos-ui/" {
		t.Fatalf("Location = %q", loc)
	}
}
