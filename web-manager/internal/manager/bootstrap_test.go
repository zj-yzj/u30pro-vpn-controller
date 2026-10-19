package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"u30pro/webmanager/internal/subscription"
)

func bootstrapManager(t *testing.T, m *Manager) error {
	t.Helper()
	bootstrap, ok := any(m).(interface{ Bootstrap() error })
	if !ok {
		t.Fatal("Manager does not implement the clean Bootstrap operation")
	}
	return bootstrap.Bootstrap()
}

type bootstrapNoCore struct{ t *testing.T }

func (c bootstrapNoCore) Snapshot(context.Context) (NativeSnapshot, error) {
	c.t.Fatal("bootstrap contacted the core")
	return NativeSnapshot{}, nil
}
func (c bootstrapNoCore) Refresh(context.Context) error { c.t.Fatal("core refresh"); return nil }
func (c bootstrapNoCore) Select(context.Context, string) error {
	c.t.Fatal("core selection")
	return nil
}
func (c bootstrapNoCore) Health(context.Context) error { c.t.Fatal("core health"); return nil }
func (c bootstrapNoCore) Probe(context.Context) error  { c.t.Fatal("core probe"); return nil }
func (c bootstrapNoCore) ProbeNode(context.Context, string) error {
	c.t.Fatal("node probe")
	return nil
}

func bootstrapBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func bootstrapSnapshot(t *testing.T, m *Manager) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for _, path := range []string{m.webPath("private-key.pem"), m.providerPath(), m.webPath("state.json")} {
		b, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		out[path] = b
	}
	return out
}

func assertBootstrapUnchanged(t *testing.T, m *Manager, before map[string][]byte) {
	t.Helper()
	after := bootstrapSnapshot(t, m)
	if len(after) != len(before) {
		t.Fatal("bootstrap added or removed existing bootstrap data")
	}
	for path, b := range before {
		if !bytes.Equal(b, after[path]) {
			t.Fatalf("bootstrap changed %s", filepath.Base(path))
		}
	}
}

func TestBootstrapCreatesEmptyOffStateWithoutExternalWork(t *testing.T) {
	m := testManager(t)
	m.Core = bootstrapNoCore{t}
	m.Control = func(context.Context, string) error { t.Fatal("bootstrap changed networking"); return nil }
	m.Fetch = func(context.Context, string) ([]byte, error) {
		t.Fatal("bootstrap downloaded a subscription")
		return nil, nil
	}
	m.Validate = func(context.Context, []map[string]any) error { t.Fatal("bootstrap ran the core validator"); return nil }
	if err := bootstrapManager(t, m); err != nil {
		t.Fatal(err)
	}
	state, err := m.readState()
	if err != nil {
		t.Fatal(err)
	}
	if state.DesiredEnabled || state.SelectedID != "Auto" || state.SubscriptionURL != "" || state.Nodes == nil || len(state.Nodes) != 0 {
		t.Fatal("first bootstrap is not an empty, unconfigured, disabled state")
	}
	if got := string(bootstrapBytes(t, m.providerPath())); got != "proxies: []\n" {
		t.Fatal("invalid empty provider")
	}
	key, err := m.privateKey()
	if err != nil || key.Validate() != nil {
		t.Fatal("no valid device-generated identity")
	}
	if err := m.Boot(); err != nil {
		t.Fatal(err)
	}
	m.Core = nil
	status, err := m.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !status.OK || status.Running || status.DesiredEnabled || status.Subscription.Configured || status.PublicKeyPEM == "" {
		t.Fatal("new installation is not controllable while VPN is off")
	}
	raw, _ := json.Marshal(status)
	if !bytes.Contains(raw, []byte(`"nodes":[]`)) {
		t.Fatal("empty status nodes are not an array")
	}
}

func TestBootstrapResumesAfterKeyAndProviderWritesWithoutRotatingIdentity(t *testing.T) {
	for _, phase := range []string{"key", "key-and-provider", "complete"} {
		t.Run(phase, func(t *testing.T) {
			m := testManager(t)
			if err := m.createKey(); err != nil {
				t.Fatal(err)
			}
			key := bootstrapBytes(t, m.webPath("private-key.pem"))
			if phase != "key" {
				if err := atomicWrite(m.providerPath(), []byte("proxies: []\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "complete" {
				if err := m.writeState(State{Schema: 1, SelectedID: "Auto", Nodes: []subscription.Node{}}); err != nil {
					t.Fatal(err)
				}
			}
			if err := bootstrapManager(t, m); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(key, bootstrapBytes(t, m.webPath("private-key.pem"))) {
				t.Fatal("identity rotated on recovery")
			}
			before := bootstrapSnapshot(t, m)
			if err := bootstrapManager(t, m); err != nil {
				t.Fatal(err)
			}
			assertBootstrapUnchanged(t, m, before)
		})
	}
}

func TestBootstrapPreservesConfiguredStateAndProvider(t *testing.T) {
	m, _, _, _ := setupUpdate(t)
	if err := m.createKey(); err != nil {
		t.Fatal(err)
	}
	before := bootstrapSnapshot(t, m)
	if err := bootstrapManager(t, m); err != nil {
		t.Fatal(err)
	}
	assertBootstrapUnchanged(t, m, before)
}

func TestBootstrapRejectsUnknownOrIncompleteDataWithoutMutation(t *testing.T) {
	for _, kind := range []string{"provider-only", "unknown-provider", "state-without-key", "state-without-provider", "bad-key", "bad-state"} {
		t.Run(kind, func(t *testing.T) {
			m := testManager(t)
			if kind != "provider-only" && kind != "state-without-key" {
				if err := m.createKey(); err != nil {
					t.Fatal(err)
				}
			}
			if kind != "state-without-provider" {
				provider := []byte("proxies: []\n")
				if kind == "unknown-provider" {
					provider = []byte("proxies: [{name: unknown}]\n")
				}
				if err := atomicWrite(m.providerPath(), provider, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if strings.HasPrefix(kind, "state-") || kind == "bad-state" {
				if err := m.writeState(State{Schema: 1, SelectedID: "Auto", Nodes: []subscription.Node{}}); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "bad-key" {
				if err := atomicWrite(m.webPath("private-key.pem"), []byte("invalid key\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "bad-state" {
				if err := atomicWrite(m.webPath("state.json"), []byte(`{"schema":999}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := bootstrapSnapshot(t, m)
			if bootstrapManager(t, m) == nil {
				t.Fatal("unknown or incomplete bootstrap data was accepted")
			}
			assertBootstrapUnchanged(t, m, before)
		})
	}
}

func TestBootstrapRejectsSymlinkProviderAndKeyWithoutCreatingState(t *testing.T) {
	for _, kind := range []string{"provider", "key"} {
		t.Run(kind, func(t *testing.T) {
			m := testManager(t)
			if err := m.createKey(); err != nil {
				t.Fatal(err)
			}
			path := m.providerPath()
			data := []byte("proxies: []\n")
			if kind == "key" {
				path = m.webPath("private-key.pem")
				data = bootstrapBytes(t, path)
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			target := filepath.Join(t.TempDir(), "untouched")
			if err := os.WriteFile(target, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			if bootstrapManager(t, m) == nil {
				t.Fatal("symlink accepted")
			}
			if !bytes.Equal(data, bootstrapBytes(t, target)) {
				t.Fatal("symlink target changed")
			}
			if _, err := os.Lstat(m.webPath("state.json")); !os.IsNotExist(err) {
				t.Fatal("state created despite unsafe path")
			}
		})
	}
}

func TestBootstrapRejectsSymlinkLockBeforeCreatingIdentity(t *testing.T) {
	m := testManager(t)
	target := filepath.Join(t.TempDir(), "untouched-lock")
	if err := os.WriteFile(target, []byte("keep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, m.webPath("manager.lock")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if bootstrapManager(t, m) == nil {
		t.Fatal("unsafe lock path accepted")
	}
	if string(bootstrapBytes(t, target)) != "keep\n" {
		t.Fatal("lock target changed")
	}
	if _, err := os.Lstat(m.webPath("private-key.pem")); !os.IsNotExist(err) {
		t.Fatal("identity created despite unsafe lock path")
	}
}

func TestBootstrapRejectsUnsafeProductionFileModes(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() != 0 {
		t.Skip("requires POSIX root ownership/modes")
	}
	for _, name := range []string{"key", "provider", "state"} {
		t.Run(name, func(t *testing.T) {
			m := testManager(t)
			if err := os.Chmod(m.Base, 0700); err != nil {
				t.Fatal(err)
			}
			if err := bootstrapManager(t, m); err != nil {
				t.Fatal(err)
			}
			paths := map[string]string{"key": m.webPath("private-key.pem"), "provider": m.providerPath(), "state": m.webPath("state.json")}
			if err := os.Chmod(paths[name], 0644); err != nil {
				t.Fatal(err)
			}
			m.Test = false
			before := bootstrapSnapshot(t, m)
			if err := bootstrapManager(t, m); ErrorCode(err) != "insecure_storage" {
				t.Fatalf("unsafe permissions accepted: %v", err)
			}
			assertBootstrapUnchanged(t, m, before)
		})
	}
}

func TestEnableWithoutNodesPreservesStateAndDoesNotCallControl(t *testing.T) {
	m := testManager(t)
	if err := m.writeState(State{Schema: 1, SelectedID: "Auto", Nodes: []subscription.Node{}}); err != nil {
		t.Fatal(err)
	}
	before := bootstrapBytes(t, m.webPath("state.json"))
	calls := 0
	m.Control = func(context.Context, string) error { calls++; return fail("control_failed") }
	enabled := true
	err := m.perform(context.Background(), Request{Action: "set-enabled", Enabled: &enabled})
	if ErrorCode(err) != "not_configured" {
		t.Fatalf("want not_configured, got %v", err)
	}
	if calls != 0 || !bytes.Equal(before, bootstrapBytes(t, m.webPath("state.json"))) {
		t.Fatal("unconfigured enable changed state or networking")
	}
}

func TestFirstSubscriptionUpdateKeepsBootstrappedVPNOff(t *testing.T) {
	m := testManager(t)
	if err := bootstrapManager(t, m); err != nil {
		t.Fatal(err)
	}
	key := bootstrapBytes(t, m.webPath("private-key.pem"))
	m.Core = bootstrapNoCore{t}
	m.Control = func(context.Context, string) error { t.Fatal("first subscription enabled VPN"); return nil }
	fetched, validated := false, false
	m.Fetch = func(_ context.Context, url string) ([]byte, error) {
		if url != "https://example.com/first" {
			t.Fatal("unexpected subscription URL")
		}
		fetched = true
		return []byte(`proxies: [{name: first, type: vless, server: first.example.com, port: 443, uuid: 00000000-0000-0000-0000-000000000001}]`), nil
	}
	m.Validate = func(_ context.Context, proxies []map[string]any) error {
		if len(proxies) != 1 || proxies[0]["type"] != "vless" || proxies[0]["server"] != "first.example.com" {
			t.Fatal("unexpected validation payload")
		}
		validated = true
		return nil
	}
	if err := m.perform(context.Background(), Request{Action: "update-subscription", URL: "https://example.com/first"}); err != nil {
		t.Fatal(err)
	}
	state, err := m.readState()
	if err != nil {
		t.Fatal(err)
	}
	if !fetched || !validated || state.DesiredEnabled || state.SubscriptionURL != "https://example.com/first" || len(state.Nodes) != 1 {
		t.Fatal("first subscription was not committed while remaining off")
	}
	if !bytes.Equal(key, bootstrapBytes(t, m.webPath("private-key.pem"))) {
		t.Fatal("subscription changed identity")
	}
}

func TestNotConfiguredErrorTellsUserToAddSubscription(t *testing.T) {
	err := SafeError("not_configured")
	if err.Error.Code != "not_configured" || !strings.Contains(err.Error.Message, "订阅") || !strings.Contains(err.Error.Message, "VPN") {
		t.Fatal("unconfigured error is not actionable")
	}
}
