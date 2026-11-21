package archive

import (
	"fmt"
	"path/filepath"
	"strings"
)

/**
 * SafeJoinUnder 将 rel 拼到 root 下，拒绝逃逸（防 zip slip）。
 */
func SafeJoinUnder(root, rel string) (string, error) {
	rel = filepath.Clean(strings.TrimSpace(rel))
	if rel == "." || rel == "" {
		return filepath.Clean(root), nil
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("压缩包内路径不能为绝对路径")
	}
	if strings.HasPrefix(rel, "..") || strings.Contains(rel, string(filepath.Separator)+"..") {
		return "", fmt.Errorf("压缩包内含非法相对路径")
	}
	full := filepath.Clean(filepath.Join(root, rel))
	rootClean := filepath.Clean(root)
	sep := string(filepath.Separator)
	if full != rootClean && !strings.HasPrefix(full, rootClean+sep) {
		return "", fmt.Errorf("压缩包路径越界")
	}
	return full, nil
}
