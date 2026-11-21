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

// DataDir 返回数据目录绝对路径：优先可执行文件同级；go run 临时目录时回退到工作目录。
func DataDir() string {
	exe, err := os.Executable()
	var dir string
	if err != nil {
		dir = filepath.Join(".", DataDirName)
	} else {
		base := filepath.Dir(exe)
		if strings.Contains(base, "go-build") {
			dir = filepath.Join(".", DataDirName)
		} else {
			dir = filepath.Join(base, DataDirName)
		}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
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

// TrashDir 返回文件管理回收站目录。
func TrashDir() string {
	return filepath.Join(DataDir(), "trash")
}

// AppsDir 返回应用目录根（~/download，一应用一子目录）。
func AppsDir() string {
	return DownloadDir()
}

// ServicesDir 返回应用运行日志等旁路目录（不写应用配置）。
func ServicesDir() string {
	return filepath.Join(DataDir(), "services")
}

/**
 * DownloadDir 返回用户下载目录 ~/download（由 HOME 解析，不写死绝对路径）。
 */
func DownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		home = "."
	}
	dir := filepath.Join(home, "download")
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}

/**
 * DownloadTempDir 返回下载临时目录 ~/download/temp。
 */
func DownloadTempDir() string {
	return filepath.Join(DownloadDir(), "temp")
}

// MarketDir 兼容旧名，指向下载临时目录。
func MarketDir() string {
	return DownloadTempDir()
}
