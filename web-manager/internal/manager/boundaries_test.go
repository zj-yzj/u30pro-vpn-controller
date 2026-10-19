package manager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"u30pro/webmanager/internal/subscription"
)

type brokenSnapshot struct{ Controller }

func (b *brokenSnapshot) Snapshot(context.Context) (NativeSnapshot, error) {
	return NativeSnapshot{Running: true}, fail("apply_failed")
}
func TestStatusDoesNotTrustPartialControllerSnapshot(t *testing.T) {
	m, _, _, _ := setupUpdate(t)
	m.Core = &brokenSnapshot{m.Core}
	m.createKey()
	atomicWrite(filepath.Join(m.Base, "state"), []byte("RUNNING\n"), 0600)
	s, e := m.Status()
	if e != nil || s.Running || s.State != "error" {
		t.Fatalf("partial snapshot misreported: %+v %v", s, e)
	}
}

func TestEnvelopeTamperAndUnknownFields(t *testing.T) {
	m := testManager(t)
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	enabled := true
	r := Request{V: 1, IssuedAt: m.Now().UnixMilli(), Nonce: base64.RawURLEncoding.EncodeToString(make([]byte, 16)), Action: "set-enabled", Enabled: &enabled}
	encoded := seal(t, k, r)
	raw, _ := base64.RawURLEncoding.DecodeString(encoded)
	var envelope Envelope
	json.Unmarshal(raw, &envelope)
	ciphertext, _ := base64.StdEncoding.DecodeString(envelope.Data)
	ciphertext[0] ^= 1
	envelope.Data = base64.StdEncoding.EncodeToString(ciphertext)
	tampered, _ := json.Marshal(envelope)
	if _, err := Decrypt(base64.RawURLEncoding.EncodeToString(tampered), k, m.Now()); err == nil {
		t.Fatal("tampered request")
	}
	unknown := strings.TrimSuffix(string(raw), "}") + `,"shell":"SECRET"}`
	if _, err := Decrypt(base64.RawURLEncoding.EncodeToString([]byte(unknown)), k, m.Now()); err == nil {
		t.Fatal("unknown outer field")
	}
	r.URL = "SECRET"
	if _, err := Decrypt(seal(t, k, r), k, m.Now()); err == nil {
		t.Fatal("unexpected action payload")
	}
}
func TestRecoveryAfterPartialCleanup(t *testing.T) {
	for _, phase := range []string{"COMMITTED", "ROLLED_BACK"} {
		t.Run(phase, func(t *testing.T) {
			m := testManager(t)
			s := State{Schema: 1, SelectedID: "Auto"}
			m.writeState(s)
			old := []byte("old")
			next := []byte("next")
			atomicWrite(m.providerPath(), old, 0600)
			m.beginTransaction(next, s)
			if phase == "COMMITTED" {
				atomicWrite(m.providerPath(), next, 0600)
				m.commitTransaction()
			} else {
				b, _ := os.ReadFile(m.txPath("journal.json"))
				var j journal
				json.Unmarshal(b, &j)
				j.Phase = phase
				b, _ = json.Marshal(j)
				atomicWrite(m.txPath("journal.json"), b, 0600)
			}
			os.Remove(m.txPath("old-provider.yaml"))
			if err := m.recoverTransaction(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestCorruptRollbackPreservesEvidence(t *testing.T) {
	m := testManager(t)
	s := State{Schema: 1, SelectedID: "Auto"}
	m.writeState(s)
	atomicWrite(m.providerPath(), []byte("old"), 0600)
	m.beginTransaction([]byte("new"), s)
	atomicWrite(m.providerPath(), []byte("new"), 0600)
	atomicWrite(m.txPath("old-provider.yaml"), []byte("corrupt"), 0600)
	if ErrorCode(m.recoverTransaction()) != "rollback_failed" {
		t.Fatal("corruption accepted")
	}
	if _, err := os.Stat(m.txPath("journal.json")); err != nil {
		t.Fatal("evidence lost")
	}
}
func TestInitImportsPrivateURLWithoutReturningIt(t *testing.T) {
	m, _, _, _ := setupUpdate(t)
	os.Remove(m.webPath("state.json"))
	normalized, _ := m.Fetch(context.Background(), "")
	result, err := subscription.Normalize(normalized)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(result.Nodes)
	atomicWrite(m.webPath("initial-nodes.json"), b, 0600)
	atomicWrite(m.providerPath(), result.Provider, 0600)
	atomicWrite(m.webPath("initial-subscription.txt"), []byte("https://example.com/sub?token=SECRET\n"), 0600)
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}
	state, _ := m.readState()
	if !strings.Contains(state.SubscriptionURL, "SECRET") {
		t.Fatal("URL not imported")
	}
	status, err := m.Status()
	if err != nil {
		t.Fatal(err)
	}
	public, _ := json.Marshal(status)
	if strings.Contains(string(public), "SECRET") {
		t.Fatal("URL leaked")
	}
}
func TestUnixControllerUsesFixedRoutesAndSanitizesErrors(t *testing.T) {
	dir, err := os.MkdirTemp("", "u30-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "s")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Skip("Host Unix socket not available")
	}
	defer listener.Close()
	methods := []string{}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/proxies/VPN":
			if r.Method == "PUT" {
				w.WriteHeader(204)
			} else {
				w.Write([]byte(`{"now":"Auto","all":["Auto","n-0000000000000000"]}`))
			}
		case "/proxies/Auto":
			w.Write([]byte(`{"now":"n-0000000000000000"}`))
		case "/providers/proxies/Subscription":
			if r.Method == "PUT" {
				w.WriteHeader(503)
				w.Write([]byte("SECRET"))
			} else {
				w.Write([]byte(`{"proxies":[{"name":"n-0000000000000000","alive":true,"history":[{"delay":42}],"password":"SECRET"}]}`))
			}
		default:
			t.Error("unexpected route")
			w.WriteHeader(404)
		}
	})}
	go server.Serve(listener)
	defer server.Close()
	core := &UnixController{Socket: socket}
	snapshot, e := core.Snapshot(context.Background())
	if e != nil || snapshot.Active != "n-0000000000000000" || *snapshot.Nodes[snapshot.Active].Delay != 42 {
		t.Fatalf("snapshot: %v", e)
	}
	if e = core.Select(context.Background(), "Auto"); e != nil {
		t.Fatal(e)
	}
	if e = core.Refresh(context.Background()); ErrorCode(e) != "apply_failed" || strings.Contains(e.Error(), "SECRET") {
		t.Fatal("error not sanitized")
	}
	if len(methods) != 5 {
		t.Fatal("unexpected request count")
	}
}
