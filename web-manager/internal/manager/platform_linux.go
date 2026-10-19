//go:build linux || android

package manager

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func ownedRoot(s os.FileInfo) bool        { v, ok := s.Sys().(*syscall.Stat_t); return ok && v.Uid == 0 }
func directorySyncUnsupported(error) bool { return false }
func acquireLock(path string, wait bool) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fail("storage_failed")
	}
	flags := syscall.LOCK_EX
	if !wait {
		flags |= syscall.LOCK_NB
	}
	if err = syscall.Flock(int(f.Fd()), flags); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fail("busy")
		}
		return nil, fail("storage_failed")
	}
	return func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
}
func detach(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
