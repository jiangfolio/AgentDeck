package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompressAndExtractZip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "hello")
	_ = os.MkdirAll(src, 0o755)
	_ = os.WriteFile(filepath.Join(src, "a.txt"), []byte("hi"), 0o644)

	zipPath := filepath.Join(dir, "hello.zip")
	if err := Compress([]string{src}, zipPath, FormatZIP); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := Extract(zipPath, out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "hello", "a.txt"))
	if err != nil || string(data) != "hi" {
		t.Fatalf("content=%q err=%v", data, err)
	}
}

func TestCompressAndExtractTarGZ(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "pkg")
	_ = os.MkdirAll(filepath.Join(src, "bin"), 0o755)
	_ = os.WriteFile(filepath.Join(src, "bin", "x"), []byte("x"), 0o755)

	tgz := filepath.Join(dir, "pkg.tar.gz")
	if err := Compress([]string{src}, tgz, FormatTarGZ); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "extracted")
	if err := Extract(tgz, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "pkg", "bin", "x")); err != nil {
		t.Fatal(err)
	}
}
