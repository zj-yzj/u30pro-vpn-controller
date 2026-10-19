package manager

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"u30pro/webmanager/internal/subscription"
)

func (m *Manager) Init() error {
	if err := m.storageReady(); err != nil {
		return err
	}
	unlock, err := acquireLock(m.webPath("manager.lock"), false)
	if err != nil {
		return err
	}
	defer unlock()
	if err = m.createKey(); err != nil {
		return err
	}
	if _, err = os.Lstat(m.webPath("state.json")); err == nil {
		s, readErr := m.readState()
		if readErr != nil {
			return readErr
		}
		if s.SubscriptionURL == "" {
			initialURL, e := m.initialSubscription()
			if e != nil {
				return e
			}
			if initialURL != "" {
				s.SubscriptionURL = initialURL
				return m.writeState(s)
			}
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fail("storage_failed")
	}
	b, err := readBounded(m.webPath("initial-nodes.json"), 4<<20)
	if err != nil {
		return fail("not_initialized")
	}
	var nodes []subscription.Node
	if json.Unmarshal(b, &nodes) != nil || len(nodes) == 0 || len(nodes) > 4096 {
		return fail("not_initialized")
	}
	seen := map[string]bool{}
	for _, n := range nodes {
		if !stableID.MatchString(n.ID) || seen[n.ID] || n.Name == "" {
			return fail("not_initialized")
		}
		seen[n.ID] = true
	}
	provider, err := readBounded(m.providerPath(), 8<<20)
	if err != nil || len(provider) == 0 {
		return fail("not_initialized")
	}
	initialURL, err := m.initialSubscription()
	if err != nil {
		return err
	}
	return m.writeState(State{Schema: 1, DesiredEnabled: true, SelectedID: "Auto", Nodes: nodes, SubscriptionURL: initialURL})
}
func (m *Manager) initialSubscription() (string, error) {
	path := m.webPath("initial-subscription.txt")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil || !info.Mode().IsRegular() || (!m.Test && (info.Mode().Perm() != 0600 || !ownedRoot(info))) {
		return "", fail("insecure_storage")
	}
	b, err := readBounded(path, 8193)
	if err != nil {
		return "", fail("storage_failed")
	}
	raw := strings.TrimSpace(string(b))
	if _, err = subscriptionURL(raw); err != nil {
		return "", fail("invalid_request")
	}
	return raw, nil
}
func (m *Manager) Status() (Status, error) {
	out := Status{OK: true, Installed: true, Nodes: []PublicNode{}, ServerTime: m.Now().UnixMilli()}
	if err := m.storageReady(); err != nil {
		return out, err
	}
	unlock, lockErr := acquireLock(m.webPath("manager.lock"), false)
	busy := lockErr != nil && ErrorCode(lockErr) == "busy"
	if lockErr != nil && !busy {
		return out, lockErr
	}
	if !busy {
		defer unlock()
		if err := m.recoverTransaction(); err != nil {
			return out, err
		}
		m.reapJob()
	}
	s, err := m.readState()
	if err != nil {
		return out, err
	}
	out.DesiredEnabled = s.DesiredEnabled
	out.SelectedID = s.SelectedID
	out.Message = s.Message
	out.PublicKeyPEM, err = m.publicKey()
	if err != nil {
		return out, err
	}
	host := ""
	if u, e := url.Parse(s.SubscriptionURL); e == nil {
		host = u.Hostname()
	}
	out.Subscription = PublicSubscription{s.SubscriptionURL != "", s.UpdatedAt, host, len(s.Nodes)}
	snap := NativeSnapshot{}
	if m.Core != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var snapshotErr error
		snap, snapshotErr = m.Core.Snapshot(ctx)
		if snapshotErr != nil {
			snap = NativeSnapshot{}
		}
		cancel()
	}
	out.Running = snap.Running
	out.ActiveID = snap.Active
	for _, n := range s.Nodes {
		p := PublicNode{ID: n.ID, Name: n.Name, Type: n.Type}
		if v, ok := snap.Nodes[n.ID]; ok {
			p.Delay = v.Delay
			p.Alive = v.Alive
		}
		out.Nodes = append(out.Nodes, p)
	}
	svc, _ := readBounded(filepath.Join(m.Base, "state"), 128)
	if busy {
		out.State = "busy"
	} else if snap.Running && strings.TrimSpace(string(svc)) == "RUNNING" && s.DesiredEnabled {
		out.State = "running"
	} else if !snap.Running && !s.DesiredEnabled {
		out.State = "stopped"
	} else if strings.TrimSpace(string(svc)) == "STARTING" {
		out.State = "starting"
	} else {
		out.State = "error"
	}
	if j, e := m.readJob(); e == nil {
		public := j.Job
		out.Job = &public
	}
	return out, nil
}
func (m *Manager) Boot() error {
	if err := m.storageReady(); err != nil {
		return err
	}
	unlock, err := acquireLock(m.webPath("manager.lock"), true)
	if err != nil {
		return err
	}
	defer unlock()
	if err = m.recoverTransaction(); err != nil {
		return err
	}
	m.reapJob()
	s, err := m.readState()
	if err != nil {
		return err
	}
	if !s.DesiredEnabled {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()
	if err = m.Control(ctx, "start"); err != nil {
		return err
	}
	return m.waitAndSelect(ctx, s.SelectedID)
}
func (m *Manager) waitAndSelect(ctx context.Context, id string) error {
	for attempt := 0; attempt < 30; attempt++ {
		if snapshot, err := m.Core.Snapshot(ctx); err == nil && snapshot.Running {
			if err = m.Core.Select(ctx, id); err == nil {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fail("control_failed")
		case <-time.After(time.Second):
		}
	}
	return fail("control_failed")
}
func (m *Manager) perform(ctx context.Context, r Request) error {
	s, err := m.readState()
	if err != nil {
		return err
	}
	switch r.Action {
	case "set-enabled":
		enabled := *r.Enabled
		if enabled && len(s.Nodes) == 0 {
			return fail("not_configured")
		}
		s.DesiredEnabled = enabled
		s.Message = ""
		if err = m.writeState(s); err != nil {
			return err
		}
		action := "stop"
		if enabled {
			action = "start"
		}
		if err = m.Control(ctx, action); err != nil {
			return err
		}
		if enabled {
			return m.waitAndSelect(ctx, s.SelectedID)
		}
		return nil
	case "select":
		found := r.ID == "Auto"
		for _, n := range s.Nodes {
			found = found || n.ID == r.ID
		}
		if !found {
			return fail("invalid_node")
		}
		previous := s.SelectedID
		if s.DesiredEnabled {
			if err = m.Core.Select(ctx, r.ID); err != nil {
				return fail("apply_failed")
			}
		}
		s.SelectedID = r.ID
		s.Message = ""
		if err = m.writeState(s); err != nil {
			if s.DesiredEnabled {
				m.Core.Select(ctx, previous)
			}
			return err
		}
		return nil
	case "update-subscription":
		return m.update(ctx, r.URL, s)
	case "probe-node":
		if r.Enabled != nil || r.URL != "" || !stableID.MatchString(r.ID) {
			return fail("invalid_request")
		}
		found := false
		for _, node := range s.Nodes {
			found = found || node.ID == r.ID
		}
		if !found {
			return fail("invalid_node")
		}
		if !s.DesiredEnabled || m.Core == nil {
			return fail("vpn_stopped")
		}
		probeCtx, cancel := context.WithTimeout(ctx, singleProbeTotalTimeout)
		defer cancel()
		snapshot, err := m.Core.Snapshot(probeCtx)
		if err != nil || !snapshot.Running {
			return fail("vpn_stopped")
		}
		if _, exists := snapshot.Nodes[r.ID]; !exists {
			return fail("invalid_node")
		}
		if err = m.Core.ProbeNode(probeCtx, r.ID); err != nil {
			if ErrorCode(err) == "invalid_node" {
				return err
			}
			return fail("probe_failed")
		}
		return nil
	case "probe-nodes":
		if r.Enabled != nil || r.ID != "" || r.URL != "" {
			return fail("invalid_request")
		}
		if !s.DesiredEnabled || m.Core == nil {
			return fail("vpn_stopped")
		}
		probeCtx, cancel := context.WithTimeout(ctx, manualProbeTotalTimeout)
		defer cancel()
		snapshot, err := m.Core.Snapshot(probeCtx)
		if err != nil || !snapshot.Running {
			return fail("vpn_stopped")
		}
		if m.Core.Probe(probeCtx) != nil {
			return fail("probe_failed")
		}
		return nil
	}
	return fail("invalid_request")
}
func (m *Manager) update(ctx context.Context, newURL string, s State) error {
	target := newURL
	if target == "" {
		target = s.SubscriptionURL
	}
	if _, err := subscriptionURL(target); err != nil {
		return fail("download_failed")
	}
	raw, err := m.Fetch(ctx, target)
	if err != nil {
		return fail("download_failed")
	}
	normalized, err := subscription.Normalize(raw)
	if err != nil {
		return fail("invalid_subscription")
	}
	validationCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	err = m.Validate(validationCtx, normalized.Proxies)
	cancel()
	if err != nil {
		return fail("invalid_subscription")
	}
	next := s
	next.SubscriptionURL = target
	next.Nodes = normalized.Nodes
	next.UpdatedAt = m.Now().UTC().Format(time.RFC3339)
	next.Message = ""
	found := s.SelectedID == "Auto"
	for _, n := range next.Nodes {
		found = found || n.ID == s.SelectedID
	}
	if !found {
		next.SelectedID = "Auto"
		next.Message = "原来选择的节点已移除，已恢复自动选择"
	}
	if err = m.beginTransaction(normalized.Provider, next); err != nil {
		return err
	}
	rollback := func() error {
		if m.recoverTransaction() != nil {
			return fail("rollback_failed")
		}
		return fail("apply_failed")
	}
	if err = atomicWrite(m.providerPath(), normalized.Provider, 0600); err != nil {
		return rollback()
	}
	if s.DesiredEnabled {
		snapshot, e := m.Core.Snapshot(ctx)
		if e != nil || !snapshot.Running {
			return rollback()
		}
		if m.Core.Refresh(ctx) != nil {
			return rollback()
		}
		snapshot, e = m.Core.Snapshot(ctx)
		if e != nil || len(snapshot.Nodes) != len(next.Nodes) {
			return rollback()
		}
		for _, n := range next.Nodes {
			if _, exists := snapshot.Nodes[n.ID]; !exists {
				return rollback()
			}
		}
		if m.Core.Select(ctx, next.SelectedID) != nil || m.Core.Health(ctx) != nil {
			return rollback()
		}
	}
	if err = m.writeState(next); err != nil {
		return rollback()
	}
	if err = m.commitTransaction(); err != nil {
		return rollback()
	}
	return m.cleanupTransaction()
}
