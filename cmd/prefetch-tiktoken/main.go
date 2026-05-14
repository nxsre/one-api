package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	"github.com/songquanpeng/one-api/common/cfg"
	"github.com/songquanpeng/one-api/common/env"
)

// 与 tiktoken-go 中各 encoding 使用的 BPE 文件一一对应，构建时下载并写入 tiktoken_cache_dir，
// 避免容器启动或首次按需加载时再访问外网。库本身仍读取进程环境变量 TIKTOKEN_CACHE_DIR，此处从配置注入。
func main() {
	if err := cfg.Init(); err != nil {
		log.Fatal(err)
	}
	env.BindViper(cfg.V)
	// Docker/Makefile 构建阶段常用环境变量 TIKTOKEN_CACHE_DIR（见 Dockerfile）；主配置可为空时回退到该变量。
	dir := strings.TrimSpace(env.StringAlways("tiktoken_cache_dir"))
	if dir == "" {
		dir = strings.TrimSpace(os.Getenv("TIKTOKEN_CACHE_DIR"))
	}
	if dir == "" {
		log.Fatal("prefetch-tiktoken: set tiktoken_cache_dir in config, or export TIKTOKEN_CACHE_DIR (e.g. Dockerfile build stage)")
	}
	if err := os.Setenv("TIKTOKEN_CACHE_DIR", dir); err != nil {
		log.Fatal(err)
	}
	names := []string{
		"o200k_base",
		"cl100k_base",
		"p50k_base",
		"r50k_base",
		"p50k_edit",
	}
	for _, name := range names {
		if _, err := tiktoken.GetEncoding(name); err != nil {
			log.Fatalf("prefetch-tiktoken: %s: %v", name, err)
		}
		fmt.Fprintf(os.Stderr, "prefetch-tiktoken: ok %s\n", name)
	}
}
