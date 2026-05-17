package protocolbridge

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	adaptorpkg "github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

const ginNoWritten = -1

// ginRecorder 实现 gin.ResponseWriter，便于在无真实 HTTP 连接时捕获 DoResponse 输出。
type ginRecorder struct {
	*httptest.ResponseRecorder
	bytesWritten int
	status       int
	headerSent   bool
}

func newGinRecorder() *ginRecorder {
	return &ginRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		bytesWritten:     ginNoWritten,
		status:           http.StatusOK,
	}
}

func (g *ginRecorder) WriteHeader(code int) {
	if g.headerSent {
		return
	}
	g.status = code
	g.ResponseRecorder.WriteHeader(code)
	g.headerSent = true
	if g.bytesWritten < 0 {
		g.bytesWritten = 0
	}
}

func (g *ginRecorder) WriteHeaderNow() {
	if !g.headerSent {
		if g.status == 0 {
			g.status = http.StatusOK
		}
		g.WriteHeader(g.status)
	}
}

func (g *ginRecorder) Write(b []byte) (int, error) {
	g.WriteHeaderNow()
	n, err := g.ResponseRecorder.Write(b)
	g.bytesWritten += n
	return n, err
}

func (g *ginRecorder) WriteString(s string) (int, error) {
	return g.Write([]byte(s))
}

func (g *ginRecorder) Status() int {
	return g.status
}

func (g *ginRecorder) Size() int {
	if g.bytesWritten < 0 {
		return 0
	}
	return g.bytesWritten
}

func (g *ginRecorder) Written() bool {
	return g.bytesWritten != ginNoWritten
}

func (g *ginRecorder) CloseNotify() <-chan bool {
	ch := make(chan bool, 1)
	return ch
}

func (g *ginRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("hijack not supported")
}

func (g *ginRecorder) Flush() {
	g.WriteHeaderNow()
	g.ResponseRecorder.Flush()
}

func (g *ginRecorder) Pusher() http.Pusher {
	return nil
}

// CaptureAdaptorResponse 在同一次请求上下文副本上执行 Adaptor.DoResponse，把写入 Gin 的响应体完整存入内存（用于协议转换后再写给真实客户端）。
func CaptureAdaptorResponse(c *gin.Context, adaptor adaptorpkg.Adaptor, resp *http.Response, meta *meta.Meta) (usage *model.Usage, rawBody []byte, statusCode int, header http.Header, bizErr *model.ErrorWithStatusCode) {
	if c == nil || adaptor == nil || meta == nil {
		return nil, nil, http.StatusInternalServerError, nil, nil
	}
	cp := c.Copy()
	rec := newGinRecorder()
	cp.Writer = rec
	usage, bizErr = adaptor.DoResponse(cp, resp, meta)
	return usage, rec.Body.Bytes(), rec.Code, rec.Header(), bizErr
}

var _ gin.ResponseWriter = (*ginRecorder)(nil)
