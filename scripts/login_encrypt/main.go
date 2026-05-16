// 登录密码 AES-256-GCM 加密（与 common.EncryptLoginPayloadAES 一致），供 curl 联调脚本使用。
// 用法: go run ./scripts/login_encrypt <login_enc_key_base64> <明文密码>
package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/songquanpeng/one-api/common"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: go run ./scripts/login_encrypt <login_enc_key_base64> <plaintext>\n")
		os.Exit(2)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(os.Args[1]))
	if err != nil || len(key) != 32 {
		fmt.Fprintf(os.Stderr, "invalid login_enc_key (need 32 bytes after base64 decode)\n")
		os.Exit(1)
	}
	enc, err := common.EncryptLoginPayloadAES(key, os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "encrypt: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(enc)
}
