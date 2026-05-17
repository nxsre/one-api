package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/relayctx"
	"github.com/songquanpeng/one-api/common/requestaudit"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/service"
)

const amapProxyAuditModel = "amap"

// amapProxyRequest one-api 高德 REST 代理：**POST /amap**（与 OpenAI `/v1` 无关）。
// 客户端 Bearer 为用户令牌；服务端注入高德 Web Key 并转发至 https://restapi.amap.com 。
//
// 允许的 path 前缀：/v3/direction/、/v4/direction/、/v5/direction/、/v5/place/
// （涵盖路径规划 v3/v4、路径规划 2.0（v5）、POI 2.0（v5））。
//
// 文档：
// - 路径规划 v3/v4：https://lbs.amap.com/api/webservice/guide/api/direction
// - 路径规划 2.0：https://lbs.amap.com/api/webservice/guide/api/newroute
// - POI 2.0：https://lbs.amap.com/api/webservice/guide/api-advanced/newpoisearch
type amapProxyRequest struct {
	Method string `json:"method"` // GET 或 POST，默认 GET
	Path   string `json:"path"`   // 如 "/v5/direction/walking"、"/v3/direction/driving"
	// Query 查询参数（不要传 key；服务端会丢弃客户端传入的 key 并注入配置的高德 Key）
	Query map[string]string `json:"query"`
	// Body POST 时的表单原文（application/x-www-form-urlencoded），可与 query 合并
	Body string `json:"body"`
}

func RelayAmapProxy(c *gin.Context) {
	// 与 Relay 栈中 Distribute() 一致：把客户端 IP 写入 context，供 RecordConsumeLog 等写入「日志」中的 ip 字段。
	c.Request = c.Request.WithContext(relayctx.WithClientIP(c.Request.Context(), c.ClientIP()))

	startAt := time.Now()
	requestaudit.Attach(c, startAt)

	rawBody, err := common.GetRequestBody(c)
	if err != nil {
		recordAmapProxyConsume(c, startAt, fmt.Sprintf("读取请求体失败: %v", err))
		requestaudit.FinalizeHTTPResult(c, amapProxyAuditModel, http.StatusBadRequest, false, err.Error(), 0)
		abortAmapAPIError(c, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	var req amapProxyRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		recordAmapProxyConsume(c, startAt, fmt.Sprintf("JSON 无效: %v", err))
		requestaudit.FinalizeHTTPResult(c, amapProxyAuditModel, http.StatusBadRequest, false, err.Error(), 0)
		abortAmapAPIError(c, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if uid := c.GetInt(ctxkey.Id); uid > 0 {
		if g, _ := model.CacheGetUserGroup(uid); g != "" {
			c.Set(ctxkey.Group, g)
		}
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	if method != http.MethodGet && method != http.MethodPost {
		recordAmapProxyConsume(c, startAt, "method 须为 GET 或 POST")
		requestaudit.FinalizeHTTPResult(c, amapProxyAuditModel, http.StatusBadRequest, false, "method 须为 GET 或 POST", 0)
		abortAmapAPIError(c, http.StatusBadRequest, "method 须为 GET 或 POST")
		return
	}
	pathNorm := service.NormalizeAmapRelayPath(req.Path)
	if pathNorm == "" || !service.IsAllowedAmapRelayPath(req.Path) {
		msg := "path 无效或不在允许列表内（允许前缀: /v3/direction/、/v4/direction/、/v5/direction/、/v5/place/）"
		recordAmapProxyConsume(c, startAt, msg)
		requestaudit.FinalizeHTTPResult(c, amapProxyAuditModel, http.StatusBadRequest, false, msg, 0)
		abortAmapAPIError(c, http.StatusBadRequest, msg)
		return
	}

	q := url.Values{}
	for k, v := range req.Query {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		q.Set(k, v)
	}

	ctx := c.Request.Context()
	snap := func(upReq *http.Request, upResp *http.Response) {
		requestaudit.SnapUpstreamHTTP(c, upReq, upResp)
	}

	var (
		status      int
		contentType string
		raw         []byte
	)
	switch method {
	case http.MethodGet:
		status, contentType, raw, err = service.CallAmapUpstreamGET(ctx, req.Path, q, "", snap)
	case http.MethodPost:
		status, contentType, raw, err = service.CallAmapUpstreamPOSTForm(ctx, req.Path, q, req.Body, "", snap)
	}

	if err != nil {
		recordAmapProxyConsume(c, startAt, fmt.Sprintf("%s %s 上游错误: %v", method, pathNorm, err))
		requestaudit.FinalizeHTTPResult(c, amapProxyAuditModel, http.StatusInternalServerError, false, err.Error(), 0)
		abortAmapAPIError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if rec := requestaudit.FromContext(c); rec != nil {
		requestaudit.SetNonStreamResponse(rec, string(raw))
	}
	success := status >= 200 && status < 400
	errMsg := ""
	if !success {
		errMsg = fmt.Sprintf("upstream HTTP %d", status)
	}
	requestaudit.FinalizeHTTPResult(c, amapProxyAuditModel, status, success, errMsg, 0)

	recordAmapProxyConsume(c, startAt, fmt.Sprintf("%s %s upstream_http=%d", method, pathNorm, status))

	c.Header("X-OneAPI-Amap-Upstream-Status", strconv.Itoa(status))
	c.Data(status, contentType, raw)
}

func recordAmapProxyConsume(c *gin.Context, startAt time.Time, content string) {
	model.RecordConsumeLog(c.Request.Context(), &model.Log{
		UserId:      c.GetInt(ctxkey.Id),
		TokenId:     c.GetInt(ctxkey.TokenId),
		TokenName:   c.GetString(ctxkey.TokenName),
		Group:       c.GetString(ctxkey.Group),
		ModelName:   amapProxyAuditModel,
		Content:     content,
		Other:       requestaudit.UpstreamHeadersJSONForLog(c),
		ElapsedTime: time.Since(startAt).Milliseconds(),
	})
}

func abortAmapAPIError(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{
		"error": gin.H{
			"message": helper.MessageWithRequestId(msg, c.GetString(helper.RequestIdKey)),
			"type":    "invalid_request_error",
		},
	})
}
