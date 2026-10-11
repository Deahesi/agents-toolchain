package files_test

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/files"
)

func openRoot(t *testing.T, dir string) *files.Root {
	t.Helper()
	root, err := files.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	return root
}

func TestRootFiles(t *testing.T) {
	dir := t.TempDir()
	root := openRoot(t, dir)
	if err := root.MkdirAll("nested/deeper", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteNewFile("nested/note.txt", []byte("long long"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteNewFile("nested/note.txt", []byte("overwrite"), 0o600); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("exclusive creation = %v; want ErrExist", err)
	}
	if err := root.Replace("nested/note.txt", "long", "x", 1); err != nil {
		t.Fatal(err)
	}
	content, err := root.ReadFile("nested/note.txt")
	if err != nil || string(content) != "x long" {
		t.Fatalf("replacement = %q, %v; want %q", content, err, "x long")
	}
	if err := root.Replace("nested/note.txt", "long", "", -1); err != nil {
		t.Fatal(err)
	}
	content, err = root.ReadFile("nested/note.txt")
	if err != nil || string(content) != "x " {
		t.Fatalf("shorter replacement = %q, %v", content, err)
	}
	info, err := root.Stat("nested/note.txt")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("permissions changed: %v", info.Mode())
	}
	if err := root.Replace("missing.txt", "a", "b", -1); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("edit missing file = %v; want ErrNotExist", err)
	}
	if _, err := root.Stat("missing.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("edit created missing file: %v", err)
	}
	if err := root.WriteFile("nested/another.txt", []byte("created"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("nested/another.txt", []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	content, err = root.ReadFile("nested/another.txt")
	if err != nil || string(content) != "new" {
		t.Fatalf("overwrite = %q, %v", content, err)
	}
	entries, err := root.ReadDir("nested")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if !slices.Equal(names, []string{"another.txt", "deeper", "note.txt"}) {
		t.Errorf("entries = %v", names)
	}
	matches, err := root.Search("nested", "*.txt")
	want := []string{filepath.Join("nested", "another.txt"), filepath.Join("nested", "note.txt")}
	if err != nil || !slices.Equal(matches, want) {
		t.Errorf("matches = %v, %v; want %v", matches, err, want)
	}
	for _, pattern := range []string{"[", "../*", "deeper/*", `deeper\*`, "bad\x00*"} {
		if _, err := root.Search("nested/deeper", pattern); !errors.Is(err, filepath.ErrBadPattern) {
			t.Errorf("pattern %q = %v; want ErrBadPattern", pattern, err)
		}
	}
	path, err := root.Path("nested/note.txt")
	if err != nil || path != filepath.Join(dir, "nested", "note.txt") {
		t.Errorf("display path = %q, %v", path, err)
	}
	if err := root.Remove("nested/note.txt"); err != nil {
		t.Fatal(err)
	}
	if err := root.Remove("."); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("remove root = %v; want ErrPermission", err)
	}
}

// Exercise each operation against the same boundary, including destructive ones.
func operations(root *files.Root) map[string]func(string) error {
	return map[string]func(string) error{
		"read":        func(p string) error { _, err := root.ReadFile(p); return err },
		"stat":        func(p string) error { _, err := root.Stat(p); return err },
		"list":        func(p string) error { _, err := root.ReadDir(p); return err },
		"search":      func(p string) error { _, err := root.Search(p, "*"); return err },
		"path":        func(p string) error { _, err := root.Path(p); return err },
		"write":       func(p string) error { return root.WriteFile(p, []byte("modified"), 0o600) },
		"replacefile": func(p string) error { return root.ReplaceFile(p, []byte("modified")) },
		"create":      func(p string) error { return root.WriteNewFile(p, []byte("modified"), 0o600) },
		"edit":        func(p string) error { return root.Replace(p, "secret", "modified", -1) },
		"mkdir":       func(p string) error { return root.Mkdir(p, 0o755) },
		"mkdirall":    func(p string) error { return root.MkdirAll(p, 0o755) },
		"remove":      root.Remove,
		"openroot": func(p string) error {
			child, err := root.OpenRoot(p)
			if err == nil {
				child.Close()
			}
			return err
		},
	}
}

func TestRootRejectsNonLocalPaths(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "root")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(parent, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := openRoot(t, dir)
	paths := []string{"", "..", "../secret.txt", "sub/../../secret.txt", "sub/../file", "bad\x00path", secret, dir}
	if runtime.GOOS == "windows" {
		paths = append(paths, `..\secret.txt`, `sub\..\file`, `C:secret.txt`, `\secret.txt`, `\\server\share\secret`, `\\?\C:\secret`, `NUL`, `sub\COM1`, `file:stream`)
	}
	for name, operation := range operations(root) {
		t.Run(name, func(t *testing.T) {
			for _, path := range paths {
				if err := operation(path); !errors.Is(err, fs.ErrPermission) {
					t.Errorf("%q = %v; want ErrPermission", path, err)
				}
			}
		})
	}
	content, err := os.ReadFile(secret)
	if err != nil || string(content) != "secret" {
		t.Fatalf("outside file changed: %q, %v", content, err)
	}
}

func TestRootRejectsDirectoryLinkEscape(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "root")
	outside := filepath.Join(parent, "outside")
	for _, path := range []string{dir, outside} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "escape")
	if runtime.GOOS == "windows" {
		// Junctions do not require the symlink privilege on Windows.
		output, err := exec.Command("cmd.exe", "/d", "/c", "mklink", "/j", link, outside).CombinedOutput()
		if err != nil {
			t.Fatalf("create junction: %v: %s", err, output)
		}
	} else if err := os.Symlink("../outside", link); err != nil {
		t.Fatal(err)
	}
	root := openRoot(t, dir)
	for name, operation := range operations(root) {
		if name == "path" {
			continue // Informational only, intentionally does not resolve links.
		}
		t.Run(name, func(t *testing.T) {
			path := "escape/secret.txt"
			if name == "list" || name == "search" || name == "openroot" {
				path = "escape"
			}
			if err := operation(path); err == nil {
				t.Fatalf("operation escaped the root through %q", path)
			}
		})
	}
	if err := root.WriteFile("escape/new.txt", []byte("new"), 0o600); err == nil {
		t.Fatal("created file outside root")
	}
	if err := root.MkdirAll("escape/new/sub", 0o755); err == nil {
		t.Fatal("created directory outside root")
	}
	content, err := os.ReadFile(secret)
	if err != nil || string(content) != "secret" {
		t.Fatalf("outside file changed: %q, %v", content, err)
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 1 {
		t.Fatalf("outside directory changed: %v, %v", entries, err)
	}
}

func TestChildRootAndRelativeSymlinks(t *testing.T) {
	dir := t.TempDir()
	root := openRoot(t, dir)
	if err := root.Mkdir("child", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("secret.txt", []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "child", "link.txt")
	if err := os.Symlink("../secret.txt", link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
	content, err := root.ReadFile("child/link.txt")
	if err != nil || string(content) != "secret" {
		t.Fatalf("relative link inside parent root = %q, %v", content, err)
	}
	child, err := root.OpenRoot("child")
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	if _, err := child.ReadFile("link.txt"); err == nil {
		t.Fatal("child accessed a file in its parent through a relative symlink")
	}
	if err := child.Replace("link.txt", "secret", "changed", -1); err == nil {
		t.Fatal("child edited a file in its parent through a relative symlink")
	}
}
