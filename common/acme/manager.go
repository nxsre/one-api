package acme

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/caddyserver/certmagic"
	"github.com/songquanpeng/one-api/common/cfg"
	"github.com/songquanpeng/one-api/common/logger"
)

var (
	mu      sync.RWMutex
	enabled bool
	magic   *certmagic.Config
	issuer  *certmagic.ACMEIssuer
	names   []string
	cancel  context.CancelFunc
)

// Enabled reports whether embedded ACME auto-TLS is active.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

// Init obtains certificates and starts automatic renewal (Let's Encrypt / compatible ACME CA).
func Init() error {
	if !cfg.V.GetBool("acme_enabled") {
		return nil
	}

	email := strings.TrimSpace(cfg.V.GetString("acme_email"))
	if email == "" {
		return fmt.Errorf("acme_enabled=true requires acme_email")
	}

	names = parseIdentifiers(
		cfg.V.GetString("acme_domains"),
		cfg.V.GetString("acme_ips"),
	)
	if len(names) == 0 {
		return fmt.Errorf("acme_enabled=true requires acme_domains and/or acme_ips")
	}

	storageDir := strings.TrimSpace(cfg.V.GetString("acme_storage_dir"))
	if storageDir == "" {
		storageDir = "data/acme"
	}

	m := certmagic.NewDefault()
	m.Storage = &certmagic.FileStorage{Path: storageDir}
	if anyIP(names) {
		m.RenewalWindowRatio = 0.33
		if ip, fallback := managedIPDefaults(names); ip != "" {
			m.DefaultServerName = ip
			if fallback != "" {
				m.FallbackServerName = fallback
			}
			logger.SysLog("ACME: IP certificate default server name " + ip)
		}
	}

	template := certmagic.DefaultACME
	template.Email = email
	template.Agreed = true
	if cfg.V.GetBool("acme_use_staging") {
		template.CA = certmagic.LetsEncryptStagingCA
	}
	if cfg.V.GetBool("acme_disable_http_challenge") {
		template.DisableHTTPChallenge = true
	}
	httpPort := cfg.V.GetInt("acme_http_port")
	if httpPort <= 0 {
		httpPort = certmagic.HTTPChallengePort
	}
	template.AltHTTPPort = httpPort

	iss := certmagic.NewACMEIssuer(m, template)
	iss.Email = email
	iss.Agreed = true
	if cfg.V.GetBool("acme_use_staging") {
		iss.CA = certmagic.LetsEncryptStagingCA
	}
	if cfg.V.GetBool("acme_disable_http_challenge") {
		iss.DisableHTTPChallenge = true
	}
	iss.AltHTTPPort = httpPort

	if anyIP(names) {
		iss.Profile = "shortlived"
	} else if profile := strings.TrimSpace(cfg.V.GetString("acme_profile")); profile != "" {
		iss.Profile = profile
	}

	m.Issuers = []certmagic.Issuer{iss}

	ctx, cancelFn := context.WithCancel(context.Background())

	logger.SysLog("ACME: obtaining certificates for " + strings.Join(names, ", "))
	if err := m.ManageSync(ctx, names); err != nil {
		cancelFn()
		return fmt.Errorf("acme obtain certificates: %w", err)
	}
	if err := m.ManageAsync(ctx, names); err != nil {
		cancelFn()
		return fmt.Errorf("acme start renewal: %w", err)
	}

	mu.Lock()
	enabled = true
	magic = m
	issuer = iss
	cancel = cancelFn
	mu.Unlock()

	logger.SysLog("ACME: certificate auto-renewal enabled (storage: " + storageDir + ")")
	return nil
}

// Shutdown stops background renewal goroutines.
func Shutdown() {
	mu.Lock()
	defer mu.Unlock()
	if cancel != nil {
		cancel()
		cancel = nil
	}
	enabled = false
	magic = nil
	issuer = nil
}

// TLSConfig returns a tls.Config that serves managed certificates (hot-reloaded on renew).
func TLSConfig() *tls.Config {
	mu.RLock()
	defer mu.RUnlock()
	if magic == nil {
		return nil
	}
	return magic.TLSConfig()
}

// WrapHTTPHandler adds HTTP-01 challenge handling for the ACME issuer.
func WrapHTTPHandler(h http.Handler) http.Handler {
	mu.RLock()
	iss := issuer
	mu.RUnlock()
	if iss == nil {
		return h
	}
	return iss.HTTPChallengeHandler(h)
}

func parseIdentifiers(domainsCSV, ipsCSV string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(raw string) {
		s := strings.TrimSpace(raw)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, part := range strings.Split(domainsCSV, ",") {
		add(part)
	}
	for _, part := range strings.Split(ipsCSV, ",") {
		add(part)
	}
	return out
}

func anyIP(identifiers []string) bool {
	for _, id := range identifiers {
		if net.ParseIP(id) != nil {
			return true
		}
	}
	return false
}

// managedIPDefaults picks DefaultServerName / FallbackServerName for IP certs.
// Inside Docker/NAT, empty SNI falls back to the container's local IP (e.g. 172.18.0.4),
// which is not on the certificate; defaulting to the configured public IP fixes handshake.
func managedIPDefaults(identifiers []string) (primary, fallback string) {
	for _, id := range identifiers {
		if net.ParseIP(id) == nil {
			continue
		}
		if primary == "" {
			primary = id
			continue
		}
		if fallback == "" {
			fallback = id
			return
		}
	}
	return
}
