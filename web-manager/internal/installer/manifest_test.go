package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fixtureModule(t *testing.T) (string, Manifest) {
	t.Helper()
	dir := t.TempDir()
	m := Manifest{Schema: 1, Version: Version, ModuleID: ModuleID}
	for _, p := range requiredPaths {
		b := []byte("fixture " + p + "\n")
		h := sha256.Sum256(b)
		mode := uint32(0600)
		if executablePath(p) {
			mode = 0700
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, p), b, os.FileMode(mode)); err != nil {
			t.Fatal(err)
		}
		m.Files = append(m.Files, FileRecord{Path: p, Bytes: int64(len(b)), SHA256: hex.EncodeToString(h[:]), Mode: mode})
	}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	return dir, m
}

func TestManifestRejectsTraversalDuplicateMissingAndTampering(t *testing.T) {
	for _, kind := range []string{"traversal", "duplicate", "missing", "tamper", "self", "unknown", "mode", "schema", "version", "extra-json"} {
		t.Run(kind, func(t *testing.T) {
			dir, m := fixtureModule(t)
			switch kind {
			case "traversal":
				m.Files[0].Path = "../outside"
			case "duplicate":
				m.Files = append(m.Files, m.Files[0])
			case "missing":
				m.Files = m.Files[1:]
			case "tamper":
				os.WriteFile(filepath.Join(dir, m.Files[0].Path), []byte("changed"), 0700)
			case "self":
				m.Files = append(m.Files, FileRecord{Path: "manifest.json"})
			case "unknown":
				m.Files[0].Path = "payload/unapproved"
			case "mode":
				m.Files[0].Mode = 0777
			case "schema":
				m.Schema = 2
			case "version":
				m.Version = "other"
			}
			b, _ := json.Marshal(m)
			if kind == "extra-json" {
				b = append(b, []byte(" {}")...)
			}
			os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0600)
			if _, err := ReadManifest(dir); err == nil {
				t.Fatalf("accepted %s", kind)
			}
		})
	}
}
func TestManifestAllowsLicenseWithoutExtensionAndRejectsUnlistedPayload(t *testing.T) {
	dir, m := fixtureModule(t)
	b := []byte("license")
	h := sha256.Sum256(b)
	os.Mkdir(filepath.Join(dir, "licenses"), 0700)
	os.WriteFile(filepath.Join(dir, "licenses/LICENSE"), b, 0600)
	m.Files = append(m.Files, FileRecord{"licenses/LICENSE", int64(len(b)), hex.EncodeToString(h[:]), 0600})
	raw, _ := json.Marshal(m)
	os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0600)
	if _, err := ReadManifest(dir); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "payload/unlisted"), []byte("bad"), 0600)
	if _, err := ReadManifest(dir); err == nil {
		t.Fatal("unlisted payload accepted")
	}
}
func TestModuleDirectoryIsFixedAndCanonical(t *testing.T) {
	for _, p := range []string{"/data/adb/modules/" + ModuleID, "/data/adb/modules_update/" + ModuleID} {
		if !ValidModuleDir(p) {
			t.Fatal(p)
		}
	}
	for _, p := range []string{"", "/tmp/" + ModuleID, "/data/adb/modules/x", "/data/adb/modules/../modules/" + ModuleID, "/data/adb/modules/" + ModuleID + "/"} {
		if ValidModuleDir(p) {
			t.Fatal(p)
		}
	}
}
func TestMagiskMayRemoveOnlyCustomizeAfterPrepare(t *testing.T) {
	dir, _ := fixtureModule(t)
	os.Remove(filepath.Join(dir, "customize.sh"))
	if _, err := ReadManifest(dir); err == nil {
		t.Fatal("prepare accepted missing customize")
	}
	if _, err := readManifest(dir, true); err != nil {
		t.Fatal("boot rejected documented Magisk cleanup", err)
	}
	os.Remove(filepath.Join(dir, "service.sh"))
	if _, err := readManifest(dir, true); err == nil {
		t.Fatal("boot accepted missing service")
	}
}
func TestManifestRejectsUnlistedMagiskStartupScriptButAllowsOwnedDynamicCore(t *testing.T) {
	dir, _ := fixtureModule(t)
	os.WriteFile(filepath.Join(dir, "post-fs-data.sh"), []byte("unsafe"), 0700)
	if _, err := ReadManifest(dir); err == nil {
		t.Fatal("unlisted Magisk hook accepted")
	}
	os.Remove(filepath.Join(dir, "post-fs-data.sh"))
	os.Mkdir(filepath.Join(dir, "core"), 0700)
	os.WriteFile(filepath.Join(dir, "core/mihomo"), []byte("dynamic"), 0700)
	os.WriteFile(filepath.Join(dir, "prepared.json"), []byte("{}"), 0600)
	if _, err := ReadManifest(dir); err != nil {
		t.Fatal("dynamic preparation files rejected", err)
	}
}
