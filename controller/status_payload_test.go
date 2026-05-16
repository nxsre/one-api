package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestGetStatusPublicHidesInternalFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := cookie.NewStore([]byte("test-session-secret-status-payload"))
	r.Use(sessions.Sessions("session", store))
	r.GET("/api/status", GetStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code %d", rec.Code)
	}
	var body struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data["status_scope"] != "public" {
		t.Fatalf("scope: %v", body.Data["status_scope"])
	}
	for _, key := range []string{
		"nacos_enabled", "server_address", "global_access_mode",
		"oidc_token_endpoint", "outbound_url_whitelist_enabled",
	} {
		if _, ok := body.Data[key]; ok {
			t.Fatalf("public status must not include %q", key)
		}
	}
	if _, ok := body.Data["system_name"]; !ok {
		t.Fatal("missing system_name")
	}
	if _, ok := body.Data["login_math_captcha"]; !ok {
		t.Fatal("missing login_math_captcha")
	}
}
