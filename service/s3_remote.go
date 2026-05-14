package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/songquanpeng/one-api/common"
)

const (
	remoteMetaExpires = "one-api-expires"
	remoteMetaCT      = "one-api-ct"
	remoteMetaSize    = "one-api-size"
	remoteMetaETag    = "one-api-etag"
)

var (
	remoteS3Client  *s3.Client
	remoteS3Once  sync.Once
	remoteS3InitE error
)

func remoteCtx() context.Context { return context.Background() }

func remoteS3API() (*s3.Client, error) {
	remoteS3Once.Do(func() {
		opts := []func(*config.LoadOptions) error{
			config.WithRegion(common.S3RemoteRegion),
		}
		if common.S3RemoteAccessKeyID != "" && common.S3RemoteSecretAccessKey != "" {
			opts = append(opts, config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(common.S3RemoteAccessKeyID, common.S3RemoteSecretAccessKey, "")))
		}
		cfg, err := config.LoadDefaultConfig(remoteCtx(), opts...)
		if err != nil {
			remoteS3InitE = err
			return
		}
		client := s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = common.S3RemotePathStyle
			if common.S3RemoteEndpoint != "" {
				o.BaseEndpoint = aws.String(common.S3RemoteEndpoint)
			}
		})
		remoteS3Client = client
	})
	if remoteS3InitE != nil {
		return nil, remoteS3InitE
	}
	return remoteS3Client, nil
}

func remoteUserPrefix(userId int, bucket string) string {
	b := strings.ToLower(strings.TrimSpace(bucket))
	return common.S3RemoteKeyPrefix + fmt.Sprintf("u%d/%s/", userId, b)
}

func remoteObjectFullKey(userId int, bucket, objectKey string) (string, error) {
	if userId <= 0 {
		return "", errors.New("invalid user")
	}
	bucket = strings.ToLower(strings.TrimSpace(bucket))
	if !bucketNameRe.MatchString(bucket) {
		return "", errors.New("invalid bucket name")
	}
	k, err := sanitizeObjectKey(objectKey)
	if err != nil {
		return "", err
	}
	base := remoteUserPrefix(userId, bucket)
	full := base + k
	rel := strings.TrimPrefix(full, base)
	if strings.Contains(rel, "..") {
		return "", errors.New("invalid path")
	}
	return full, nil
}

func mapRemoteErr(err error) error {
	if err == nil {
		return nil
	}
	var nk *s3types.NoSuchKey
	var nf *s3types.NotFound
	var nsb *s3types.NoSuchBucket
	if errors.As(err, &nk) || errors.As(err, &nf) || errors.As(err, &nsb) {
		return os.ErrNotExist
	}
	return err
}

func remotePutMeta(exp int64, ct string, size int64, etagHex string) map[string]string {
	return map[string]string{
		remoteMetaExpires: strconv.FormatInt(exp, 10),
		remoteMetaCT:      ct,
		remoteMetaSize:    strconv.FormatInt(size, 10),
		remoteMetaETag:    strings.Trim(etagHex, "\""),
	}
}

func remoteReadMeta(md map[string]string) (exp int64, ct string, size int64, etag string) {
	if md == nil {
		return 0, "", 0, ""
	}
	if v := md[remoteMetaExpires]; v != "" {
		exp, _ = strconv.ParseInt(v, 10, 64)
	}
	ct = md[remoteMetaCT]
	if v := md[remoteMetaSize]; v != "" {
		size, _ = strconv.ParseInt(v, 10, 64)
	}
	if eh := md[remoteMetaETag]; eh != "" {
		etag = "\"" + strings.Trim(eh, "\"") + "\""
	}
	return
}

func putS3ObjectRemote(userId int, bucket, key, contentType string, ttl time.Duration, body io.Reader, contentMD5Base64 string) (etag string, size int64, err error) {
	if ttl <= 0 {
		ttl = common.S3DefaultTTL
	}
	if ttl > common.S3MaxTTL {
		ttl = common.S3MaxTTL
	}
	fullKey, err := remoteObjectFullKey(userId, bucket, key)
	if err != nil {
		return "", 0, err
	}
	cli, err := remoteS3API()
	if err != nil {
		return "", 0, err
	}
	var md5Expect []byte
	if strings.TrimSpace(contentMD5Base64) != "" {
		md5Expect, err = base64.StdEncoding.DecodeString(strings.TrimSpace(contentMD5Base64))
		if err != nil || len(md5Expect) != md5.Size {
			return "", 0, errors.New("invalid Content-MD5")
		}
	}
	max := common.S3MaxObjectBytes + 1
	data, err := io.ReadAll(io.LimitReader(body, max))
	if err != nil {
		return "", 0, err
	}
	if int64(len(data)) > common.S3MaxObjectBytes {
		return "", 0, fmt.Errorf("object too large")
	}
	if md5Expect != nil {
		sum := md5.Sum(data)
		if !bytesEqual(sum[:], md5Expect) {
			return "", 0, errors.New("bad digest: Content-MD5 does not match")
		}
	}
	h := sha256.Sum256(data)
	n := int64(len(data))
	etag = "\"" + hex.EncodeToString(h[:]) + "\""
	exp := time.Now().Add(ttl).Unix()
	meta := remotePutMeta(exp, contentType, n, etag)
	in := &s3.PutObjectInput{
		Bucket:        aws.String(common.S3RemoteBucket),
		Key:           aws.String(fullKey),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(n),
		Metadata:      meta,
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	_, err = cli.PutObject(remoteCtx(), in)
	if err != nil {
		return "", 0, err
	}
	return etag, n, nil
}

// Fix: PutObject Body - use bytes.Reader not string(data) for binary
// I'll fix in next edit - strings.NewReader(string(data)) breaks binary - use bytes.NewReader(data)

func getS3ObjectRemote(userId int, bucket, key string) (rc io.ReadCloser, contentType string, size int64, etag string, err error) {
	fullKey, err := remoteObjectFullKey(userId, bucket, key)
	if err != nil {
		return nil, "", 0, "", err
	}
	cli, err := remoteS3API()
	if err != nil {
		return nil, "", 0, "", err
	}
	ctx := remoteCtx()
	hout, err := cli.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return nil, "", 0, "", mapRemoteErr(err)
	}
	exp, mct, msize, metag := remoteReadMeta(hout.Metadata)
	if exp > 0 && exp <= time.Now().Unix() {
		_, _ = cli.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(common.S3RemoteBucket),
			Key:    aws.String(fullKey),
		})
		return nil, "", 0, "", os.ErrNotExist
	}
	gout, err := cli.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return nil, "", 0, "", mapRemoteErr(err)
	}
	ct := mct
	if ct == "" && gout.ContentType != nil {
		ct = *gout.ContentType
	}
	sz := msize
	if sz <= 0 && gout.ContentLength != nil {
		sz = *gout.ContentLength
	}
	et := metag
	if et == "" && gout.ETag != nil {
		et = *gout.ETag
	}
	return gout.Body, ct, sz, et, nil
}

func headS3ObjectRemote(userId int, bucket, key string) (contentType string, size int64, etag string, err error) {
	fullKey, err := remoteObjectFullKey(userId, bucket, key)
	if err != nil {
		return "", 0, "", err
	}
	cli, err := remoteS3API()
	if err != nil {
		return "", 0, "", err
	}
	ctx := remoteCtx()
	hout, err := cli.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return "", 0, "", mapRemoteErr(err)
	}
	exp, mct, msize, metag := remoteReadMeta(hout.Metadata)
	if exp > 0 && exp <= time.Now().Unix() {
		_, _ = cli.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(common.S3RemoteBucket),
			Key:    aws.String(fullKey),
		})
		return "", 0, "", os.ErrNotExist
	}
	return mct, msize, metag, nil
}

func deleteS3ObjectRemote(userId int, bucket, key string) error {
	fullKey, err := remoteObjectFullKey(userId, bucket, key)
	if err != nil {
		return err
	}
	cli, err := remoteS3API()
	if err != nil {
		return err
	}
	_, err = cli.DeleteObject(remoteCtx(), &s3.DeleteObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(fullKey),
	})
	return err
}

func statS3ObjectRemote(userId int, bucket, key string) (contentType string, size int64, etag string, modTime time.Time, err error) {
	fullKey, err := remoteObjectFullKey(userId, bucket, key)
	if err != nil {
		return "", 0, "", time.Time{}, err
	}
	cli, err := remoteS3API()
	if err != nil {
		return "", 0, "", time.Time{}, err
	}
	ctx := remoteCtx()
	hout, err := cli.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return "", 0, "", time.Time{}, mapRemoteErr(err)
	}
	exp, mct, msize, metag := remoteReadMeta(hout.Metadata)
	if exp > 0 && exp <= time.Now().Unix() {
		_, _ = cli.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(common.S3RemoteBucket),
			Key:    aws.String(fullKey),
		})
		return "", 0, "", time.Time{}, os.ErrNotExist
	}
	mod := time.Now().UTC()
	if hout.LastModified != nil {
		mod = *hout.LastModified
	}
	return mct, msize, metag, mod, nil
}

func openS3ObjectRangeRemote(userId int, bucket, key string, start, length int64) (rc io.ReadCloser, contentType string, totalSize int64, etag string, err error) {
	fullKey, err := remoteObjectFullKey(userId, bucket, key)
	if err != nil {
		return nil, "", 0, "", err
	}
	cli, err := remoteS3API()
	if err != nil {
		return nil, "", 0, "", err
	}
	ctx := remoteCtx()
	hout, err := cli.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return nil, "", 0, "", mapRemoteErr(err)
	}
	exp, mct, msize, metag := remoteReadMeta(hout.Metadata)
	if exp > 0 && exp <= time.Now().Unix() {
		_, _ = cli.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(common.S3RemoteBucket),
			Key:    aws.String(fullKey),
		})
		return nil, "", 0, "", os.ErrNotExist
	}
	total := msize
	if total <= 0 && hout.ContentLength != nil {
		total = *hout.ContentLength
	}
	if start < 0 || start > total {
		return nil, "", total, metag, fmt.Errorf("invalid range")
	}
	var end int64
	if length < 0 {
		end = total - 1
	} else {
		if start+length > total {
			return nil, "", total, metag, fmt.Errorf("invalid range")
		}
		end = start + length - 1
	}
	rng := fmt.Sprintf("bytes=%d-%d", start, end)
	gout, err := cli.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(fullKey),
		Range:  aws.String(rng),
	})
	if err != nil {
		return nil, "", total, metag, mapRemoteErr(err)
	}
	ct := mct
	if ct == "" && gout.ContentType != nil {
		ct = *gout.ContentType
	}
	if ct == "" {
		ct = "application/octet-stream"
	}
	return gout.Body, ct, total, metag, nil
}

func copyS3ObjectRemote(userId int, dstBucket, dstKey, srcBucket, srcKey, contentType string, ttl time.Duration) (etag string, size int64, err error) {
	srcFull, err := remoteObjectFullKey(userId, srcBucket, srcKey)
	if err != nil {
		return "", 0, err
	}
	dstFull, err := remoteObjectFullKey(userId, dstBucket, dstKey)
	if err != nil {
		return "", 0, err
	}
	cli, err := remoteS3API()
	if err != nil {
		return "", 0, err
	}
	ctx := remoteCtx()
	srcHead, err := cli.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(srcFull),
	})
	if err != nil {
		return "", 0, mapRemoteErr(err)
	}
	exp, mct, msize, _ := remoteReadMeta(srcHead.Metadata)
	if exp > 0 && exp <= time.Now().Unix() {
		return "", 0, os.ErrNotExist
	}
	ct := contentType
	if ct == "" {
		ct = mct
	}
	if ct == "" {
		ct = "application/octet-stream"
	}
	if ttl <= 0 {
		ttl = common.S3DefaultTTL
	}
	if ttl > common.S3MaxTTL {
		ttl = common.S3MaxTTL
	}
	newExp := time.Now().Add(ttl).Unix()
	_, _, _, srcETag := remoteReadMeta(srcHead.Metadata)
	copySrc := common.S3RemoteBucket + "/" + srcFull
	in := &s3.CopyObjectInput{
		Bucket:            aws.String(common.S3RemoteBucket),
		Key:               aws.String(dstFull),
		CopySource:        aws.String(copySrc),
		ContentType:       aws.String(ct),
		MetadataDirective: s3types.MetadataDirectiveReplace,
		Metadata:          remotePutMeta(newExp, ct, msize, srcETag),
	}
	out, err := cli.CopyObject(ctx, in)
	if err != nil {
		return "", 0, err
	}
	// Copy may not return body etag in meta - Head dst
	h2, err := cli.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(dstFull),
	})
	if err != nil {
		return "", 0, err
	}
	_, _, _, etag = remoteReadMeta(h2.Metadata)
	if etag == "" && out.CopyObjectResult != nil && out.CopyObjectResult.ETag != nil {
		etag = *out.CopyObjectResult.ETag
	}
	if msize <= 0 && h2.ContentLength != nil {
		msize = *h2.ContentLength
	}
	return etag, msize, nil
}

func ensureS3BucketRemote(userId int, bucket string) error {
	if userId <= 0 {
		return errors.New("invalid user")
	}
	b := strings.ToLower(strings.TrimSpace(bucket))
	if !bucketNameRe.MatchString(b) {
		return errors.New("invalid bucket name")
	}
	key := remoteUserPrefix(userId, bucket) + ".one-api-bucket"
	cli, err := remoteS3API()
	if err != nil {
		return err
	}
	ctx := remoteCtx()
	_, err = cli.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return nil
	}
	far := time.Now().Add(100 * 365 * 24 * time.Hour).Unix()
	sum := sha256.Sum256(nil)
	etag := "\"" + hex.EncodeToString(sum[:]) + "\""
	_, err = cli.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(common.S3RemoteBucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(nil),
		ContentLength: aws.Int64(0),
		ContentType:   aws.String("application/octet-stream"),
		Metadata:      remotePutMeta(far, "application/octet-stream", 0, etag),
	})
	return err
}

func bucketExistsS3Remote(userId int, bucket string) bool {
	if userId <= 0 || !IsValidS3BucketName(bucket) {
		return false
	}
	cli, err := remoteS3API()
	if err != nil {
		return false
	}
	base := remoteUserPrefix(userId, bucket)
	out, err := cli.ListObjectsV2(remoteCtx(), &s3.ListObjectsV2Input{
		Bucket:  aws.String(common.S3RemoteBucket),
		Prefix:  aws.String(base),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false
	}
	return len(out.Contents) > 0 || len(out.CommonPrefixes) > 0
}

func s3BucketIsEmptyRemote(userId int, bucket string) (bool, error) {
	cli, err := remoteS3API()
	if err != nil {
		return false, err
	}
	base := remoteUserPrefix(userId, bucket)
	var token *string
	for {
		out, err := cli.ListObjectsV2(remoteCtx(), &s3.ListObjectsV2Input{
			Bucket:            aws.String(common.S3RemoteBucket),
			Prefix:            aws.String(base),
			MaxKeys:           aws.Int32(1000),
			ContinuationToken: token,
		})
		if err != nil {
			return false, err
		}
		for _, c := range out.Contents {
			if c.Key == nil || *c.Key == "" {
				continue
			}
			rel := strings.TrimPrefix(*c.Key, base)
			if rel != ".one-api-bucket" {
				return false, nil
			}
		}
		if !aws.ToBool(out.IsTruncated) {
			break
		}
		if out.NextContinuationToken == nil || strings.TrimSpace(*out.NextContinuationToken) == "" {
			break
		}
		token = out.NextContinuationToken
	}
	return true, nil
}

func removeS3BucketRemote(userId int, bucket string) error {
	if !bucketExistsS3Remote(userId, bucket) {
		return ErrS3NoSuchBucket
	}
	empty, err := s3BucketIsEmptyRemote(userId, bucket)
	if err != nil {
		return err
	}
	if !empty {
		return errors.New("bucket not empty")
	}
	return remoteDeletePrefix(remoteUserPrefix(userId, bucket))
}

func remoteDeletePrefix(prefix string) error {
	cli, err := remoteS3API()
	if err != nil {
		return err
	}
	ctx := remoteCtx()
	var token *string
	for {
		out, err := cli.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(common.S3RemoteBucket),
			Prefix:            aws.String(prefix),
			MaxKeys:           aws.Int32(1000),
			ContinuationToken: token,
		})
		if err != nil {
			return err
		}
		for _, c := range out.Contents {
			if c.Key == nil {
				continue
			}
			_, _ = cli.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(common.S3RemoteBucket),
				Key:    c.Key,
			})
		}
		if !aws.ToBool(out.IsTruncated) {
			break
		}
		if out.NextContinuationToken == nil || strings.TrimSpace(*out.NextContinuationToken) == "" {
			break
		}
		token = out.NextContinuationToken
	}
	return nil
}

func sweepExpiredS3Remote() error {
	cli, err := remoteS3API()
	if err != nil {
		return err
	}
	pfx := common.S3RemoteKeyPrefix
	if pfx == "" {
		pfx = ""
	}
	var token *string
	for {
		out, err := cli.ListObjectsV2(remoteCtx(), &s3.ListObjectsV2Input{
			Bucket:            aws.String(common.S3RemoteBucket),
			Prefix:            aws.String(pfx),
			MaxKeys:           aws.Int32(500),
			ContinuationToken: token,
		})
		if err != nil {
			return err
		}
		for _, c := range out.Contents {
			if c.Key == nil {
				continue
			}
			h, err := cli.HeadObject(remoteCtx(), &s3.HeadObjectInput{
				Bucket: aws.String(common.S3RemoteBucket),
				Key:    c.Key,
			})
			if err != nil {
				continue
			}
			exp, _, _, _ := remoteReadMeta(h.Metadata)
			if exp > 0 && exp <= time.Now().Unix() {
				_, _ = cli.DeleteObject(remoteCtx(), &s3.DeleteObjectInput{
					Bucket: aws.String(common.S3RemoteBucket),
					Key:    c.Key,
				})
			}
		}
		if !aws.ToBool(out.IsTruncated) {
			break
		}
		if out.NextContinuationToken == nil || strings.TrimSpace(*out.NextContinuationToken) == "" {
			break
		}
		token = out.NextContinuationToken
	}
	return nil
}
