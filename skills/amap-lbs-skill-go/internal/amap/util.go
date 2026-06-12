package amap

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// flexString 兼容高德接口中"有时是字符串、有时是数字、有时是空数组"的字段。
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*f = ""
		return nil
	}
	switch b[0] {
	case '"':
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
	case '[':
		var a []string
		if err := json.Unmarshal(b, &a); err == nil {
			*f = flexString(strings.Join(a, ","))
		} else {
			*f = ""
		}
	default: // 数字、布尔等：原样保留
		*f = flexString(strings.Trim(string(b), `"`))
	}
	return nil
}

func (f flexString) String() string { return string(f) }

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

// parseLngLat 解析 "经度,纬度" 字符串。
func parseLngLat(loc string) (lng, lat float64) {
	parts := strings.Split(loc, ",")
	if len(parts) != 2 {
		return 0, 0
	}
	return atof(parts[0]), atof(parts[1])
}
