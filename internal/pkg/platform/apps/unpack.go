package apps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiangbyte/agentdeck/internal/pkg/platform/archive"
)

/**
 * UnpackArchive 清空 destDir 后解压（市场/批量落盘用）；底层走跨平台 archive 包。
 */
func UnpackArchive(archivePath, destDir string) error {
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("清理目标目录失败: %w", err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}
	return archive.Extract(archivePath, destDir)
}

/**
 * FlattenIfSingleRoot 若解压后只有一层包装根目录（如 redis-7.2.5/），则上提内容；
 * 已是 bin/sbin 等布局时不处理。
 */
func FlattenIfSingleRoot(destDir string) error {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return err
	}
	var onlyDir string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			if onlyDir != "" {
				return nil
			}
			onlyDir = name
			continue
		}
		if name != "manifest.json" {
			return nil
		}
	}
	if onlyDir == "" {
		return nil
	}
	switch strings.ToLower(onlyDir) {
	case "bin", "sbin", "lib", "lib64", "usr", "etc", "conf", "share", "include", "data", "logs":
		return nil
	}
	root := filepath.Join(destDir, onlyDir)
	children, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	if len(children) == 0 {
		return nil
	}
	for _, c := range children {
		from := filepath.Join(root, c.Name())
		to := filepath.Join(destDir, c.Name())
		if err := os.Rename(from, to); err != nil {
			return err
		}
	}
	return os.RemoveAll(root)
}
