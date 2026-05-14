package service

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/common"
)

// ListObjectsParams 对齐 AWS ListObjects / ListObjectsV2 请求参数子集。
type ListObjectsParams struct {
	UserID int // 登录用户 / AK 所属用户，存储根路径 namespace

	Bucket   string
	Prefix   string
	MaxKeys  int
	Delimiter string
	ListType string // "1" 或 "2"

	// ListObjectsV2
	ContinuationToken string
	StartAfter        string
	// ListObjects (v1)
	Marker string

	FetchOwner   bool   // 仅影响 XML 中 Owner 节点
	EncodingType string // 请求 echo，如 "url"
}

// ListObjectsResult 用于构造 S3 XML 响应。
type ListObjectsResult struct {
	Contents       []ListObjectEntry
	CommonPrefixes []string

	IsTruncated bool
	// V2
	NextContinuationToken string
	// V1
	NextMarker string

	// 回显（与 AWS 行为一致）
	EchoContinuationToken string
	EchoStartAfter        string
	EchoMarker            string
	KeyCount              int
}

type listMergeItem struct {
	isCommonPrefix bool
	key            string // 对象键或 CommonPrefixes 的完整前缀串
}

// ListS3BucketObjects 按 AWS S3 ListObjects(V1)/ListObjectsV2 语义列举（扁平 + 可选 delimiter）。
func ListS3BucketObjects(p ListObjectsParams) (ListObjectsResult, error) {
	var out ListObjectsResult
	out.EchoContinuationToken = p.ContinuationToken
	out.EchoStartAfter = p.StartAfter
	out.EchoMarker = p.Marker

	if p.MaxKeys <= 0 {
		p.MaxKeys = 1000
	}
	if p.MaxKeys > 1000 {
		p.MaxKeys = 1000
	}
	lt := strings.TrimSpace(p.ListType)
	if lt != "1" && lt != "2" {
		return out, errors.New("invalid argument: unsupported value for list-type")
	}

	if lt == "1" {
		if p.ContinuationToken != "" || p.StartAfter != "" {
			return out, errors.New("invalid argument: continuation-token and start-after are not supported with list-type=1")
		}
	}
	if lt == "2" {
		if p.ContinuationToken != "" && p.StartAfter != "" {
			return out, errors.New("invalid argument: you cannot specify both continuation-token and start-after")
		}
		if p.Marker != "" {
			return out, errors.New("invalid argument: marker is not supported with list-type=2")
		}
	}

	bucket := strings.ToLower(strings.TrimSpace(p.Bucket))
	if !bucketNameRe.MatchString(bucket) {
		return out, errors.New("invalid bucket name")
	}
	if p.UserID <= 0 {
		return out, errors.New("invalid user")
	}
	if common.S3RemoteEnabled {
		return listS3BucketObjectsRemote(p)
	}
	root := filepath.Join(s3UserDir(p.UserID), bucket)
	fi, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
	}
	if !fi.IsDir() {
		return out, errors.New("invalid bucket")
	}

	var keys []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".meta.json") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil
		}
		key := filepath.ToSlash(rel)
		if p.Prefix != "" && !strings.HasPrefix(key, p.Prefix) {
			return nil
		}
		metaPath := path + ".meta.json"
		if mb, err := os.ReadFile(metaPath); err == nil {
			var m tempObjectMeta
			if json.Unmarshal(mb, &m) == nil && m.ExpiresAtUnix > 0 && m.ExpiresAtUnix <= time.Now().Unix() {
				_ = os.Remove(path)
				_ = os.Remove(metaPath)
				return nil
			}
		}
		keys = append(keys, key)
		return nil
	})
	sort.Strings(keys)

	// StartAfter（仅 v2，无 continuation 时）：键严格大于 StartAfter
	if lt == "2" && p.ContinuationToken == "" && p.StartAfter != "" {
		var cut []string
		for _, k := range keys {
			if k > p.StartAfter {
				cut = append(cut, k)
			}
		}
		keys = cut
	}

	merged := buildListMergeItems(keys, p.Prefix, p.Delimiter)

	startOff := 0
	if lt == "2" && p.ContinuationToken != "" {
		off, err := resolveListContinuationOffsetV2(p.ContinuationToken, p.Delimiter, merged)
		if err != nil {
			return out, err
		}
		startOff = off
	} else if lt == "1" && p.Marker != "" {
		startOff = mergedFirstIndexAfterKey(merged, p.Marker)
	}
	if lt == "2" && p.ContinuationToken != "" && (startOff < 0 || startOff > len(merged)) {
		return out, errors.New("invalid continuation-token")
	}

	page := merged[startOff:]
	if len(page) > p.MaxKeys {
		page = page[:p.MaxKeys]
	}
	truncated := startOff+len(page) < len(merged)

	for _, it := range page {
		if it.isCommonPrefix {
			out.CommonPrefixes = append(out.CommonPrefixes, it.key)
			continue
		}
		ent, err := listFillObjectEntry(p.UserID, bucket, it.key)
		if err != nil {
			continue
		}
		out.Contents = append(out.Contents, ent)
	}
	out.KeyCount = len(out.Contents) + len(out.CommonPrefixes)
	out.IsTruncated = truncated
	sort.Strings(out.CommonPrefixes)
	if truncated {
		nextOff := startOff + len(page)
		if lt == "2" {
			tok, err := encodeListContinuationToken(nextOff)
			if err != nil {
				return out, err
			}
			out.NextContinuationToken = tok
		} else if p.Delimiter != "" {
			out.NextMarker = page[len(page)-1].key
		}
	}
	return out, nil
}

func buildListMergeItems(keys []string, prefix, delimiter string) []listMergeItem {
	if delimiter == "" {
		out := make([]listMergeItem, len(keys))
		for i, k := range keys {
			out[i] = listMergeItem{isCommonPrefix: false, key: k}
		}
		return out
	}
	var out []listMergeItem
	for i := 0; i < len(keys); {
		k := keys[i]
		rest := strings.TrimPrefix(k, prefix)
		di := strings.Index(rest, delimiter)
		if di < 0 {
			out = append(out, listMergeItem{isCommonPrefix: false, key: k})
			i++
			continue
		}
		cp := k[:len(prefix)+di+len(delimiter)]
		out = append(out, listMergeItem{isCommonPrefix: true, key: cp})
		for i < len(keys) && strings.HasPrefix(keys[i], cp) {
			i++
		}
	}
	return out
}

func mergedFirstIndexAfterKey(merged []listMergeItem, marker string) int {
	for i, it := range merged {
		if it.key > marker {
			return i
		}
	}
	return len(merged)
}

func resolveListContinuationOffsetV2(token, delimiter string, merged []listMergeItem) (int, error) {
	raw, err := decodeBase64Flexible(token)
	if err != nil {
		return 0, errors.New("invalid argument: The continuation token provided is incorrect")
	}
	var jp struct {
		Version int `json:"version"`
		O       int `json:"o"`
	}
	if json.Unmarshal(raw, &jp) == nil && jp.Version == 2 && jp.O >= 0 {
		if jp.O > len(merged) {
			return 0, errors.New("invalid argument: The continuation token provided is incorrect")
		}
		return jp.O, nil
	}
	if delimiter != "" {
		return 0, errors.New("invalid argument: The continuation token provided is incorrect")
	}
	lastKey := string(raw)
	for o := 0; o < len(merged); o++ {
		if !merged[o].isCommonPrefix && merged[o].key > lastKey {
			return o, nil
		}
	}
	return len(merged), nil
}

func encodeListContinuationToken(offset int) (string, error) {
	b, err := json.Marshal(struct {
		Version int `json:"version"`
		O       int `json:"o"`
	}{Version: 2, O: offset})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func decodeBase64Flexible(s string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}

func listFillObjectEntry(userId int, bucket, key string) (ListObjectEntry, error) {
	if common.S3RemoteEnabled {
		return listFillObjectEntryRemote(userId, bucket, key)
	}
	_, objectPath, metaPath, err := objectPaths(userId, bucket, key)
	if err != nil {
		return ListObjectEntry{}, err
	}
	st, err := os.Stat(objectPath)
	if err != nil {
		return ListObjectEntry{}, err
	}
	var etag string
	var size int64
	if mb, err := os.ReadFile(metaPath); err == nil {
		var m tempObjectMeta
		if json.Unmarshal(mb, &m) == nil {
			etag = m.ETag
			size = m.Size
		}
	}
	if size == 0 && st.Size() > 0 {
		size = st.Size()
	}
	if etag == "" {
		var err2 error
		etag, err2 = objectSHA256ETag(objectPath)
		if err2 != nil {
			return ListObjectEntry{}, err2
		}
	}
	return ListObjectEntry{
		Key:          key,
		LastModified: st.ModTime().UTC(),
		ETag:         etag,
		Size:         size,
	}, nil
}
