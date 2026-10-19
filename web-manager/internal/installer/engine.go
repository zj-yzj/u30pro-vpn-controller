package installer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	OK             bool   `json:"ok"`
	Version        string `json:"version"`
	State          string `json:"state"`
	Complete       bool   `json:"complete"`
	Deferred       bool   `json:"deferred,omitempty"`
	RebootRequired bool   `json:"rebootRequired,omitempty"`
	Backup         string `json:"backup,omitempty"`
}
type Engine struct {
	host      Host
	fs        Filesystem
	core      corePin
	client    *http.Client
	bootstrap func(string) error
}

func New() *Engine { return NewWith(SystemHost{}, OSFiles{}) }
func NewWith(host Host, files Filesystem) *Engine {
	return &Engine{host: host, fs: files, core: upstreamCore, bootstrap: bootstrapManager}
}
func result(state string) Result { return Result{OK: true, Version: Version, State: state} }
func (e *Engine) run(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	b, err := e.host.Run(ctx, name, args...)
	return strings.TrimSpace(string(b)), err
}
func (e *Engine) exists(p string) bool                       { _, err := os.Lstat(e.fs.Local(p)); return !os.IsNotExist(err) }
func (e *Engine) read(p string, limit int64) ([]byte, error) { return readSmall(e.fs.Local(p), limit) }
func (e *Engine) mkdir(p string) error {
	if e.exists(p) {
		if !e.fs.Safe(p, 0700, true) {
			return problem("unsafe_storage")
		}
		return nil
	}
	if os.Mkdir(e.fs.Local(p), 0700) != nil {
		return problem("storage_failed")
	}
	return syncDirectory(filepath.Dir(e.fs.Local(p)))
}
func (e *Engine) save(p string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return problem("storage_failed")
	}
	if err = e.fs.Atomic(p, b, 0600); err != nil {
		return problem("storage_failed")
	}
	return nil
}
func (e *Engine) safeParents() error {
	for _, p := range []string{"/", "/data"} {
		s, err := os.Lstat(e.fs.Local(p))
		if err != nil || !s.IsDir() || s.Mode()&os.ModeSymlink != 0 {
			return problem("unsafe_storage")
		}
	}
	if !e.fs.Safe("/data/adb", 0, true) {
		return problem("unsafe_storage")
	}
	return nil
}
func (e *Engine) packageFiles(module string, installed ...bool) (Manifest, string, error) {
	var m Manifest
	if !ValidModuleDir(module) || e.safeParents() != nil || !e.fs.Safe(filepath.ToSlash(filepath.Dir(module)), 0, true) || !e.fs.Safe(module, 0, true) {
		return m, "", problem("unsafe_storage")
	}
	afterCustomize := len(installed) > 0 && installed[0]
	m, err := readManifest(e.fs.Local(module), afterCustomize)
	if err != nil {
		return m, "", err
	}
	for _, f := range m.Files {
		if afterCustomize && f.Path == "customize.sh" && !e.exists(module+"/customize.sh") {
			continue
		}
		if !e.fs.Safe(module+"/"+f.Path, os.FileMode(f.Mode), false) {
			return m, "", problem("unsafe_storage")
		}
	}
	for _, tree := range []string{"payload", "licenses"} {
		if !e.exists(module + "/" + tree) {
			continue
		}
		if err = filepath.WalkDir(e.fs.Local(module+"/"+tree), func(p string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				rel, err := filepath.Rel(e.fs.Local(module), p)
				if err != nil || !e.fs.Safe(module+"/"+filepath.ToSlash(rel), 0, true) {
					return problem("unsafe_storage")
				}
			}
			return nil
		}); err != nil {
			return m, "", problem("unsafe_storage")
		}
	}
	if !e.fs.Safe(module+"/manifest.json", 0600, false) {
		return m, "", problem("unsafe_storage")
	}
	b, err := e.read(module+"/manifest.json", 1<<20)
	return m, sum(b), err
}
func (e *Engine) profile(ctx context.Context, needPM bool) error {
	if e.safeParents() != nil {
		return problem("unsafe_storage")
	}
	uid, err := e.run(ctx, "/system/bin/id", "-u")
	if err != nil || uid != "0" {
		return problem("root_required")
	}
	expected := map[string]string{"ro.product.model": "MU5358", "ro.product.device": "MU5358", "ro.product.cpu.abi": "arm64-v8a", "ro.build.version.sdk": "35", "ro.build.display.id": "U30ProV1.0.0B23"}
	for key, want := range expected {
		got, err := e.run(ctx, "/system/bin/getprop", key)
		if err != nil || got != want {
			return problem("unsupported_device")
		}
	}
	enforce, err := e.run(ctx, "/system/bin/getenforce")
	if err != nil || enforce != "Permissive" {
		return problem("unsupported_device")
	}
	magisk, err := e.run(ctx, "magisk", "-V")
	version, parseErr := strconv.Atoi(magisk)
	if err != nil || parseErr != nil || version < 28100 {
		return problem("magisk_unavailable")
	}
	const tools = "for t in ip iptables ip6tables stat sha256sum nohup setsid timeout awk grep tr readlink tail chmod chown cp mv cat sleep ss; do [ -x /system/bin/$t ] || exit 1; done"
	if _, err = e.run(ctx, "/system/bin/sh", "-c", tools); err != nil {
		return problem("unsupported_device")
	}
	if needPM {
		ufi, err := e.run(ctx, "/system/bin/cmd", "package", "list", "packages", UfiPackage)
		if err != nil || ufi != "package:"+UfiPackage {
			return problem("ufi_required")
		}
	}
	return nil
}
func (e *Engine) networkAbsent(ctx context.Context) error {
	rules, err := e.run(ctx, "/system/bin/ip", "-4", "rule", "show")
	if err != nil || !strings.Contains(rules, "lookup local") {
		return problem("network_unavailable")
	}
	for _, line := range strings.Split(rules, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "9490:") || strings.HasPrefix(line, "9491:") {
			return problem("network_collision")
		}
	}
	routes, err := e.run(ctx, "/system/bin/ip", "-4", "route", "show", "table", "all")
	if err != nil {
		return problem("network_unavailable")
	}
	for _, line := range strings.Split(routes, "\n") {
		fields := strings.Fields(line)
		for i := 0; i+1 < len(fields); i++ {
			if fields[i] == "table" && fields[i+1] == "20131" {
				return problem("network_collision")
			}
		}
	}
	links, err := e.run(ctx, "/system/bin/ip", "link", "show")
	if err != nil {
		return problem("network_unavailable")
	}
	for _, line := range strings.Split(links, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 && strings.TrimSuffix(strings.Split(fields[1], "@")[0], ":") == "U30VPN" {
			return problem("network_collision")
		}
	}
	reserved := map[string]bool{"U30VPN4FORWARD": true, "U30VPN6FORWARD": true, "U30VPND4NAT": true, "U30VPNI4INPUT": true, "U30VPNI6INPUT": true}
	for _, spec := range [][2]string{{"iptables", "filter"}, {"ip6tables", "filter"}, {"iptables", "nat"}} {
		snapshot, err := e.run(ctx, "/system/bin/"+spec[0], "-w", "1", "-t", spec[1], "-S")
		if err != nil {
			return problem("network_unavailable")
		}
		policies := map[string]bool{}
		for _, line := range strings.Split(snapshot, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[0] == "-P" {
				policies[fields[1]] = true
			}
			for _, field := range fields {
				if reserved[field] {
					return problem("network_collision")
				}
			}
		}
		if spec[1] == "filter" && (!policies["INPUT"] || !policies["FORWARD"]) {
			return problem("network_unavailable")
		}
		if spec[1] == "nat" && !policies["PREROUTING"] {
			return problem("network_unavailable")
		}
	}
	sfp, err := e.read("/proc/net/sfp/enable", 16)
	if err != nil || strings.TrimSpace(string(sfp)) != "1" {
		return problem("network_collision")
	}
	return nil
}
func (e *Engine) networkProfile(ctx context.Context) error {
	if err := e.networkAbsent(ctx); err != nil {
		return err
	}
	forward, err := e.read("/proc/sys/net/ipv4/ip_forward", 16)
	if err != nil || strings.TrimSpace(string(forward)) != "1" {
		return problem("unsupported_network")
	}
	addr, err := e.run(ctx, "/system/bin/ip", "-4", "addr", "show", "dev", "br0")
	if err != nil || !strings.Contains(addr, "inet 192.168.0.1/24 ") {
		return problem("unsupported_network")
	}
	if !e.exists("/dev/net/tun") && !e.exists("/dev/tun") {
		return problem("unsupported_network")
	}
	sockets, err := e.run(ctx, "/system/bin/ss", "-H", "-lntup")
	if err != nil {
		return problem("network_unavailable")
	}
	for _, field := range strings.Fields(sockets) {
		if strings.HasSuffix(field, ":17890") {
			return problem("network_collision")
		}
	}
	for _, c := range [][4]string{{"iptables", "filter", "REJECT", ""}, {"ip6tables", "filter", "REJECT", ""}, {"iptables", "nat", "DNAT", ""}} {
		if _, err = e.run(ctx, "/system/bin/"+c[0], "-t", c[1], "-j", c[2], "-h"); err != nil {
			return problem("unsupported_network")
		}
	}
	return nil
}
func (e *Engine) packageAbsent(ctx context.Context) error {
	out, err := e.run(ctx, "/system/bin/cmd", "package", "list", "packages", "-U", PackageName)
	if err != nil {
		return problem("pm_unavailable")
	}
	if out != "" {
		return problem("existing_package")
	}
	return nil
}
func (e *Engine) Check(ctx context.Context, module string) (Result, error) {
	if module != "" {
		if _, _, err := e.packageFiles(module); err != nil {
			return Result{}, err
		}
	} else {
		if e.exists("/data/adb/modules/"+ModuleID) || e.exists("/data/adb/modules_update/"+ModuleID) {
			return Result{}, problem("existing_installation")
		}
	}
	if err := e.profile(ctx, true); err != nil {
		return Result{}, err
	}
	if e.exists(Base) || e.exists(stageRoot) || e.exists(journalPath) {
		return Result{}, problem("existing_installation")
	}
	if module == "/data/adb/modules_update/"+ModuleID && e.exists("/data/adb/modules/"+ModuleID) {
		return Result{}, problem("existing_installation")
	}
	if err := e.packageAbsent(ctx); err != nil {
		return Result{}, err
	}
	if err := e.networkProfile(ctx); err != nil {
		return Result{}, err
	}
	free, err := e.host.FreeBytes(e.fs.Local("/data/adb"))
	if err != nil || free < uint64(e.core.RawBytes*4)+(32<<20) {
		return Result{}, problem("insufficient_space")
	}
	r := result("supported")
	r.Complete = true
	return r, nil
}

type prepared struct {
	Schema         int    `json:"schema"`
	Version        string `json:"version"`
	ManifestSHA256 string `json:"manifestSha256"`
	CoreSHA256     string `json:"coreSha256"`
	CoreBytes      int64  `json:"coreBytes"`
}

func (e *Engine) validatePrepared(module, manifestHash string) error {
	if !e.fs.Safe(module+"/prepared.json", 0600, false) || !e.fs.Safe(module+"/core", 0700, true) || !e.fs.Safe(module+"/core/mihomo", 0700, false) {
		return problem("not_prepared")
	}
	b, err := e.read(module+"/prepared.json", 4096)
	var p prepared
	if err != nil || decodeStrict(b, &p) != nil || p.Schema != 1 || p.Version != Version || p.ManifestSHA256 != manifestHash || p.CoreSHA256 != e.core.RawSHA256 || p.CoreBytes != e.core.RawBytes {
		return problem("not_prepared")
	}
	return verifyFile(e.fs.Local(module+"/core/mihomo"), p.CoreBytes, p.CoreSHA256)
}
func (e *Engine) Prepare(ctx context.Context, module string) (Result, error) {
	if _, err := e.Check(ctx, module); err != nil {
		return Result{}, err
	}
	_, mh, err := e.packageFiles(module)
	if err != nil {
		return Result{}, err
	}
	if e.exists(module + "/prepared.json") {
		if err = e.validatePrepared(module, mh); err != nil {
			return Result{}, err
		}
	} else {
		if err = e.mkdir(module + "/core"); err != nil {
			return Result{}, err
		}
		target := module + "/core/mihomo"
		if e.exists(target) {
			if !e.fs.Safe(target, 0700, false) || verifyFile(e.fs.Local(target), e.core.RawBytes, e.core.RawSHA256) != nil {
				return Result{}, problem("not_prepared")
			}
		} else {
			client := e.client
			if client == nil {
				client, err = androidClient()
				if err != nil {
					return Result{}, err
				}
			}
			if err = downloadPinned(ctx, client, e.core, e.fs.Local(target)); err != nil {
				return Result{}, err
			}
		}
		if err = e.save(module+"/prepared.json", prepared{1, Version, mh, e.core.RawSHA256, e.core.RawBytes}); err != nil {
			return Result{}, err
		}
	}
	r := result("prepared")
	r.Complete = true
	r.RebootRequired = true
	return r, nil
}
