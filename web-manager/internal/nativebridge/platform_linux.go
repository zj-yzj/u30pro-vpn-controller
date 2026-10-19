//go:build linux

package nativebridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const lockPath = NativeDir + "/bridge.lock"
const identityPath = NativeDir + "/bridge-process.json"

func noSymlinkPath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errDenied
	}
	current := "/"
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return errDenied
		}
		if i < len(parts)-1 && !info.IsDir() {
			return errDenied
		}
	}
	return nil
}
func privatePath(path string, mode os.FileMode, directory bool) error {
	if err := noSymlinkPath(path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode().Perm() != mode || info.IsDir() != directory {
		return errDenied
	}
	if !directory && !info.Mode().IsRegular() {
		return errDenied
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 || st.Gid != 0 {
		return errDenied
	}
	return nil
}
func checkPrivateLayout() error {
	for _, path := range []string{Base, NativeDir} {
		if err := privatePath(path, 0700, true); err != nil {
			return err
		}
	}
	if err := privatePath(BridgePath, 0700, false); err != nil {
		return err
	}
	return nil
}
func LoadTrust() (Trust, error) {
	if checkPrivateLayout() != nil || privatePath(TrustPath, 0600, false) != nil {
		return Trust{}, errDenied
	}
	f, err := os.OpenFile(TrustPath, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return Trust{}, errDenied
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil {
		return Trust{}, errDenied
	}
	return ParseTrust(data)
}
func HashAPK(path string) (string, error) {
	if !validAPKPath(path) || noSymlinkPath(path) != nil {
		return "", errDenied
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Size() < 1 || before.Size() > 64<<20 {
		return "", errDenied
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return "", errDenied
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return "", errDenied
	}
	h := sha256.New()
	if _, err = io.Copy(h, io.LimitReader(f, 64<<20+1)); err != nil {
		return "", errDenied
	}
	after, err := f.Stat()
	if err != nil || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return "", errDenied
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func peerUID(c *net.UnixConn) (uint32, error) {
	raw, err := c.SyscallConn()
	if err != nil {
		return 0, errDenied
	}
	var credential *syscall.Ucred
	var syscallErr error
	err = raw.Control(func(fd uintptr) {
		credential, syscallErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	if err != nil || syscallErr != nil || credential == nil {
		return 0, errDenied
	}
	return credential.Uid, nil
}
func writeResponse(c *net.UnixConn, data []byte) {
	c.SetWriteDeadline(time.Now().Add(2 * time.Second))
	c.Write(append(data, '\n'))
}
func handleConnection(s *Service, c *net.UnixConn) {
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	uid, err := peerUID(c)
	if err != nil {
		writeResponse(c, SafeError("unauthorized"))
		return
	}
	req, err := ReadRequest(c)
	if err != nil {
		writeResponse(c, SafeError("invalid_request"))
		return
	}
	if s.Authorize(ctx, uid) != nil {
		writeResponse(c, SafeError("unauthorized"))
		return
	}
	out, err := s.Forward(ctx, req)
	if err != nil {
		writeResponse(c, SafeError("unavailable"))
		return
	}
	writeResponse(c, out)
}

type processIdentity struct {
	Schema    int    `json:"schema"`
	PID       int    `json:"pid"`
	StartTime string `json:"startTime"`
	BootID    string `json:"bootId"`
	ExePath   string `json:"exePath"`
}

func procIdentity(pid int) (processIdentity, error) {
	if pid < 2 {
		return processIdentity{}, errUnavailable
	}
	root := "/proc/" + strconv.Itoa(pid)
	stat, err := os.ReadFile(root + "/stat")
	if err != nil {
		return processIdentity{}, errUnavailable
	}
	start, err := ParseProcStartTime(string(stat))
	if err != nil {
		return processIdentity{}, errUnavailable
	}
	exe, err := os.Readlink(root + "/exe")
	if err != nil || exe != BridgePath {
		return processIdentity{}, errUnavailable
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return processIdentity{}, errUnavailable
	}
	status, err := os.ReadFile(root + "/status")
	if err != nil {
		return processIdentity{}, errUnavailable
	}
	rootUser := false
	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, "Uid:") {
			rootUser = strings.Join(strings.Fields(line), " ") == "Uid: 0 0 0 0"
		}
	}
	if !rootUser {
		return processIdentity{}, errUnavailable
	}
	cmdline, err := os.ReadFile(root + "/cmdline")
	if err != nil || string(cmdline) != BridgePath+"\x00serve\x00" {
		return processIdentity{}, errUnavailable
	}
	return processIdentity{1, pid, start, strings.TrimSpace(string(boot)), exe}, nil
}
func loadIdentity() (processIdentity, error) {
	if privatePath(identityPath, 0600, false) != nil {
		return processIdentity{}, errUnavailable
	}
	data, err := os.ReadFile(identityPath)
	if err != nil || len(data) > 2048 {
		return processIdentity{}, errUnavailable
	}
	var id processIdentity
	if json.Unmarshal(data, &id) != nil || id.Schema != 1 || id.PID < 2 {
		return processIdentity{}, errUnavailable
	}
	actual, err := procIdentity(id.PID)
	if err != nil || actual != id {
		return processIdentity{}, errUnavailable
	}
	return id, nil
}
func writeIdentity(id processIdentity) error {
	data, _ := json.Marshal(id)
	// Only the locked daemon writes this task-owned atomic identity file.
	tmp, err := os.OpenFile(identityPath+".new", os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		if noSymlinkPath(identityPath+".new") == nil && privatePath(identityPath+".new", 0600, false) == nil {
			os.Remove(identityPath + ".new")
			tmp, err = os.OpenFile(identityPath+".new", os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
		}
	}
	if err != nil {
		return errUnavailable
	}
	ok := false
	defer func() {
		tmp.Close()
		if !ok {
			os.Remove(identityPath + ".new")
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return errUnavailable
	}
	if tmp.Sync() != nil {
		return errUnavailable
	}
	if tmp.Close() != nil {
		return errUnavailable
	}
	if err = os.Rename(identityPath+".new", identityPath); err != nil {
		return errUnavailable
	}
	ok = true
	dir, err := os.Open(NativeDir)
	if err == nil {
		dir.Sync()
		dir.Close()
	}
	return nil
}

func Start() error {
	if os.Geteuid() != 0 {
		return errDenied
	}
	if checkPrivateLayout() != nil {
		return errUnavailable
	}
	if _, err := loadIdentity(); err == nil {
		return nil
	}
	cmd := exec.Command(BridgePath, "serve")
	cmd.Env = []string{"PATH=/system/bin:/system/xbin", "HOME=/", "LANG=C", "LC_ALL=C"}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	devnull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		return errUnavailable
	}
	defer devnull.Close()
	cmd.Stdin = devnull
	cmd.Stdout = devnull
	cmd.Stderr = devnull
	if cmd.Start() != nil {
		return errUnavailable
	}
	cmd.Process.Release()
	deadline := time.Now().Add(750 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, err = loadIdentity(); err == nil {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return errUnavailable
}
func Serve() error {
	if os.Geteuid() != 0 || checkPrivateLayout() != nil {
		return errDenied
	}
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return errUnavailable
	}
	defer lock.Close()
	if privatePath(lockPath, 0600, false) != nil || syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return errUnavailable
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: "@" + SocketName, Net: "unix"})
	if err != nil {
		return errUnavailable
	}
	defer listener.Close()
	id, err := procIdentity(os.Getpid())
	if err != nil {
		return errUnavailable
	}
	if writeIdentity(id) != nil {
		return errUnavailable
	}
	defer func() {
		if current, err := loadIdentity(); err == nil && current == id {
			os.Remove(identityPath)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(stop)
	closing := make(chan struct{})
	defer close(closing)
	go func() {
		select {
		case <-stop:
			listener.Close()
		case <-closing:
		}
	}()
	sem := make(chan struct{}, 4)
	var workers sync.WaitGroup
	defer workers.Wait()
	service := NewService()
	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return errUnavailable
		}
		select {
		case sem <- struct{}{}:
			workers.Add(1)
			go func(c *net.UnixConn) { defer workers.Done(); defer func() { <-sem }(); handleConnection(service, c) }(conn)
		default:
			writeResponse(conn, SafeError("busy"))
			conn.Close()
		}
	}
}
func Stop() error {
	if os.Geteuid() != 0 || checkPrivateLayout() != nil {
		return errDenied
	}
	id, err := loadIdentity()
	if err != nil {
		if _, statErr := os.Lstat(identityPath); os.IsNotExist(statErr) {
			return nil
		}
		return errUnavailable
	}
	// Pin the process before the final identity check. pidfd signaling cannot hit
	// a different process if the recorded PID exits and is immediately reused.
	fd, _, errno := syscall.Syscall(434, uintptr(id.PID), 0, 0)
	if errno != 0 {
		return errUnavailable
	}
	defer syscall.Close(int(fd))
	current, err := procIdentity(id.PID)
	if err != nil || current != id {
		return errUnavailable
	}
	_, _, errno = syscall.Syscall6(424, fd, uintptr(syscall.SIGTERM), 0, 0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("%w", errUnavailable)
	}
	return nil
}
