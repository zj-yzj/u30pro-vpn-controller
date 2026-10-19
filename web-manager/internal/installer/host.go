package installer

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type Host interface {
	Run(context.Context, string, ...string) ([]byte, error)
	InstallAPK(context.Context, string, int64) ([]byte, error)
	FreeBytes(string) (uint64, error)
	Lock(string) (func(), error)
	Sleep(context.Context, time.Duration) error
}
type SystemHost struct{}
type boundedWriter struct {
	b     []byte
	limit int
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	if len(p) > w.limit-len(w.b) {
		return 0, problem("command_failed")
	}
	w.b = append(w.b, p...)
	return len(p), nil
}
func (SystemHost) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "magisk" {
		name = ""
		for _, candidate := range []string{"/debug_ramdisk/magisk", "/sbin/magisk", "/system/bin/magisk", "/data/adb/magisk/magisk"} {
			if s, e := os.Lstat(candidate); e == nil && s.Mode().IsRegular() && s.Mode()&os.ModeSymlink == 0 && rootOwned(s) && s.Mode().Perm()&0022 == 0 {
				name = candidate
				break
			}
		}
		if name == "" {
			return nil, problem("magisk_unavailable")
		}
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = []string{"PATH=/system/bin:/system/xbin:/sbin:/debug_ramdisk:/data/adb/magisk", "HOME=/", "LANG=C", "LC_ALL=C"}
	cmd.WaitDelay = time.Second
	out := &boundedWriter{limit: 1 << 20}
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	if e := cmd.Run(); e != nil {
		return nil, e
	}
	return out.b, nil
}

// Stream from the already private, checked payload. Package Manager runs in a
// different SELinux/UID context and cannot reliably open /data/adb paths itself.
func (SystemHost) InstallAPK(ctx context.Context, p string, size int64) ([]byte, error) {
	s, err := os.Lstat(p)
	if err != nil || !s.Mode().IsRegular() || !rootOwned(s) || s.Size() != size {
		return nil, problem("payload_mismatch")
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cmd := exec.CommandContext(ctx, "/system/bin/pm", "install", "-S", strconv.FormatInt(size, 10))
	cmd.Env = []string{"PATH=/system/bin:/system/xbin", "HOME=/", "LANG=C", "LC_ALL=C"}
	cmd.Stdin = io.LimitReader(f, size)
	cmd.Stderr = io.Discard
	out := &boundedWriter{limit: 1 << 20}
	cmd.Stdout = out
	cmd.WaitDelay = time.Second
	err = cmd.Run()
	return out.b, err
}
func (SystemHost) Sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
