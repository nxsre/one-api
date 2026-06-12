package amap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	openclawDir     = ".openclaw"
	configFileName  = ".amap-lbs-skill.json"
	fixedConfigPath = "/home/node/.openclaw/.amap-lbs-skill.json" // openclaw 容器默认落点
)

// Config 为配置文件结构（命名中性，不绑定具体网关实现）。
type Config struct {
	BaseURL  string `json:"base_url"` // 网关地址，如 http://127.0.0.1:13000
	APIKey   string `json:"apikey"`   // 网关 API Key，作 Authorization: Bearer 用
	Insecure bool   `json:"insecure"` // 跳过自签 HTTPS 证书校验（仅调试/内网自签）
}

// candidateConfigPaths 按优先级返回候选配置文件路径：
// 显式 path > AMAP_SKILL_CONFIG > $HOME/.openclaw/.amap-lbs-skill.json > /home/node/.openclaw/.amap-lbs-skill.json > ./.amap-lbs-skill.json。
func candidateConfigPaths(explicit string) []string {
	var paths []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		for _, e := range paths {
			if e == p {
				return
			}
		}
		paths = append(paths, p)
	}
	add(explicit)
	add(os.Getenv("AMAP_SKILL_CONFIG"))
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, openclawDir, configFileName))
	}
	add(fixedConfigPath)
	add(configFileName) // ./.amap-lbs-skill.json（开发用）
	return paths
}

// DefaultConfigPath 返回推荐的落点，用于提示与初始化。
func DefaultConfigPath() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, openclawDir, configFileName)
	}
	return fixedConfigPath
}

// 环境变量（兜底）：配置文件缺省字段时从这里取。命名中性，便于 openclaw 注入。
const (
	envBaseURL  = "AMAP_SKILL_BASE_URL"
	envAPIKey   = "AMAP_SKILL_APIKEY"
	envInsecure = "AMAP_SKILL_INSECURE"
)

// LoadConfig 综合「配置文件」与「环境变量」得到最终配置：**文件优先、env 兜底**。
// 找不到配置文件不算错误（可纯用环境变量）；仅读/解析失败才返回错误。
// 返回配置、命中的文件路径（纯 env 时为空）、错误。
func LoadConfig(explicit string) (Config, string, error) {
	cfg, path, _, err := loadConfigFile(explicit)
	if err != nil {
		return cfg, path, err
	}
	return applyEnvFallback(cfg), path, nil
}

// loadConfigFile 按优先级查找并读取配置文件；未找到时返回 found=false（非错误）。
func loadConfigFile(explicit string) (cfg Config, path string, found bool, err error) {
	for _, p := range candidateConfigPaths(explicit) {
		info, e := os.Stat(p)
		if e != nil || info.IsDir() {
			continue
		}
		data, e := os.ReadFile(p)
		if e != nil {
			return cfg, p, true, fmt.Errorf("读取配置文件 %s 失败: %w", p, e)
		}
		if e := json.Unmarshal(data, &cfg); e != nil {
			return cfg, p, true, fmt.Errorf("配置文件 %s 不是合法 JSON: %w", p, e)
		}
		warnIfPermissive(p, info)
		return cfg, p, true, nil
	}
	return cfg, "", false, nil
}

// applyEnvFallback 用环境变量填补配置文件中缺省的字段（文件优先）。
func applyEnvFallback(cfg Config) Config {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = strings.TrimSpace(os.Getenv(envBaseURL))
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = strings.TrimSpace(os.Getenv(envAPIKey))
	}
	if !cfg.Insecure {
		cfg.Insecure = envTruthy(os.Getenv(envInsecure))
	}
	return cfg
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	}
	return false
}

func warnIfPermissive(path string, info os.FileInfo) {
	if runtime.GOOS == "windows" {
		return
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "⚠️  配置文件 %s 权限过松（%#o），内含密钥，建议 chmod 600\n", path, perm)
	}
}

// credential 为访问网关所需的凭证。
type credential struct {
	Base     string
	Key      string
	Insecure bool
}

// credential 从配置导出凭证；base_url / apikey 任一为空则视为未就绪。
func (c Config) credential() (credential, bool) {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	key := strings.TrimSpace(c.APIKey)
	if base == "" || key == "" {
		return credential{}, false
	}
	return credential{Base: base, Key: key, Insecure: c.Insecure}, true
}
