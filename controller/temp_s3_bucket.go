package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/awsv4"
	"github.com/songquanpeng/one-api/service"
)

// --- 桶级路由（path-style 与 /s3/ 前缀共用逻辑）---

func S3BucketGETPrefixed(c *gin.Context) {
	if !common.S3SiteOpen() {
		c.Status(http.StatusNotFound)
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.Param("bucket")))
	s3BucketGET(c, bucket)
}

func S3BucketWritePrefixed(c *gin.Context) {
	if !common.S3SiteOpen() {
		c.Status(http.StatusNotFound)
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.Param("bucket")))
	s3BucketWrite(c, bucket)
}

func S3BucketDeletePrefixed(c *gin.Context) {
	if !common.S3SiteOpen() {
		c.Status(http.StatusNotFound)
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.Param("bucket")))
	s3BucketDelete(c, bucket)
}

func S3BucketHeadPrefixed(c *gin.Context) {
	if !common.S3SiteOpen() {
		c.Status(http.StatusNotFound)
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.Param("bucket")))
	s3BucketHead(c, bucket)
}

func S3PathStyleBucketGET(c *gin.Context) {
	if !common.S3SiteOpen() || !common.S3PathStyleAtRoot {
		c.Status(http.StatusNotFound)
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.Param("bucket")))
	if service.IsReservedS3Bucket(bucket) {
		c.Status(http.StatusNotFound)
		return
	}
	s3BucketGET(c, bucket)
}

func S3PathStyleBucketWrite(c *gin.Context) {
	if !common.S3SiteOpen() || !common.S3PathStyleAtRoot {
		c.Status(http.StatusNotFound)
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.Param("bucket")))
	if service.IsReservedS3Bucket(bucket) {
		c.Status(http.StatusNotFound)
		return
	}
	s3BucketWrite(c, bucket)
}

func S3PathStyleBucketDelete(c *gin.Context) {
	if !common.S3SiteOpen() || !common.S3PathStyleAtRoot {
		c.Status(http.StatusNotFound)
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.Param("bucket")))
	if service.IsReservedS3Bucket(bucket) {
		c.Status(http.StatusNotFound)
		return
	}
	s3BucketDelete(c, bucket)
}

func S3PathStyleBucketHead(c *gin.Context) {
	if !common.S3SiteOpen() || !common.S3PathStyleAtRoot {
		c.Status(http.StatusNotFound)
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.Param("bucket")))
	if service.IsReservedS3Bucket(bucket) {
		c.Status(http.StatusNotFound)
		return
	}
	s3BucketHead(c, bucket)
}

func s3BucketGET(c *gin.Context, bucket string) {
	rid := s3RequestID(c)
	q := c.Request.URL.Query()
	if _, has := q["location"]; has {
		if c.Request.Method != http.MethodGet {
			writeS3APIError(c, http.StatusMethodNotAllowed, "MethodNotAllowed", "wrong method", rid, bucket, "", c.Request.URL.Path)
			return
		}
		if err := s3Verify(c); err != nil {
			writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		s3WriteGetBucketLocation(c, rid, bucket)
		return
	}
	if _, has := q["acl"]; has {
		if c.Request.Method != http.MethodGet {
			writeS3APIError(c, http.StatusMethodNotAllowed, "MethodNotAllowed", "wrong method", rid, bucket, "", c.Request.URL.Path)
			return
		}
		if err := s3Verify(c); err != nil {
			writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		s3WriteBucketACL(c, rid)
		return
	}
	if _, has := q["versioning"]; has {
		if err := s3Verify(c); err != nil {
			writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		s3WriteBucketVersioning(c, rid)
		return
	}
	if _, has := q["requestPayment"]; has {
		if err := s3Verify(c); err != nil {
			writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		s3WriteRequestPayment(c, rid)
		return
	}
	if _, has := q["encryption"]; has {
		if err := s3Verify(c); err != nil {
			writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		s3WriteBucketEncryptionDefault(c, rid)
		return
	}
	if _, has := q["lifecycle"]; has {
		if err := s3Verify(c); err != nil {
			writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		writeS3APIError(c, http.StatusNotFound, "NoSuchLifecycleConfiguration", "The lifecycle configuration does not exist", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if _, has := q["cors"]; has {
		if err := s3Verify(c); err != nil {
			writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		writeS3APIError(c, http.StatusNotFound, "NoSuchCORSConfiguration", "The CORS configuration does not exist", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if _, has := q["policy"]; has {
		if err := s3Verify(c); err != nil {
			writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		writeS3APIError(c, http.StatusNotFound, "NoSuchBucketPolicy", "The bucket policy does not exist", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if _, has := q["tagging"]; has {
		if err := s3Verify(c); err != nil {
			writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		writeS3APIError(c, http.StatusNotFound, "NoSuchTagSet", "The TagSet does not exist", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if _, has := q["publicAccessBlock"]; has {
		if err := s3Verify(c); err != nil {
			writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
			return
		}
		s3WritePublicAccessBlockOff(c, rid)
		return
	}
	// 列举
	s3ListBucket(c, bucket)
}

func s3BucketWrite(c *gin.Context, bucket string) {
	rid := s3RequestID(c)
	if c.Request.Method != http.MethodPut {
		writeS3APIError(c, http.StatusMethodNotAllowed, "MethodNotAllowed", "wrong method", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if err := s3Verify(c); err != nil {
		writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
		return
	}
	uid := s3AuthUID(c)
	if uid <= 0 {
		writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", "access denied", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if err := service.EnsureS3Bucket(uid, bucket); err != nil {
		writeS3APIError(c, http.StatusBadRequest, "InvalidBucketName", err.Error(), rid, bucket, "", c.Request.URL.Path)
		return
	}
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Status(http.StatusOK)
}

func s3BucketDelete(c *gin.Context, bucket string) {
	rid := s3RequestID(c)
	if c.Request.Method != http.MethodDelete {
		writeS3APIError(c, http.StatusMethodNotAllowed, "MethodNotAllowed", "wrong method", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if err := s3Verify(c); err != nil {
		writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
		return
	}
	uid := s3AuthUID(c)
	if uid <= 0 {
		writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", "access denied", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if err := service.RemoveS3Bucket(uid, bucket); err != nil {
		if errors.Is(err, service.ErrS3NoSuchBucket) {
			writeS3APIError(c, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", rid, bucket, "", c.Request.URL.Path)
			return
		}
		if strings.Contains(err.Error(), "not empty") {
			writeS3APIError(c, http.StatusConflict, "BucketNotEmpty", "The bucket you tried to delete is not empty", rid, bucket, "", c.Request.URL.Path)
			return
		}
		writeS3APIError(c, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", rid, bucket, "", c.Request.URL.Path)
		return
	}
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Status(http.StatusNoContent)
}

func s3BucketHead(c *gin.Context, bucket string) {
	rid := s3RequestID(c)
	if c.Request.Method != http.MethodHead {
		writeS3APIError(c, http.StatusMethodNotAllowed, "MethodNotAllowed", "wrong method", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if err := s3Verify(c); err != nil {
		writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", err.Error(), rid, bucket, "", c.Request.URL.Path)
		return
	}
	uid := s3AuthUID(c)
	if uid <= 0 {
		writeS3APIError(c, http.StatusForbidden, "SignatureDoesNotMatch", "access denied", rid, bucket, "", c.Request.URL.Path)
		return
	}
	if !awsv4.HasSigV4Auth(c.Request) && common.S3PathStyleAtRoot {
		RelayNotFound(c)
		return
	}
	if !service.BucketExistsS3(uid, bucket) {
		writeS3APIError(c, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", rid, bucket, "", c.Request.URL.Path)
		return
	}
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Status(http.StatusOK)
}

func s3WriteGetBucketLocation(c *gin.Context, rid, bucket string) {
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	loc := common.S3Region
	body := `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	if loc == "" || loc == "us-east-1" {
		body += `<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>`
	} else {
		body += `<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` + escapeXML(loc) + `</LocationConstraint>`
	}
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(body))
	_ = bucket
}

func s3WriteBucketACL(c *gin.Context, rid string) {
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(s3BucketACLXML()))
}

func s3WriteBucketVersioning(c *gin.Context, rid string) {
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Status>Disabled</Status>
</VersioningConfiguration>
`))
	_ = rid
}

func s3WriteRequestPayment(c *gin.Context, rid string) {
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<RequestPaymentConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Payer>BucketOwner</Payer>
</RequestPaymentConfiguration>
`))
	_ = rid
}

func s3WriteBucketEncryptionDefault(c *gin.Context, rid string) {
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ServerSideEncryptionConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>
`))
	_ = rid
}

func s3WritePublicAccessBlockOff(c *gin.Context, rid string) {
	applyS3CORS(c)
	setS3CommonResponseHeaders(c, rid)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<PublicAccessBlockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <BlockPublicAcls>false</BlockPublicAcls>
  <IgnorePublicAcls>false</IgnorePublicAcls>
  <BlockPublicPolicy>false</BlockPublicPolicy>
  <RestrictPublicBuckets>false</RestrictPublicBuckets>
</PublicAccessBlockConfiguration>
`))
	_ = rid
}

func s3BucketACLXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<AccessControlPolicy xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Owner>
    <ID>temp-s3-owner</ID>
    <DisplayName>temp-s3</DisplayName>
  </Owner>
  <AccessControlList>
    <Grant>
      <Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="CanonicalUser">
        <ID>temp-s3-owner</ID>
        <DisplayName>temp-s3</DisplayName>
      </Grantee>
      <Permission>FULL_CONTROL</Permission>
    </Grant>
  </AccessControlList>
</AccessControlPolicy>
`
}
