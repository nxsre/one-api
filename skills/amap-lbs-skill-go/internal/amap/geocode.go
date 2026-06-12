package amap

import (
	"fmt"
	"strings"
)

// Geocode 为地理编码（地名→坐标）单条结果。
type Geocode struct {
	FormattedAddress flexString `json:"formatted_address"`
	Province         flexString `json:"province"`
	City             flexString `json:"city"`
	District         flexString `json:"district"`
	Adcode           flexString `json:"adcode"`
	Location         string     `json:"location"` // "经度,纬度"
}

// GeocodeResponse 为 /v3/geocode/geo 响应。
type GeocodeResponse struct {
	Status   string    `json:"status"`
	Info     string    `json:"info"`
	Count    string    `json:"count"`
	Geocodes []Geocode `json:"geocodes"`
}

// ReGeocodeResponse 为 /v3/geocode/regeo（逆地理编码，坐标→地址）响应。
type ReGeocodeResponse struct {
	Status    string `json:"status"`
	Info      string `json:"info"`
	Regeocode struct {
		FormattedAddress flexString `json:"formatted_address"`
	} `json:"regeocode"`
}

// Geocode 地理编码：把地名/地址转为经纬度（高德 v3，经网关）。
func (c *Client) Geocode(address, city string) (*GeocodeResponse, error) {
	if strings.TrimSpace(address) == "" {
		return nil, fmt.Errorf("地理编码须提供 address")
	}
	q := map[string]string{"address": strings.TrimSpace(address)}
	if c := strings.TrimSpace(city); c != "" {
		q["city"] = c
	}
	var r GeocodeResponse
	if err := c.getJSON("/v3/geocode/geo", q, &r); err != nil {
		return nil, err
	}
	if r.Status != "1" {
		return nil, fmt.Errorf("地理编码失败: %s", r.Info)
	}
	if len(r.Geocodes) == 0 {
		return nil, fmt.Errorf("未找到「%s」的坐标", address)
	}
	return &r, nil
}

// GeocodeLocation 是 Geocode 的便捷封装，直接返回首个结果的坐标与规范化地址。
func (c *Client) GeocodeLocation(address, city string) (location, formattedAddress string, err error) {
	r, err := c.Geocode(address, city)
	if err != nil {
		return "", "", err
	}
	g := r.Geocodes[0]
	return g.Location, g.FormattedAddress.String(), nil
}

// ReverseGeocode 逆地理编码：把经纬度转为结构化地址（高德 v3，经网关）。
func (c *Client) ReverseGeocode(location string) (*ReGeocodeResponse, error) {
	if strings.TrimSpace(location) == "" {
		return nil, fmt.Errorf("逆地理编码须提供 location（经度,纬度）")
	}
	var r ReGeocodeResponse
	if err := c.getJSON("/v3/geocode/regeo", map[string]string{
		"location": strings.TrimSpace(location),
	}, &r); err != nil {
		return nil, err
	}
	if r.Status != "1" {
		return nil, fmt.Errorf("逆地理编码失败: %s", r.Info)
	}
	return &r, nil
}
