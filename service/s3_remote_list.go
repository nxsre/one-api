package service

import (
	"errors"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/songquanpeng/one-api/common"
)

// listS3BucketObjectsRemote 在远程 S3 桶内列举（拉取前缀下全部键后复用本地分页/ delimiter 逻辑）。
func listS3BucketObjectsRemote(p ListObjectsParams) (ListObjectsResult, error) {
	var out ListObjectsResult
	out.EchoContinuationToken = p.ContinuationToken
	out.EchoStartAfter = p.StartAfter
	out.EchoMarker = p.Marker

	bucket := strings.ToLower(strings.TrimSpace(p.Bucket))
	base := remoteUserPrefix(p.UserID, bucket)
	s3Prefix := base + p.Prefix

	cli, err := remoteS3API()
	if err != nil {
		return out, err
	}
	ctx := remoteCtx()

	var keys []string
	var token *string
	for {
		lout, err := cli.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(common.S3RemoteBucket),
			Prefix:            aws.String(s3Prefix),
			ContinuationToken: token,
		})
		if err != nil {
			return out, err
		}
		for _, c := range lout.Contents {
			if c.Key == nil {
				continue
			}
			rel := strings.TrimPrefix(*c.Key, base)
			if rel == "" || rel == ".one-api-bucket" {
				continue
			}
			keys = append(keys, rel)
		}
		if !aws.ToBool(lout.IsTruncated) {
			break
		}
		if lout.NextContinuationToken == nil || strings.TrimSpace(*lout.NextContinuationToken) == "" {
			// 部分 S3 实现在 IsTruncated=true 时未返回 token，会导致死循环与客户端超时重试
			break
		}
		token = lout.NextContinuationToken
	}

	sort.Strings(keys)

	lt := strings.TrimSpace(p.ListType)
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
		return out, errors.New("invalid argument: The continuation token provided is incorrect")
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

func listFillObjectEntryRemote(userId int, bucket, key string) (ListObjectEntry, error) {
	_, size, etag, mod, err := statS3ObjectRemote(userId, bucket, key)
	if err != nil {
		return ListObjectEntry{}, err
	}
	return ListObjectEntry{
		Key:          key,
		LastModified: mod,
		ETag:         etag,
		Size:         size,
	}, nil
}
