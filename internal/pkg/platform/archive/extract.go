package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

/**
 * Extract 将压缩包解压到 destDir（不存在则创建；已存在且非空则拒绝，避免覆盖）。
 * 纯 Go 实现，跨平台支持 zip / tar / tar.gz / tar.xz。
 */
func Extract(archivePath, destDir string) error {
	// 1. 识别格式
	format := DetectFormat(archivePath)
	if format == "" {
		return fmt.Errorf("不支持的压缩格式，请使用 zip / tar / tar.gz / tar.xz")
	}
	st, err := os.Stat(archivePath)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return fmt.Errorf("路径是目录，无法解压")
	}
	// 2. 准备目标目录
	if destDir == "" {
		destDir = DefaultExtractDir(archivePath)
	}
	if err := prepareDestDir(destDir); err != nil {
		return err
	}
	// 3. 按格式解压
	switch format {
	case FormatZIP:
		return extractZip(archivePath, destDir)
	case FormatTarGZ:
		return extractTarGZ(archivePath, destDir)
	case FormatTarXZ:
		return extractTarXZ(archivePath, destDir)
	case FormatTar:
		return extractTar(archivePath, destDir)
	default:
		return fmt.Errorf("不支持的压缩格式")
	}
}

func prepareDestDir(destDir string) error {
	st, err := os.Stat(destDir)
	if err == nil {
		if !st.IsDir() {
			return fmt.Errorf("目标已存在且不是目录")
		}
		entries, err := os.ReadDir(destDir)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("目标目录非空: %s", destDir)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(destDir, 0o755)
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if err := writeZipEntry(destDir, f); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(destDir string, f *zip.File) error {
	name := strings.ReplaceAll(f.Name, "\\", "/")
	target, err := SafeJoinUnder(destDir, name)
	if err != nil {
		return err
	}
	if f.FileInfo().IsDir() || strings.HasSuffix(name, "/") {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func extractTarGZ(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	return readTar(gz, destDir)
}

func extractTarXZ(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	xzr, err := xz.NewReader(f)
	if err != nil {
		return err
	}
	return readTar(xzr, destDir)
}

func extractTar(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return readTar(f, destDir)
}

func readTar(r io.Reader, destDir string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := strings.ReplaceAll(hdr.Name, "\\", "/")
		target, err := SafeJoinUnder(destDir, name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode)
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			_ = out.Close()
		default:
			// 跳过链接等，保持跨平台可移植
		}
	}
}
