package controller

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestIsChannelModelTestTimeout(t *testing.T) {
	if !isChannelModelTestTimeout(context.DeadlineExceeded) {
		t.Fatal("expected deadline exceeded")
	}
	var netErr net.Error = &timeoutError{}
	if !isChannelModelTestTimeout(netErr) {
		t.Fatal("expected net timeout")
	}
	if isChannelModelTestTimeout(errors.New("http status code: 429")) {
		t.Fatal("429 should not be timeout")
	}
	if !isChannelModelTestTimeout(fmt.Errorf("do request failed: %w", context.DeadlineExceeded)) {
		t.Fatal("wrapped deadline")
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func TestChannelModelTestTimeoutConstant(t *testing.T) {
	if channelModelTestTimeout != 30*time.Second {
		t.Fatalf("timeout want 30s got %v", channelModelTestTimeout)
	}
}
