package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/requestaudit"
)

type consumeLogWriter struct {
	gin.ResponseWriter
	c            *gin.Context
	bytesWritten int
	snapped      bool
}

func (w *consumeLogWriter) snapResponse() {
	if w.snapped || w.c == nil {
		return
	}
	w.snapped = true
	requestaudit.SnapClientResponse(w.c, w.ResponseWriter)
}

func (w *consumeLogWriter) WriteHeader(statusCode int) {
	w.snapResponse()
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *consumeLogWriter) Write(b []byte) (int, error) {
	w.snapResponse()
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

// ConsumeLogCapture 记录客户端响应头与响应体字节数，供消费日志 other 字段展示。
func ConsumeLogCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		w := &consumeLogWriter{ResponseWriter: c.Writer, c: c}
		c.Writer = w
		c.Next()
		c.Set(ctxkey.ResponseBodyBytesLog, w.bytesWritten)
		if !w.snapped {
			w.snapResponse()
		}
	}
}
