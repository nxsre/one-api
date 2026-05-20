package config

import "sync/atomic"

var exposeRatioEnabled atomic.Bool

func SetExposeRatioEnabled(enabled bool) {
	exposeRatioEnabled.Store(enabled)
}

func IsExposeRatioEnabled() bool {
	return exposeRatioEnabled.Load()
}
