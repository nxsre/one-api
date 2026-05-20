package common

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/songquanpeng/one-api/common/cfg"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/env"
	"github.com/songquanpeng/one-api/common/logger"
)

var (
	Port         int
	PrintVersion bool
	PrintHelp    bool
)

func printHelp() {
	fmt.Println("One API " + Version + " - All in one API service for OpenAI API.")
	fmt.Println("Copyright (C) 2023 JustSong. All rights reserved.")
	fmt.Println("GitHub: https://github.com/songquanpeng/one-api")
	fmt.Println("Usage: one-api [--config path] [--port <port>] [--log-dir <dir>] [--version] [--help]")
}

func Init() {
	if err := cfg.Init(); err != nil {
		log.Fatal(err)
	}
	PrintVersion = cfg.PrintVersion
	PrintHelp = cfg.PrintHelp

	cfg.MustExitForFlags(func() { fmt.Println(Version) }, func() { printHelp() }, os.Exit)

	env.BindViper(cfg.V)
	config.LoadRuntime()

	Port = cfg.V.GetInt("port")

	if secret := strings.TrimSpace(env.StringAlways("session_secret")); secret != "" {
		if secret == "random_string" {
			logger.SysError("session_secret is set to an example value, please change it to a random string.")
		} else {
			config.SessionSecret = secret
		}
	}
	if p := strings.TrimSpace(env.StringAlways("sqlite_path")); p != "" {
		SQLitePath = p
	}

	SQLiteBusyTimeout = env.Int("SQLITE_BUSY_TIMEOUT", 3000)

	if err := cfg.EnsureLogDir(); err != nil {
		log.Fatal(err)
	}
	logger.LogDir = cfg.LogDir

	InitIdGenerator()
	InitSecurityEnv()
	InitS3Config()
}
