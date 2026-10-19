package manager

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func sealProbePlain(t *testing.T, key *rsa.PrivateKey, plain []byte) string {
	t.Helper()
	aesKey := make([]byte, 32)
	iv := make([]byte, 12)
	rand.Read(aesKey)
	rand.Read(iv)
	block, _ := aes.NewCipher(aesKey)
	gcm, _ := cipher.NewGCM(block)
	wrapped, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &key.PublicKey, aesKey, []byte(envelopeLabel))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(Envelope{V: 1, Key: base64.StdEncoding.EncodeToString(wrapped), IV: base64.StdEncoding.EncodeToString(iv), Data: base64.StdEncoding.EncodeToString(gcm.Seal(nil, iv, plain, []byte(envelopeLabel)))})
	return base64.RawURLEncoding.EncodeToString(b)
}

func TestProbeEncryptedActionAcceptsNoCallerControlledTargets(t *testing.T) {
	m := testManager(t)
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	base := fmt.Sprintf(`{"v":1,"issuedAt":%d,"nonce":"AAAAAAAAAAAAAAAAAAAAAA","action":"probe-nodes"}`, m.Now().UnixMilli())
	if _, err := Decrypt(sealProbePlain(t, key, []byte(base)), key, m.Now()); err != nil {
		t.Fatalf("manual probe rejected: %v", err)
	}
	for _, field := range []string{`"enabled":true`, `"enabled":null`, `"id":"Auto"`, `"id":""`, `"id":null`, `"url":"https://example.invalid/"`, `"url":""`, `"url":null`, `"command":"id"`} {
		plain := strings.TrimSuffix(base, "}") + "," + field + "}"
		if _, err := Decrypt(sealProbePlain(t, key, []byte(plain)), key, m.Now()); err == nil {
			t.Errorf("probe accepted forbidden field %s", field)
		}
	}
}

type probeCore struct {
	Controller
	probes       int
	probeErr     error
	probeContext context.Context
}

func (c *probeCore) Probe(ctx context.Context) error {
	c.probes++
	c.probeContext = ctx
	return c.probeErr
}

func TestProbeWorkerPublishesResultsWithoutChangingStateOrProvider(t *testing.T) {
	for _, tc := range []struct {
		name              string
		desired, running  bool
		probeErr          error
		wantCode, message string
		wantProbes        int
	}{
		{"success", true, true, nil, "", "测速已完成", 1},
		{"desired-off", false, true, nil, "vpn_stopped", "请先开启 VPN 再测速", 0},
		{"not-running", true, false, nil, "vpn_stopped", "请先开启 VPN 再测速", 0},
		{"failed", true, true, fail("apply_failed"), "probe_failed", "测速未完成，请重试", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, core, state, _ := setupUpdate(t)
			state.DesiredEnabled = tc.desired
			m.writeState(state)
			core.running = tc.running
			spy := &probeCore{Controller: core, probeErr: tc.probeErr}
			m.Core = spy
			controls := 0
			m.Control = func(context.Context, string) error { controls++; return nil }
			beforeState, _ := os.ReadFile(m.webPath("state.json"))
			beforeProvider, _ := os.ReadFile(m.providerPath())
			id := strings.Repeat("a", 32)
			m.writeJob(jobDisk{Job: Job{ID: id, Kind: "probe-nodes", Status: "running"}, CreatedAt: m.Now().UnixMilli()})
			request, _ := json.Marshal(Request{V: 1, Action: "probe-nodes"})
			atomicWrite(m.webPath("job-request.json"), request, 0600)
			err := m.Worker(id)
			if err != nil && ErrorCode(err) != tc.wantCode || err == nil && tc.wantCode != "" {
				t.Fatalf("worker code %v want %s", err, tc.wantCode)
			}
			job, err := m.readJob()
			if err != nil {
				t.Fatal(err)
			}
			wantStatus := "succeeded"
			if tc.wantCode != "" {
				wantStatus = "failed"
			}
			if job.Kind != "probe-nodes" || job.Status != wantStatus || job.Message != tc.message {
				t.Fatalf("unexpected probe job: %+v", job.Job)
			}
			afterState, _ := os.ReadFile(m.webPath("state.json"))
			afterProvider, _ := os.ReadFile(m.providerPath())
			if string(beforeState) != string(afterState) || string(beforeProvider) != string(afterProvider) || controls != 0 || core.selects != 0 || core.refreshes != 0 || spy.probes != tc.wantProbes {
				t.Fatal("probe mutated configuration, selection, or service")
			}
			if spy.probes > 0 {
				deadline, ok := spy.probeContext.Deadline()
				if !ok || time.Until(deadline) > 121*time.Second {
					t.Fatal("probe lacks total timeout")
				}
			}
		})
	}
}

func TestRepeatedStatusNeverStartsProbe(t *testing.T) {
	m, core, _, _ := setupUpdate(t)
	spy := &probeCore{Controller: core}
	m.Core = spy
	m.createKey()
	atomicWrite(filepath.Join(m.Base, "state"), []byte("RUNNING\n"), 0600)
	for n := 0; n < 5; n++ {
		if _, err := m.Status(); err != nil {
			t.Fatal(err)
		}
	}
	if spy.probes != 0 || core.refreshes != 0 || core.selects != 0 {
		t.Fatal("status initiated a node measurement")
	}
}

type publishingProbeCore struct {
	Controller
	measured bool
	id       string
}

func (c *publishingProbeCore) Probe(context.Context) error { c.measured = true; return nil }
func (c *publishingProbeCore) Snapshot(ctx context.Context) (NativeSnapshot, error) {
	snap, err := c.Controller.Snapshot(ctx)
	if c.measured {
		delay := uint16(47)
		alive := true
		snap.Nodes[c.id] = NativeNode{Delay: &delay, Alive: &alive}
	}
	return snap, err
}
func TestProbeCompletionIsAvailableThroughExistingStatusPoll(t *testing.T) {
	m, core, state, _ := setupUpdate(t)
	m.createKey()
	m.Core = &publishingProbeCore{Controller: core, id: state.Nodes[0].ID}
	atomicWrite(filepath.Join(m.Base, "state"), []byte("RUNNING\n"), 0600)
	before, err := m.Status()
	if err != nil || before.Nodes[0].Delay != nil {
		t.Fatal("unexpected pre-probe status")
	}
	id := strings.Repeat("b", 32)
	m.writeJob(jobDisk{Job: Job{ID: id, Kind: "probe-nodes", Status: "running"}, CreatedAt: m.Now().UnixMilli()})
	raw, _ := json.Marshal(Request{V: 1, Action: "probe-nodes"})
	atomicWrite(m.webPath("job-request.json"), raw, 0600)
	if err = m.Worker(id); err != nil {
		t.Fatal(err)
	}
	after, err := m.Status()
	if err != nil {
		t.Fatal(err)
	}
	if after.Job == nil || after.Job.Kind != "probe-nodes" || after.Job.Status != "succeeded" || after.Job.Message != "测速已完成" || after.Nodes[0].Delay == nil || *after.Nodes[0].Delay != 47 || after.Nodes[0].Alive == nil || !*after.Nodes[0].Alive {
		t.Fatal("poll did not expose completed measurement")
	}
	if after.SelectedID != before.SelectedID || after.DesiredEnabled != before.DesiredEnabled {
		t.Fatal("poll changed user settings")
	}
}
func TestInterruptedProbeDoesNotClaimConfigurationRollback(t *testing.T) {
	m, _, _, _ := setupUpdate(t)
	id := strings.Repeat("c", 32)
	m.writeJob(jobDisk{Job: Job{ID: id, Kind: "probe-nodes", Status: "running"}, CreatedAt: m.Now().Add(-10 * time.Second).UnixMilli()})
	m.reapJob()
	job, err := m.readJob()
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "failed" || job.Message != "上次测速被中断，请重试" || strings.Contains(job.Message, "恢复") {
		t.Fatal("probe interruption was described as configuration rollback")
	}
}

func TestProbeQueueRetainsReplayAndOperationLock(t *testing.T) {
	m, _, _, _ := setupUpdate(t)
	m.createKey()
	key, _ := m.privateKey()
	r := Request{V: 1, IssuedAt: m.Now().UnixMilli(), Nonce: base64.RawURLEncoding.EncodeToString(make([]byte, 16)), Action: "probe-nodes"}
	encoded := seal(t, key, r)
	unlock, err := acquireLock(m.webPath("manager.lock"), false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Queue(encoded)
	unlock()
	if ErrorCode(err) != "busy" {
		t.Fatal("probe bypassed lock")
	}
	_, err = m.Queue(encoded)
	if ErrorCode(err) != "operation_failed" {
		t.Fatalf("expected absent test worker, got %v", err)
	}
	_, err = m.Queue(encoded)
	if ErrorCode(err) != "replayed_request" {
		t.Fatalf("probe nonce replay accepted: %v", err)
	}
}

func serveProbeController(t *testing.T, handler http.HandlerFunc) *UnixController {
	t.Helper()
	dir, err := os.MkdirTemp("", "u30p-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "s")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Skip("Host Unix socket unavailable")
	}
	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	return &UnixController{Socket: socket}
}
func invokeProbe(ctx context.Context, core *UnixController) error {
	method, ok := any(core).(interface{ Probe(context.Context) error })
	if !ok {
		return fmt.Errorf("manual Probe operation is not implemented")
	}
	return method.Probe(ctx)
}

const probeURLForTest = "https://www.gstatic.com/generate_204"

func providerForProbe(ids []string, measured bool, extraAlive map[string]bool) map[string]any {
	proxies := []any{}
	for _, id := range ids {
		p := map[string]any{"name": id, "alive": true, "history": []any{map[string]any{"delay": 1}}}
		if measured {
			alive := true
			if value, ok := extraAlive[id]; ok {
				alive = value
			}
			delay := uint16(25)
			if !alive {
				delay = 0
			}
			p["extra"] = map[string]any{probeURLForTest: map[string]any{"alive": alive, "history": []any{map[string]any{"delay": delay}}}}
		}
		proxies = append(proxies, p)
	}
	return map[string]any{"proxies": proxies}
}

func TestControllerProbeUsesOnlySubscriptionAndCurrentRoundToSelectInnerAuto(t *testing.T) {
	ids := []string{"n-2222222222222222", "n-1111111111111111", "n-3333333333333333", "n-4444444444444444"}
	var measured atomic.Bool
	var mu sync.Mutex
	calls := map[string]int{}
	autoSelected := "old"
	outerSelected := "manual-stays"
	core := serveProbeController(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls[r.Method+" "+r.URL.Path]++
		mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/providers/proxies/Subscription":
			json.NewEncoder(w).Encode(providerForProbe(ids, measured.Load(), map[string]bool{ids[2]: false, ids[3]: false}))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/providers/proxies/Subscription/") && strings.HasSuffix(r.URL.Path, "/healthcheck"):
			q := r.URL.Query()
			if len(q) != 3 || q.Get("url") != probeURLForTest || q.Get("timeout") != "5000" || q.Get("expected") != "204" {
				t.Error("caller-controlled or missing probe parameters")
			}
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/providers/proxies/Subscription/"), "/healthcheck")
			if id == ids[3] {
				w.WriteHeader(503)
				w.Write([]byte(`{"message":"SECRET"}`))
			} else {
				json.NewEncoder(w).Encode(map[string]int{"delay": 25})
			}
			measured.Store(true)
		case r.Method == http.MethodPut && r.URL.Path == "/proxies/Auto":
			var body struct {
				Name string `json:"name"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			autoSelected = body.Name
			mu.Unlock()
			w.WriteHeader(204)
		default:
			t.Errorf("probe escaped fixed provider/Auto routes: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	})
	if err := invokeProbe(context.Background(), core); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if autoSelected != ids[1] || outerSelected != "manual-stays" {
		t.Fatal("probe did not use fresh successful delays and stable-ID tie break")
	}
	for _, id := range ids {
		if calls["GET /providers/proxies/Subscription/"+id+"/healthcheck"] != 1 {
			t.Fatal("node was not measured exactly once")
		}
	}
	if calls["PUT /proxies/Auto"] != 1 || calls["PUT /proxies/VPN"] != 0 {
		t.Fatal("unexpected selector mutation")
	}
}

func TestControllerProbeAllFailedCompletesWithoutChangingAuto(t *testing.T) {
	ids := []string{"n-1111111111111111", "n-2222222222222222"}
	puts := 0
	core := serveProbeController(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/providers/proxies/Subscription" {
			json.NewEncoder(w).Encode(providerForProbe(ids, false, nil))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/healthcheck") {
			if strings.Contains(r.URL.Path, ids[0]) {
				w.WriteHeader(503)
			} else {
				w.WriteHeader(504)
			}
			w.Write([]byte(`{"message":"SECRET"}`))
			return
		}
		puts++
		w.WriteHeader(404)
	})
	if err := invokeProbe(context.Background(), core); err != nil {
		t.Fatalf("failed measurements are still a completed round: %v", err)
	}
	if puts != 0 {
		t.Fatal("all-failed round changed selector")
	}
}

func TestControllerProbeSelectsLowestCurrentDelayNotOldHistory(t *testing.T) {
	ids := []string{"n-1111111111111111", "n-2222222222222222", "n-3333333333333333"}
	delays := map[string]uint16{ids[0]: 40, ids[1]: 12, ids[2]: 30}
	selected := "unchanged"
	core := serveProbeController(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/providers/proxies/Subscription" {
			provider := providerForProbe(ids, true, nil)
			for _, value := range provider["proxies"].([]any) {
				p := value.(map[string]any)
				id := p["name"].(string)
				p["extra"] = map[string]any{probeURLForTest: map[string]any{"alive": true, "history": []any{map[string]any{"delay": delays[id]}}}}
			}
			json.NewEncoder(w).Encode(provider)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/healthcheck") {
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/providers/proxies/Subscription/"), "/healthcheck")
			json.NewEncoder(w).Encode(map[string]uint16{"delay": delays[id]})
			return
		}
		if r.Method == "PUT" && r.URL.Path == "/proxies/Auto" {
			var body struct {
				Name string `json:"name"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			selected = body.Name
			w.WriteHeader(204)
			return
		}
		t.Error("unexpected endpoint")
	})
	if err := invokeProbe(context.Background(), core); err != nil {
		t.Fatal(err)
	}
	if selected != ids[1] {
		t.Fatal("best node was not chosen from this round's actual replies")
	}
}

func TestControllerProbeBoundsConcurrencyAndCancellation(t *testing.T) {
	ids := []string{}
	for i := 0; i < 12; i++ {
		ids = append(ids, fmt.Sprintf("n-%016x", i+1))
	}
	var active, maxActive, started atomic.Int32
	release := make(chan struct{})
	four := make(chan struct{})
	var once sync.Once
	core := serveProbeController(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/providers/proxies/Subscription" {
			json.NewEncoder(w).Encode(providerForProbe(ids, true, nil))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/healthcheck") {
			now := active.Add(1)
			defer active.Add(-1)
			started.Add(1)
			for {
				old := maxActive.Load()
				if now <= old || maxActive.CompareAndSwap(old, now) {
					break
				}
			}
			if now == 4 {
				once.Do(func() { close(four) })
			}
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
			json.NewEncoder(w).Encode(map[string]int{"delay": 25})
			return
		}
		if r.Method == "PUT" && r.URL.Path == "/proxies/Auto" {
			w.WriteHeader(204)
			return
		}
		t.Error("unexpected route")
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- invokeProbe(ctx, core) }()
	select {
	case <-four:
	case <-time.After(2 * time.Second):
		cancel()
		close(release)
		t.Fatal("bounded workers did not start")
	}
	if maxActive.Load() != 4 {
		t.Error("unexpected concurrency limit")
	}
	cancel()
	select {
	case err := <-done:
		if ErrorCode(err) != "probe_failed" {
			t.Errorf("cancel error=%v", err)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("probe did not cancel promptly")
	}
	close(release)
	if started.Load() > 4 {
		t.Fatal("cancellation launched additional probes")
	}
}

func TestControllerProbeRejectsUnsafeNamesAndProtocolFailures(t *testing.T) {
	for _, kind := range []string{"unsafe-name", "duplicate-name", "transport", "bad-delay", "oversize", "bad-status"} {
		t.Run(kind, func(t *testing.T) {
			id := "n-1111111111111111"
			healthcalls := 0
			core := serveProbeController(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/providers/proxies/Subscription" {
					ids := []string{id}
					if kind == "unsafe-name" {
						ids = []string{"Auto/../../SECRET"}
					}
					if kind == "duplicate-name" {
						ids = append(ids, id)
					}
					json.NewEncoder(w).Encode(providerForProbe(ids, true, nil))
					return
				}
				healthcalls++
				switch kind {
				case "transport":
					conn, _, _ := w.(http.Hijacker).Hijack()
					conn.Close()
				case "bad-delay":
					w.Write([]byte(`{"delay":0}`))
				case "oversize":
					w.Write([]byte(strings.Repeat("SECRET", 1024)))
				default:
					w.WriteHeader(500)
					w.Write([]byte("SECRET"))
				}
			})
			err := invokeProbe(context.Background(), core)
			if ErrorCode(err) != "probe_failed" || strings.Contains(err.Error(), "SECRET") {
				t.Fatalf("unsafe failure result: %v", err)
			}
			if (kind == "unsafe-name" || kind == "duplicate-name") && healthcalls != 0 {
				t.Fatal("unsafe provider reached a probe")
			}
		})
	}
}

func TestSnapshotPrefersFixedProbeURLExtraOverMisleadingTopLevelHistory(t *testing.T) {
	core := serveProbeController(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/proxies/VPN":
			w.Write([]byte(`{"now":"manual"}`))
		case "/providers/proxies/Subscription":
			w.Write([]byte(`{"proxies":[{"name":"n-1111111111111111","alive":true,"history":[{"delay":12}],"extra":{"https://www.gstatic.com/generate_204":{"alive":false,"history":[{"delay":0}]}}}]}`))
		default:
			t.Error("snapshot started a probe")
			w.WriteHeader(404)
		}
	})
	snap, err := core.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	node := snap.Nodes["n-1111111111111111"]
	if node.Alive == nil || *node.Alive || node.Delay == nil || *node.Delay != 0 {
		t.Fatal("snapshot ignored failed expected-status probe")
	}
}
