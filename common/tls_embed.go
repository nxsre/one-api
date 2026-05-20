package common

import (
	"fmt"
	"os"
	"strings"

	"github.com/songquanpeng/one-api/common/cfg"
	"github.com/songquanpeng/one-api/common/env"
	"github.com/songquanpeng/one-api/common/logger"
)

// TLSCertFile / TLSKeyFile 用于进程内 ListenAndServeTLS，须同时配置。
var (
	TLSCertFile      string
	TLSKeyFile       string
	HTTPSOnly        bool
	TLSDualHTTPSPort string
)

// InitEmbeddedTLSFromEnv 从配置加载嵌入式 HTTPS；未配置证书时保持 HTTP。
// 若启用 acme_enabled，证书由 ACME 自动申请与续期，无需 tls_cert_file / tls_key_file。
func InitEmbeddedTLSFromEnv() {
	if cfg.V.GetBool("acme_enabled") {
		HTTPSOnly = true
		if cfg.V.IsSet("https_only") {
			HTTPSOnly = cfg.V.GetBool("https_only")
		}
		TLSDualHTTPSPort = ""
		if !HTTPSOnly {
			TLSDualHTTPSPort = strings.TrimSpace(env.StringAlways("https_port"))
			if TLSDualHTTPSPort == "" {
				TLSDualHTTPSPort = "3443"
			}
		}
		TLSCertFile = "acme-managed"
		TLSKeyFile = "acme-managed"
		return
	}

	TLSCertFile = strings.TrimSpace(env.StringAlways("tls_cert_file"))
	TLSKeyFile = strings.TrimSpace(env.StringAlways("tls_key_file"))
	HTTPSOnly = true
	TLSDualHTTPSPort = ""

	if TLSCertFile == "" && TLSKeyFile == "" {
		if cfg.V.GetBool("https_only") {
			logger.FatalLog("https_only=true requires tls_cert_file and tls_key_file")
		}
		return
	}
	if TLSCertFile == "" || TLSKeyFile == "" {
		logger.FatalLog("tls_cert_file and tls_key_file must both be set when enabling embedded HTTPS")
	}
	for _, pair := range []struct{ name, path string }{
		{"tls_cert_file", TLSCertFile},
		{"tls_key_file", TLSKeyFile},
	} {
		st, err := os.Stat(pair.path)
		if err != nil {
			logger.FatalLog(fmt.Sprintf("%s: invalid path %q: %v", pair.name, pair.path, err))
		}
		if st.IsDir() {
			logger.FatalLog(fmt.Sprintf("%s: path %q is a directory, expected a file", pair.name, pair.path))
		}
	}

	HTTPSOnly = true
	if cfg.V.IsSet("https_only") {
		HTTPSOnly = cfg.V.GetBool("https_only")
	}
	if !HTTPSOnly {
		TLSDualHTTPSPort = strings.TrimSpace(env.StringAlways("https_port"))
		if TLSDualHTTPSPort == "" {
			TLSDualHTTPSPort = "3443"
		}
	}
}
