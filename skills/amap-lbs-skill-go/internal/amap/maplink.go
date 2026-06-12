package amap

import (
	"encoding/json"
	"net/url"
	"strings"
)

// MapTask 为地图可视化任务（兼容 poi 与 route 两种形态）。
type MapTask struct {
	Type      string    `json:"type"`
	Lnglat    []float64 `json:"lnglat,omitempty"`
	Sort      string    `json:"sort,omitempty"`
	Text      string    `json:"text,omitempty"`
	Remark    string    `json:"remark,omitempty"`
	RouteType string    `json:"routeType,omitempty"`
	Start     []float64 `json:"start,omitempty"`
	End       []float64 `json:"end,omitempty"`
	City      string    `json:"city,omitempty"`
}

// GenerateMapLink 生成地图可视化链接（与 JS encodeURIComponent 对齐：空格编码为 %20）。
func GenerateMapLink(tasks []MapTask) string {
	data, _ := json.Marshal(tasks)
	encoded := strings.ReplaceAll(url.QueryEscape(string(data)), "+", "%20")
	return "https://a.amap.com/jsapi_demo_show/static/openclaw/travel_plan.html?data=" + encoded
}
