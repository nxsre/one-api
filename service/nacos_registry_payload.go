package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
)

// NacosRegistryPayloadBackend MCP/A2A/Prompt 大 JSON 与 Skill ZIP 共用 nacos_registry_zip_storage：db | local | s3。
func NacosRegistryPayloadBackend() string {
	return NacosZipStorageBackend()
}

func nacosPayloadSanitizeNs(ns string) string {
	s := strings.TrimSpace(ns)
	if s == "" {
		s = "public"
	}
	s = strings.ReplaceAll(s, string(os.PathSeparator), "_")
	s = strings.ReplaceAll(s, "..", "_")
	return s
}

func nacosPayloadLocalPath(namespace, family string, id int64) (string, error) {
	ns := nacosPayloadSanitizeNs(namespace)
	base := filepath.Join(NacosRegistryEffectiveZipLocalDir(), "nacos-ai-payloads", family, ns)
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", err
	}
	return filepath.Join(base, fmt.Sprintf("%d.json", id)), nil
}

func nacosPayloadS3Key(namespace, family string, id int64) string {
	ns := nacosPayloadSanitizeNs(namespace)
	prefix := strings.Trim(config.NacosRegistryS3KeyPrefix, "/")
	if prefix == "" {
		prefix = "nacos-ai-registry"
	}
	return fmt.Sprintf("%s/payloads/%s/%s/%d.json", prefix, family, ns, id)
}

// NacosWriteRegistryPayload 写入 MCP/A2A/Prompt 等 JSON 负载；db 时返回 kind=db、ref 空、dbInline 为应写入列的文本。
func NacosWriteRegistryPayload(namespace, family string, id int64, body []byte) (kind string, ref string, dbInline string, err error) {
	switch NacosRegistryPayloadBackend() {
	case "db":
		return "db", "", string(body), nil
	case "local":
		p, err := nacosPayloadLocalPath(namespace, family, id)
		if err != nil {
			return "", "", "", err
		}
		if err := os.WriteFile(p, body, 0644); err != nil {
			return "", "", "", err
		}
		return "local", p, "", nil
	case "s3":
		key := nacosPayloadS3Key(namespace, family, id)
		if err := nacosPutS3DirectWithContentType(key, body, "application/json"); err != nil {
			return "", "", "", err
		}
		return "s3", key, "", nil
	default:
		return "", "", "", errors.New("未知 payload 存储后端")
	}
}

// NacosReadRegistryPayload 按存储元数据读取 JSON 字节（兼容历史仅列内联数据）。
func NacosReadRegistryPayload(kind, ref, dbInline string) ([]byte, error) {
	k := strings.ToLower(strings.TrimSpace(kind))
	if k == "" || k == "db" {
		if s := strings.TrimSpace(dbInline); s != "" {
			return []byte(s), nil
		}
		// 历史：未写 kind 列但内容在本地路径（极少见）
		if strings.TrimSpace(ref) != "" && k == "" {
			if b, err := os.ReadFile(ref); err == nil {
				return b, nil
			}
		}
		return nil, errors.New("payload 为空")
	}
	switch k {
	case "local":
		if strings.TrimSpace(ref) == "" {
			return nil, errors.New("local payload 路径为空")
		}
		return os.ReadFile(ref)
	case "s3":
		if strings.TrimSpace(ref) == "" {
			return nil, errors.New("s3 payload key 为空")
		}
		return nacosGetS3Direct(ref)
	default:
		return nil, fmt.Errorf("无法解析 payload 存储方式 %q", kind)
	}
}

// NacosRemoveRegistryPayload 删除外置对象（db 模式无操作）。
func NacosRemoveRegistryPayload(kind, ref string) error {
	k := strings.ToLower(strings.TrimSpace(kind))
	switch k {
	case "", "db":
		return nil
	case "local":
		if ref != "" {
			_ = os.Remove(ref)
		}
		return nil
	case "s3":
		return nacosDeleteS3Key(ref)
	default:
		return nil
	}
}
