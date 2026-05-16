package service

import (
	"testing"

	"github.com/songquanpeng/one-api/common/config"
)

func TestNacosCsDecryptStored_KeyRotation(t *testing.T) {
	oldK := config.NacosCsEncryptionKey
	oldP := config.NacosCsEncryptionKeyPrevious
	t.Cleanup(func() {
		config.NacosCsEncryptionKey = oldK
		config.NacosCsEncryptionKeyPrevious = oldP
	})

	config.NacosCsEncryptionKey = "primary-material-rotated"
	config.NacosCsEncryptionKeyPrevious = ""

	ct, encField, err := nacosCsEncryptPlaintext("payload-xyz")
	if err != nil {
		t.Fatal(err)
	}
	if encField != nacosCsEncEnvelope {
		t.Fatalf("enc field %q", encField)
	}

	// 仅旧密钥在 previous 中、主密钥已换新：仍能解密历史密文
	config.NacosCsEncryptionKey = "new-primary-after-rotation"
	config.NacosCsEncryptionKeyPrevious = "primary-material-rotated"

	got, err := NacosCsDecryptStored(ct, encField)
	if err != nil {
		t.Fatal(err)
	}
	if got != "payload-xyz" {
		t.Fatalf("got %q", got)
	}
}

func TestNacosCsDecryptStored_MultiLinePrevious(t *testing.T) {
	oldK := config.NacosCsEncryptionKey
	oldP := config.NacosCsEncryptionKeyPrevious
	t.Cleanup(func() {
		config.NacosCsEncryptionKey = oldK
		config.NacosCsEncryptionKeyPrevious = oldP
	})

	config.NacosCsEncryptionKey = "k0"
	config.NacosCsEncryptionKeyPrevious = ""

	ct, encField, err := nacosCsEncryptPlaintext("data")
	if err != nil {
		t.Fatal(err)
	}

	config.NacosCsEncryptionKey = "wrong-new"
	config.NacosCsEncryptionKeyPrevious = "noise\nk0\nother"

	got, err := NacosCsDecryptStored(ct, encField)
	if err != nil {
		t.Fatal(err)
	}
	if got != "data" {
		t.Fatalf("got %q", got)
	}
}
