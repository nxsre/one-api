package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
)

// setS3CommonResponseHeaders 设置与 AWS S3 常见响应接近的通用头。
func setS3CommonResponseHeaders(c *gin.Context, rid string) {
	c.Header("x-amz-request-id", rid)
	sum := sha256.Sum256([]byte(rid + "|" + c.Request.URL.Path))
	c.Header("x-amz-id-2", hex.EncodeToString(sum[:]))
	c.Header("Date", time.Now().UTC().Format(http.TimeFormat))
	c.Header("Server", "AmazonS3")
	c.Header("x-amz-bucket-region", common.S3Region)
}

// writeS3APIError 输出与 S3 更接近的 Error XML（含可选 BucketName / Key / Resource / HostId）。
func writeS3APIError(c *gin.Context, status int, code, message, rid, bucket, key, resource string) {
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	hostID := rid + "oneapi"
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString("<Error>\n")
	fmt.Fprintf(&b, "  <Code>%s</Code>\n", escapeXML(code))
	fmt.Fprintf(&b, "  <Message>%s</Message>\n", escapeXML(message))
	if resource != "" {
		fmt.Fprintf(&b, "  <Resource>%s</Resource>\n", escapeXML(resource))
	}
	if bucket != "" {
		fmt.Fprintf(&b, "  <BucketName>%s</BucketName>\n", escapeXML(bucket))
	}
	if key != "" {
		fmt.Fprintf(&b, "  <Key>%s</Key>\n", escapeXML(key))
	}
	fmt.Fprintf(&b, "  <RequestId>%s</RequestId>\n", escapeXML(rid))
	fmt.Fprintf(&b, "  <HostId>%s</HostId>\n", escapeXML(hostID))
	b.WriteString("</Error>\n")
	c.Data(status, "application/xml; charset=utf-8", []byte(b.String()))
}
