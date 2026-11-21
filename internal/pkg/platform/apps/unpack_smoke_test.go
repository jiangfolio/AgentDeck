package apps

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestUnpackZipAndManifest(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "pkg")
	_ = os.MkdirAll(filepath.Join(pkg, "bin"), 0o755)
	_ = os.WriteFile(filepath.Join(pkg, "bin", "redis-server"), []byte("#!/bin/sh\n"), 0o755)
	_ = os.WriteFile(filepath.Join(pkg, "manifest.json"), []byte(`{"id":"redis","version":"7.2.5","bins":["bin/redis-server"]}`), 0o644)

	zipPath := filepath.Join(dir, "redis.zip")
	if err := writeTestZip(pkg, zipPath); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "dest")
	if err := UnpackArchive(zipPath, dest); err != nil {
		t.Fatal(err)
	}
	_ = FlattenIfSingleRoot(dest)
	m, err := ReadManifest(dest)
	if err != nil || m == nil || m.ID != "redis" {
		t.Fatalf("manifest: %+v %v", m, err)
	}
	if err := VerifyBins(dest, "bin/redis-server", m.Bins); err != nil {
		t.Fatal(err)
	}
}

func writeTestZip(srcDir, zipPath string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()
		_, err = io.Copy(w, r)
		return err
	})
}
