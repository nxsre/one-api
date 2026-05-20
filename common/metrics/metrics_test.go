package metrics

import (
	"testing"
)

func TestMetricsRegistered(t *testing.T) {
	// The metrics are registered in init(). If they are registered multiple times, it will panic.
	// We just check if they are non-nil.
	if RatioFallback == nil {
		t.Fatal("RatioFallback is nil")
	}
	if RelayRequestTotal == nil {
		t.Fatal("RelayRequestTotal is nil")
	}
	if RelayRequestDuration == nil {
		t.Fatal("RelayRequestDuration is nil")
	}
	if ClientDisconnectTotal == nil {
		t.Fatal("ClientDisconnectTotal is nil")
	}
	if CircuitFailureTotal == nil {
		t.Fatal("CircuitFailureTotal is nil")
	}
	if QuotaConsumedTotal == nil {
		t.Fatal("QuotaConsumedTotal is nil")
	}
	
	// Test if we can successfully get a metric from the registry.
	// Since init is run, we just check if it's not nil.
}
