package service

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common"
)

var bucketNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)

func s3UserDir(userId int) string {
	return filepath.Join(common.S3StorageDir, fmt.Sprintf("u%d", userId))
}

// ErrS3NoSuchBucket 删除或依赖桶存在性时使用。
var ErrS3NoSuchBucket = errors.New("no such bucket")

// IsValidS3BucketName 报告桶名是否符合 S3 DNS 命名子集规则。
func IsValidS3BucketName(bucket string) bool {
	return bucketNameRe.MatchString(strings.ToLower(strings.TrimSpace(bucket)))
}

// BucketExistsS3 报告存储目录下是否存在该桶目录。
func BucketExistsS3(userId int, bucket string) bool {
	if common.S3RemoteEnabled {
		return bucketExistsS3Remote(userId, bucket)
	}
	if userId <= 0 || !IsValidS3BucketName(bucket) {
		return false
	}
	b := strings.ToLower(strings.TrimSpace(bucket))
	root := filepath.Join(s3UserDir(userId), b)
	fi, err := os.Stat(root)
	return err == nil && fi.IsDir()
}

// 与 one-api 既有 HTTP 路径首段冲突的桶名；仅在 S3_ADDRESSING=path 时拒绝。
var reservedS3Buckets = map[string]struct{}{
	"api": {}, "v1": {}, "v1beta": {}, "openai": {}, "anthropic": {}, "gemini": {},
	"dashboard": {}, "pg": {}, "mj": {}, "s3": {}, "web": {},
}

// IsReservedS3Bucket 报告桶名是否与站点根路径冲突（path-style 下不可用）。
func IsReservedS3Bucket(bucket string) bool {
	_, ok := reservedS3Buckets[strings.ToLower(strings.TrimSpace(bucket))]
	return ok
}

// ListObjectEntry 对应 ListObjects 中单个对象。
type ListObjectEntry struct {
	Key          string
	LastModified time.Time
	ETag         string
	Size         int64
}

func objectSHA256ETag(objectPath string) (string, error) {
	h := sha256.New()
	f, err := os.Open(objectPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "\"" + hex.EncodeToString(h.Sum(nil)) + "\"", nil
}

type tempObjectMeta struct {
	ExpiresAtUnix int64  `json:"expires_at_unix"`
	ContentType   string `json:"content_type,omitempty"`
	Size          int64  `json:"size"`
	ETag          string `json:"etag,omitempty"` // 带引号的十六进制 SHA256，与 S3 常见形态一致
}

var s3CleanerOnce sync.Once

// StartS3Cleaner 周期性删除过期临时对象。
func StartS3Cleaner() {
	if !common.S3Enabled {
		return
	}
	s3CleanerOnce.Do(func() {
		go runS3Cleaner()
	})
}

func runS3Cleaner() {
	sweep := func() { _ = sweepExpiredS3() }
	sweep()
	t := time.NewTicker(common.S3CleanerInterval)
	defer t.Stop()
	for range t.C {
		sweep()
	}
}

func sweepExpiredS3() error {
	if common.S3RemoteEnabled {
		return sweepExpiredS3Remote()
	}
	root := common.S3StorageDir
	now := time.Now().Unix()
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Base(path) != ".meta.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var m tempObjectMeta
		if json.Unmarshal(data, &m) != nil || m.ExpiresAtUnix <= 0 {
			return nil
		}
		if m.ExpiresAtUnix > now {
			return nil
		}
		objPath := strings.TrimSuffix(path, ".meta.json")
		_ = os.Remove(objPath)
		_ = os.Remove(path)
		return nil
	})
}

func sanitizeObjectKey(key string) (string, error) {
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "", errors.New("invalid object key")
	}
	for _, seg := range strings.Split(filepath.ToSlash(key), "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", errors.New("invalid object key")
		}
	}
	return key, nil
}

func objectPaths(userId int, bucket, key string) (dir, objectPath, metaPath string, err error) {
	if userId <= 0 {
		return "", "", "", errors.New("invalid user")
	}
	bucket = strings.ToLower(bucket)
	if !bucketNameRe.MatchString(bucket) {
		return "", "", "", errors.New("invalid bucket name")
	}
	k, err := sanitizeObjectKey(key)
	if err != nil {
		return "", "", "", err
	}
	dir = filepath.Join(s3UserDir(userId), bucket)
	objectPath = filepath.Join(dir, filepath.FromSlash(k))
	metaPath = objectPath + ".meta.json"
	rel, err := filepath.Rel(dir, objectPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", "", "", errors.New("invalid path")
	}
	return dir, objectPath, metaPath, nil
}

type countingHashWriter struct {
	h   hash.Hash
	out io.Writer
}

func (w *countingHashWriter) Write(p []byte) (int, error) {
	_, _ = w.h.Write(p)
	return w.out.Write(p)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

// PutS3Object 写入对象及元数据，返回 ETag（带引号的十六进制 SHA256）与大小。
// contentMD5Base64 非空时须为对象 MD5 的 Base64（与 S3 Content-MD5 头一致），否则返回错误。
func PutS3Object(userId int, bucket, key, contentType string, ttl time.Duration, body io.Reader, contentMD5Base64 string) (etag string, size int64, err error) {
	if common.S3RemoteEnabled {
		return putS3ObjectRemote(userId, bucket, key, contentType, ttl, body, contentMD5Base64)
	}
	if ttl <= 0 {
		ttl = common.S3DefaultTTL
	}
	if ttl > common.S3MaxTTL {
		ttl = common.S3MaxTTL
	}
	_, objectPath, metaPath, err := objectPaths(userId, bucket, key)
	if err != nil {
		return "", 0, err
	}
	if err = os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		return "", 0, err
	}
	var md5Expect []byte
	if strings.TrimSpace(contentMD5Base64) != "" {
		md5Expect, err = base64.StdEncoding.DecodeString(strings.TrimSpace(contentMD5Base64))
		if err != nil || len(md5Expect) != md5.Size {
			return "", 0, errors.New("invalid Content-MD5")
		}
	}
	tmp := objectPath + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return "", 0, err
	}
	h := sha256.New()
	wh := &countingHashWriter{h: h, out: f}
	var src io.Reader = io.LimitReader(body, common.S3MaxObjectBytes+1)
	var md5h hash.Hash
	if md5Expect != nil {
		md5h = md5.New()
		src = io.TeeReader(src, md5h)
	}
	n, err := io.Copy(wh, src)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", 0, err
	}
	if n > common.S3MaxObjectBytes {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", 0, fmt.Errorf("object too large")
	}
	if md5Expect != nil {
		if !bytesEqual(md5h.Sum(nil), md5Expect) {
			_ = f.Close()
			_ = os.Remove(tmp)
			return "", 0, errors.New("bad digest: Content-MD5 does not match")
		}
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", 0, err
	}
	if err = os.Rename(tmp, objectPath); err != nil {
		_ = os.Remove(tmp)
		return "", 0, err
	}
	exp := time.Now().Add(ttl).Unix()
	sum := h.Sum(nil)
	etag = "\"" + hex.EncodeToString(sum) + "\""
	meta := tempObjectMeta{
		ExpiresAtUnix: exp,
		ContentType:   contentType,
		Size:          n,
		ETag:          etag,
	}
	mb, _ := json.Marshal(meta)
	if err = os.WriteFile(metaPath, mb, 0o644); err != nil {
		_ = os.Remove(objectPath)
		return "", 0, err
	}
	return etag, n, nil
}

// GetS3Object 打开对象只读流；若过期则删除。etag 来自元数据（新写入对象均有）。
func GetS3Object(userId int, bucket, key string) (rc io.ReadCloser, contentType string, size int64, etag string, err error) {
	if common.S3RemoteEnabled {
		return getS3ObjectRemote(userId, bucket, key)
	}
	_, objectPath, metaPath, err := objectPaths(userId, bucket, key)
	if err != nil {
		return nil, "", 0, "", err
	}
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, "", 0, "", err
	}
	var m tempObjectMeta
	if json.Unmarshal(metaBytes, &m) != nil {
		return nil, "", 0, "", errors.New("corrupt meta")
	}
	if m.ExpiresAtUnix <= time.Now().Unix() {
		_ = os.Remove(objectPath)
		_ = os.Remove(metaPath)
		return nil, "", 0, "", os.ErrNotExist
	}
	f, err := os.Open(objectPath)
	if err != nil {
		return nil, "", 0, "", err
	}
	return f, m.ContentType, m.Size, m.ETag, nil
}

// HeadS3Object 返回元信息；过期则删除。
func HeadS3Object(userId int, bucket, key string) (contentType string, size int64, etag string, err error) {
	if common.S3RemoteEnabled {
		return headS3ObjectRemote(userId, bucket, key)
	}
	_, objectPath, metaPath, err := objectPaths(userId, bucket, key)
	if err != nil {
		return "", 0, "", err
	}
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return "", 0, "", err
	}
	var m tempObjectMeta
	if json.Unmarshal(metaBytes, &m) != nil {
		return "", 0, "", errors.New("corrupt meta")
	}
	if m.ExpiresAtUnix <= time.Now().Unix() {
		_ = os.Remove(objectPath)
		_ = os.Remove(metaPath)
		return "", 0, "", os.ErrNotExist
	}
	if m.ETag != "" {
		return m.ContentType, m.Size, m.ETag, nil
	}
	etag, err = objectSHA256ETag(objectPath)
	if err != nil {
		return "", 0, "", err
	}
	return m.ContentType, m.Size, etag, nil
}

// DeleteS3Object 删除对象与元数据。
func DeleteS3Object(userId int, bucket, key string) error {
	if common.S3RemoteEnabled {
		return deleteS3ObjectRemote(userId, bucket, key)
	}
	_, objectPath, metaPath, err := objectPaths(userId, bucket, key)
	if err != nil {
		return err
	}
	_ = os.Remove(objectPath)
	_ = os.Remove(metaPath)
	return nil
}

// ParseTTLFromMetaHeaders 从 x-amz-meta-* 解析 TTL（秒）。
func ParseTTLFromMetaHeaders(h http.Header) time.Duration {
	for _, k := range []string{"X-Amz-Meta-One-Api-Ttl-Seconds", "X-Amz-Meta-Ttl-Seconds"} {
		if v := h.Get(k); v != "" {
			if sec, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && sec > 0 {
				return time.Duration(sec) * time.Second
			}
		}
	}
	return 0
}

// EnsureS3Bucket 创建桶目录（等价于 CreateBucket）。
func EnsureS3Bucket(userId int, bucket string) error {
	if common.S3RemoteEnabled {
		return ensureS3BucketRemote(userId, bucket)
	}
	if userId <= 0 {
		return errors.New("invalid user")
	}
	b := strings.ToLower(strings.TrimSpace(bucket))
	if !bucketNameRe.MatchString(b) {
		return errors.New("invalid bucket name")
	}
	root := filepath.Join(s3UserDir(userId), b)
	return os.MkdirAll(root, 0o755)
}

// S3BucketIsEmpty 报告桶目录下是否无任何对象（无非 .meta.json 的对象文件）。
func S3BucketIsEmpty(userId int, bucket string) (bool, error) {
	if common.S3RemoteEnabled {
		return s3BucketIsEmptyRemote(userId, bucket)
	}
	if userId <= 0 {
		return false, errors.New("invalid user")
	}
	b := strings.ToLower(strings.TrimSpace(bucket))
	root := filepath.Join(s3UserDir(userId), b)
	fi, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	if !fi.IsDir() {
		return false, errors.New("not a bucket directory")
	}
	var hasObject bool
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".meta.json") {
			return nil
		}
		hasObject = true
		return filepath.SkipAll
	})
	if err != nil {
		return false, err
	}
	return !hasObject, nil
}

// RemoveS3Bucket 删除空桶目录。
func RemoveS3Bucket(userId int, bucket string) error {
	if common.S3RemoteEnabled {
		return removeS3BucketRemote(userId, bucket)
	}
	if userId <= 0 {
		return errors.New("invalid user")
	}
	b := strings.ToLower(strings.TrimSpace(bucket))
	if !bucketNameRe.MatchString(b) {
		return errors.New("invalid bucket name")
	}
	root := filepath.Join(s3UserDir(userId), b)
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return ErrS3NoSuchBucket
		}
		return err
	}
	empty, err := S3BucketIsEmpty(userId, bucket)
	if err != nil {
		return err
	}
	if !empty {
		return errors.New("bucket not empty")
	}
	return os.Remove(root)
}

// StatS3Object 返回对象元数据与 Last-Modified（用于条件 GET/HEAD）。
func StatS3Object(userId int, bucket, key string) (contentType string, size int64, etag string, modTime time.Time, err error) {
	if common.S3RemoteEnabled {
		return statS3ObjectRemote(userId, bucket, key)
	}
	_, objectPath, metaPath, err := objectPaths(userId, bucket, key)
	if err != nil {
		return "", 0, "", time.Time{}, err
	}
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return "", 0, "", time.Time{}, err
	}
	var m tempObjectMeta
	if json.Unmarshal(metaBytes, &m) != nil {
		return "", 0, "", time.Time{}, errors.New("corrupt meta")
	}
	if m.ExpiresAtUnix <= time.Now().Unix() {
		_ = os.Remove(objectPath)
		_ = os.Remove(metaPath)
		return "", 0, "", time.Time{}, os.ErrNotExist
	}
	st, err := os.Stat(objectPath)
	if err != nil {
		return "", 0, "", time.Time{}, err
	}
	modTime = st.ModTime().UTC()
	etag = m.ETag
	if etag == "" {
		etag, err = objectSHA256ETag(objectPath)
		if err != nil {
			return "", 0, "", time.Time{}, err
		}
	}
	return m.ContentType, m.Size, etag, modTime, nil
}

// OpenS3ObjectRange 按字节范围打开对象；length<0 表示直到文件末尾。
func OpenS3ObjectRange(userId int, bucket, key string, start, length int64) (rc io.ReadCloser, contentType string, totalSize int64, etag string, err error) {
	if common.S3RemoteEnabled {
		return openS3ObjectRangeRemote(userId, bucket, key, start, length)
	}
	_, objectPath, metaPath, err := objectPaths(userId, bucket, key)
	if err != nil {
		return nil, "", 0, "", err
	}
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, "", 0, "", err
	}
	var m tempObjectMeta
	if json.Unmarshal(metaBytes, &m) != nil {
		return nil, "", 0, "", errors.New("corrupt meta")
	}
	if m.ExpiresAtUnix <= time.Now().Unix() {
		_ = os.Remove(objectPath)
		_ = os.Remove(metaPath)
		return nil, "", 0, "", os.ErrNotExist
	}
	if m.ETag == "" {
		m.ETag, err = objectSHA256ETag(objectPath)
		if err != nil {
			return nil, "", 0, "", err
		}
	}
	f, err := os.Open(objectPath)
	if err != nil {
		return nil, "", 0, "", err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, "", 0, "", err
	}
	total := st.Size()
	if start < 0 || start > total {
		_ = f.Close()
		return nil, "", total, m.ETag, fmt.Errorf("invalid range")
	}
	if _, err = f.Seek(start, io.SeekStart); err != nil {
		_ = f.Close()
		return nil, "", total, m.ETag, err
	}
	var lim int64
	if length < 0 {
		lim = total - start
	} else {
		if start+length > total {
			_ = f.Close()
			return nil, "", total, m.ETag, fmt.Errorf("invalid range")
		}
		lim = length
	}
	ct := m.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	return &s3LimitReadCloser{f: f, lr: io.LimitReader(f, lim)}, ct, total, m.ETag, nil
}

type s3LimitReadCloser struct {
	f  *os.File
	lr io.Reader
}

func (s *s3LimitReadCloser) Read(p []byte) (int, error) { return s.lr.Read(p) }
func (s *s3LimitReadCloser) Close() error               { return s.f.Close() }

// CopyS3Object 服务端复制对象（x-amz-copy-source）。
func CopyS3Object(userId int, dstBucket, dstKey, srcBucket, srcKey, contentType string, ttl time.Duration) (etag string, size int64, err error) {
	if common.S3RemoteEnabled {
		return copyS3ObjectRemote(userId, dstBucket, dstKey, srcBucket, srcKey, contentType, ttl)
	}
	rc, ct, sz, _, err := GetS3Object(userId, srcBucket, srcKey)
	if err != nil {
		return "", 0, err
	}
	defer rc.Close()
	if contentType == "" {
		contentType = ct
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return PutS3Object(userId, dstBucket, dstKey, contentType, ttl, io.LimitReader(rc, sz), "")
}
