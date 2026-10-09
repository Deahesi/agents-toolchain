// Package files provides filesystem operations confined to an explicitly opened
// directory. Callers choose a trusted root; all subsequent paths are relative to it.
package files

import (
	"cmp"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Root owns a directory handle. Close it after all operations have finished.
// Operations use os.Root so symlinks and concurrent path changes cannot escape
// the directory. This is a filesystem boundary, not a sandbox for shell commands,
// hard links, mount points or special files.
type Root struct {
	dir *os.Root
}

// OpenRoot opens the trusted base directory, which must already exist.
// The base may be absolute or relative to the process working directory.
func OpenRoot(dir string) (*Root, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, &fs.PathError{Op: "openroot", Path: dir, Err: fs.ErrInvalid}
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, err
	}
	return &Root{dir: root}, nil
}

func (r *Root) Close() error {
	return r.dir.Close()
}

// OpenRoot opens a narrower boundary inside r. The returned root has its own
// handle and must be closed separately; it cannot access siblings in r.
func (r *Root) OpenRoot(name string) (*Root, error) {
	if err := checkPath("openroot", name); err != nil {
		return nil, err
	}
	dir, err := r.dir.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	return &Root{dir: dir}, nil
}

// Path returns an absolute display path. It does not resolve symlinks and must
// not be used for filesystem access; use Root methods to retain confinement.
func (r *Root) Path(name string) (string, error) {
	if err := checkPath("path", name); err != nil {
		return "", err
	}
	return filepath.Join(r.dir.Name(), name), nil
}

func (r *Root) Stat(name string) (fs.FileInfo, error) {
	if err := checkPath("stat", name); err != nil {
		return nil, err
	}
	return r.dir.Stat(name)
}

func (r *Root) ReadFile(name string) ([]byte, error) {
	if err := checkPath("read", name); err != nil {
		return nil, err
	}
	return r.dir.ReadFile(name)
}

// ReadDir returns entries sorted by name, without following entry symlinks.
func (r *Root) ReadDir(name string) ([]fs.DirEntry, error) {
	if err := checkPath("readdir", name); err != nil {
		return nil, err
	}
	dir, err := r.dir.Open(name)
	if err != nil {
		return nil, err
	}
	entries, readErr := dir.ReadDir(-1)
	if err := errors.Join(readErr, dir.Close()); err != nil {
		return nil, err
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return entries, nil
}

// WriteFile creates or truncates a file. Existing permissions are preserved.
func (r *Root) WriteFile(name string, content []byte, perm fs.FileMode) error {
	if err := checkPath("write", name); err != nil {
		return err
	}
	return r.dir.WriteFile(name, content, perm)
}

// WriteNewFile creates a file exclusively, without overwriting existing files.
func (r *Root) WriteNewFile(name string, content []byte, perm fs.FileMode) error {
	if err := checkPath("create", name); err != nil {
		return err
	}
	file, err := r.dir.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(content)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return errors.Join(err, r.dir.Remove(name))
	}
	return nil
}

// ReplaceFile stages a complete replacement beside an existing regular file,
// preserves its permission bits, and renames it into place. Final symlinks are
// rejected. A failed write leaves the original intact. Rename is atomic on Unix;
// other platforms have the guarantees described by os.Rename. Concurrent edits
// require caller coordination; ownership and extended attributes are not copied.
func (r *Root) ReplaceFile(name string, content []byte) (err error) {
	if err := checkPath("replacefile", name); err != nil {
		return err
	}
	dir, err := r.OpenRoot(filepath.Dir(name))
	if err != nil {
		return err
	}
	defer dir.Close()
	base := filepath.Base(name)
	info, err := dir.dir.Lstat(base)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return &fs.PathError{Op: "replacefile", Path: name, Err: fmt.Errorf("regular file required: %w", fs.ErrInvalid)}
	}
	temporary := ".atc-" + rand.Text() + ".tmp"
	file, err := dir.dir.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	renamed := false
	defer func() {
		if !renamed {
			err = errors.Join(err, dir.dir.Remove(temporary))
		}
	}()
	_, err = file.Write(content)
	if err == nil {
		err = file.Chmod(info.Mode().Perm())
	}
	if err == nil {
		err = file.Sync()
	}
	if err = errors.Join(err, file.Close()); err != nil {
		return err
	}
	err = dir.dir.Rename(temporary, base)
	renamed = err == nil
	return err
}

func (r *Root) Mkdir(name string, perm fs.FileMode) error {
	if err := checkPath("mkdir", name); err != nil {
		return err
	}
	return r.dir.Mkdir(name, perm)
}

func (r *Root) MkdirAll(name string, perm fs.FileMode) error {
	if err := checkPath("mkdirall", name); err != nil {
		return err
	}
	return r.dir.MkdirAll(name, perm)
}

// Remove removes a file or an empty directory, but never the root itself.
func (r *Root) Remove(name string) error {
	if err := checkPath("remove", name); err != nil {
		return err
	}
	if filepath.Clean(name) == "." {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrPermission}
	}
	return r.dir.Remove(name)
}

// Search matches entry names in a single directory using filepath.Match syntax.
// Patterns cannot contain directories. Results are sorted, root-relative paths;
// directory and symlink entries may be returned, but are never traversed.
func (r *Root) Search(dir, pattern string) ([]string, error) {
	if strings.ContainsAny(pattern, "/\\\x00") {
		return nil, fmt.Errorf("search pattern must match entry names only: %w", filepath.ErrBadPattern)
	}
	if _, err := filepath.Match(pattern, ""); err != nil {
		return nil, err
	}
	entries, err := r.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, entry := range entries {
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, filepath.Join(dir, entry.Name()))
		}
	}
	return matches, nil
}

// Replace edits an existing file through one handle, preserving its permissions.
// A negative count replaces all occurrences, as in strings.Replace. The update
// is not atomic; concurrent edits to the same file need caller coordination.
func (r *Root) Replace(name, old, replacement string, count int) (err error) {
	if err := checkPath("replace", name); err != nil {
		return err
	}
	file, err := r.dir.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	updated := strings.Replace(string(content), old, replacement, count)
	if updated == string(content) {
		return nil
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := io.WriteString(file, updated); err != nil {
		return err
	}
	return file.Truncate(int64(len(updated)))
}

// Reject traversal before cleaning: even "sub/../file" is not accepted.
// os.Root supplies the separate, race-resistant symlink confinement check.
func checkPath(op, name string) error {
	invalid := !filepath.IsLocal(name) || strings.ContainsRune(name, 0)
	if filepath.Separator == '\\' && strings.Contains(name, ":") {
		invalid = true // Reject Windows alternate data streams as well as drives.
	}
	for _, component := range strings.FieldsFunc(name, func(c rune) bool {
		return c == '/' || c == rune(filepath.Separator)
	}) {
		if component == ".." {
			invalid = true
		}
	}
	if invalid {
		return &fs.PathError{Op: op, Path: name, Err: fmt.Errorf("relative path without '..' required: %w", fs.ErrPermission)}
	}
	return nil
}
