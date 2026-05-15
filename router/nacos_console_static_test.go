package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestNacosUILegacyTrailingSlashServesLegacyIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dist := fstest.MapFS{
		"legacy/index.html": {Data: []byte(`<!doctype html><html><body>legacy-nacos-console</body></html>`)},
		"legacy/js/main.js": {Data: []byte("// legacy bundle")},
	}

	r := gin.New()
	r.Use(nacosUILegacyIndexMiddleware(dist))
	r.Use(nacosUIStaticFilesMiddleware(dist))

	req := httptest.NewRequest(http.MethodGet, "/nacos-ui/legacy/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, location = %q, want 200", rec.Code, rec.Header().Get("Location"))
	}
	body := rec.Body.String()
	if strings.Contains(body, "One API") || strings.Contains(body, "/static/js/main.") {
		t.Fatalf("expected legacy console HTML, got one-api SPA: %q", body[:min(120, len(body))])
	}
	if !strings.Contains(body, "legacy-nacos-console") {
		t.Fatalf("expected legacy index marker, got: %q", body)
	}
}

func TestNacosUILegacyCSSFontAlias(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dist := fstest.MapFS{
		"legacy/fonts/font.woff": {Data: []byte("font")},
		"legacy/index.html":      {Data: []byte(`<html></html>`)},
	}

	r := gin.New()
	r.Use(nacosUILegacyIndexMiddleware(dist))
	r.Use(nacosUIStaticFilesMiddleware(dist))

	req := httptest.NewRequest(http.MethodGet, "/nacos-ui/legacy/css/fonts/font.woff", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
