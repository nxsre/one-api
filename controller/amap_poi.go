package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/service"
)

// amapPOIRequest 与高德 v5/place/* 查询参数对齐（snake_case），见
// https://lbs.amap.com/api/webservice/guide/api-advanced/newpoisearch
type amapPOIRequest struct {
	Model string `json:"model"`

	// text_search | around | polygon | detail
	Operation string `json:"operation"`

	Keywords  string `json:"keywords"`
	Types     string `json:"types"`
	Region    string `json:"region"`
	CityLimit string `json:"city_limit"`
	Location  string `json:"location"`
	Radius    string `json:"radius"`
	Sortrule  string `json:"sortrule"`
	Polygon   string `json:"polygon"`
	// 详情接口高德参数名为 id；支持 ids 作为别名（内容直接传给高德，多个用 |）
	ID  string `json:"id"`
	IDs string `json:"ids"`

	ShowFields string `json:"show_fields"`
	PageSize   string `json:"page_size"`
	PageNum    string `json:"page_num"`
	Sig        string `json:"sig"`
	Callback   string `json:"callback"`
}

func RelayAmapPOI(c *gin.Context) {
	var req amapPOIRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		abortAmapOpenAIError(c, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	op := strings.TrimSpace(strings.ToLower(req.Operation))
	var suffix string
	switch op {
	case "text_search", "text":
		suffix = "text"
	case "around":
		suffix = "around"
	case "polygon":
		suffix = "polygon"
	case "detail":
		suffix = "detail"
	default:
		abortAmapOpenAIError(c, http.StatusBadRequest, `operation 须为 text_search、around、polygon 或 detail`)
		return
	}

	q := url.Values{}
	add := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			q.Set(k, v)
		}
	}
	add("show_fields", req.ShowFields)
	add("page_size", req.PageSize)
	add("page_num", req.PageNum)
	add("sig", req.Sig)
	add("callback", req.Callback)

	switch suffix {
	case "text":
		add("keywords", req.Keywords)
		add("types", req.Types)
		if req.Keywords == "" && req.Types == "" {
			abortAmapOpenAIError(c, http.StatusBadRequest, "关键字搜索须至少提供 keywords 或 types 之一")
			return
		}
		add("region", req.Region)
		add("city_limit", req.CityLimit)
	case "around":
		if strings.TrimSpace(req.Location) == "" {
			abortAmapOpenAIError(c, http.StatusBadRequest, "周边搜索须提供 location（经度,纬度）")
			return
		}
		add("location", req.Location)
		add("keywords", req.Keywords)
		add("types", req.Types)
		add("radius", req.Radius)
		add("sortrule", req.Sortrule)
		add("region", req.Region)
		add("city_limit", req.CityLimit)
	case "polygon":
		if strings.TrimSpace(req.Polygon) == "" {
			abortAmapOpenAIError(c, http.StatusBadRequest, "多边形搜索须提供 polygon")
			return
		}
		add("polygon", req.Polygon)
		add("keywords", req.Keywords)
		add("types", req.Types)
	case "detail":
		id := strings.TrimSpace(req.ID)
		if id == "" {
			id = strings.TrimSpace(req.IDs)
		}
		if id == "" {
			abortAmapOpenAIError(c, http.StatusBadRequest, "ID 详情须提供 id 或 ids（高德参数 id，多个 POI 用 | 分隔）")
			return
		}
		q.Set("id", id)
	}

	key := service.ResolveAmapWebServiceKey()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()

	status, raw, err := service.CallAmapPlaceV5(ctx, key, suffix, q)
	if err != nil {
		abortAmapOpenAIError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		msg := fmt.Sprintf("高德 HTTP %d", status)
		if len(raw) > 0 {
			msg += ": " + string(raw)
		}
		abortAmapOpenAIError(c, http.StatusBadGateway, msg)
		return
	}

	content := string(raw)
	resp := gin.H{
		"id":      fmt.Sprintf("amap-%s", uuid.NewString()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "amap-place-v5",
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": gin.H{
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
		},
	}
	c.JSON(http.StatusOK, resp)
}

func abortAmapOpenAIError(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{
		"error": gin.H{
			"message": helper.MessageWithRequestId(msg, c.GetString(helper.RequestIdKey)),
			"type":    "invalid_request_error",
		},
	})
}
