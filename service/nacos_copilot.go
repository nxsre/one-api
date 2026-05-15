package service

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/songquanpeng/one-api/model"
)

const nacosCopilotOptionKey = "NacosCopilotConfigJSON"

// NacosCopilotConfig 控制台 Copilot 配置（持久化到 options 表）。
type NacosCopilotConfig struct {
	Enabled bool   `json:"enabled"`
	ApiKey  string `json:"apiKey"`
	Model   string `json:"model"`
	BaseURL string `json:"baseUrl"`
}

func NacosCopilotLoadConfig() NacosCopilotConfig {
	var row model.Option
	if err := model.DB.Where("key = ?", nacosCopilotOptionKey).First(&row).Error; err != nil || strings.TrimSpace(row.Value) == "" {
		return NacosCopilotConfig{Enabled: true, Model: "gpt-4o-mini"}
	}
	var c NacosCopilotConfig
	if json.Unmarshal([]byte(row.Value), &c) != nil {
		return NacosCopilotConfig{Enabled: true, Model: "gpt-4o-mini"}
	}
	return c
}

func NacosCopilotSaveConfig(patch NacosCopilotConfig) error {
	cur := NacosCopilotLoadConfig()
	if patch.ApiKey != "" {
		cur.ApiKey = patch.ApiKey
	}
	if strings.TrimSpace(patch.Model) != "" {
		cur.Model = strings.TrimSpace(patch.Model)
	}
	if strings.TrimSpace(patch.BaseURL) != "" {
		cur.BaseURL = strings.TrimSpace(patch.BaseURL)
	}
	if patch.Enabled {
		cur.Enabled = true
	} else if strings.TrimSpace(patch.ApiKey) != "" {
		cur.Enabled = true
	}
	b, err := json.Marshal(cur)
	if err != nil {
		return err
	}
	return model.UpdateOption(nacosCopilotOptionKey, string(b))
}

// NacosCopilotEffectiveConfig 合并数据库与环境变量（环境变量优先覆盖密钥与端点）。
func NacosCopilotEffectiveConfig() NacosCopilotConfig {
	base := NacosCopilotLoadConfig()
	if k := strings.TrimSpace(os.Getenv("NACOS_COPILOT_API_KEY")); k != "" {
		base.ApiKey = k
	}
	if u := strings.TrimSpace(os.Getenv("NACOS_COPILOT_BASE_URL")); u != "" {
		base.BaseURL = u
	}
	if m := strings.TrimSpace(os.Getenv("NACOS_COPILOT_MODEL")); m != "" {
		base.Model = m
	}
	if strings.TrimSpace(base.Model) == "" {
		base.Model = "gpt-4o-mini"
	}
	if strings.TrimSpace(base.BaseURL) == "" {
		if strings.Contains(strings.ToLower(base.Model), "qwen") {
			base.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		} else {
			base.BaseURL = "https://api.openai.com/v1"
		}
	}
	if !base.Enabled {
		base.Enabled = strings.TrimSpace(os.Getenv("NACOS_COPILOT_ENABLED")) == "1" ||
			strings.EqualFold(strings.TrimSpace(os.Getenv("NACOS_COPILOT_ENABLED")), "true")
	}
	if !base.Enabled && base.ApiKey != "" {
		base.Enabled = true
	}
	return base
}
