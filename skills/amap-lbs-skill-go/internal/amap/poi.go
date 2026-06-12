package amap

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// SearchParams 为 POI 搜索参数。
type SearchParams struct {
	Keywords  string
	City      string
	Types     string
	Location  string // "经度,纬度"，非空则走周边搜索 /v5/place/around
	Radius    string
	Page      int
	Offset    int
	CityLimit *bool // nil 视为 true
}

// POI 为单条地点结果。
type POI struct {
	Name     string     `json:"name"`
	Address  flexString `json:"address"`
	Type     flexString `json:"type"`
	Tel      flexString `json:"tel"`
	Location string     `json:"location"`
	Distance flexString `json:"distance"`
}

// POIResponse 为归一化后的搜索结果。
type POIResponse struct {
	Status string
	Info   string
	Count  string
	Pois   []POI
}

// CountInt 以整数返回结果总数（解析失败时为 0）。
func (r *POIResponse) CountInt() int {
	n, _ := strconv.Atoi(strings.TrimSpace(r.Count))
	return n
}

func normalizeGaodePOIResponse(raw []byte) (*POIResponse, error) {
	var probe struct {
		Status string          `json:"status"`
		Info   flexString      `json:"info"`
		Count  flexString      `json:"count"`
		Pois   json.RawMessage `json:"pois"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	resp := &POIResponse{Status: probe.Status, Info: probe.Info.String(), Count: probe.Count.String(), Pois: []POI{}}
	if len(probe.Pois) > 0 {
		var arr []POI
		if err := json.Unmarshal(probe.Pois, &arr); err == nil {
			resp.Pois = arr
		} else {
			// 兼容 v3 形态：pois 为 {"poi":[...]}
			var obj struct {
				Poi []POI `json:"poi"`
			}
			if err := json.Unmarshal(probe.Pois, &obj); err == nil && obj.Poi != nil {
				resp.Pois = obj.Poi
			}
		}
	}
	return resp, nil
}

// SearchPOI 经网关 POST /amap 调用高德 POI 2.0（/v5/place/text 或 /v5/place/around）。
func (c *Client) SearchPOI(p SearchParams) (*POIResponse, error) {
	page := p.Page
	if page <= 0 {
		page = 1
	}
	offset := p.Offset
	if offset <= 0 {
		offset = 10
	}

	req := amapRequest{Method: http.MethodGet, Query: map[string]string{}}
	q := req.Query
	q["page_num"] = strconv.Itoa(page)
	q["page_size"] = strconv.Itoa(offset)
	if t := strings.TrimSpace(p.Types); t != "" {
		q["types"] = t
	}
	cityLimit := "true"
	if p.CityLimit != nil && !*p.CityLimit {
		cityLimit = "false"
	}

	if loc := strings.TrimSpace(p.Location); loc != "" {
		req.Path = "/v5/place/around"
		q["location"] = loc
		q["keywords"] = p.Keywords
		if r := strings.TrimSpace(p.Radius); r != "" {
			q["radius"] = r
		} else {
			q["radius"] = "1000"
		}
		if city := strings.TrimSpace(p.City); city != "" {
			q["region"] = city
		}
		q["city_limit"] = cityLimit
	} else {
		req.Path = "/v5/place/text"
		q["keywords"] = p.Keywords
		if city := strings.TrimSpace(p.City); city != "" {
			q["region"] = city
		}
		q["city_limit"] = cityLimit
		if strings.TrimSpace(q["keywords"]) == "" && strings.TrimSpace(q["types"]) == "" {
			return nil, errors.New("关键字搜索须至少提供 keywords 或 types")
		}
	}

	status, body, err := c.callAmap(req)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		snippet := string(body)
		if len(snippet) > 400 {
			snippet = snippet[:400]
		}
		return nil, fmt.Errorf("网关 HTTP %d %s", status, snippet)
	}

	norm, err := normalizeGaodePOIResponse(body)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if norm.Status != "1" {
		return nil, fmt.Errorf("高德返回失败: %s", norm.Info)
	}
	return norm, nil
}
