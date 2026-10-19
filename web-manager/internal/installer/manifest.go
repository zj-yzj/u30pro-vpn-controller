// Package installer implements the fixed-profile, first-install-only Magisk
// lifecycle. Runtime paths and upstream core pins are deliberately not CLI inputs.
package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const Version = "1.3.0-alpha.1"
const ModuleID = "codex_u30pro_vpn"
const Base = "/data/adb/u30pro-vpn"
const stageRoot = "/data/adb/.u30pro-vpn-install-" + Version
const journalPath = "/data/adb/.u30pro-vpn-install-" + Version + ".json"
const lockPath = "/data/adb/.u30pro-vpn-setup.lock"
const PackageName = "com.codex.u30vpn"
const UfiPackage = "com.minikano.f50_sms"

type category string

func (e category) Error() string { return string(e) }
func problem(code string) error  { return category(code) }
func ErrorCode(err error) string {
	var e category
	if errors.As(err, &e) {
		return string(e)
	}
	return "operation_failed"
}

type FileRecord struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
	Mode   uint32 `json:"mode"`
}
type Manifest struct {
	Schema   int          `json:"schema"`
	Version  string       `json:"version"`
	ModuleID string       `json:"moduleId"`
	Files    []FileRecord `json:"files"`
}

var requiredPaths = []string{"module.prop", "customize.sh", "service.sh", "action.sh", "uninstall.sh", "u30setup", "payload/webctl", "payload/webbridge", "payload/U30ProVPN.apk", "payload/config.yaml", "payload/control.sh", "payload/vpn-service.sh", "payload/U30ProVPN.txt"}

func executablePath(p string) bool {
	return p == "u30setup" || strings.HasSuffix(p, ".sh") || p == "payload/webctl" || p == "payload/webbridge"
}
func ValidModuleDir(p string) bool {
	return p == "/data/adb/modules/"+ModuleID || p == "/data/adb/modules_update/"+ModuleID
}
func validHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && strings.ToLower(s) == s
}
func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// uniqueJSON also rejects duplicate fields that encoding/json otherwise accepts.
func uniqueJSON(raw []byte) error {
	d := json.NewDecoder(strings.NewReader(string(raw)))
	var value func() error
	value = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					k, e := d.Token()
					if e != nil {
						return e
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return problem("invalid_manifest")
					}
					seen[key] = true
					if e = value(); e != nil {
						return e
					}
				}
				end, e := d.Token()
				if e != nil || end != json.Delim('}') {
					return problem("invalid_manifest")
				}
			case '[':
				for d.More() {
					if e := value(); e != nil {
						return e
					}
				}
				end, e := d.Token()
				if e != nil || end != json.Delim(']') {
					return problem("invalid_manifest")
				}
			default:
				return problem("invalid_manifest")
			}
		}
		return nil
	}
	if err := value(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return problem("invalid_manifest")
	}
	return nil
}
func decodeStrict(raw []byte, out any) error {
	if uniqueJSON(raw) != nil {
		return problem("invalid_manifest")
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return problem("invalid_manifest")
	}
	return nil
}
func readSmall(file string, limit int64) ([]byte, error) {
	s, e := os.Lstat(file)
	if e != nil || !s.Mode().IsRegular() || s.Size() > limit {
		return nil, problem("unsafe_storage")
	}
	f, e := os.Open(file)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil || int64(len(b)) > limit {
		return nil, problem("unsafe_storage")
	}
	return b, nil
}

func ReadManifest(dir string) (Manifest, error) { return readManifest(dir, false) }

// Magisk's installer removes customize.sh after sourcing it. All other release
// files remain mandatory. A retained customize.sh must still match its pin.
func readManifest(dir string, afterCustomize bool) (Manifest, error) {
	var m Manifest
	raw, err := readSmall(filepath.Join(dir, "manifest.json"), 1<<20)
	if err != nil || decodeStrict(raw, &m) != nil || m.Schema != 1 || m.Version != Version || m.ModuleID != ModuleID || len(m.Files) > 512 {
		return m, problem("invalid_manifest")
	}
	allowed := map[string]bool{}
	for _, p := range requiredPaths {
		allowed[p] = true
	}
	seen := map[string]bool{}
	for _, f := range m.Files {
		safe := f.Path != "" && path.Clean(f.Path) == f.Path && !strings.ContainsAny(f.Path, "\\\x00\r\n") && !strings.HasPrefix(f.Path, "/") && !strings.HasPrefix(f.Path, "../")
		license := strings.HasPrefix(f.Path, "licenses/") && f.Path != "licenses/" && len(f.Path) < 256
		mode := uint32(0600)
		if executablePath(f.Path) {
			mode = 0700
		}
		if !safe || (!allowed[f.Path] && !license) || seen[f.Path] || !validHash(f.SHA256) || f.Bytes < 1 || f.Bytes > 128<<20 || f.Mode != mode {
			return m, problem("invalid_manifest")
		}
		seen[f.Path] = true
		full := filepath.Join(dir, filepath.FromSlash(f.Path))
		rel, _ := filepath.Rel(dir, full)
		if strings.HasPrefix(rel, "..") {
			return m, problem("invalid_manifest")
		}
		if afterCustomize && f.Path == "customize.sh" {
			if _, err := os.Lstat(full); os.IsNotExist(err) {
				continue
			}
		}
		current := dir
		parts := strings.Split(f.Path, "/")
		for i, p := range parts {
			current = filepath.Join(current, p)
			s, e := os.Lstat(current)
			if e != nil || s.Mode()&os.ModeSymlink != 0 || (i < len(parts)-1 && !s.IsDir()) {
				return m, problem("invalid_manifest")
			}
		}
		info, e := os.Lstat(full)
		if e != nil || !info.Mode().IsRegular() || info.Size() != f.Bytes {
			return m, problem("invalid_manifest")
		}
		file, e := os.Open(full)
		if e != nil {
			return m, problem("invalid_manifest")
		}
		h := sha256.New()
		n, e := io.Copy(h, io.LimitReader(file, f.Bytes+1))
		file.Close()
		if e != nil || n != f.Bytes || hex.EncodeToString(h.Sum(nil)) != f.SHA256 {
			return m, problem("invalid_manifest")
		}
	}
	for p := range allowed {
		if !seen[p] {
			return m, problem("invalid_manifest")
		}
	}
	// Unexpected top-level Magisk hooks would be executed by Magisk itself and
	// must not be accepted just because every listed payload still verifies.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return m, problem("invalid_manifest")
	}
	for _, entry := range entries {
		name := entry.Name()
		full := filepath.Join(dir, name)
		info, err := os.Lstat(full)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return m, problem("invalid_manifest")
		}
		if entry.IsDir() {
			if name != "payload" && name != "licenses" && name != "core" {
				return m, problem("invalid_manifest")
			}
			continue
		}
		dynamic := name == "manifest.json" || name == "prepared.json" || name == "setup.lock" || name == "disable" || name == "remove" || name == "update" || name == "skip_mount" || strings.HasPrefix(name, ".u30setup-")
		if !info.Mode().IsRegular() || (!seen[name] && !dynamic) {
			return m, problem("invalid_manifest")
		}
	}
	for _, tree := range []string{"payload", "licenses"} {
		base := filepath.Join(dir, tree)
		if _, e := os.Lstat(base); os.IsNotExist(e) && tree == "licenses" {
			continue
		}
		if err := filepath.WalkDir(base, func(p string, entry os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return problem("invalid_manifest")
			}
			if entry.IsDir() {
				return nil
			}
			rel, e := filepath.Rel(dir, p)
			if e != nil || !seen[filepath.ToSlash(rel)] {
				return problem("invalid_manifest")
			}
			return nil
		}); err != nil {
			return m, problem("invalid_manifest")
		}
	}
	return m, nil
}
func (m Manifest) file(p string) (FileRecord, bool) {
	for _, f := range m.Files {
		if f.Path == p {
			return f, true
		}
	}
	return FileRecord{}, false
}
