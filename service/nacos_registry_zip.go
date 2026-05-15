package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/model"
)

func NacosZipStorageBackend() string {
	s := strings.ToLower(strings.TrimSpace(config.NacosRegistryZipStorage))
	switch s {
	case "s3":
		return "s3"
	case "db":
		return "db"
	case "local", "":
		return "local"
	default:
		return "local"
	}
}

// NacosRegistryEffectiveZipLocalDir 返回本地 ZIP 根目录：显式配置优先，否则默认 data/nacos-ai-registry（相对进程工作目录）。
func NacosRegistryEffectiveZipLocalDir() string {
	base := strings.TrimSpace(config.NacosRegistryZipLocalDir)
	if base != "" {
		return base
	}
	return filepath.Clean("data/nacos-ai-registry")
}

func nacosRegistryS3ObjectKey(namespace, kind string, artifactID int64, version string) string {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "public"
	}
	prefix := strings.Trim(config.NacosRegistryS3KeyPrefix, "/")
	if prefix == "" {
		prefix = "nacos-ai-registry"
	}
	return fmt.Sprintf("%s/%s/%s/%d/%s.zip", prefix, ns, kind, artifactID, version)
}

func nacosPutS3DirectWithContentType(key string, body []byte, contentType string) error {
	if strings.TrimSpace(common.S3RemoteBucket) == "" {
		return errors.New("未配置 s3_remote_bucket，无法使用 S3 存储")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	cli, err := remoteS3API()
	if err != nil {
		return err
	}
	_, err = cli.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        aws.String(common.S3RemoteBucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(body),
		ContentLength: aws.Int64(int64(len(body))),
		ContentType:   aws.String(contentType),
	})
	return err
}

func nacosPutS3Direct(key string, body []byte) error {
	return nacosPutS3DirectWithContentType(key, body, "application/zip")
}

func nacosGetS3Direct(key string) ([]byte, error) {
	cli, err := remoteS3API()
	if err != nil {
		return nil, err
	}
	out, err := cli.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func nacosDeleteS3Key(key string) error {
	if strings.TrimSpace(common.S3RemoteBucket) == "" || strings.TrimSpace(key) == "" {
		return nil
	}
	cli, err := remoteS3API()
	if err != nil {
		return err
	}
	_, err = cli.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(common.S3RemoteBucket),
		Key:    aws.String(key),
	})
	return err
}

func nacosLocalZipPath(artifactID int64, version string) (string, error) {
	base := NacosRegistryEffectiveZipLocalDir()
	dir := filepath.Join(base, "nacos-ai", fmt.Sprintf("%d", artifactID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, version+".zip"), nil
}

// NacosPersistVersionZIP 将 ZIP 按配置写入 db/local/s3，并更新版本行。
func NacosPersistVersionZIP(v *model.NacosAIArtifactVersion, namespace, kind string, artifactID int64, zip []byte) error {
	backend := NacosZipStorageBackend()
	switch backend {
	case "db":
		return model.DB.Model(v).Updates(map[string]interface{}{
			"zip_bytes":        zip,
			"zip_storage_kind": "db",
			"zip_storage_ref":  "",
		}).Error
	case "local":
		p, err := nacosLocalZipPath(artifactID, v.Version)
		if err != nil {
			return err
		}
		if err := os.WriteFile(p, zip, 0644); err != nil {
			return err
		}
		return model.DB.Model(v).Updates(map[string]interface{}{
			"zip_bytes":        nil,
			"zip_storage_kind": "local",
			"zip_storage_ref":  p,
		}).Error
	case "s3":
		key := nacosRegistryS3ObjectKey(namespace, kind, artifactID, v.Version)
		if err := nacosPutS3Direct(key, zip); err != nil {
			return err
		}
		return model.DB.Model(v).Updates(map[string]interface{}{
			"zip_bytes":        nil,
			"zip_storage_kind": "s3",
			"zip_storage_ref":  key,
		}).Error
	default:
		return errors.New("未知 zip 存储后端")
	}
}

// NacosRemoveStoredZIP 删除版本在 local/s3 上的对象（db 模式随行删除即可，此处不处理）。
func NacosRemoveStoredZIP(v *model.NacosAIArtifactVersion) error {
	k := strings.ToLower(strings.TrimSpace(v.ZipStorageKind))
	switch k {
	case "", "db":
		return nil
	case "local":
		if v.ZipStorageRef != "" {
			_ = os.Remove(v.ZipStorageRef)
		}
		return nil
	case "s3":
		return nacosDeleteS3Key(v.ZipStorageRef)
	default:
		return nil
	}
}

// NacosLoadVersionZIP 读取版本 ZIP 字节（兼容历史仅 zip_bytes 数据）。
func NacosLoadVersionZIP(v *model.NacosAIArtifactVersion) ([]byte, error) {
	kind := strings.ToLower(strings.TrimSpace(v.ZipStorageKind))
	if kind == "" || kind == "db" {
		if len(v.ZipBytes) > 0 {
			return v.ZipBytes, nil
		}
		if v.ZipStorageRef == "" {
			return nil, errors.New("ZIP 数据为空")
		}
	}
	switch kind {
	case "local":
		return os.ReadFile(v.ZipStorageRef)
	case "s3":
		return nacosGetS3Direct(v.ZipStorageRef)
	default:
		if len(v.ZipBytes) > 0 {
			return v.ZipBytes, nil
		}
		return nil, errors.New("无法解析 ZIP 存储方式")
	}
}
