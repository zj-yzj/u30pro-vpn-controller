package manager

import (
	"context"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"u30pro/webmanager/internal/subscription"
)

type fakeCore struct {
	m           *Manager
	running     bool
	selected    string
	refreshes   int
	selects     int
	healthErr   bool
	failRefresh bool
}

func (c *fakeCore) Snapshot(context.Context) (NativeSnapshot, error) {
	s := NativeSnapshot{Running: c.running, Selected: c.selected, Active: c.selected, Nodes: map[string]NativeNode{}}
	if !c.running {
		return s, fail("apply_failed")
	}
	b, _ := os.ReadFile(c.m.providerPath())
	r, e := subscription.Normalize(b)
	if e != nil {
		return s, e
	}
	for _, n := range r.Nodes {
		s.Nodes[n.ID] = NativeNode{}
	}
	return s, nil
}
func (c *fakeCore) Refresh(context.Context) error {
	c.refreshes++
	if c.failRefresh {
		return fail("apply_failed")
	}
	return nil
}
func (c *fakeCore) Select(_ context.Context, id string) error {
	c.selects++
	c.selected = id
	return nil
}
func (c *fakeCore) Health(context.Context) error {
	if c.healthErr {
		return fail("apply_failed")
	}
	return nil
}
func (c *fakeCore) Probe(context.Context) error             { return nil }
func (c *fakeCore) ProbeNode(context.Context, string) error { return nil }
func setupUpdate(t *testing.T) (*Manager, *fakeCore, State, []byte) {
	t.Helper()
	m := testManager(t)
	oldRaw := []byte(`proxies: [{name: old, type: vless, server: old.example.com, port: 443, uuid: 00000000-0000-0000-0000-000000000001}]`)
	old, e := subscription.Normalize(oldRaw)
	if e != nil {
		t.Fatal(e)
	}
	s := State{Schema: 1, DesiredEnabled: true, SelectedID: old.Nodes[0].ID, SubscriptionURL: "https://example.com/private?SECRET", Nodes: old.Nodes}
	m.writeState(s)
	atomicWrite(m.providerPath(), old.Provider, 0600)
	m.Validate = func(context.Context, []map[string]any) error { return nil }
	m.Fetch = func(context.Context, string) ([]byte, error) {
		return []byte(`proxies: [{name: new, type: vless, server: new.example.com, port: 443, uuid: 00000000-0000-0000-0000-000000000002}]`), nil
	}
	core := &fakeCore{m: m, running: true, selected: s.SelectedID}
	m.Core = core
	return m, core, s, old.Provider
}
func TestUpdateFailureRollsBackProviderStateAndSelection(t *testing.T) {
	m, core, s, old := setupUpdate(t)
	core.healthErr = true
	err := m.update(context.Background(), "", s)
	if ErrorCode(err) != "apply_failed" {
		t.Fatalf("code: %v", err)
	}
	got, _ := os.ReadFile(m.providerPath())
	if string(got) != string(old) {
		t.Fatal("old provider lost")
	}
	state, _ := m.readState()
	if state.SelectedID != s.SelectedID || core.selected != s.SelectedID || core.refreshes != 2 {
		t.Fatal("selection/runtime not rolled back")
	}
	if _, e := os.Stat(m.txPath("journal.json")); !os.IsNotExist(e) {
		t.Fatal("journal remained")
	}
}
func TestUpdateWhenOffDoesNotStartCore(t *testing.T) {
	m, core, s, _ := setupUpdate(t)
	s.DesiredEnabled = false
	m.writeState(s)
	core.running = false
	calls := 0
	m.Control = func(context.Context, string) error { calls++; return nil }
	if err := m.update(context.Background(), "", s); err != nil {
		t.Fatal(err)
	}
	state, _ := m.readState()
	if state.DesiredEnabled || core.refreshes != 0 || core.selects != 0 || calls != 0 {
		t.Fatal("off update touched VPN")
	}
	if state.SelectedID != "Auto" || state.Message == "" {
		t.Fatal("removed selection not reset")
	}
}
func TestBadDownloadAndValidationKeepOldBytes(t *testing.T) {
	for _, kind := range []string{"fetch", "normalize", "validate"} {
		t.Run(kind, func(t *testing.T) {
			m, _, s, old := setupUpdate(t)
			switch kind {
			case "fetch":
				m.Fetch = func(context.Context, string) ([]byte, error) { return nil, fail("download_failed") }
			case "normalize":
				m.Fetch = func(context.Context, string) ([]byte, error) { return []byte("SECRET garbage"), nil }
			case "validate":
				m.Validate = func(context.Context, []map[string]any) error { return fail("invalid_subscription") }
			}
			if m.update(context.Background(), "", s) == nil {
				t.Fatal("failure accepted")
			}
			b, _ := os.ReadFile(m.providerPath())
			if string(b) != string(old) {
				t.Fatal("provider touched before validation")
			}
			if _, e := os.Stat(m.txPath("journal.json")); !os.IsNotExist(e) {
				t.Fatal("transaction begun")
			}
		})
	}
}
func TestPublicStatusHasNoURLCredentials(t *testing.T) {
	m, _, _, _ := setupUpdate(t)
	if err := m.createKey(); err != nil {
		t.Fatal(err)
	}
	atomicWrite(filepath.Join(m.Base, "state"), []byte("RUNNING\n"), 0600)
	out, e := m.Status()
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(out)
	if strings.Contains(string(b), "SECRET") || strings.Contains(string(b), "private?") {
		t.Fatal("private URL leaked")
	}
	if out.Subscription.Host != "example.com" || out.State != "running" {
		t.Fatal("status fields")
	}
}
func TestDesiredOffSurvivesControlFailureAndBoot(t *testing.T) {
	m, core, _, _ := setupUpdate(t)
	enabled := false
	m.Control = func(context.Context, string) error { return fail("control_failed") }
	if m.perform(context.Background(), Request{Action: "set-enabled", Enabled: &enabled}) == nil {
		t.Fatal("failed stop accepted")
	}
	s, _ := m.readState()
	if s.DesiredEnabled {
		t.Fatal("off intention lost")
	}
	core.running = false
	calls := 0
	m.Control = func(context.Context, string) error { calls++; return nil }
	if err := m.Boot(); err != nil || calls != 0 {
		t.Fatal("off boot started VPN")
	}
}
func TestSSRFBounds(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "192.168.0.1", "100.64.0.1", "198.18.0.1", "169.254.169.254", "224.0.0.1", "0.0.0.0", "::1", "::ffff:192.168.0.1"} {
		if publicIPv4(netip.MustParseAddr(raw)) {
			t.Fatalf("unsafe address %s", raw)
		}
	}
	if !publicIPv4(netip.MustParseAddr("1.1.1.1")) {
		t.Fatal("public rejected")
	}
	for _, raw := range []string{"http://example.com", "https://user:SECRET@example.com", "https://example.com:8080/", "https://example.com/#SECRET"} {
		if _, e := subscriptionURL(raw); e == nil {
			t.Fatal("unsafe URL")
		}
	}
}
