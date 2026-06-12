package amap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// RoutePath 为步行/驾车单条路径（v3）。
type RoutePath struct {
	Distance      flexString `json:"distance"`
	Duration      flexString `json:"duration"`
	Tolls         flexString `json:"tolls"`
	TrafficLights flexString `json:"traffic_lights"`
}

// V3RouteResponse 为步行/驾车响应（v3）。
type V3RouteResponse struct {
	Status string `json:"status"`
	Info   string `json:"info"`
	Route  struct {
		Paths []RoutePath `json:"paths"`
	} `json:"route"`
}

// RidingPath 为骑行单条路径（v4）。
type RidingPath struct {
	Distance flexString `json:"distance"`
	Duration flexString `json:"duration"`
}

// V4RidingResponse 为骑行响应（v4，成功为 errcode==0）。
type V4RidingResponse struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
	Data    struct {
		Paths []RidingPath `json:"paths"`
	} `json:"data"`
}

// Transit 为单个公交方案（v3）。
type Transit struct {
	Duration        flexString `json:"duration"`
	Cost            flexString `json:"cost"`
	WalkingDistance flexString `json:"walking_distance"`
}

// V3TransitResponse 为公交响应（v3）。
type V3TransitResponse struct {
	Status string `json:"status"`
	Info   string `json:"info"`
	Route  struct {
		Transits []Transit `json:"transits"`
	} `json:"route"`
}

// getJSON 经网关 POST /amap 发起 GET 并解析 JSON 到 out。
func (c *Client) getJSON(path string, query map[string]string, out any) error {
	status, body, err := c.callAmap(amapRequest{Method: http.MethodGet, Path: path, Query: query})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		snippet := string(body)
		if len(snippet) > 400 {
			snippet = snippet[:400]
		}
		return fmt.Errorf("网关 HTTP %d %s", status, snippet)
	}
	return json.Unmarshal(body, out)
}

// WalkingRoute 步行路径规划（高德 v3，经网关）。
func (c *Client) WalkingRoute(origin, destination string) (*V3RouteResponse, error) {
	var r V3RouteResponse
	if err := c.getJSON("/v3/direction/walking", map[string]string{
		"origin": origin, "destination": destination,
	}, &r); err != nil {
		return nil, err
	}
	if r.Status != "1" {
		return nil, fmt.Errorf("步行路线规划失败: %s", r.Info)
	}
	return &r, nil
}

// DrivingParams 为驾车路径规划参数。
type DrivingParams struct {
	Origin      string
	Destination string
	Waypoints   string
	Strategy    int // 默认 10
}

// DrivingRoute 驾车路径规划（高德 v3，经网关）。
func (c *Client) DrivingRoute(p DrivingParams) (*V3RouteResponse, error) {
	strategy := p.Strategy
	if strategy == 0 {
		strategy = 10
	}
	q := map[string]string{
		"origin": p.Origin, "destination": p.Destination,
		"strategy": strconv.Itoa(strategy), "extensions": "base",
	}
	if p.Waypoints != "" {
		q["waypoints"] = p.Waypoints
	}
	var r V3RouteResponse
	if err := c.getJSON("/v3/direction/driving", q, &r); err != nil {
		return nil, err
	}
	if r.Status != "1" {
		return nil, fmt.Errorf("驾车路线规划失败: %s", r.Info)
	}
	return &r, nil
}

// RidingRoute 骑行路径规划（高德 v4，经网关）。
func (c *Client) RidingRoute(origin, destination string) (*V4RidingResponse, error) {
	var r V4RidingResponse
	if err := c.getJSON("/v4/direction/bicycling", map[string]string{
		"origin": origin, "destination": destination,
	}, &r); err != nil {
		return nil, err
	}
	if r.Errcode != 0 {
		return nil, fmt.Errorf("骑行路线规划失败: %s", r.Errmsg)
	}
	return &r, nil
}

// TransitParams 为公交路径规划参数。
type TransitParams struct {
	Origin      string
	Destination string
	City        string
	Strategy    int
	Nightflag   bool
}

// TransitRoute 公交路径规划（高德 v3，经网关）。
func (c *Client) TransitRoute(p TransitParams) (*V3TransitResponse, error) {
	night := "0"
	if p.Nightflag {
		night = "1"
	}
	var r V3TransitResponse
	if err := c.getJSON("/v3/direction/transit/integrated", map[string]string{
		"origin": p.Origin, "destination": p.Destination, "city": p.City,
		"strategy": strconv.Itoa(p.Strategy), "nightflag": night,
	}, &r); err != nil {
		return nil, err
	}
	if r.Status != "1" {
		return nil, fmt.Errorf("公交路线规划失败: %s", r.Info)
	}
	return &r, nil
}
