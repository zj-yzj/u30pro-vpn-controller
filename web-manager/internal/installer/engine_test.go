package installer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"u30pro/webmanager/internal/manager"
)

type sandboxFiles struct {
	root          string
	failPhase     string
	unsafePath    string
	modeOverrides map[string]os.FileMode
}

func (f *sandboxFiles) Local(p string) string {
	return filepath.Join(f.root, filepath.FromSlash(strings.TrimPrefix(p, "/")))
}
func (f *sandboxFiles) Safe(p string, mode os.FileMode, dir bool) bool {
	if actual, ok := f.modeOverrides[p]; ok && mode != 0 && mode != actual {
		return false
	}
	if p == f.unsafePath {
		return false
	}
	s, e := os.Lstat(f.Local(p))
	return e == nil && s.Mode()&os.ModeSymlink == 0 && s.IsDir() == dir && (dir || s.Mode().IsRegular())
}
func TestInterruptedInstallDoesNotWriteIntoUnverifiedStage(t *testing.T) {
	e, _, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	fs.failPhase = "RUNTIME_READY"
	if _, err := e.Boot(context.Background(), module); err == nil {
		t.Fatal("missing interruption")
	}
	os.Rename(fs.Local(stageRoot), fs.Local(stageRoot+"-preserved"))
	os.Mkdir(fs.Local(stageRoot), 0700)
	fs.unsafePath = stageRoot
	if _, err := e.Boot(context.Background(), module); err == nil {
		t.Fatal("unverified stage accepted")
	}
	if _, err := os.Stat(fs.Local(stageRoot + "/installation.json")); !os.IsNotExist(err) {
		t.Fatal("wrote to unverified staging directory")
	}
}
func (f *sandboxFiles) Atomic(p string, b []byte, mode os.FileMode) error {
	if p == journalPath && f.failPhase != "" && strings.Contains(string(b), `"phase":"`+f.failPhase+`"`) {
		f.failPhase = ""
		return errors.New("simulated power loss")
	}
	return writeAtomic(f.Local(p), b, mode)
}
func (f *sandboxFiles) Publish(src, dst string) error {
	if _, e := os.Lstat(f.Local(dst)); !os.IsNotExist(e) {
		return errors.New("exists")
	}
	return os.Rename(f.Local(src), f.Local(dst))
}

type fakeHost struct {
	props               map[string]string
	installed           bool
	unknownAPK          bool
	calls               []string
	space               uint64
	fs                  *sandboxFiles
	apk                 []byte
	uid                 string
	ready               bool
	streamedInstalls    int
	forbidDirectInstall bool
	locks               map[string]bool
	onBridgeStop        func()
	networkActive       bool
	bridgeLingers       bool
	snapshotFailure     string
	collision           string
}

func (h *fakeHost) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	h.calls = append(h.calls, key)
	switch name {
	case "/system/bin/id":
		return []byte("0\n"), nil
	case "/system/bin/getprop":
		return []byte(h.props[args[0]] + "\n"), nil
	case "/system/bin/getenforce":
		return []byte("Permissive\n"), nil
	case "magisk":
		return []byte("28103\n"), nil
	case "/system/bin/cmd":
		if len(args) > 2 && args[0] == "package" && args[1] == "list" && args[2] == "packages" {
			if !h.ready {
				return nil, errors.New("pm not ready")
			}
			pkg := args[len(args)-1]
			if pkg == UfiPackage {
				return []byte("package:" + UfiPackage + "\n"), nil
			}
			if h.installed {
				return []byte("package:" + PackageName + " uid:" + h.uid + "\n"), nil
			}
			return nil, nil
		}
	case "/system/bin/pm":
		if !h.ready {
			return nil, errors.New("pm not ready")
		}
		switch args[0] {
		case "path":
			if !h.installed {
				return nil, nil
			}
			return []byte("package:/data/app/fixture/" + PackageName + "-fixture/base.apk\n"), nil
		case "install":
			if h.forbidDirectInstall {
				return nil, errors.New("private APK path inaccessible to Package Manager")
			}
			h.installed = true
			dir := h.fs.Local("/data/app/fixture/" + PackageName + "-fixture")
			os.MkdirAll(dir, 0700)
			b := h.apk
			if h.unknownAPK {
				b = []byte("unknown apk")
			}
			os.WriteFile(filepath.Join(dir, "base.apk"), b, 0600)
			return []byte("Success\n"), nil
		case "uninstall":
			h.installed = false
			return []byte("Success\n"), nil
		}
	case "/system/bin/ip":
		joined := strings.Join(args, " ")
		if joined == h.snapshotFailure {
			return nil, errors.New("snapshot unavailable")
		}
		switch joined {
		case "-4 rule show":
			return []byte("0: from all lookup local\n9999: from all lookup main\n"), nil
		case "-4 addr show dev br0":
			return []byte("inet 192.168.0.1/24 scope global br0\n"), nil
		case "-4 route show table all":
			if h.collision == "route" {
				return []byte("default via 10.0.0.1 table 20131\n"), nil
			}
			return []byte("192.168.0.0/24 dev br0 table main\n"), nil
		case "link show":
			if h.networkActive || h.collision == "tun" {
				return []byte("22: U30VPN: <UP>\n"), nil
			}
			return []byte("1: lo: <UP>\n2: br0: <UP>\n"), nil
		case "link show dev U30VPN":
			if h.networkActive {
				return []byte("U30VPN"), nil
			}
			return nil, errors.New("absent")
		}
		return nil, nil
	case "/system/bin/iptables", "/system/bin/ip6tables":
		if strings.Join(args, " ") == h.snapshotFailure {
			return nil, errors.New("snapshot unavailable")
		}
		if strings.Join(args, " ") == "-w 1 -t filter -S" {
			s := "-P FORWARD ACCEPT\n-P INPUT ACCEPT\n"
			if h.collision == "chain" && strings.HasSuffix(name, "/iptables") {
				s += "-N U30VPN4FORWARD\n"
			}
			return []byte(s), nil
		}
		if strings.Join(args, " ") == "-w 1 -t nat -S" {
			return []byte("-P PREROUTING ACCEPT\n"), nil
		}
		if strings.Contains(key, "U30VPN") {
			return nil, errors.New("absent")
		}
		return []byte("-P FORWARD ACCEPT\n"), nil
	case "/system/bin/ss":
		if strings.Join(args, " ") == "-H -xl" && h.bridgeLingers {
			return []byte("u_str LISTEN 0 128 @codex_u30vpn_v1 1234\n"), nil
		}
		return []byte("State Recv-Q Send-Q Local Address:Port Peer Address:Port\n"), nil
	case "/system/bin/sh":
		return []byte("STOPPED\n"), nil
	}
	if strings.HasSuffix(name, "/webbridge") && len(args) > 0 && args[0] == "stop" && h.onBridgeStop != nil {
		h.onBridgeStop()
	}
	if strings.HasSuffix(name, "/webbridge") || strings.HasSuffix(name, "/webctl") {
		return []byte("{}\n"), nil
	}
	return nil, errors.New("unexpected command " + key)
}
func (h *fakeHost) InstallAPK(ctx context.Context, p string, size int64) ([]byte, error) {
	h.streamedInstalls++
	old := h.forbidDirectInstall
	h.forbidDirectInstall = false
	defer func() { h.forbidDirectInstall = old }()
	return h.Run(ctx, "/system/bin/pm", "install", "-S", "123")
}
func (h *fakeHost) FreeBytes(string) (uint64, error) { return h.space, nil }
func (h *fakeHost) Lock(p string) (func(), error) {
	if h.locks == nil {
		h.locks = map[string]bool{}
	}
	if h.locks[p] {
		return nil, errors.New("busy")
	}
	h.locks[p] = true
	return func() { h.locks[p] = false }, nil
}
func (h *fakeHost) Sleep(ctx context.Context, d time.Duration) error { return ctx.Err() }
func fixtureEngine(t *testing.T) (*Engine, *fakeHost, *sandboxFiles, string) {
	t.Helper()
	fs := &sandboxFiles{root: t.TempDir()}
	for _, p := range []string{"/data/adb/modules/" + ModuleID, "/data/app", "/proc/net/sfp", "/proc/sys/net/ipv4", "/sys/class/net/br0", "/dev/net"} {
		os.MkdirAll(fs.Local(p), 0700)
	}
	os.WriteFile(fs.Local("/proc/net/sfp/enable"), []byte("1\n"), 0600)
	os.WriteFile(fs.Local("/proc/sys/net/ipv4/ip_forward"), []byte("1\n"), 0600)
	os.WriteFile(fs.Local("/dev/net/tun"), []byte("tun"), 0600)
	module := "/data/adb/modules/" + ModuleID
	source, m := fixtureModule(t)
	for _, r := range m.Files {
		b, _ := os.ReadFile(filepath.Join(source, r.Path))
		dest := fs.Local(module + "/" + r.Path)
		os.MkdirAll(filepath.Dir(dest), 0700)
		os.WriteFile(dest, b, os.FileMode(r.Mode))
	}
	b, _ := os.ReadFile(filepath.Join(source, "manifest.json"))
	os.WriteFile(fs.Local(module+"/manifest.json"), b, 0600)
	apk, _ := os.ReadFile(fs.Local(module + "/payload/U30ProVPN.apk"))
	host := &fakeHost{fs: fs, props: map[string]string{"ro.product.model": "MU5358", "ro.product.device": "MU5358", "ro.product.cpu.abi": "arm64-v8a", "ro.build.version.sdk": "35", "ro.build.display.id": "U30ProV1.0.0B23", "sys.boot_completed": "1"}, space: 1 << 30, apk: apk, uid: "10123", ready: true}
	e := NewWith(host, fs)
	e.bootstrap = func(base string) error { return manager.NewForTest(base).Bootstrap() }
	return e, host, fs, module
}
func TestCheckRejectsWrongProfileWithoutGlobalWrites(t *testing.T) {
	for _, key := range []string{"ro.product.model", "ro.product.device", "ro.product.cpu.abi", "ro.build.version.sdk", "ro.build.display.id"} {
		t.Run(key, func(t *testing.T) {
			e, h, fs, module := fixtureEngine(t)
			h.props[key] = "wrong"
			if _, err := e.Check(context.Background(), module); err == nil {
				t.Fatal("wrong profile accepted")
			}
			if _, err := os.Lstat(fs.Local(Base)); !os.IsNotExist(err) {
				t.Fatal("base mutated")
			}
			for _, call := range h.calls {
				if strings.Contains(call, " install ") || strings.Contains(call, "control.sh") {
					t.Fatal("mutation " + call)
				}
			}
		})
	}
}
func TestCheckRefusesUnknownRuntimeAndInsufficientDisk(t *testing.T) {
	for _, kind := range []string{"base", "disk", "package"} {
		t.Run(kind, func(t *testing.T) {
			e, h, fs, module := fixtureEngine(t)
			switch kind {
			case "base":
				os.Mkdir(fs.Local(Base), 0700)
			case "disk":
				h.space = 1
			case "package":
				h.installed = true
			}
			if _, err := e.Check(context.Background(), module); err == nil {
				t.Fatal("preflight accepted collision")
			}
		})
	}
}
func preparedFixture(t *testing.T, e *Engine, fs *sandboxFiles, module string) {
	t.Helper()
	raw, _, pins := coreFixture()
	e.core = pins
	os.Mkdir(fs.Local(module+"/core"), 0700)
	os.WriteFile(fs.Local(module+"/core/mihomo"), raw, 0700)
	m, err := ReadManifest(fs.Local(module))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(fs.Local(module + "/manifest.json"))
	p := prepared{1, Version, sum(b), pins.RawSHA256, pins.RawBytes}
	data, _ := json.Marshal(p)
	fs.Atomic(module+"/prepared.json", data, 0600)
	_ = m
}
func TestBootIsEmptyOffAndReentryKeepsKeyAndPackage(t *testing.T) {
	e, h, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(fs.Local(Base + "/web/private-key.pem"))
	if len(first) == 0 {
		t.Fatal("no identity")
	}
	s, _ := os.ReadFile(fs.Local(Base + "/web/state.json"))
	if !strings.Contains(string(s), `"desiredEnabled":false`) || !strings.Contains(string(s), `"nodes":[]`) {
		t.Fatal(string(s))
	}
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(fs.Local(Base + "/web/private-key.pem"))
	if string(first) != string(second) {
		t.Fatal("identity rotated")
	}
	installs := 0
	for _, c := range h.calls {
		if strings.Contains(c, "/system/bin/pm install") {
			installs++
		}
		if strings.Contains(c, "control.sh start") {
			t.Fatal("VPN started during install")
		}
	}
	if installs != 1 {
		t.Fatalf("installed %d times", installs)
	}
}
func TestBootResumesAfterTrustJournalInterruption(t *testing.T) {
	e, h, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	fs.failPhase = "TRUST_READY"
	if _, err := e.Boot(context.Background(), module); err == nil {
		t.Fatal("injected interruption missing")
	}
	key, _ := os.ReadFile(fs.Local(Base + "/web/private-key.pem"))
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	same, _ := os.ReadFile(fs.Local(Base + "/web/private-key.pem"))
	if string(key) != string(same) || !h.installed {
		t.Fatal("resume changed identity/package")
	}
}
func TestBootRefusesMismatchedInstalledAPKBeforeTrust(t *testing.T) {
	e, h, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	h.unknownAPK = true
	if _, err := e.Boot(context.Background(), module); err == nil {
		t.Fatal("unknown apk trusted")
	}
	if _, err := os.Stat(fs.Local(Base + "/native-ui/trust.json")); !os.IsNotExist(err) {
		t.Fatal("trust written")
	}
}
func TestUninstallPreservesBackupAndRejectsUnknownOwnership(t *testing.T) {
	e, h, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(fs.Local(Base+"/web/private-note"), []byte("keep"), 0600)
	r, err := e.Uninstall(context.Background(), module)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Complete || h.installed {
		t.Fatal("not uninstalled")
	}
	if _, err = os.Stat(fs.Local(r.Backup + "/web/private-note")); err != nil {
		t.Fatal("backup lost")
	}
	if _, err = os.Stat(fs.Local(Base)); !os.IsNotExist(err) {
		t.Fatal("base retained")
	}
	e2, _, fs2, module2 := fixtureEngine(t)
	os.Mkdir(fs2.Local(Base), 0700)
	os.WriteFile(fs2.Local(Base+"/unknown"), []byte("keep"), 0600)
	if _, err = e2.Uninstall(context.Background(), module2); err == nil {
		t.Fatal("unknown uninstall accepted")
	}
	if _, err = os.Stat(fs2.Local(Base + "/unknown")); err != nil {
		t.Fatal("unknown removed")
	}
}
func TestUninstallDefersWhenPMUnavailableAndKeepsRuntime(t *testing.T) {
	e, h, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	h.ready = false
	r, err := e.Uninstall(context.Background(), module)
	if err == nil || r.Complete || !r.Deferred {
		t.Fatal("deferred cleanup reported success")
	}
	if _, err = os.Stat(fs.Local(Base + "/web/private-key.pem")); err != nil {
		t.Fatal("runtime removed during deferral")
	}
}
func TestBootStreamsPrivateAPKAndAcceptsOnlyMagiskCustomizeCleanup(t *testing.T) {
	e, h, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	h.forbidDirectInstall = true
	os.Remove(fs.Local(module + "/customize.sh"))
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	if h.streamedInstalls != 1 {
		t.Fatal("APK was not streamed to Package Manager")
	}
}
func TestEveryInstallJournalBoundaryResumesWithoutRotatingIdentity(t *testing.T) {
	for _, phase := range []string{"PREPARED", "RUNTIME_READY", "PUBLISHED", "APK_REQUESTED", "APK_INSTALLED", "TRUST_READY", "COMMITTED"} {
		t.Run(phase, func(t *testing.T) {
			e, _, fs, module := fixtureEngine(t)
			preparedFixture(t, e, fs, module)
			fs.failPhase = phase
			if _, err := e.Boot(context.Background(), module); err == nil {
				t.Fatal("interruption not injected")
			}
			key, _ := os.ReadFile(fs.Local(Base + "/web/private-key.pem"))
			if len(key) == 0 {
				key, _ = os.ReadFile(fs.Local(stageRoot + "/web/private-key.pem"))
			}
			if _, err := e.Boot(context.Background(), module); err != nil {
				t.Fatal(err)
			}
			next, _ := os.ReadFile(fs.Local(Base + "/web/private-key.pem"))
			if len(key) > 0 && string(key) != string(next) {
				t.Fatal("identity changed on resume")
			}
		})
	}
}
func TestUninstallResumesAfterBackupRenameBeforeFinalJournal(t *testing.T) {
	e, _, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	fs.failPhase = "UNINSTALLED"
	r, err := e.Uninstall(context.Background(), module)
	if err == nil || !r.Deferred {
		t.Fatal("missing uninstall interruption")
	}
	r, err = e.Uninstall(context.Background(), module)
	if err != nil || !r.Complete {
		t.Fatal("uninstall did not reconcile backup", err)
	}
}
func TestCommittedTrustMismatchCannotBeRepairedOrUninstalled(t *testing.T) {
	e, h, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	p := fs.Local(Base + "/native-ui/trust.json")
	os.WriteFile(p, []byte(`{"schema":1,"packageName":"com.codex.u30vpn","uid":12345,"apkPath":"/data/app/wrong/base.apk","apkSha256":"bad"}`), 0600)
	before := len(h.calls)
	if _, err := e.Boot(context.Background(), module); err == nil {
		t.Fatal("bad trust accepted")
	}
	if _, err := e.Uninstall(context.Background(), module); err == nil {
		t.Fatal("bad trust uninstalled")
	}
	for _, call := range h.calls[before:] {
		if strings.Contains(call, "control.sh stop") || strings.Contains(call, "pm uninstall") {
			t.Fatal("mutated unknown trust")
		}
	}
}
func TestUninstallBlocksConcurrentEnableUntilRuntimeIsBackedUp(t *testing.T) {
	e, h, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	h.onBridgeStop = func() {
		if !h.locks[fs.Local(Base+"/web/manager.lock")] {
			h.networkActive = true
		}
	}
	r, err := e.Uninstall(context.Background(), module)
	if err != nil {
		t.Fatal(err)
	}
	if r.Complete && h.networkActive {
		t.Fatal("uninstall completed while concurrent enable restarted VPN")
	}
}
func TestUninstallWaitsForBridgeListenerToCloseBeforeRemovingAPK(t *testing.T) {
	e, h, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	h.bridgeLingers = true
	r, err := e.Uninstall(context.Background(), module)
	if err == nil || !r.Deferred || r.Complete || !h.installed {
		t.Fatal("bridge still served while uninstall completed")
	}
	if _, err := os.Stat(fs.Local(Base + "/web/private-key.pem")); err != nil {
		t.Fatal("private state moved while bridge alive")
	}
}
func TestUninstallAcceptsRootSafeMagiskRemovalFlag(t *testing.T) {
	e, _, fs, module := fixtureEngine(t)
	preparedFixture(t, e, fs, module)
	if _, err := e.Boot(context.Background(), module); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(fs.Local(module+"/remove"), nil, 0644)
	fs.modeOverrides = map[string]os.FileMode{module + "/remove": 0644}
	r, err := e.Uninstall(context.Background(), module)
	if err != nil || !r.Complete {
		t.Fatal("safe Magisk flag rejected", err)
	}
}
func TestPreflightRequiresCompleteSuccessfulNetworkSnapshots(t *testing.T) {
	for _, snapshot := range []string{"-4 route show table all", "link show", "-w 1 -t filter -S", "-w 1 -t nat -S"} {
		t.Run(snapshot, func(t *testing.T) {
			e, h, _, module := fixtureEngine(t)
			h.snapshotFailure = snapshot
			if _, err := e.Check(context.Background(), module); err == nil {
				t.Fatal("snapshot error treated as absence")
			}
		})
	}
	for _, collision := range []string{"route", "tun", "chain"} {
		t.Run(collision, func(t *testing.T) {
			e, h, _, module := fixtureEngine(t)
			h.collision = collision
			if _, err := e.Check(context.Background(), module); err == nil {
				t.Fatal("reserved object in full snapshot ignored")
			}
		})
	}
}
