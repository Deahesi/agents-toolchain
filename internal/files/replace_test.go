package files_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReplaceFile(t *testing.T) {
	dir := t.TempDir()
	root := openRoot(t, dir)
	if err := root.Mkdir("nested", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteNewFile("nested/config.yml", []byte("old content"), 0o640); err != nil {
		t.Fatal(err)
	}
	before, err := root.Stat("nested/config.yml")
	if err != nil {
		t.Fatal(err)
	}
	if err := root.ReplaceFile("nested/config.yml", []byte("new")); err != nil {
		t.Fatal(err)
	}
	content, err := root.ReadFile("nested/config.yml")
	if err != nil || string(content) != "new" {
		t.Fatalf("replacement = %q, %v", content, err)
	}
	after, err := root.Stat("nested/config.yml")
	if err != nil || after.Mode().Perm() != before.Mode().Perm() {
		t.Fatalf("permissions were not preserved: %v", err)
	}
	entries, err := root.ReadDir("nested")
	if err != nil || len(entries) != 1 || entries[0].Name() != "config.yml" {
		t.Fatalf("temporary file was not removed: %v, %v", entries, err)
	}
	if err := root.ReplaceFile("nested/missing.yml", []byte("new")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("replacement must not create missing files: %v", err)
	}
	if err := root.ReplaceFile("nested", []byte("new")); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("replacement must not overwrite directories: %v", err)
	}
}

func TestReplaceFileRejectsFinalSymlink(t *testing.T) {
	dir := t.TempDir()
	root := openRoot(t, dir)
	if err := root.WriteNewFile("target", []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink("target", link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if err := root.ReplaceFile("link", []byte("new")); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("replacement must reject final symlinks: %v", err)
	}
	content, err := root.ReadFile("target")
	if err != nil || string(content) != "original" {
		t.Fatalf("symlink target was modified: %q, %v", content, err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("symlink was replaced: %v", err)
	}
}

func TestReplaceFileDoesNotModifyHardLinkTarget(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "root")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside")
	if err := os.WriteFile(outside, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(outside, filepath.Join(dir, "config")); err != nil {
		t.Fatal(err)
	}
	root := openRoot(t, dir)
	if err := root.ReplaceFile("config", []byte("new")); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(outside)
	if err != nil || string(content) != "original" {
		t.Fatalf("hard link target outside root was modified: %q, %v", content, err)
	}
}
