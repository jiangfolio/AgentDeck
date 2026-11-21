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
)

/**
 * Compress 将 paths 打成压缩包写入 destFile；纯 Go，跨平台。
 * format 支持 zip / tar.gz；paths 至少一个，均为已存在的绝对路径。
 */
func Compress(paths []string, destFile string, format Format) error {
	if len(paths) == 0 {
		return fmt.Errorf("请选择要压缩的文件或目录")
	}
	if strings.TrimSpace(destFile) == "" {
		return fmt.Errorf("请指定输出路径")
	}
	if _, err := os.Stat(destFile); err == nil {
		return fmt.Errorf("目标已存在: %s", filepath.Base(destFile))
	}
	if err := os.MkdirAll(filepath.Dir(destFile), 0o755); err != nil {
		return err
	}
	switch format {
	case FormatZIP:
		return compressZip(paths, destFile)
	case FormatTarGZ:
		return compressTarGZ(paths, destFile)
	default:
		return fmt.Errorf("压缩仅支持 zip / tar.gz")
	}
}

func compressZip(paths []string, destFile string) error {
	out, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()
	for _, p := range paths {
		if err := addPathToZip(zw, p); err != nil {
			_ = os.Remove(destFile)
			return err
		}
	}
	return nil
}

func addPathToZip(zw *zip.Writer, root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	base := filepath.Base(root)
	if !info.IsDir() {
		return writeZipFile(zw, root, base)
	}
	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join(base, rel))
		if fi.IsDir() {
			if !strings.HasSuffix(name, "/") {
				name += "/"
			}
			_, err := zw.Create(name)
			return err
		}
		return writeZipFile(zw, path, name)
	})
}

func writeZipFile(zw *zip.Writer, path, name string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(st)
	if err != nil {
		return err
	}
	hdr.Name = name
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

func compressTarGZ(paths []string, destFile string) error {
	out, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	for _, p := range paths {
		if err := addPathToTar(tw, p); err != nil {
			_ = os.Remove(destFile)
			return err
		}
	}
	return nil
}

func addPathToTar(tw *tar.Writer, root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	base := filepath.Base(root)
	if !info.IsDir() {
		return writeTarFile(tw, root, base, info)
	}
	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join(base, rel))
		if fi.IsDir() {
			hdr, err := tar.FileInfoHeader(fi, "")
			if err != nil {
				return err
			}
			hdr.Name = name + "/"
			return tw.WriteHeader(hdr)
		}
		return writeTarFile(tw, path, name, fi)
	})
}

func writeTarFile(tw *tar.Writer, path, name string, fi os.FileInfo) error {
	hdr, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	hdr.Name = name
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}
