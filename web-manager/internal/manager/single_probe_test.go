package manager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSingleProbeEncryptedShapeRequiresOnlyStableID(t *testing.T) {
	m := testManager(t)
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	base := fmt.Sprintf(`{"v":1,"issuedAt":%d,"nonce":"AAAAAAAAAAAAAAAAAAAAAA","action":"probe-node","id":"n-1111111111111111"}`, m.Now().UnixMilli())
	if got, err := Decrypt(sealProbePlain(t, key, []byte(base)), key, m.Now()); err != nil || got.ID != "n-1111111111111111" {
		t.Fatalf("single probe rejected: %v", err)
	}
	for _, plain := range []string{
		strings.Replace(base, `,"id":"n-1111111111111111"`, "", 1),
		strings.Replace(base, `"n-1111111111111111"`, `"Auto"`, 1),
		strings.Replace(base, `"n-1111111111111111"`, `"n-ABCDABCDABCDABCD"`, 1),
		strings.Replace(base, `"n-1111111111111111"`, `null`, 1),
		strings.TrimSuffix(base, "}") + `,"enabled":null}`,
		strings.TrimSuffix(base, "}") + `,"enabled":false}`,
		strings.TrimSuffix(base, "}") + `,"url":""}`,
		strings.TrimSuffix(base, "}") + `,"url":null}`,
		strings.TrimSuffix(base, "}") + `,"command":"id"}`,
		strings.TrimSuffix(base, "}") + `,"id":"n-2222222222222222"}`,
	} {
		if _, err := Decrypt(sealProbePlain(t, key, []byte(plain)), key, m.Now()); err == nil {
			t.Error("single probe accepted invalid shape")
		}
	}
}

type singleProbeCore struct {
	Controller
	ids        []string
	batchCalls int
	err        error
	ctx        context.Context
}

func (c *singleProbeCore) Probe(context.Context) error { c.batchCalls++; return nil }
func (c *singleProbeCore) ProbeNode(ctx context.Context, id string) error {
	c.ids = append(c.ids, id)
	c.ctx = ctx
	return c.err
}

func setSingleProbeJob(t *testing.T, m *Manager, id, nodeID, request string) {
	t.Helper()
	jobRaw, _ := json.Marshal(map[string]any{"id": id, "kind": "probe-node", "nodeId": nodeID, "status": "running", "createdAt": m.Now().UnixMilli()})
	var job jobDisk
	json.Unmarshal(jobRaw, &job)
	if err := m.writeJob(job); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(m.webPath("job-request.json"), []byte(request), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestSingleProbeWorkerTargetsOneSavedAndLiveNodeWithoutChangingConfig(t *testing.T) {
	for _, kind := range []string{"success", "saved-unknown", "provider-unknown", "off", "core-stopped", "error", "controller-unknown"} {
		t.Run(kind, func(t *testing.T) {
			m, base, state, _ := setupUpdate(t)
			core := &singleProbeCore{Controller: base}
			m.Core = core
			target := state.Nodes[0].ID
			wantCode := ""
			wantCalls := 1
			switch kind {
			case "saved-unknown":
				target = "n-9999999999999999"
				wantCode = "invalid_node"
				wantCalls = 0
			case "provider-unknown":
				atomicWrite(m.providerPath(), []byte(`proxies: [{name: replacement, type: vless, server: new.example.com, port: 443, uuid: 00000000-0000-0000-0000-000000000002}]`), 0600)
				wantCode = "invalid_node"
				wantCalls = 0
			case "off":
				state.DesiredEnabled = false
				m.writeState(state)
				wantCode = "vpn_stopped"
				wantCalls = 0
			case "core-stopped":
				base.running = false
				wantCode = "vpn_stopped"
				wantCalls = 0
			case "error":
				core.err = fail("apply_failed")
				wantCode = "probe_failed"
			case "controller-unknown":
				core.err = fail("invalid_node")
				wantCode = "invalid_node"
			}
			controls := 0
			m.Control = func(context.Context, string) error { controls++; return nil }
			beforeState, _ := os.ReadFile(m.webPath("state.json"))
			beforeProvider, _ := os.ReadFile(m.providerPath())
			request, _ := json.Marshal(Request{V: 1, Action: "probe-node", ID: target})
			id := strings.Repeat("d", 32)
			setSingleProbeJob(t, m, id, target, string(request))
			err := m.Worker(id)
			if err == nil && wantCode != "" || err != nil && ErrorCode(err) != wantCode {
				t.Fatalf("unexpected result: %v want %s", err, wantCode)
			}
			if len(core.ids) != wantCalls || core.batchCalls != 0 || controls != 0 || base.selects != 0 || base.refreshes != 0 {
				t.Fatal("single probe affected other nodes, selection or service")
			}
			if len(core.ids) > 0 {
				if core.ids[0] != target {
					t.Fatal("wrong probe target")
				}
				deadline, ok := core.ctx.Deadline()
				if !ok || time.Until(deadline) > 12*time.Second {
					t.Fatal("single-node context exceeds 12 seconds")
				}
			}
			job, err := m.readJob()
			if err != nil {
				t.Fatal(err)
			}
			public, _ := json.Marshal(job.Job)
			var fields map[string]any
			json.Unmarshal(public, &fields)
			if fields["nodeId"] != target || job.Kind != "probe-node" {
				t.Fatal("job did not retain target identity")
			}
			if wantCode == "" && (job.Status != "succeeded" || job.Message != "单节点测速已完成") {
				t.Fatal("single-node completion missing")
			}
			afterState, _ := os.ReadFile(m.webPath("state.json"))
			afterProvider, _ := os.ReadFile(m.providerPath())
			if string(beforeState) != string(afterState) || string(beforeProvider) != string(afterProvider) {
				t.Fatal("single probe wrote configuration")
			}
		})
	}
}

func TestSingleProbeWorkerIndependentlyRejectsForbiddenShapeAndIdentityMismatch(t *testing.T) {
	for _, kind := range []string{"extra-url-null", "extra-enabled-null", "unknown-null", "malformed-json", "missing-id", "auto", "duplicate-id", "job-id-mismatch", "job-action-mismatch"} {
		t.Run(kind, func(t *testing.T) {
			m, base, state, _ := setupUpdate(t)
			core := &singleProbeCore{Controller: base}
			m.Core = core
			target := state.Nodes[0].ID
			request := fmt.Sprintf(`{"v":1,"issuedAt":%d,"nonce":"AAAAAAAAAAAAAAAAAAAAAA","action":"probe-node","id":"%s"}`, m.Now().UnixMilli(), target)
			jobTarget := target
			switch kind {
			case "unknown-null":
				request = strings.TrimSuffix(request, "}") + `,"command":null}`
			case "malformed-json":
				request = `{`
			case "extra-url-null":
				request = strings.TrimSuffix(request, "}") + `,"url":null}`
			case "extra-enabled-null":
				request = strings.TrimSuffix(request, "}") + `,"enabled":null}`
			case "missing-id":
				request = strings.Replace(request, `,"id":"`+target+`"`, "", 1)
			case "auto":
				request = strings.Replace(request, target, "Auto", 1)
			case "duplicate-id":
				request = strings.TrimSuffix(request, "}") + `,"id":"` + target + `"}`
			case "job-id-mismatch":
				jobTarget = "n-2222222222222222"
			case "job-action-mismatch":
				request = strings.Replace(request, "probe-node", "select", 1)
			}
			id := strings.Repeat("e", 32)
			setSingleProbeJob(t, m, id, jobTarget, request)
			if err := m.Worker(id); ErrorCode(err) != "invalid_request" {
				t.Fatalf("worker accepted invalid single request: %v", err)
			}
			if len(core.ids) != 0 || base.selects != 0 || base.refreshes != 0 || core.batchCalls != 0 {
				t.Fatal("invalid worker request reached controller")
			}
			job, readErr := m.readJob()
			if readErr != nil || job.Status != "failed" {
				t.Fatal("invalid single request left an active job")
			}
		})
	}
}
func TestSingleProbeQueueKeepsLockReplayAndPublicNodeID(t *testing.T) {
	m, _, state, _ := setupUpdate(t)
	m.createKey()
	key, _ := m.privateKey()
	target := state.Nodes[0].ID
	r := Request{V: 1, IssuedAt: m.Now().UnixMilli(), Nonce: base64.RawURLEncoding.EncodeToString(make([]byte, 16)), Action: "probe-node", ID: target}
	encoded := seal(t, key, r)
	unlock, _ := acquireLock(m.webPath("manager.lock"), false)
	_, err := m.Queue(encoded)
	unlock()
	if ErrorCode(err) != "busy" {
		t.Fatal("single probe bypassed manager lock")
	}
	_, err = m.Queue(encoded)
	if ErrorCode(err) != "operation_failed" {
		t.Fatalf("expected missing host worker, got %v", err)
	}
	job, _ := m.readJob()
	data, _ := json.Marshal(job.Job)
	var fields map[string]any
	json.Unmarshal(data, &fields)
	if fields["nodeId"] != target || job.Kind != "probe-node" {
		t.Fatal("queue omitted row identity")
	}
	_, err = m.Queue(encoded)
	if ErrorCode(err) != "replayed_request" {
		t.Fatal("single probe replay accepted")
	}
}

func invokeSingleProbe(ctx context.Context, core *UnixController, id string) error {
	method, ok := any(core).(interface {
		ProbeNode(context.Context, string) error
	})
	if !ok {
		return fmt.Errorf("single ProbeNode operation is not implemented")
	}
	return method.ProbeNode(ctx, id)
}
func TestControllerSingleProbeUsesExactlyOneTargetAndNoSelector(t *testing.T) {
	for _, status := range []int{200, 503, 504} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			id := "n-1111111111111111"
			core := serveProbeController(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/providers/proxies/Subscription/"+id+"/healthcheck" {
					t.Error("single probe escaped its one target")
				}
				q := r.URL.Query()
				if len(q) != 3 || q.Get("url") != probeURLForTest || q.Get("timeout") != "5000" || q.Get("expected") != "204" {
					t.Error("single probe parameters changed")
				}
				w.WriteHeader(status)
				if status == 200 {
					w.Write([]byte(`{"delay":37}`))
				} else {
					w.Write([]byte(`{"message":"SECRET"}`))
				}
			})
			if err := invokeSingleProbe(context.Background(), core, id); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatal("single probe made more than one request")
			}
		})
	}
}
func TestControllerSingleProbeRejectsUnsafeUnknownAndProtocolErrors(t *testing.T) {
	for _, kind := range []string{"auto", "unsafe", "unknown", "oversize", "bad-json", "bad-delay", "bad-status"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			id := "n-1111111111111111"
			wantCode := "probe_failed"
			if kind == "auto" {
				id = "Auto"
				wantCode = "invalid_node"
			}
			if kind == "unsafe" {
				id = "n-/../SECRET"
				wantCode = "invalid_node"
			}
			if kind == "unknown" {
				wantCode = "invalid_node"
			}
			core := serveProbeController(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch kind {
				case "unknown":
					w.WriteHeader(404)
				case "oversize":
					w.Write([]byte(strings.Repeat("SECRET", 1024)))
				case "bad-json":
					w.Write([]byte("SECRET"))
				case "bad-delay":
					w.Write([]byte(`{"delay":0}`))
				default:
					w.WriteHeader(500)
				}
			})
			err := invokeSingleProbe(context.Background(), core, id)
			if ErrorCode(err) != wantCode || strings.Contains(err.Error(), "SECRET") {
				t.Fatalf("unsafe failure: %v", err)
			}
			if (kind == "auto" || kind == "unsafe") && calls != 0 {
				t.Fatal("unsafe ID reached HTTP")
			}
		})
	}
}
func TestControllerSingleProbeCancelsWithoutOtherMeasurements(t *testing.T) {
	started := make(chan struct{})
	calls := 0
	core := serveProbeController(t, func(w http.ResponseWriter, r *http.Request) { calls++; close(started); <-r.Context().Done() })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- invokeSingleProbe(ctx, core, "n-1111111111111111") }()
	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("single measurement not started")
	}
	cancel()
	select {
	case err := <-done:
		if ErrorCode(err) != "probe_failed" {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("single probe did not cancel")
	}
	if calls != 1 {
		t.Fatal("cancel caused another request")
	}
}
func TestStatusNeverStartsSingleProbe(t *testing.T) {
	m, base, _, _ := setupUpdate(t)
	m.createKey()
	core := &singleProbeCore{Controller: base}
	m.Core = core
	for n := 0; n < 3; n++ {
		if _, err := m.Status(); err != nil {
			t.Fatal(err)
		}
	}
	if len(core.ids) != 0 || core.batchCalls != 0 {
		t.Fatal("polling initiated a measurement")
	}
}
