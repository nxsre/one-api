// Package awsv4 提供对 AWS Signature Version 4（UNSIGNED-PAYLOAD）请求的校验，用于 S3 兼容子集。
package awsv4

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	unsignedPayloadMarker = "UNSIGNED-PAYLOAD"
	algorithm             = "AWS4-HMAC-SHA256"
)

// Config 与服务端配置的访问密钥及区域一致。
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

// VerifyRequest 校验 Authorization: AWS4-HMAC-SHA256 ... 与 x-amz-date。
// 当 SignedHeaders 包含 x-amz-content-sha256 且值为 UNSIGNED-PAYLOAD 时不读取 body；
// 否则读取 body 计算 SHA256（适合小对象）。
func VerifyRequest(r *http.Request, cfg Config, clockSkew time.Duration) error {
	if cfg.SecretAccessKey == "" || cfg.AccessKeyID == "" || cfg.Region == "" {
		return fmt.Errorf("awsv4: invalid server config")
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, algorithm+" ") {
		return fmt.Errorf("awsv4: unsupported authorization scheme")
	}
	parts := parseAuthParams(auth[len(algorithm)+1:])
	cred, ok := parts["Credential"]
	if !ok {
		return fmt.Errorf("awsv4: missing Credential")
	}
	sig, ok := parts["Signature"]
	if !ok {
		return fmt.Errorf("awsv4: missing Signature")
	}
	signedHeadersStr, ok := parts["SignedHeaders"]
	if !ok {
		return fmt.Errorf("awsv4: missing SignedHeaders")
	}
	credParts := strings.Split(cred, "/")
	if len(credParts) != 5 || credParts[4] != "aws4_request" {
		return fmt.Errorf("awsv4: invalid credential scope")
	}
	accessKey := credParts[0]
	dateStamp := credParts[1]
	region := credParts[2]
	service := credParts[3]
	if accessKey != cfg.AccessKeyID {
		return fmt.Errorf("awsv4: access key mismatch")
	}
	if region != cfg.Region {
		return fmt.Errorf("awsv4: region mismatch")
	}
	if service != "s3" {
		return fmt.Errorf("awsv4: service must be s3")
	}
	amzDate := r.Header.Get("X-Amz-Date")
	if amzDate == "" {
		return fmt.Errorf("awsv4: missing x-amz-date")
	}
	if len(amzDate) < 8 || amzDate[:8] != dateStamp {
		return fmt.Errorf("awsv4: x-amz-date does not match credential date")
	}
	if err := checkAmzDateSkew(amzDate, clockSkew); err != nil {
		return err
	}

	signedHeaders := strings.Split(signedHeadersStr, ";")
	for i := range signedHeaders {
		signedHeaders[i] = strings.TrimSpace(strings.ToLower(signedHeaders[i]))
	}
	sort.Strings(signedHeaders)

	var canonicalHeaders strings.Builder
	for _, name := range signedHeaders {
		val := headerValue(r, name)
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(val))
		canonicalHeaders.WriteString("\n")
	}

	payloadHash := unsignedPayloadMarker
	if containsFold(signedHeadersStr, "x-amz-content-sha256") {
		ph := strings.TrimSpace(r.Header.Get("X-Amz-Content-Sha256"))
		if ph != "" {
			payloadHash = ph
		}
	}
	if payloadHash != unsignedPayloadMarker {
		body, err := readBodyForHash(r)
		if err != nil {
			return err
		}
		h := sha256.Sum256(body)
		payloadHash = hex.EncodeToString(h[:])
	}

	canonicalURI := r.URL.EscapedPath()
	canonicalQuery := canonicalQueryString(r.URL.RawQuery)

	cr := strings.Join([]string{
		r.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders.String(),
		strings.Join(signedHeaders, ";"),
		payloadHash,
	}, "\n")
	crHash := sha256.Sum256([]byte(cr))
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, cfg.Region)
	stringToSign := strings.Join([]string{
		algorithm,
		amzDate,
		credentialScope,
		hex.EncodeToString(crHash[:]),
	}, "\n")

	signingKey := signingKeyV4(cfg.SecretAccessKey, dateStamp, cfg.Region, "s3")
	expected := hmacSHA256Hex(signingKey, stringToSign)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(sig)) != 1 {
		return fmt.Errorf("awsv4: signature mismatch")
	}
	return nil
}

func readBodyForHash(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	return data, nil
}

func containsFold(signedHeadersStr, h string) bool {
	for _, p := range strings.Split(signedHeadersStr, ";") {
		if strings.EqualFold(strings.TrimSpace(p), h) {
			return true
		}
	}
	return false
}

func headerValue(r *http.Request, lowerName string) string {
	if lowerName == "host" {
		// Go 服务端解析请求后把 Host 放在 r.Host，且不会放进 r.Header（见 net/http readRequest）。
		// SigV4 的 canonical headers 必须使用与客户端一致的 Host，否则恒为 signature mismatch。
		if h := strings.TrimSpace(r.Host); h != "" {
			return h
		}
	}
	for k, vs := range r.Header {
		if strings.ToLower(k) == lowerName && len(vs) > 0 {
			return vs[0]
		}
	}
	return ""
}

func parseAuthParams(s string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		idx := strings.IndexByte(part, '=')
		if idx <= 0 {
			continue
		}
		k := strings.TrimSpace(part[:idx])
		v := strings.TrimSpace(part[idx+1:])
		out[k] = v
	}
	return out
}

func canonicalQueryString(raw string) string {
	if raw == "" {
		return ""
	}
	// 须先按表单规则解码再按 SigV4 规则编码。若直接对 RawQuery 子串 awsEncode，
	// 例如 delimiter=%2F 会把「%」编成 %252F，与 AWS SDK / s5cmd 不一致（ls 等带 query 的请求会签名校验失败）。
	q, err := url.ParseQuery(raw)
	if err != nil {
		return ""
	}
	type pair struct{ k, v string }
	var pairs []pair
	for k, vs := range q {
		if len(vs) == 0 {
			pairs = append(pairs, pair{k, ""})
			continue
		}
		for _, v := range vs {
			pairs = append(pairs, pair{k, v})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].k != pairs[j].k {
			return pairs[i].k < pairs[j].k
		}
		return pairs[i].v < pairs[j].v
	})
	var b strings.Builder
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(awsEncode(p.k))
		b.WriteByte('=')
		b.WriteString(awsEncode(p.v))
	}
	return b.String()
}

func awsEncode(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			b.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return b.String()
}

func signingKeyV4(secret, date, region, service string) []byte {
	kDate := hmacSHA256Raw([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256Raw(kDate, []byte(region))
	kService := hmacSHA256Raw(kRegion, []byte(service))
	return hmacSHA256Raw(kService, []byte("aws4_request"))
}

func hmacSHA256Raw(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func hmacSHA256Hex(key []byte, data string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func checkAmzDateSkew(amzDate string, skew time.Duration) error {
	if len(amzDate) < 14 {
		return fmt.Errorf("awsv4: invalid x-amz-date")
	}
	t, err := time.Parse("20060102T150405Z", amzDate)
	if err != nil {
		return fmt.Errorf("awsv4: parse x-amz-date: %w", err)
	}
	now := time.Now().UTC()
	if t.After(now.Add(skew)) || t.Before(now.Add(-skew)) {
		return fmt.Errorf("awsv4: x-amz-date out of allowed skew")
	}
	return nil
}

// ExtractSigV4AccessKey 从 Authorization 头或预签名 URL 解析 Access Key Id（Credential 首段）。
func ExtractSigV4AccessKey(r *http.Request) (string, error) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(auth, algorithm+" ") {
		parts := parseAuthParams(auth[len(algorithm)+1:])
		cred, ok := parts["Credential"]
		if !ok {
			return "", fmt.Errorf("awsv4: missing Credential")
		}
		credParts := strings.Split(cred, "/")
		if len(credParts) != 5 || credParts[4] != "aws4_request" {
			return "", fmt.Errorf("awsv4: invalid credential scope")
		}
		return credParts[0], nil
	}
	q := r.URL.Query()
	if q.Get("X-Amz-Signature") != "" && q.Get("X-Amz-Credential") != "" {
		cred := q.Get("X-Amz-Credential")
		credParts := strings.Split(cred, "/")
		if len(credParts) != 5 || credParts[4] != "aws4_request" {
			return "", fmt.Errorf("awsv4: invalid X-Amz-Credential")
		}
		return credParts[0], nil
	}
	return "", fmt.Errorf("awsv4: no sigv4 authentication")
}

// HasSigV4Auth 判断请求是否带有 Authorization SigV4 或预签名查询参数（用于根 path-style 与其它路由分流）。
func HasSigV4Auth(r *http.Request) bool {
	if strings.HasPrefix(strings.TrimSpace(r.Header.Get("Authorization")), algorithm+" ") {
		return true
	}
	q := r.URL.Query()
	return q.Get("X-Amz-Signature") != "" && q.Get("X-Amz-Credential") != ""
}
