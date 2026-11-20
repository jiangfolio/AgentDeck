package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 内置运行常量，不依赖外部配置文件。
const (
	ListenAddr    = ":8080"
	DBFileName    = "agentdeck.db"
	DataDirName   = "data"
	SessionCookie = "agentdeck_sid"
	SessionTTL    = 7 * 24 * time.Hour
	DefaultAdmin  = "admin"
	DefaultPass   = "admin123"
	CryptoKeyName = "sm2_server"
	HeaderSM4Key  = "X-SM4-Key"
	PublicKeyPath = "/api/crypto/public-key"
	LoginPath     = "/api/auth/login"

	CronLogDirName     = "cron-logs"
	CronMarkerPrefix   = "AgentDeck Cron"
	CronTaskNamePrefix = "AgentDeck-Cron-"
	StartupTaskPrefix  = "AgentDeck-Startup-"
	StartupSelfTask    = "AgentDeck-Self"
)

// DataDir 返回数据目录：优先可执行文件同级；go run 临时目录时回退到工作目录。
func DataDir() string {
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join(".", DataDirName)
	}
	dir := filepath.Dir(exe)
	if strings.Contains(dir, "go-build") {
		return filepath.Join(".", DataDirName)
	}
	return filepath.Join(dir, DataDirName)
}

// DBPath 返回 SQLite 文件绝对路径。
func DBPath() string {
	return filepath.Join(DataDir(), DBFileName)
}

// CronLogDir 返回定时任务运行日志目录。
func CronLogDir() string {
	return filepath.Join(DataDir(), CronLogDirName)
}

// CronWrapperDir 返回定时任务 wrapper 脚本目录。
func CronWrapperDir() string {
	return filepath.Join(DataDir(), "cron-wrappers")
}

// StartupWrapperDir 返回自启相关脚本/单元文件工作目录。
func StartupWrapperDir() string {
	return filepath.Join(DataDir(), "startup")
}
