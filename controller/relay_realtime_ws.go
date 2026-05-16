package controller

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	relaymeta "github.com/songquanpeng/one-api/relay/meta"
)

var realtimeUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// RelayRealtimeWebSocket 透明代理 OpenAI Realtime WebSocket（客户端 GET /v1/realtime?model=… Upgrade）。
func RelayRealtimeWebSocket(c *gin.Context) {
	ctx := c.Request.Context()
	meta := relaymeta.GetByContext(c)
	target := realtimeUpstreamWSURL(meta.BaseURL, c.Request.URL.RawQuery)
	if target == "" {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"type":    "one_api_error",
				"message": "无法构造上游 WebSocket URL，请检查渠道的 Base URL",
			},
		})
		return
	}

	hdr := http.Header{}
	if key := strings.TrimSpace(c.GetString(ctxkey.ChannelKey)); key != "" {
		hdr.Set("Authorization", "Bearer "+key)
	}

	clientConn, err := realtimeUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Errorf(ctx, "RelayRealtimeWebSocket: upgrade client: %v", err)
		return
	}
	defer clientConn.Close()

	d := websocket.Dialer{HandshakeTimeout: 45 * time.Second}
	backendConn, resp, err := d.Dial(target, hdr)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		logger.Errorf(ctx, "RelayRealtimeWebSocket: dial upstream %s: %v", target, err)
		deadline := time.Now().Add(time.Second)
		_ = clientConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "upstream dial failed"),
			deadline,
		)
		return
	}
	defer backendConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		pipeRealtimeFrames(clientConn, backendConn)
	}()
	go func() {
		defer wg.Done()
		pipeRealtimeFrames(backendConn, clientConn)
	}()
	wg.Wait()
}

func pipeRealtimeFrames(from, to *websocket.Conn) {
	for {
		mt, msg, err := from.ReadMessage()
		if err != nil {
			break
		}
		if err := to.WriteMessage(mt, msg); err != nil {
			break
		}
	}
}

func realtimeUpstreamWSURL(baseURL, rawQuery string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return ""
	}
	scheme := "wss"
	if u.Scheme == "http" {
		scheme = "ws"
	}
	prefix := strings.TrimSuffix(u.Path, "/")
	path := prefix + "/v1/realtime"
	if rawQuery != "" {
		path += "?" + rawQuery
	}
	return scheme + "://" + u.Host + path
}
