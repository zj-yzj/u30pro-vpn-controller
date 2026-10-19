package installer

import (
	"os"
	"path/filepath"
	"runtime"
)

// Filesystem is an injectable filesystem boundary. Production always uses
// OSFiles, whose absolute virtual paths are the actual fixed device paths.
type Filesystem interface {
	Local(string) string
	Safe(string, os.FileMode, bool) bool
	Atomic(string, []byte, os.FileMode) error
	Publish(string, string) error
}
type OSFiles struct{}

func (OSFiles) Local(p string) string { return p }
func (OSFiles) Safe(p string, mode os.FileMode, dir bool) bool {
	info, err := os.Lstat(p)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || info.IsDir() != dir || (!dir && !info.Mode().IsRegular()) || !rootOwned(info) {
		return false
	}
	if mode == 0 {
		return info.Mode().Perm()&0022 == 0
	}
	return info.Mode().Perm() == mode
}
func (OSFiles) Atomic(p string, b []byte, mode os.FileMode) error { return writeAtomic(p, b, mode) }
func (OSFiles) Publish(src, dst string) error                     { return noReplace(src, dst) }
func syncDirectory(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	f, e := os.Open(dir)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
func writeAtomic(p string, b []byte, mode os.FileMode) error {
	f, e := os.CreateTemp(filepath.Dir(p), ".u30setup-")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()
	if e = f.Chmod(mode); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	if e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Rename(name, p); e != nil {
		return e
	}
	return syncDirectory(filepath.Dir(p))
}
