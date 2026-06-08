package anthropic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
)

const claudeSessionHeader = "X-Claude-Session-Id"
const claudeTenantHeader = "X-Claude-Tenant-Key"

type anthropicSessionRawRequest struct {
	Model    string `json:"model"`
	System   any    `json:"system"`
	Messages []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"messages"`
	Metadata *struct {
		UserID string `json:"user_id"`
	} `json:"metadata"`
}

func deriveStableClaudeSessionID(c *gin.Context) string {
	raw, ok := c.Get(ctxkey.KeyRequestBody)
	if !ok || raw == nil {
		return ""
	}
	body, ok := raw.([]byte)
	if !ok || len(body) == 0 {
		return ""
	}

	var req anthropicSessionRawRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return ""
	}

	systemAnchor := normalizeAny(req.System)
	firstUserAnchor := firstUserTextAnchor(req.Messages)
	if firstUserAnchor == "" {
		return ""
	}

	tokenID := c.GetInt(ctxkey.TokenId)
	userID := c.GetInt(ctxkey.Id)
	metaUserID := ""
	if req.Metadata != nil {
		metaUserID = strings.TrimSpace(req.Metadata.UserID)
	}

	parts := []string{
		"anthropic-compatible",
		model,
		metaUserID,
		intToString(tokenID),
		intToString(userID),
		systemAnchor,
		firstUserAnchor,
	}
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "oas_" + hex.EncodeToString(h[:16])
}

func deriveStableClaudeTenantKey(c *gin.Context) string {
	tokenID := c.GetInt(ctxkey.TokenId)
	userID := c.GetInt(ctxkey.Id)
	channelID := c.GetInt(ctxkey.ChannelId)
	model := strings.TrimSpace(c.GetString(ctxkey.RequestModel))
	group := strings.TrimSpace(c.GetString(ctxkey.Group))
	parts := []string{
		"anthropic-compatible-tenant",
		intToString(tokenID),
		intToString(userID),
		intToString(channelID),
		model,
		group,
	}
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "oat_" + hex.EncodeToString(h[:12])
}

func firstUserTextAnchor(messages []struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}) string {
	for _, m := range messages {
		if !strings.EqualFold(strings.TrimSpace(m.Role), "user") {
			continue
		}
		s := extractContentText(m.Content)
		if s != "" {
			return s
		}
	}
	return ""
}

func extractContentText(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return normalizeAnchor(s)
	}
	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var parts []string
	for _, b := range blocks {
		t, _ := b["type"].(string)
		if t != "text" {
			continue
		}
		text, _ := b["text"].(string)
		if strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}
	return normalizeAnchor(strings.Join(parts, "\n"))
}

func normalizeAny(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return normalizeAnchor(t)
	case []any:
		var parts []string
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok && strings.TrimSpace(text) != "" {
					parts = append(parts, text)
				}
			}
		}
		return normalizeAnchor(strings.Join(parts, "\n"))
	default:
		b, _ := json.Marshal(v)
		return normalizeAnchor(string(b))
	}
}

func normalizeAnchor(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	if len(s) > 512 {
		return s[:512]
	}
	return s
}

func intToString(v int) string {
	return strconv.Itoa(v)
}

