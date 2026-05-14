package controller

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/awsv4"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/service"
)

const s3XMLNamespace = "http://s3.amazonaws.com/doc/2006-03-01/"

// --- 路由入口（见 router.SetS3CompatRouter / SetS3PathStyleRootRouter）---

func S3OptionsPrefixed(c *gin.Context) { s3Options(c) }

func S3ObjectPrefixed(c *gin.Context) {
	if !common.S3SiteOpen() {
		c.Status(http.StatusNotFound)
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.Param("bucket")))
	key := strings.TrimPrefix(strings.TrimSpace(c.Param("key")), "/")
	s3ObjectOps(c, bucket, key)
}

func S3PathStyleOPTIONS(c *gin.Context) { s3Options(c) }

func S3PathStyleObject(c *gin.Context) {
	if !common.S3SiteOpen() || !common.S3PathStyleAtRoot {
		c.Status(http.StatusNotFound)
		return
	}
	// 与 Gin 路由一致：不可对 /:bucket/*objKey 再单独注册 OPTIONS（会与 Any 冲突），预检需在签名校验前处理。
	if c.Request.Method == http.MethodOptions {
		s3Options(c)
		return
	}
	if !awsv4.HasSigV4Auth(c.Request) {
		RelayNotFound(c)
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.Param("bucket")))
	if service.IsReservedS3Bucket(bucket) {
		rid := s3RequestID(c)
		c.Header("x-amz-request-id", rid)
		applyS3CORS(c)
		writeS3Error(c, http.StatusBadRequest, "InvalidBucketName", "this bucket name is reserved when using path-style addressing at site root; use another bucket name or S3_ADDRESSING=prefixed", rid)
		return
	}
	key := strings.TrimPrefix(strings.TrimSpace(c.Param("objKey")), "/")
	s3ObjectOps(c, bucket, key)
}

// --- 共享逻辑 ---

func s3Options(c *gin.Context) {
	if !common.S3SiteOpen() {
		c.Status(http.StatusNotFound)
		return
	}
	applyS3CORS(c)
	c.Header("Allow", "GET, HEAD, PUT, DELETE, OPTIONS")
	c.Status(http.StatusNoContent)
}

func s3RequestID(c *gin.Context) string {
	rid := c.Writer.Header().Get(helper.RequestIdKey)
	if rid == "" {
		rid = helper.GenRequestID()
	}
	return rid
}

func s3Verify(c *gin.Context) error {
	ak, err := awsv4.ExtractSigV4AccessKey(c.Request)
	if err != nil {
		return err
	}
	user, err := model.GetUserByS3AccessKey(ak)
	if err != nil || user == nil {
		return fmt.Errorf("access denied")
	}
	if user.S3AccessKey == nil || user.S3SecretKey == nil || *user.S3SecretKey == "" {
		return fmt.Errorf("access denied")
	}
	cfg := awsv4.Config{
		Region:          common.S3Region,
		AccessKeyID:     *user.S3AccessKey,
		SecretAccessKey: *user.S3SecretKey,
	}
	q := c.Request.URL.Query()
	if q.Get("X-Amz-Signature") != "" && q.Get("X-Amz-Credential") != "" {
		err = awsv4.VerifyPresignedRequest(c.Request, cfg, common.S3ClockSkew)
	} else {
		err = awsv4.VerifyRequest(c.Request, cfg, common.S3ClockSkew)
	}
	if err != nil {
		return err
	}
	c.Set("s3_user_id", user.Id)
	return nil
}

func s3AuthUID(c *gin.Context) int {
	v, ok := c.Get("s3_user_id")
	if !ok {
		return 0
	}
	id, ok := v.(int)
	if !ok {
		return 0
	}
	return id
}

func s3ListBucket(c *gin.Context, bucket string) {
	rid := s3RequestID(c)

	if c.Request.Method != http.MethodGet {
		writeS3APIError(c, http.StatusMethodNotAllowed, "MethodNotAllowed", "only GET is supported for bucket listing", rid, bucket, "", c.Request.URL.Path)
		return
	}

	listType := c.Query("list-type")
	if listType == "" {
		if !awsv4.HasSigV4Auth(c.Request) {
			RelayNotFound(c)
			return
		}
		writeS3APIError(c, http.StatusBadRequest, "InvalidArgument", "list-type query parameter must be 1 or 2", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if listType != "2" && listType != "1" {
		writeS3APIError(c, http.StatusBadRequest, "InvalidArgument", "list-type query parameter must be 1 or 2", rid, bucket, "", c.Request.URL.Path)
		return
	}

	if err := s3Verify(c); err != nil {
		writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
		return
	}

	if service.IsValidS3BucketName(bucket) && !service.BucketExistsS3(s3AuthUID(c), bucket) {
		writeS3APIError(c, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", rid, bucket, "", c.Request.URL.Path)
		return
	}

	prefix := c.Query("prefix")
	delimiter := c.Query("delimiter")
	maxKeys := 1000
	if mk := c.Query("max-keys"); mk != "" {
		if n, err := strconv.Atoi(mk); err == nil && n > 0 {
			maxKeys = n
		}
	}
	fetchOwner := strings.EqualFold(strings.TrimSpace(c.Query("fetch-owner")), "true")
	encType := strings.TrimSpace(c.Query("encoding-type"))
	if encType != "" && encType != "url" {
		writeS3APIError(c, http.StatusBadRequest, "InvalidArgument", "encoding-type must be url", rid, bucket, "", c.Request.URL.Path)
		return
	}

	res, err := service.ListS3BucketObjects(service.ListObjectsParams{
		UserID:              s3AuthUID(c),
		Bucket:              bucket,
		Prefix:              prefix,
		MaxKeys:             maxKeys,
		Delimiter:           delimiter,
		ListType:            listType,
		ContinuationToken:   c.Query("continuation-token"),
		StartAfter:          c.Query("start-after"),
		Marker:              c.Query("marker"),
		FetchOwner:          fetchOwner,
		EncodingType:        encType,
	})
	if err != nil {
		if strings.HasPrefix(err.Error(), "invalid argument:") {
			msg := strings.TrimSpace(strings.TrimPrefix(err.Error(), "invalid argument:"))
			writeS3APIError(c, http.StatusBadRequest, "InvalidArgument", msg, rid, bucket, "", c.Request.URL.Path)
			return
		}
		if strings.Contains(err.Error(), "invalid bucket") {
			writeS3APIError(c, http.StatusBadRequest, "InvalidBucketName", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		writeS3APIError(c, http.StatusInternalServerError, "InternalError", err.Error(), rid, bucket, "", c.Request.URL.Path)
		return
	}

	xmlBytes, err := marshalListBucketXML(listType, bucket, prefix, delimiter, maxKeys, encType, fetchOwner, res)
	if err != nil {
		writeS3APIError(c, http.StatusInternalServerError, "InternalError", err.Error(), rid, bucket, "", c.Request.URL.Path)
		return
	}
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", append([]byte(xml.Header), xmlBytes...))
}

func parseAmzCopySource(raw string) (bucket, key string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}
	u, err := url.QueryUnescape(raw)
	if err != nil {
		u = raw
	}
	u = strings.TrimPrefix(u, "/")
	if i := strings.IndexByte(u, '?'); i >= 0 {
		u = u[:i]
	}
	slash := strings.IndexByte(u, '/')
	if slash <= 0 || slash >= len(u)-1 {
		return "", "", false
	}
	return strings.ToLower(u[:slash]), u[slash+1:], true
}

func etagMatchesHeader(etag, header string) bool {
	need := strings.Trim(strings.TrimSpace(etag), `"`)
	for _, p := range strings.Split(header, ",") {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(t), "w/") {
			t = strings.TrimSpace(t[2:])
		}
		t = strings.Trim(t, `"`)
		if t == need {
			return true
		}
	}
	return false
}

func s3Precondition412(c *gin.Context, etag string, mod time.Time) bool {
	if im := c.GetHeader("If-Match"); im != "" && im != "*" && !etagMatchesHeader(etag, im) {
		return true
	}
	if ius := c.GetHeader("If-Unmodified-Since"); ius != "" {
		if t, err := http.ParseTime(ius); err == nil {
			if mod.UTC().Truncate(time.Second).After(t.UTC().Truncate(time.Second)) {
				return true
			}
		}
	}
	return false
}

func s3NotModified304(c *gin.Context, meth, etag string, mod time.Time) bool {
	if meth != http.MethodGet && meth != http.MethodHead {
		return false
	}
	if inm := c.GetHeader("If-None-Match"); inm != "" {
		if strings.TrimSpace(inm) == "*" {
			return true
		}
		if etagMatchesHeader(etag, inm) {
			return true
		}
	}
	if ims := c.GetHeader("If-Modified-Since"); ims != "" {
		if t, err := http.ParseTime(ims); err == nil {
			if !mod.UTC().Truncate(time.Second).After(t.UTC().Truncate(time.Second)) {
				return true
			}
		}
	}
	return false
}

// parseS3SingleRange 解析单个 bytes=… Range；返回半开区间 [start,end)；ok 表示应返回 206。
func parseS3SingleRange(rangeHdr string, size int64) (start, end int64, ok bool, err416 bool) {
	if rangeHdr == "" || size <= 0 {
		return 0, size, false, false
	}
	rh := strings.TrimSpace(rangeHdr)
	const pfx = "bytes="
	if !strings.HasPrefix(strings.ToLower(rh), pfx) {
		return 0, 0, false, true
	}
	spec := strings.TrimSpace(strings.TrimPrefix(rh, pfx))
	if spec == "" || strings.Contains(spec, ",") {
		return 0, 0, false, true
	}
	d := strings.IndexByte(spec, '-')
	if d < 0 {
		return 0, 0, false, true
	}
	left, right := spec[:d], spec[d+1:]
	if left == "" {
		suff, err := strconv.ParseInt(strings.TrimSpace(right), 10, 64)
		if err != nil || suff <= 0 {
			return 0, 0, false, true
		}
		if suff >= size {
			return 0, size, true, false
		}
		return size - suff, size, true, false
	}
	st, err := strconv.ParseInt(strings.TrimSpace(left), 10, 64)
	if err != nil || st < 0 || st >= size {
		return 0, 0, false, true
	}
	if right == "" {
		return st, size, true, false
	}
	endIncl, err := strconv.ParseInt(strings.TrimSpace(right), 10, 64)
	if err != nil || endIncl < st || endIncl >= size {
		return 0, 0, false, true
	}
	return st, endIncl + 1, true, false
}

func s3EncodeObjectKeyForList(key string) string {
	segs := strings.Split(filepath.ToSlash(key), "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "%2F")
}

func s3ObjectOps(c *gin.Context, bucket, key string) {
	rid := s3RequestID(c)

	if c.Request.Method == http.MethodOptions {
		applyS3CORS(c)
		setS3CommonResponseHeaders(c, rid)
		c.Header("Allow", "GET, HEAD, PUT, DELETE, OPTIONS")
		c.Status(http.StatusNoContent)
		return
	}

	if err := s3Verify(c); err != nil {
		writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, key, c.Request.URL.Path)
		return
	}
	uid := s3AuthUID(c)
	if uid <= 0 {
		writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", "access denied", rid, bucket, key, c.Request.URL.Path)
		return
	}

	switch c.Request.Method {
	case http.MethodPut:
		ttl := service.ParseTTLFromMetaHeaders(c.Request.Header)
		ct := c.GetHeader("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		copySrc := strings.TrimSpace(c.GetHeader("x-amz-copy-source"))
		if copySrc != "" {
			sb, sk, ok := parseAmzCopySource(copySrc)
			if !ok {
				writeS3APIError(c, http.StatusBadRequest, "InvalidArgument", "Invalid copy source", rid, bucket, key, c.Request.URL.Path)
				return
			}
			etag, _, err := service.CopyS3Object(uid, bucket, key, sb, sk, "", ttl)
			if c.Request.Body != nil {
				_ = c.Request.Body.Close()
			}
			if err != nil {
				if os.IsNotExist(err) {
					writeS3APIError(c, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", rid, sb, sk, c.Request.URL.Path)
					return
				}
				writeS3APIError(c, http.StatusBadRequest, "InvalidRequest", err.Error(), rid, bucket, key, c.Request.URL.Path)
				return
			}
			writeCopyObjectResultXML(c, rid, etag, time.Now().UTC())
			return
		}

		md5h := strings.TrimSpace(c.GetHeader("Content-Md5"))
		etag, _, err := service.PutS3Object(uid, bucket, key, ct, ttl, c.Request.Body, md5h)
		if c.Request.Body != nil {
			_ = c.Request.Body.Close()
		}
		if err != nil {
			if strings.Contains(err.Error(), "too large") {
				writeS3APIError(c, http.StatusBadRequest, "EntityTooLarge", "Your proposed upload exceeds the maximum allowed object size.", rid, bucket, key, c.Request.URL.Path)
				return
			}
			if strings.Contains(err.Error(), "invalid Content-MD5") {
				writeS3APIError(c, http.StatusBadRequest, "InvalidDigest", "The Content-MD5 you specified was invalid.", rid, bucket, key, c.Request.URL.Path)
				return
			}
			if strings.Contains(err.Error(), "bad digest") {
				writeS3APIError(c, http.StatusBadRequest, "BadDigest", "The Content-MD5 you specified did not match what we received.", rid, bucket, key, c.Request.URL.Path)
				return
			}
			if strings.Contains(err.Error(), "invalid") {
				writeS3APIError(c, http.StatusBadRequest, "InvalidRequest", err.Error(), rid, bucket, key, c.Request.URL.Path)
				return
			}
			writeS3APIError(c, http.StatusInternalServerError, "InternalError", err.Error(), rid, bucket, key, c.Request.URL.Path)
			return
		}
		applyS3CORS(c)
		setS3CommonResponseHeaders(c, rid)
		c.Header("ETag", etag)
		c.Header("x-amz-version-id", "null")
		c.Header("x-amz-storage-class", "STANDARD")
		c.Status(http.StatusOK)

	case http.MethodGet:
		ct, size, etag, mod, err := service.StatS3Object(uid, bucket, key)
		if err != nil {
			if os.IsNotExist(err) {
				writeS3APIError(c, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", rid, bucket, key, c.Request.URL.Path)
				return
			}
			writeS3APIError(c, http.StatusBadRequest, "InvalidRequest", err.Error(), rid, bucket, key, c.Request.URL.Path)
			return
		}
		if s3Precondition412(c, etag, mod) {
			writeS3APIError(c, http.StatusPreconditionFailed, "PreconditionFailed", "At least one of the pre-conditions you specified did not hold", rid, bucket, key, c.Request.URL.Path)
			return
		}
		if s3NotModified304(c, http.MethodGet, etag, mod) {
			applyS3CORS(c)
			setS3CommonResponseHeaders(c, rid)
			c.Header("ETag", etag)
			c.Header("Last-Modified", mod.UTC().Format(http.TimeFormat))
			c.Status(http.StatusNotModified)
			return
		}
		if ct == "" {
			ct = "application/octet-stream"
		}
		rh := c.GetHeader("Range")
		start, end, hasRange, badRange := parseS3SingleRange(rh, size)
		if rh != "" && badRange {
			writeS3APIError(c, http.StatusRequestedRangeNotSatisfiable, "InvalidRange", "The requested range is not satisfiable", rid, bucket, key, c.Request.URL.Path)
			return
		}
		if hasRange {
			rlen := end - start
			rc, rct, _, retag, err := service.OpenS3ObjectRange(uid, bucket, key, start, rlen)
			if err != nil {
				if strings.Contains(err.Error(), "invalid range") {
					writeS3APIError(c, http.StatusRequestedRangeNotSatisfiable, "InvalidRange", "The requested range is not satisfiable", rid, bucket, key, c.Request.URL.Path)
					return
				}
				writeS3APIError(c, http.StatusInternalServerError, "InternalError", err.Error(), rid, bucket, key, c.Request.URL.Path)
				return
			}
			defer rc.Close()
			if rct != "" {
				ct = rct
			}
			etag = retag
			applyS3CORS(c)
			setS3CommonResponseHeaders(c, rid)
			c.Header("ETag", etag)
			c.Header("Last-Modified", mod.UTC().Format(http.TimeFormat))
			c.Header("Accept-Ranges", "bytes")
			c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end-1, size))
			c.Header("x-amz-version-id", "null")
			c.Header("x-amz-storage-class", "STANDARD")
			c.DataFromReader(http.StatusPartialContent, rlen, ct, rc, nil)
			return
		}
		rc, rct, _, retag, err := service.GetS3Object(uid, bucket, key)
		if err != nil {
			writeS3APIError(c, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", rid, bucket, key, c.Request.URL.Path)
			return
		}
		defer rc.Close()
		if rct != "" {
			ct = rct
		}
		if retag != "" {
			etag = retag
		}
		applyS3CORS(c)
		setS3CommonResponseHeaders(c, rid)
		c.Header("ETag", etag)
		c.Header("Last-Modified", mod.UTC().Format(http.TimeFormat))
		c.Header("Accept-Ranges", "bytes")
		c.Header("Content-Length", fmt.Sprintf("%d", size))
		c.Header("x-amz-version-id", "null")
		c.Header("x-amz-storage-class", "STANDARD")
		c.DataFromReader(http.StatusOK, size, ct, rc, nil)

	case http.MethodHead:
		ct, size, etag, mod, err := service.StatS3Object(uid, bucket, key)
		if err != nil {
			if os.IsNotExist(err) {
				c.Status(http.StatusNotFound)
				return
			}
			c.Status(http.StatusInternalServerError)
			return
		}
		if s3Precondition412(c, etag, mod) {
			writeS3APIError(c, http.StatusPreconditionFailed, "PreconditionFailed", "At least one of the pre-conditions you specified did not hold", rid, bucket, key, c.Request.URL.Path)
			return
		}
		if s3NotModified304(c, http.MethodHead, etag, mod) {
			applyS3CORS(c)
			setS3CommonResponseHeaders(c, rid)
			c.Header("ETag", etag)
			c.Header("Last-Modified", mod.UTC().Format(http.TimeFormat))
			c.Status(http.StatusNotModified)
			return
		}
		if ct != "" {
			c.Header("Content-Type", ct)
		}
		applyS3CORS(c)
		setS3CommonResponseHeaders(c, rid)
		c.Header("Content-Length", fmt.Sprintf("%d", size))
		c.Header("ETag", etag)
		c.Header("Last-Modified", mod.UTC().Format(http.TimeFormat))
		c.Header("Accept-Ranges", "bytes")
		c.Header("x-amz-version-id", "null")
		c.Header("x-amz-storage-class", "STANDARD")
		c.Status(http.StatusOK)

	case http.MethodDelete:
		_ = service.DeleteS3Object(uid, bucket, key)
		applyS3CORS(c)
		setS3CommonResponseHeaders(c, rid)
		c.Status(http.StatusNoContent)

	default:
		writeS3APIError(c, http.StatusNotImplemented, "NotImplemented", "A header you provided implies functionality that is not implemented.", rid, bucket, key, c.Request.URL.Path)
	}
}

type copyObjectResultXML struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	Xmlns        string   `xml:"xmlns,attr"`
	LastModified string   `xml:"LastModified"`
	ETag         string   `xml:"ETag"`
}

func writeCopyObjectResultXML(c *gin.Context, rid, etag string, mod time.Time) {
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	body := copyObjectResultXML{
		XMLName:      xml.Name{Local: "CopyObjectResult"},
		Xmlns:        s3XMLNamespace,
		LastModified: formatS3ISO8601Millis(mod),
		ETag:         etag,
	}
	b, err := xml.MarshalIndent(body, "", "  ")
	if err != nil {
		writeS3APIError(c, http.StatusInternalServerError, "InternalError", err.Error(), rid, "", "", c.Request.URL.Path)
		return
	}
	c.Data(http.StatusOK, "application/xml; charset=utf-8", append([]byte(xml.Header), b...))
}

func applyS3CORS(c *gin.Context) {
	o := common.S3CORSAllowOrigin
	if o == "" {
		return
	}
	c.Header("Access-Control-Allow-Origin", o)
	c.Header("Access-Control-Allow-Methods", "GET, HEAD, PUT, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "authorization, content-type, x-amz-date, x-amz-content-sha256, x-amz-meta-one-api-ttl-seconds, x-amz-meta-ttl-seconds")
	c.Header("Access-Control-Max-Age", "86400")
}

func writeS3Error(c *gin.Context, status int, code, message, requestID string) {
	writeS3APIError(c, status, code, message, requestID, "", "", c.Request.URL.Path)
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func formatS3ISO8601Millis(t time.Time) string {
	t = t.UTC().Truncate(time.Millisecond)
	y, mon, d := t.Date()
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d.%03dZ",
		y, int(mon), d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1e6)
}

func marshalListBucketXML(listType, bucket, prefix, delimiter string, maxKeys int, encType string, fetchOwner bool, res service.ListObjectsResult) ([]byte, error) {
	urlEnc := encType == "url"
	contents := make([]listContentXML, 0, len(res.Contents))
	for _, it := range res.Contents {
		k := it.Key
		if urlEnc {
			k = s3EncodeObjectKeyForList(k)
		}
		item := listContentXML{
			Key:            k,
			LastModified: formatS3ISO8601Millis(it.LastModified),
			ETag:           it.ETag,
			Size:           it.Size,
			StorageClass:   "STANDARD",
		}
		if fetchOwner {
			item.Owner = &listOwnerXML{ID: "temp-s3-owner", DisplayName: "temp-s3"}
		}
		contents = append(contents, item)
	}
	cps := make([]listCommonPrefixesXML, len(res.CommonPrefixes))
	for i, p := range res.CommonPrefixes {
		pp := p
		if urlEnc {
			pp = s3EncodeObjectKeyForList(pp)
		}
		cps[i] = listCommonPrefixesXML{
			XMLName: xml.Name{Local: "CommonPrefixes"},
			Prefix:  pp,
		}
	}
	if listType == "2" {
		body := listBucketResultV2{
			XMLName:               xml.Name{Local: "ListBucketResult"},
			Xmlns:                 s3XMLNamespace,
			Name:                  bucket,
			Prefix:                prefix,
			MaxKeys:               maxKeys,
			KeyCount:              res.KeyCount,
			IsTruncated:           res.IsTruncated,
			Delimiter:             delimiter,
			StartAfter:            res.EchoStartAfter,
			ContinuationToken:     res.EchoContinuationToken,
			NextContinuationToken: res.NextContinuationToken,
			CommonPrefixes:        cps,
			Contents:              contents,
		}
		if encType != "" {
			body.EncodingType = encType
		}
		return xml.MarshalIndent(body, "", "  ")
	}
	body := listBucketResultV1{
		XMLName:        xml.Name{Local: "ListBucketResult"},
		Xmlns:          s3XMLNamespace,
		Name:           bucket,
		Prefix:         prefix,
		Marker:         res.EchoMarker,
		MaxKeys:        maxKeys,
		Delimiter:      delimiter,
		IsTruncated:    res.IsTruncated,
		NextMarker:     res.NextMarker,
		CommonPrefixes: cps,
		Contents:       contents,
	}
	if encType != "" {
		body.EncodingType = encType
	}
	return xml.MarshalIndent(body, "", "  ")
}

type listBucketResultV2 struct {
	XMLName               xml.Name               `xml:"ListBucketResult"`
	Xmlns                 string                 `xml:"xmlns,attr"`
	Name                  string                 `xml:"Name"`
	Prefix                string                 `xml:"Prefix"`
	MaxKeys               int                    `xml:"MaxKeys"`
	EncodingType          string                 `xml:"EncodingType,omitempty"`
	KeyCount              int                    `xml:"KeyCount"`
	IsTruncated           bool                   `xml:"IsTruncated"`
	Delimiter             string                 `xml:"Delimiter,omitempty"`
	StartAfter            string                 `xml:"StartAfter,omitempty"`
	ContinuationToken     string                 `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string                 `xml:"NextContinuationToken,omitempty"`
	CommonPrefixes        []listCommonPrefixesXML `xml:"CommonPrefixes,omitempty"`
	Contents              []listContentXML       `xml:"Contents,omitempty"`
}

type listBucketResultV1 struct {
	XMLName        xml.Name                `xml:"ListBucketResult"`
	Xmlns          string                  `xml:"xmlns,attr"`
	Name           string                  `xml:"Name"`
	Prefix         string                  `xml:"Prefix"`
	Marker         string                  `xml:"Marker,omitempty"`
	MaxKeys        int                     `xml:"MaxKeys"`
	EncodingType   string                  `xml:"EncodingType,omitempty"`
	Delimiter      string                  `xml:"Delimiter,omitempty"`
	IsTruncated    bool                    `xml:"IsTruncated"`
	NextMarker     string                  `xml:"NextMarker,omitempty"`
	CommonPrefixes []listCommonPrefixesXML `xml:"CommonPrefixes,omitempty"`
	Contents       []listContentXML        `xml:"Contents,omitempty"`
}

type listCommonPrefixesXML struct {
	XMLName xml.Name `xml:"CommonPrefixes"`
	Prefix  string   `xml:"Prefix"`
}

type listOwnerXML struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName,omitempty"`
}

type listContentXML struct {
	Key          string        `xml:"Key"`
	LastModified string        `xml:"LastModified"`
	ETag         string        `xml:"ETag"`
	Size         int64         `xml:"Size"`
	StorageClass string        `xml:"StorageClass"`
	Owner        *listOwnerXML `xml:"Owner,omitempty"`
}
