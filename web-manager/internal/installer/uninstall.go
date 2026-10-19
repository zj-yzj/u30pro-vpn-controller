package installer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func deferred() Result { r := result("cleanup_deferred"); r.OK = false; r.Deferred = true; return r }
func (e *Engine) removeOldLock(ctx context.Context) error {
	dir := Base + "/lock"
	if !e.exists(dir) {
		return nil
	}
	if !e.fs.Safe(dir, 0700, true) || !e.fs.Safe(dir+"/identity", 0600, false) {
		return problem("cleanup_pending")
	}
	b, err := e.read(dir+"/identity", 512)
	if err != nil {
		return problem("cleanup_pending")
	}
	parts := strings.Fields(string(b))
	boot, err := e.read("/proc/sys/kernel/random/boot_id", 128)
	if err != nil || len(parts) != 3 || parts[2] == strings.TrimSpace(string(boot)) {
		return problem("cleanup_pending")
	}
	if err = e.networkAbsent(ctx); err != nil {
		return err
	}
	entries, err := os.ReadDir(e.fs.Local(dir))
	if err != nil || len(entries) != 1 || entries[0].Name() != "identity" {
		return problem("cleanup_pending")
	}
	if os.Remove(e.fs.Local(dir+"/identity")) != nil || os.Remove(e.fs.Local(dir)) != nil {
		return problem("cleanup_pending")
	}
	return syncDirectory(e.fs.Local(Base))
}
func (e *Engine) removeOldBridgeIdentity() error {
	p := Base + "/native-ui/bridge-process.json"
	if !e.exists(p) {
		return nil
	}
	if !e.fs.Safe(p, 0600, false) {
		return problem("cleanup_pending")
	}
	b, err := e.read(p, 2048)
	if err != nil {
		return problem("cleanup_pending")
	}
	var id struct {
		Schema    int    `json:"schema"`
		PID       int    `json:"pid"`
		StartTime string `json:"startTime"`
		BootID    string `json:"bootId"`
		ExePath   string `json:"exePath"`
	}
	if decodeStrict(b, &id) != nil || id.Schema != 1 || id.PID < 2 || id.StartTime == "" || id.ExePath != Base+"/native-ui/webbridge" {
		return problem("cleanup_pending")
	}
	boot, err := e.read("/proc/sys/kernel/random/boot_id", 128)
	if err != nil {
		return problem("cleanup_pending")
	}
	if id.BootID == strings.TrimSpace(string(boot)) {
		return nil
	}
	if os.Remove(e.fs.Local(p)) != nil {
		return problem("cleanup_pending")
	}
	return syncDirectory(filepath.Dir(e.fs.Local(p)))
}
func (e *Engine) markRemoval(module string) error {
	p := module + "/remove"
	if e.exists(p) && !e.fs.Safe(p, 0, false) {
		return problem("unsafe_storage")
	}
	if err := e.fs.Atomic(p, []byte{}, 0600); err != nil {
		return problem("storage_failed")
	}
	return nil
}
func (e *Engine) waitBridgeStopped(ctx context.Context) error {
	for n := 0; n < 60; n++ {
		sockets, err := e.run(ctx, "/system/bin/ss", "-H", "-xl")
		if err != nil {
			return problem("cleanup_pending")
		}
		if !e.exists(Base+"/native-ui/bridge-process.json") && !strings.Contains(sockets, "codex_u30vpn_v1") {
			return nil
		}
		if err = e.host.Sleep(ctx, 250*time.Millisecond); err != nil {
			return problem("cleanup_pending")
		}
	}
	return problem("cleanup_pending")
}
func (e *Engine) Uninstall(ctx context.Context, module string) (Result, error) {
	m, mh, err := e.packageFiles(module, true)
	if err != nil {
		return Result{}, err
	}
	if err = e.safeParents(); err != nil {
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
	j, err := e.loadJournal(mh)
	if err != nil {
		return Result{}, err
	}
	apk, _ := m.file("payload/U30ProVPN.apk")
	if j.APKSHA256 != apk.SHA256 {
		return Result{}, problem("package_mismatch")
	}
	if j.Phase == "UNINSTALLED" {
		if j.Backup == "" || e.ownerAt(j.Backup, j) != nil {
			return Result{}, problem("unknown_ownership")
		}
		if err = e.markRemoval(module); err != nil {
			return Result{}, err
		}
		r := result("uninstalled")
		r.Complete = true
		r.Backup = j.Backup
		return r, nil
	}
	// A prior rename may have completed before its journal commit. Reconcile the
	// exact backup namespace, never search for or rename an unrelated directory.
	if !e.exists(Base) {
		if j.Phase != "UNINSTALL_PENDING" || j.Backup == "" || e.ownerAt(j.Backup, j) != nil {
			return Result{}, problem("unknown_ownership")
		}
		if err = e.phase(&j, "UNINSTALLED"); err != nil {
			return deferred(), err
		}
		if err = e.markRemoval(module); err != nil {
			return deferred(), err
		}
		r := result("uninstalled")
		r.Complete = true
		r.Backup = j.Backup
		return r, nil
	}
	if err = e.verifyRuntime(Base, m, j); err != nil {
		return Result{}, err
	}
	t, present, err := e.packageIdentity(ctx, j.APKSHA256)
	if err != nil {
		return deferred(), err
	}
	if present && e.exists(Base+"/native-ui/trust.json") {
		b, err := e.read(Base+"/native-ui/trust.json", 4096)
		var existing trust
		if !e.fs.Safe(Base+"/native-ui/trust.json", 0600, false) || err != nil || decodeStrict(b, &existing) != nil || existing != t {
			return Result{}, problem("package_mismatch")
		}
	}
	// Do not begin a destructive uninstall while PM or netfilter is unavailable
	// during Magisk's early-boot uninstall hook. Preserve all private data.
	if _, err = e.run(ctx, "/system/bin/ip", "-4", "rule", "show"); err != nil {
		return deferred(), problem("network_unavailable")
	}
	// Queue and Worker use this exact flock inode. Holding it prevents UFI and
	// native clients from changing desired state or launching a pending worker
	// while stop/rename is in progress. An active worker causes a safe deferral.
	managerLock := Base + "/web/manager.lock"
	if !e.fs.Safe(managerLock, 0600, false) {
		return deferred(), problem("unsafe_storage")
	}
	releaseManager, err := e.host.Lock(e.fs.Local(managerLock))
	if err != nil {
		return deferred(), problem("busy")
	}
	defer releaseManager()
	j.Backup = "/data/adb/u30pro-vpn-backup-" + Version + "-" + j.Instance
	if e.exists(j.Backup) {
		return Result{}, problem("unknown_ownership")
	}
	if err = e.phase(&j, "UNINSTALL_PENDING"); err != nil {
		return deferred(), err
	}
	if err = e.removeOldBridgeIdentity(); err != nil {
		return deferred(), err
	}
	if _, err = e.run(ctx, Base+"/native-ui/webbridge", "stop"); err != nil {
		return deferred(), problem("cleanup_pending")
	}
	if err = e.waitBridgeStopped(ctx); err != nil {
		return deferred(), err
	}
	// Ordinary stop delegates process identity and narrowly owned rule removal to
	// the existing controller. An old-boot lock can be removed only with no owned
	// network objects and the original SFP value restored.
	stopCtx, cancel := context.WithTimeout(ctx, 50*time.Second)
	_, stopErr := e.host.Run(stopCtx, "/system/bin/sh", Base+"/control.sh", "stop")
	cancel()
	if stopErr != nil {
		if err = e.removeOldLock(ctx); err != nil {
			return deferred(), problem("cleanup_pending")
		}
	}
	if err = e.networkAbsent(ctx); err != nil {
		return deferred(), err
	}
	if err = e.removeOldLock(ctx); err != nil {
		return deferred(), err
	}
	if present {
		latest, still, err := e.packageIdentity(ctx, j.APKSHA256)
		if err != nil || !still || latest != t {
			return deferred(), problem("package_mismatch")
		}
		if _, err = e.run(ctx, "/system/bin/pm", "uninstall", PackageName); err != nil {
			return deferred(), problem("package_uninstall_failed")
		}
		_, still, err = e.packageIdentity(ctx, j.APKSHA256)
		if err != nil || still {
			return deferred(), problem("package_uninstall_failed")
		}
	}
	if err = e.fs.Publish(Base, j.Backup); err != nil {
		return deferred(), problem("storage_failed")
	}
	if err = e.phase(&j, "UNINSTALLED"); err != nil {
		return deferred(), err
	}
	if err = e.markRemoval(module); err != nil {
		return deferred(), err
	}
	r := result("uninstalled")
	r.Complete = true
	r.Backup = j.Backup
	return r, nil
}

// ErrorResult is intentionally a fixed category; no raw shell, URL or private
// runtime data is returned to either the Magisk or Android installer entry point.
func ErrorResult(err error, r Result) map[string]any {
	code := ErrorCode(err)
	out := map[string]any{"ok": false, "version": Version, "error": map[string]string{"code": code, "message": code}, "complete": false}
	if r.Deferred {
		out["deferred"] = true
		out["state"] = "cleanup_deferred"
	}
	return out
}
