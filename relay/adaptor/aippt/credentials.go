package aippt

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseChannelKey 解析渠道「密钥」字段，支持：
//   - app_key|secret_key
//   - 两行：第一行 app，第二行 secret
//   - JSON（推荐）：{"api_key":"...","api_secret":"...","uid":"..."}
//   - JSON（兼容）：{"app_key":"...","secret_key":"...","uid":"..."}（uid 可选）
func ParseChannelKey(raw string) (appKey, secretKey, uid string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", fmt.Errorf("aippt: empty channel key")
	}
	if strings.HasPrefix(raw, "{") {
		var m struct {
			APIKey    string `json:"api_key"`
			APISecret string `json:"api_secret"`
			AppKey    string `json:"app_key"`
			SecretKey string `json:"secret_key"`
			UID       string `json:"uid"`
		}
		if e := json.Unmarshal([]byte(raw), &m); e != nil {
			return "", "", "", fmt.Errorf("aippt: invalid JSON key: %w", e)
		}
		app := strings.TrimSpace(m.APIKey)
		sec := strings.TrimSpace(m.APISecret)
		if app == "" {
			app = strings.TrimSpace(m.AppKey)
		}
		if sec == "" {
			sec = strings.TrimSpace(m.SecretKey)
		}
		if app == "" || sec == "" {
			return "", "", "", fmt.Errorf("aippt: api_key/api_secret (或 app_key/secret_key) are required in JSON")
		}
		uid = strings.TrimSpace(m.UID)
		if uid == "" {
			uid = "openclaw_default"
		}
		return app, sec, uid, nil
	}
	if idx := strings.Index(raw, "|"); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), strings.TrimSpace(raw[idx+1:]), "openclaw_default", nil
	}
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	lines = filterEmpty(lines)
	if len(lines) >= 2 {
		return strings.TrimSpace(lines[0]), strings.TrimSpace(lines[1]), "openclaw_default", nil
	}
	return "", "", "", fmt.Errorf("aippt: use app_key|secret_key or two lines or JSON")
}

func filterEmpty(in []string) []string {
	var out []string
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}
