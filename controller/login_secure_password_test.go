package controller

import (
	"testing"

	"github.com/songquanpeng/one-api/common/config"
)

func TestPublicStatusIncludesSecurePasswordLoginFlag(t *testing.T) {
	config.SecurePasswordLoginEnabled = false
	data := buildPublicStatusData()
	if data["secure_password_login"] != false {
		t.Fatalf("expected false, got %v", data["secure_password_login"])
	}
	config.SecurePasswordLoginEnabled = true
	data = buildPublicStatusData()
	if data["secure_password_login"] != true {
		t.Fatalf("expected true, got %v", data["secure_password_login"])
	}
}
