package relayctx

import "context"

type clientIPKeyType struct{}

var clientIPKey = clientIPKeyType{}

// WithClientIP 将客户端 IP 写入 context，供异步记录日志使用。
func WithClientIP(parent context.Context, ip string) context.Context {
	if ip == "" {
		return parent
	}
	return context.WithValue(parent, clientIPKey, ip)
}

// ClientIP 从 context 读取客户端 IP（可能为空）。
func ClientIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(clientIPKey).(string); ok {
		return v
	}
	return ""
}
