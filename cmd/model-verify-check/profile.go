package main

import (
	"fmt"
	"strings"
)

const (
	profileStrict      = "strict"
	profileOAuthProxy  = "oauth-proxy"
)

func normalizeProfile(raw string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(raw))
	if p == "" {
		p = profileStrict
	}
	switch p {
	case profileStrict, profileOAuthProxy:
		return p, nil
	default:
		return "", fmt.Errorf("未知 -profile %q，可选 strict、oauth-proxy", raw)
	}
}

// oauthProxyExpectedWarn reports probes that commonly WARN on OAuth/CLI proxy channels.
func oauthProxyExpectedWarn(probeID string) bool {
	switch strings.ToUpper(strings.TrimSpace(probeID)) {
	case probeModelNameEN, probeModelNameZH, probeAgentProxy:
		return true
	default:
		return false
	}
}

// shouldExitFailure decides process exit code from probe outcomes and profile.
func shouldExitFailure(outcomes []probeOutcome, profile string) bool {
	switch profile {
	case profileOAuthProxy:
		for _, o := range outcomes {
			if !o.Success {
				return true
			}
			if o.Pass || oauthProxyExpectedWarn(o.ProbeID) {
				continue
			}
			return true
		}
		return false
	default:
		for _, o := range outcomes {
			if !o.Success || !o.Pass {
				return true
			}
		}
		return false
	}
}
