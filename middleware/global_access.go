package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/cfg"
	"github.com/songquanpeng/one-api/model"

	"github.com/gin-gonic/gin"
)

const globalAccessScopeKey = "global_access_scope"

func GlobalAccessGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !shouldRunGlobalAccessCheck(c.Request.URL.Path) {
			c.Next()
			return
		}
		mode := model.GetGlobalAccessMode()
		if mode == model.GlobalAccessModeNone || mode == "" {
			c.Next()
			return
		}
		clientIP := c.ClientIP()
		apiKey := extractAPIKeyFromRequest(c.Request)
		wl, bl := model.LoadGlobalAccessListsCached()
		switch mode {
		case model.GlobalAccessModeBlacklist:
			if matchesBlacklist(clientIP, apiKey, bl) {
				abortGlobalAccess(c)
				return
			}
		case model.GlobalAccessModeWhitelist:
			if !matchesWhitelist(clientIP, apiKey, wl) {
				abortGlobalAccess(c)
				return
			}
		}
		c.Next()
	}
}

func abortGlobalAccess(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"message": "access denied by global access list policy",
			"type":    "global_access_denied",
		},
	})
}

func shouldRunGlobalAccessCheck(path string) bool {
	scope := strings.TrimSpace(cfg.V.GetString(globalAccessScopeKey))
	if scope == "" {
		scope = "relay"
	}
	if strings.HasPrefix(path, "/v1") {
		return true
	}
	if strings.HasPrefix(path, "/v1beta") {
		return true
	}
	if strings.HasPrefix(path, "/openai/") || strings.HasPrefix(path, "/anthropic/") || strings.HasPrefix(path, "/gemini/") {
		return true
	}
	if scope != "full" {
		return false
	}
	if strings.HasPrefix(path, "/api") {
		return !isGlobalAccessExemptAPI(path)
	}
	return false
}

func isGlobalAccessExemptAPI(path string) bool {
	exempt := []string{
		"/api/status",
		"/api/notice",
		"/api/about",
		"/api/home_page_content",
	}
	for _, p := range exempt {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

func extractAPIKeyFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	if k := strings.TrimSpace(r.Header.Get("x-api-key")); k != "" {
		return k
	}
	return ""
}

func ipMatchesRule(clientIP, rule string) bool {
	rule = strings.TrimSpace(rule)
	if rule == "" || clientIP == "" {
		return false
	}
	if strings.Contains(rule, "/") {
		_, ipNet, err := net.ParseCIDR(rule)
		if err != nil {
			return false
		}
		ip := net.ParseIP(clientIP)
		if ip == nil {
			return false
		}
		return ipNet.Contains(ip)
	}
	return clientIP == rule
}

func normalizeKeyFragment(k string) string {
	k = strings.TrimSpace(k)
	k = strings.TrimPrefix(k, "sk-")
	if idx := strings.Index(k, "-"); idx > 0 {
		return k[:idx]
	}
	return k
}

func apiKeyMatches(requestKey, entryValue string) bool {
	requestKey = strings.TrimSpace(requestKey)
	entryValue = strings.TrimSpace(entryValue)
	if requestKey == "" || entryValue == "" {
		return false
	}
	if requestKey == entryValue {
		return true
	}
	rq := normalizeKeyFragment(requestKey)
	ev := normalizeKeyFragment(entryValue)
	return rq != "" && ev != "" && rq == ev
}

func matchesBlacklist(clientIP, apiKey string, entries []*model.GlobalAccessBlacklist) bool {
	for _, e := range entries {
		if e == nil || !e.Enabled {
			continue
		}
		switch e.Type {
		case model.GlobalAccessEntryTypeIP:
			if ipMatchesRule(clientIP, e.Value) {
				return true
			}
		case model.GlobalAccessEntryTypeAPIKey:
			if apiKeyMatches(apiKey, e.Value) {
				return true
			}
		}
	}
	return false
}

func matchesWhitelist(clientIP, apiKey string, entries []*model.GlobalAccessWhitelist) bool {
	if len(entries) == 0 {
		return false
	}
	for _, e := range entries {
		if e == nil || !e.Enabled {
			continue
		}
		switch e.Type {
		case model.GlobalAccessEntryTypeIP:
			if ipMatchesRule(clientIP, e.Value) {
				return true
			}
		case model.GlobalAccessEntryTypeAPIKey:
			if apiKeyMatches(apiKey, e.Value) {
				return true
			}
		}
	}
	return false
}
