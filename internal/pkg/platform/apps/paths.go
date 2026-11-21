package apps

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jiangbyte/agentdeck/internal/config"
)

// PlatformKey 当前运行平台标识，如 linux-amd64。
func PlatformKey() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}

// ExpandCtx 路径/命令模板展开上下文。
type ExpandCtx struct {
	InstallDir string
	Workdir    string
	Bin        string
}

/**
 * Expand 将模板中的占位符替换为实际值。
 */
func Expand(s string, ctx ExpandCtx) string {
	replacer := strings.NewReplacer(
		"{os}", runtime.GOOS,
		"{arch}", runtime.GOARCH,
		"{os-arch}", PlatformKey(),
		"{install_dir}", ctx.InstallDir,
		"{workdir}", ctx.Workdir,
		"{bin}", ctx.Bin,
	)
	return replacer.Replace(s)
}

/**
 * SafeJoinUnder 将 rel 拼到 root 下，拒绝逃逸。
 */
func SafeJoinUnder(root, rel string) (string, error) {
	rel = filepath.Clean(strings.TrimSpace(rel))
	if rel == "." || rel == "" {
		return filepath.Clean(root), nil
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("路径不能为绝对路径")
	}
	if strings.HasPrefix(rel, "..") || strings.Contains(rel, string(filepath.Separator)+"..") {
		return "", fmt.Errorf("非法相对路径")
	}
	full := filepath.Clean(filepath.Join(root, rel))
	rootClean := filepath.Clean(root)
	sep := string(filepath.Separator)
	if full != rootClean && !strings.HasPrefix(full, rootClean+sep) {
		return "", fmt.Errorf("路径越界")
	}
	return full, nil
}

// AppDir 返回应用专属目录（一应用一目录）。
func AppDir(appID string) string {
	return filepath.Join(config.AppsDir(), appID)
}

// ServiceDir 返回运行日志等旁路目录（不写应用配置）。
func ServiceDir(appID string) string {
	return filepath.Join(config.ServicesDir(), appID)
}

/**
 * EnsureAppDir 确保应用目录存在。
 */
func EnsureAppDir(appID string) (string, error) {
	dir := AppDir(appID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
