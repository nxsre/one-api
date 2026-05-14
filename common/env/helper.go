package env

import (
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

var vv *viper.Viper

// BindViper 在 cfg.Init 之后调用，之后 env.* 从 TOML / 命令行（及 viper.Set）读取。
func BindViper(v *viper.Viper) {
	vv = v
}

func key(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func Bool(name string, defaultValue bool) bool {
	if vv == nil {
		return defaultValue
	}
	return vv.GetBool(key(name))
}

func Int(name string, defaultValue int) int {
	if vv == nil {
		return defaultValue
	}
	return vv.GetInt(key(name))
}

func Float64(name string, defaultValue float64) float64 {
	if vv == nil {
		return defaultValue
	}
	return vv.GetFloat64(key(name))
}

func String(name string, defaultValue string) string {
	if vv == nil {
		return defaultValue
	}
	return vv.GetString(key(name))
}

// StringAlways 返回 viper 中的字符串（依赖 cfg 中的 SetDefault）。
func StringAlways(name string) string {
	if vv == nil {
		return ""
	}
	return vv.GetString(key(name))
}

// IntAlways 返回整型（依赖 SetDefault）。
func IntAlways(name string) int {
	if vv == nil {
		return 0
	}
	return vv.GetInt(key(name))
}

// Int64Always 返回 int64。
func Int64Always(name string) int64 {
	if vv == nil {
		return 0
	}
	return vv.GetInt64(key(name))
}

// BoolAlways 返回布尔。
func BoolAlways(name string) bool {
	if vv == nil {
		return false
	}
	return vv.GetBool(key(name))
}

// ParseBoolLoose 解析 "1"/"true"/"yes"/"on" 等，空串返回 defaultValue。
func ParseBoolLoose(s string, defaultValue bool) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return defaultValue
	}
	switch s {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// ParseInt 十进制整数，失败返回 defaultValue。
func ParseInt(s string, defaultValue int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return n
}

// ParseInt64 十进制 int64，失败返回 defaultValue。
func ParseInt64(s string, defaultValue int64) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultValue
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return defaultValue
	}
	return n
}
