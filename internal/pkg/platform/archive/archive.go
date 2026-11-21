package archive

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Format 压缩输出格式。
type Format string

const (
	FormatZIP    Format = "zip"
	FormatTarGZ  Format = "tar.gz"
	FormatTarXZ  Format = "tar.xz"
	FormatTar    Format = "tar"
)

/**
 * DetectFormat 根据文件名后缀识别压缩格式；未知返回空。
 */
func DetectFormat(name string) Format {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return FormatTarGZ
	case strings.HasSuffix(lower, ".tar.xz"), strings.HasSuffix(lower, ".txz"):
		return FormatTarXZ
	case strings.HasSuffix(lower, ".tar"):
		return FormatTar
	case strings.HasSuffix(lower, ".zip"):
		return FormatZIP
	default:
		return ""
	}
}

// IsArchive 是否为支持的压缩包。
func IsArchive(name string) bool {
	return DetectFormat(name) != ""
}

/**
 * DefaultExtractDir 解压目标目录：与压缩包同级，名为去掉后缀的基名。
 */
func DefaultExtractDir(archivePath string) string {
	base := filepath.Base(archivePath)
	lower := strings.ToLower(base)
	name := base
	for _, suf := range []string{".tar.gz", ".tar.xz", ".tgz", ".txz", ".tar", ".zip"} {
		if strings.HasSuffix(lower, suf) {
			name = base[:len(base)-len(suf)]
			break
		}
	}
	if name == "" || name == base {
		name = base + "_extracted"
	}
	return filepath.Join(filepath.Dir(archivePath), name)
}

/**
 * DefaultArchiveName 根据源路径与格式生成默认压缩包文件名。
 */
func DefaultArchiveName(srcPath string, format Format) (string, error) {
	base := filepath.Base(srcPath)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("无效的源路径")
	}
	switch format {
	case FormatZIP:
		return base + ".zip", nil
	case FormatTarGZ:
		return base + ".tar.gz", nil
	case FormatTarXZ:
		return base + ".tar.xz", nil
	case FormatTar:
		return base + ".tar", nil
	default:
		return "", fmt.Errorf("不支持的压缩格式")
	}
}
