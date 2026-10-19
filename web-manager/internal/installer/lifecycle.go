package installer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"u30pro/webmanager/internal/manager"
)

type journal struct {
	Schema         int    `json:"schema"`
	Version        string `json:"version"`
	ModuleID       string `json:"moduleId"`
	ManifestSHA256 string `json:"manifestSha256"`
	APKSHA256      string `json:"apkSha256"`
	Instance       string `json:"instance"`
	Phase          string `json:"phase"`
	Backup         string `json:"backup,omitempty"`
}

var phases = map[string]bool{"PREPARED": true, "RUNTIME_READY": true, "PUBLISHED": true, "APK_REQUESTED": true, "APK_INSTALLED": true, "TRUST_READY": true, "COMMITTED": true, "UNINSTALL_PENDING": true, "UNINSTALLED": true}

func (e *Engine) loadJournal(mh string) (journal, error) {
	var j journal
	if !e.fs.Safe(journalPath, 0600, false) {
		return j, problem("existing_installation")
	}
	b, err := e.read(journalPath, 8192)
	if err != nil || decodeStrict(b, &j) != nil || j.Schema != 1 || j.Version != Version || j.ModuleID != ModuleID || j.ManifestSHA256 != mh || !validHash(j.APKSHA256) || len(j.Instance) != 32 || !phases[j.Phase] {
		return j, problem("existing_installation")
	}
	if _, err = hex.DecodeString(j.Instance); err != nil {
		return j, problem("existing_installation")
	}
	if j.Backup != "" && j.Backup != "/data/adb/u30pro-vpn-backup-"+Version+"-"+j.Instance {
		return j, problem("unknown_ownership")
	}
	return j, nil
}
func (e *Engine) phase(j *journal, phase string) error {
	j.Phase = phase
	return e.save(journalPath, j)
}
func (e *Engine) ownerAt(root string, j journal) error {
	if !e.fs.Safe(root, 0700, true) || !e.fs.Safe(root+"/installation.json", 0600, false) {
		return problem("unknown_ownership")
	}
	b, err := e.read(root+"/installation.json", 8192)
	var owner journal
	if err != nil || decodeStrict(b, &owner) != nil || owner.Schema != 1 || owner.Version != j.Version || owner.ModuleID != j.ModuleID || owner.ManifestSHA256 != j.ManifestSHA256 || owner.APKSHA256 != j.APKSHA256 || owner.Instance != j.Instance {
		return problem("unknown_ownership")
	}
	return nil
}
func verifyFile(p string, size int64, hash string) error {
	info, err := os.Lstat(p)
	if err != nil || !info.Mode().IsRegular() || info.Size() != size {
		return problem("payload_mismatch")
	}
	f, err := os.Open(p)
	if err != nil {
		return problem("payload_mismatch")
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, size+1))
	if err != nil || n != size || hex.EncodeToString(h.Sum(nil)) != hash {
		return problem("payload_mismatch")
	}
	return nil
}
func (e *Engine) copyPinned(src, dst string, size int64, hash string, mode os.FileMode) error {
	if e.exists(dst) {
		if !e.fs.Safe(dst, mode, false) || verifyFile(e.fs.Local(dst), size, hash) != nil {
			return problem("payload_mismatch")
		}
		return nil
	}
	f, err := os.Open(e.fs.Local(src))
	if err != nil {
		return problem("storage_failed")
	}
	defer f.Close()
	temp, err := os.CreateTemp(filepath.Dir(e.fs.Local(dst)), ".u30setup-copy-")
	if err != nil {
		return problem("storage_failed")
	}
	name := temp.Name()
	defer os.Remove(name)
	defer temp.Close()
	if temp.Chmod(mode) != nil {
		return problem("storage_failed")
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(temp, h), io.LimitReader(f, size+1))
	if err != nil || n != size || hex.EncodeToString(h.Sum(nil)) != hash {
		return problem("payload_mismatch")
	}
	if temp.Sync() != nil || temp.Close() != nil {
		return problem("storage_failed")
	}
	if e.exists(dst) {
		return problem("existing_installation")
	}
	if os.Rename(name, e.fs.Local(dst)) != nil {
		return problem("storage_failed")
	}
	return syncDirectory(filepath.Dir(e.fs.Local(dst)))
}

var runtimePayload = map[string]string{"payload/webctl": "webctl", "payload/webbridge": "native-ui/webbridge", "payload/config.yaml": "config.yaml", "payload/control.sh": "control.sh", "payload/vpn-service.sh": "vpn-service.sh"}

func bootstrapManager(base string) error { m := manager.New(); m.Base = base; return m.Bootstrap() }
func (e *Engine) verifyRuntime(root string, m Manifest, j journal) error {
	if err := e.ownerAt(root, j); err != nil {
		return err
	}
	for _, dir := range []string{"web", "cache", "runtime", "log", "native-ui"} {
		if !e.fs.Safe(root+"/"+dir, 0700, true) {
			return problem("unsafe_storage")
		}
	}
	for source, target := range runtimePayload {
		r, ok := m.file(source)
		if !ok || !e.fs.Safe(root+"/"+target, os.FileMode(r.Mode), false) || verifyFile(e.fs.Local(root+"/"+target), r.Bytes, r.SHA256) != nil {
			return problem("payload_mismatch")
		}
	}
	if !e.fs.Safe(root+"/mihomo", 0700, false) {
		return problem("unsafe_storage")
	}
	return verifyFile(e.fs.Local(root+"/mihomo"), e.core.RawBytes, e.core.RawSHA256)
}
func (e *Engine) waitReady(ctx context.Context) error {
	for n := 0; n < 90; n++ {
		ready, err := e.run(ctx, "/system/bin/getprop", "sys.boot_completed")
		if err == nil && ready == "1" {
			ufi, err := e.run(ctx, "/system/bin/cmd", "package", "list", "packages", UfiPackage)
			if err == nil {
				if ufi != "package:"+UfiPackage {
					return problem("ufi_required")
				}
				return nil
			}
		}
		if err = e.host.Sleep(ctx, 2*time.Second); err != nil {
			return problem("pm_unavailable")
		}
	}
	return problem("pm_unavailable")
}
func (e *Engine) stageRuntime(module string, m Manifest, j journal) error {
	if err := e.mkdir(stageRoot); err != nil {
		return err
	}
	if !e.exists(stageRoot + "/installation.json") {
		// Intent was durable before this namespace was created. Only an empty
		// directory or interrupted atomic writes are valid before the owner marker.
		entries, err := os.ReadDir(e.fs.Local(stageRoot))
		if err != nil {
			return problem("unsafe_storage")
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasPrefix(entry.Name(), ".u30setup-") || !e.fs.Safe(stageRoot+"/"+entry.Name(), 0600, false) {
				return problem("unknown_ownership")
			}
		}
		if err = e.save(stageRoot+"/installation.json", j); err != nil {
			return err
		}
	}
	if err := e.ownerAt(stageRoot, j); err != nil {
		return err
	}
	for _, dir := range []string{"web", "cache", "log", "runtime", "native-ui"} {
		if err := e.mkdir(stageRoot + "/" + dir); err != nil {
			return err
		}
	}
	for source, target := range runtimePayload {
		r, _ := m.file(source)
		if err := e.copyPinned(module+"/"+source, stageRoot+"/"+target, r.Bytes, r.SHA256, os.FileMode(r.Mode)); err != nil {
			return err
		}
	}
	if err := e.copyPinned(module+"/core/mihomo", stageRoot+"/mihomo", e.core.RawBytes, e.core.RawSHA256, 0700); err != nil {
		return err
	}
	baseline := map[string]any{"schema": 1, "instance": j.Instance, "sfp": 1, "ipv4Forward": 1, "lan": "192.168.0.1/24", "tun": "U30VPN", "routeTable": 20131, "rulePriorities": []int{9490, 9491}, "objectsInitiallyAbsent": true}
	if !e.exists(stageRoot + "/baseline.json") {
		if err := e.save(stageRoot+"/baseline.json", baseline); err != nil {
			return err
		}
	}
	if !e.exists(stageRoot + "/web/manager.lock") {
		if err := e.fs.Atomic(stageRoot+"/web/manager.lock", []byte{}, 0600); err != nil {
			return problem("storage_failed")
		}
	}
	if err := e.bootstrap(e.fs.Local(stageRoot)); err != nil {
		return problem("bootstrap_failed")
	}
	if !e.exists(stageRoot + "/state") {
		if err := e.fs.Atomic(stageRoot+"/state", []byte("STOPPED\n"), 0600); err != nil {
			return problem("storage_failed")
		}
	}
	if err := e.verifyRuntime(stageRoot, m, j); err != nil {
		return err
	}
	return syncDirectory(e.fs.Local(stageRoot))
}

type trust struct {
	Schema      int    `json:"schema"`
	PackageName string `json:"packageName"`
	UID         uint32 `json:"uid"`
	APKPath     string `json:"apkPath"`
	APKSHA256   string `json:"apkSha256"`
}

func validAPKPath(p string) bool {
	return strings.HasPrefix(p, "/data/app/") && strings.HasSuffix(p, "/base.apk") && filepath.ToSlash(filepath.Clean(p)) == p && !strings.ContainsAny(p, "\x00\r\n\t \\") && len(strings.Split(p, "/")) >= 5
}
func (e *Engine) packageIdentity(ctx context.Context, expected string) (trust, bool, error) {
	out, err := e.run(ctx, "/system/bin/cmd", "package", "list", "packages", "-U", PackageName)
	if err != nil {
		return trust{}, false, problem("pm_unavailable")
	}
	if out == "" {
		return trust{}, false, nil
	}
	prefix := "package:" + PackageName + " uid:"
	if !strings.HasPrefix(out, prefix) || strings.ContainsAny(out, "\r\n") {
		return trust{}, true, problem("package_mismatch")
	}
	rawUID := strings.TrimPrefix(out, prefix)
	uid, err := strconv.ParseUint(rawUID, 10, 32)
	if err != nil || uid < 10000 || uid > 2147483647 || strconv.FormatUint(uid, 10) != rawUID {
		return trust{}, true, problem("package_mismatch")
	}
	shared, err := e.run(ctx, "/system/bin/cmd", "package", "list", "packages", "-U", "--uid", rawUID)
	if err != nil || shared != out {
		return trust{}, true, problem("package_mismatch")
	}
	location, err := e.run(ctx, "/system/bin/pm", "path", PackageName)
	if err != nil || !strings.HasPrefix(location, "package:") || strings.ContainsAny(location, "\r\n") {
		return trust{}, true, problem("package_mismatch")
	}
	p := strings.TrimPrefix(location, "package:")
	if !validAPKPath(p) {
		return trust{}, true, problem("package_mismatch")
	}
	current := ""
	for _, part := range strings.Split(strings.TrimPrefix(p, "/"), "/") {
		current += "/" + part
		info, err := os.Lstat(e.fs.Local(current))
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return trust{}, true, problem("package_mismatch")
		}
	}
	b, err := e.read(p, 64<<20)
	if err != nil || sum(b) != expected {
		return trust{}, true, problem("package_mismatch")
	}
	return trust{1, PackageName, uint32(uid), p, expected}, true, nil
}
func (e *Engine) ensurePackage(ctx context.Context, module string, j *journal, apk FileRecord) error {
	t, present, err := e.packageIdentity(ctx, j.APKSHA256)
	if err != nil {
		return err
	}
	if j.Phase == "PUBLISHED" {
		if present {
			return problem("existing_package")
		}
		if err = e.phase(j, "APK_REQUESTED"); err != nil {
			return err
		}
	}
	if !present {
		if j.Phase != "APK_REQUESTED" {
			return problem("package_mismatch")
		}
		// No -r: replacement of an existing same-name package is never authorized.
		installCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		_, installErr := e.host.InstallAPK(installCtx, e.fs.Local(module+"/payload/U30ProVPN.apk"), apk.Bytes)
		cancel()
		t, present, err = e.packageIdentity(ctx, j.APKSHA256)
		if err != nil {
			return err
		}
		if !present {
			_ = installErr
			return problem("package_install_failed")
		}
	}
	if j.Phase == "APK_REQUESTED" {
		if err = e.phase(j, "APK_INSTALLED"); err != nil {
			return err
		}
	}
	if j.Phase == "APK_INSTALLED" || j.Phase == "TRUST_READY" {
		if e.exists(Base + "/native-ui/trust.json") {
			if !e.fs.Safe(Base+"/native-ui/trust.json", 0600, false) {
				return problem("package_mismatch")
			}
			b, err := e.read(Base+"/native-ui/trust.json", 4096)
			var old trust
			if err != nil || decodeStrict(b, &old) != nil || old != t {
				return problem("package_mismatch")
			}
		} else {
			if err = e.save(Base+"/native-ui/trust.json", t); err != nil {
				return err
			}
		}
		return e.phase(j, "TRUST_READY")
	}
	if j.Phase == "COMMITTED" {
		b, err := e.read(Base+"/native-ui/trust.json", 4096)
		var old trust
		if !e.fs.Safe(Base+"/native-ui/trust.json", 0600, false) || err != nil || decodeStrict(b, &old) != nil || old != t {
			return problem("package_mismatch")
		}
	}
	return nil
}
func (e *Engine) Boot(ctx context.Context, module string) (Result, error) {
	m, mh, err := e.packageFiles(module, true)
	if err != nil {
		return Result{}, err
	}
	if err = e.profile(ctx, false); err != nil {
		return Result{}, err
	}
	if err = e.waitReady(ctx); err != nil {
		return Result{}, err
	}
	if err = e.validatePrepared(module, mh); err != nil {
		return Result{}, err
	}
	if e.exists(lockPath) && !e.fs.Safe(lockPath, 0600, false) {
		return Result{}, problem("unsafe_storage")
	}
	unlock, err := e.host.Lock(e.fs.Local(lockPath))
	if err != nil {
		return Result{}, problem("busy")
	}
	defer unlock()
	var j journal
	if e.exists(journalPath) {
		j, err = e.loadJournal(mh)
		if err != nil {
			return Result{}, err
		}
		apk, _ := m.file("payload/U30ProVPN.apk")
		if j.APKSHA256 != apk.SHA256 {
			return Result{}, problem("package_mismatch")
		}
	} else {
		if e.exists(Base) || e.exists(stageRoot) {
			return Result{}, problem("existing_installation")
		}
		if err = e.packageAbsent(ctx); err != nil {
			return Result{}, err
		}
		if err = e.networkProfile(ctx); err != nil {
			return Result{}, err
		}
		free, err := e.host.FreeBytes(e.fs.Local("/data/adb"))
		if err != nil || free < uint64(e.core.RawBytes*2)+(32<<20) {
			return Result{}, problem("insufficient_space")
		}
		instance := make([]byte, 16)
		if _, err = rand.Read(instance); err != nil {
			return Result{}, problem("storage_failed")
		}
		apk, _ := m.file("payload/U30ProVPN.apk")
		j = journal{Schema: 1, Version: Version, ModuleID: ModuleID, ManifestSHA256: mh, APKSHA256: apk.SHA256, Instance: hex.EncodeToString(instance), Phase: "PREPARED"}
		if err = e.save(journalPath, j); err != nil {
			return Result{}, err
		}
	}
	if j.Phase == "UNINSTALLED" || j.Phase == "UNINSTALL_PENDING" {
		return Result{}, problem("cleanup_pending")
	}
	if !e.exists(Base) {
		if j.Phase != "PREPARED" && j.Phase != "RUNTIME_READY" {
			return Result{}, problem("unknown_ownership")
		}
		if err = e.networkProfile(ctx); err != nil {
			return Result{}, err
		}
		if err = e.packageAbsent(ctx); err != nil {
			return Result{}, err
		}
		if err = e.stageRuntime(module, m, j); err != nil {
			return Result{}, err
		}
		if err = e.phase(&j, "RUNTIME_READY"); err != nil {
			return Result{}, err
		}
		if err = e.fs.Publish(stageRoot, Base); err != nil {
			return Result{}, problem("existing_installation")
		}
	}
	if err = e.verifyRuntime(Base, m, j); err != nil {
		return Result{}, err
	}
	if j.Phase == "PREPARED" || j.Phase == "RUNTIME_READY" {
		if err = e.phase(&j, "PUBLISHED"); err != nil {
			return Result{}, err
		}
	}
	committed := j.Phase == "COMMITTED"
	apk, _ := m.file("payload/U30ProVPN.apk")
	if err = e.ensurePackage(ctx, module, &j, apk); err != nil {
		return Result{}, err
	}
	if _, err = e.run(ctx, Base+"/native-ui/webbridge", "start"); err != nil {
		return Result{}, problem("bridge_failed")
	}
	if !committed {
		if err = e.phase(&j, "COMMITTED"); err != nil {
			return Result{}, err
		}
	} else {
		bootCtx, cancel := context.WithTimeout(ctx, 100*time.Second)
		_, err = e.host.Run(bootCtx, Base+"/webctl", "boot")
		cancel()
		if err != nil {
			return Result{}, problem("control_failed")
		}
	}
	r := result("installed")
	r.Complete = true
	return r, nil
}
func (e *Engine) Status(ctx context.Context, module string) (Result, error) {
	_, mh, err := e.packageFiles(module, true)
	if err != nil {
		return Result{}, err
	}
	if !e.exists(journalPath) {
		if e.exists(Base) {
			return Result{}, problem("existing_installation")
		}
		if err = e.validatePrepared(module, mh); err != nil {
			return Result{}, err
		}
		r := result("prepared")
		r.RebootRequired = true
		return r, nil
	}
	j, err := e.loadJournal(mh)
	if err != nil {
		return Result{}, err
	}
	r := result(strings.ToLower(j.Phase))
	r.Complete = j.Phase == "COMMITTED" || j.Phase == "UNINSTALLED"
	r.Deferred = j.Phase == "UNINSTALL_PENDING"
	r.Backup = j.Backup
	return r, nil
}
