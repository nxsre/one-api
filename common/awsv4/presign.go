package awsv4

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// VerifyPresignedRequest 校验通过查询参数传递的 SigV4 预签名（常见于 GET 下载链接）。
// 要求存在 X-Amz-Algorithm、X-Amz-Credential、X-Amz-Date、X-Amz-Expires、X-Amz-SignedHeaders、X-Amz-Signature。
func VerifyPresignedRequest(r *http.Request, cfg Config, clockSkew time.Duration) error {
	if cfg.SecretAccessKey == "" || cfg.AccessKeyID == "" || cfg.Region == "" {
		return fmt.Errorf("awsv4: invalid server config")
	}
	q := r.URL.Query()
	if q.Get("X-Amz-Algorithm") != algorithm {
		return fmt.Errorf("awsv4: presigned: invalid X-Amz-Algorithm")
	}
	cred := q.Get("X-Amz-Credential")
	sig := q.Get("X-Amz-Signature")
	signedHeadersStr := q.Get("X-Amz-SignedHeaders")
	amzDate := q.Get("X-Amz-Date")
	expiresStr := q.Get("X-Amz-Expires")
	if cred == "" || sig == "" || signedHeadersStr == "" || amzDate == "" || expiresStr == "" {
		return fmt.Errorf("awsv4: presigned: missing query parameter")
	}
	credParts := strings.Split(cred, "/")
	if len(credParts) != 5 || credParts[4] != "aws4_request" {
		return fmt.Errorf("awsv4: presigned: invalid X-Amz-Credential")
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
	if len(amzDate) < 8 || amzDate[:8] != dateStamp {
		return fmt.Errorf("awsv4: x-amz-date does not match credential date")
	}
	expiresSec, err := strconv.Atoi(expiresStr)
	if err != nil || expiresSec <= 0 || expiresSec > 604800 {
		return fmt.Errorf("awsv4: presigned: invalid X-Amz-Expires")
	}
	if err := checkPresignedExpiry(amzDate, expiresSec, clockSkew, time.Now().UTC()); err != nil {
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
	if ph := strings.TrimSpace(q.Get("X-Amz-Content-Sha256")); ph != "" {
		payloadHash = ph
	}

	canonicalURI := r.URL.EscapedPath()
	canonicalQuery := canonicalQueryStringForPresign(q)

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

func canonicalQueryStringForPresign(q url.Values) string {
	type pair struct{ rawK, rawV string }
	var pairs []pair
	for k, vals := range q {
		if strings.EqualFold(k, "X-Amz-Signature") {
			continue
		}
		if len(vals) == 0 {
			pairs = append(pairs, pair{k, ""})
			continue
		}
		for _, v := range vals {
			pairs = append(pairs, pair{k, v})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].rawK != pairs[j].rawK {
			return pairs[i].rawK < pairs[j].rawK
		}
		return pairs[i].rawV < pairs[j].rawV
	})
	var b strings.Builder
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(awsEncode(p.rawK))
		b.WriteByte('=')
		b.WriteString(awsEncode(p.rawV))
	}
	return b.String()
}

// checkPresignedExpiry 校验预签名 URL 是否在有效期内。
// 起始侧仍允许 clockSkew（签名的 X-Amz-Date 略晚于服务端时钟）；
// 截止时刻为 X-Amz-Date + X-Amz-Expires，不再叠加 clockSkew，否则 s3_clock_skew_seconds 设很大时
//（如 1800）会把短时效链接错误延长数倍。
func checkPresignedExpiry(amzDate string, expiresSec int, skew time.Duration, now time.Time) error {
	t, err := time.Parse("20060102T150405Z", amzDate)
	if err != nil {
		return fmt.Errorf("awsv4: parse x-amz-date: %w", err)
	}
	if now.Before(t.Add(-skew)) {
		return fmt.Errorf("awsv4: request is not yet valid")
	}
	expiresAt := t.Add(time.Duration(expiresSec) * time.Second)
	if now.After(expiresAt) {
		return fmt.Errorf("awsv4: presigned request has expired")
	}
	return nil
}
