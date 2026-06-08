package core

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// ModelSet 指定某 Target 下各接口形态使用的模型名。
type ModelSet struct {
	Chat       string `json:"chat,omitempty"`
	Completion string `json:"completion,omitempty"`
	Embedding  string `json:"embedding,omitempty"`
}

// Target 一个被测端点：协议 + 基址 + 鉴权 + 模型。
// 既可指向 one-api 网关（OpenAI 兼容入口），也可指向各家原生 API。
type Target struct {
	Name     string            `json:"name"`
	Protocol Protocol          `json:"protocol"`
	BaseURL  string            `json:"base_url"`
	APIKey   string            `json:"api_key"`
	Models   ModelSet          `json:"models"`
	Headers  map[string]string `json:"headers,omitempty"` // 额外请求头（可选）
}

// Config 测试运行配置。
type Config struct {
	Targets        []Target          `json:"targets"`
	Scenarios      []string          `json:"scenarios,omitempty"`     // 空=全部内置场景
	CustomInputs   map[string]string `json:"custom_inputs,omitempty"` // 场景ID -> 自定义用户提示（覆盖随机生成）
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	Concurrency    int               `json:"concurrency,omitempty"`
	Seed           int64             `json:"seed,omitempty"`
}

var envRe = regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)

// expandEnv 把字符串中的 ${VAR} 替换为环境变量值，方便把密钥放到环境里而非配置文件。
func expandEnv(s string) string {
	return envRe.ReplaceAllStringFunc(s, func(m string) string {
		name := m[2 : len(m)-1]
		return os.Getenv(name)
	})
}

// LoadConfig 从 JSON 文件读取配置并做基本校验与默认值填充。
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置 JSON 失败: %w", err)
	}
	if len(cfg.Targets) == 0 {
		return nil, fmt.Errorf("配置中至少需要一个 target")
	}
	for i := range cfg.Targets {
		t := &cfg.Targets[i]
		t.APIKey = expandEnv(t.APIKey)
		t.BaseURL = expandEnv(t.BaseURL)
		if t.Name == "" {
			t.Name = fmt.Sprintf("target-%d", i+1)
		}
		if t.Protocol == "" {
			return nil, fmt.Errorf("target %q 缺少 protocol（openai|anthropic|gemini）", t.Name)
		}
		switch t.Protocol {
		case ProtocolOpenAI, ProtocolAnthropic, ProtocolGemini:
		default:
			return nil, fmt.Errorf("target %q 的 protocol=%q 不受支持", t.Name, t.Protocol)
		}
		if t.BaseURL == "" {
			return nil, fmt.Errorf("target %q 缺少 base_url", t.Name)
		}
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 60
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	return &cfg, nil
}

// ExampleConfig 返回一份可直接改用的示例配置 JSON。
func ExampleConfig() []byte {
	cfg := Config{
		Targets: []Target{
			{
				Name:     "one-api-gateway",
				Protocol: ProtocolOpenAI,
				BaseURL:  "http://localhost:13000",
				APIKey:   "${ONEAPI_KEY}",
				Models:   ModelSet{Chat: "gpt-4o-mini", Completion: "gpt-3.5-turbo-instruct", Embedding: "text-embedding-3-small"},
			},
			{
				Name:     "openai-native",
				Protocol: ProtocolOpenAI,
				BaseURL:  "https://api.openai.com",
				APIKey:   "${OPENAI_API_KEY}",
				Models:   ModelSet{Chat: "gpt-4o-mini", Embedding: "text-embedding-3-small"},
			},
			{
				Name:     "anthropic-native",
				Protocol: ProtocolAnthropic,
				BaseURL:  "https://api.anthropic.com",
				APIKey:   "${ANTHROPIC_API_KEY}",
				Models:   ModelSet{Chat: "claude-3-5-sonnet-20241022"},
			},
			{
				Name:     "gemini-native",
				Protocol: ProtocolGemini,
				BaseURL:  "https://generativelanguage.googleapis.com",
				APIKey:   "${GEMINI_API_KEY}",
				Models:   ModelSet{Chat: "gemini-1.5-flash"},
			},
		},
		TimeoutSeconds: 60,
		Concurrency:    4,
		Seed:           0,
		CustomInputs:   map[string]string{},
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return b
}
