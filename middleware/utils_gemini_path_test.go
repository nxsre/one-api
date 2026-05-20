package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetRequestModel_geminiCompatPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{"list compat", http.MethodGet, "/gemini/models", ""},
		{"get model compat", http.MethodGet, "/gemini/models/gemini-2.5-flash", "gemini-2.5-flash"},
		{"get model v1beta prefixed", http.MethodGet, "/gemini/v1beta/models/gemini-3.1-pro-preview", "gemini-3.1-pro-preview"},
		{"post generate compat", http.MethodPost, "/gemini/models/gemini-2.5-flash:generateContent", "gemini-2.5-flash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(tt.method, tt.path, nil)

			got, err := getRequestModel(c)
			if err != nil {
				t.Fatalf("getRequestModel() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("getRequestModel() = %q, want %q", got, tt.want)
			}
		})
	}
}
