//go:build linux

package installer

import (
	"os"
	"syscall"
	"unsafe"
)

func rootOwned(s os.FileInfo) bool {
	v, ok := s.Sys().(*syscall.Stat_t)
	return ok && v.Uid == 0 && v.Gid == 0
}
func (SystemHost) FreeBytes(p string) (uint64, error) {
	var s syscall.Statfs_t
	e := syscall.Statfs(p, &s)
	return s.Bavail * uint64(s.Bsize), e
}
func (SystemHost) Lock(p string) (func(), error) {
	fd, e := syscall.Open(p, syscall.O_RDWR|syscall.O_CREAT|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), p)
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || !rootOwned(s) || s.Mode().Perm() != 0600 {
		f.Close()
		return nil, problem("unsafe_storage")
	}
	if e = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, problem("busy")
	}
	return func() { syscall.Flock(fd, syscall.LOCK_UN); f.Close() }, nil
}
func noReplace(src, dst string) error {
	a, e := syscall.BytePtrFromString(src)
	if e != nil {
		return e
	}
	b, e := syscall.BytePtrFromString(dst)
	if e != nil {
		return e
	}
	const atFDCWD = ^uintptr(99)
	_, _, errno := syscall.Syscall6(syscall.SYS_RENAMEAT2, atFDCWD, uintptr(unsafe.Pointer(a)), atFDCWD, uintptr(unsafe.Pointer(b)), 1, 0)
	if errno != 0 {
		return errno
	}
	return syncDirectory("/data/adb")
}
