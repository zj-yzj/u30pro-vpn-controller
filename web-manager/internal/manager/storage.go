package manager

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

func readBounded(path string, limit int64) ([]byte, error) {
	s, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !s.Mode().IsRegular() || s.Size() > limit {
		return nil, fail("storage_failed")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, fail("storage_failed")
	}
	return b, nil
}
func atomicWrite(path string, b []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".webctl-")
	if err != nil {
		return fail("storage_failed")
	}
	temp := f.Name()
	defer os.Remove(temp)
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fail("storage_failed")
	}
	if err = os.Rename(temp, path); err != nil {
		return fail("storage_failed")
	}
	return syncDir(dir)
}
func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fail("storage_failed")
	}
	defer f.Close()
	if err = f.Sync(); err != nil && !directorySyncUnsupported(err) {
		return fail("storage_failed")
	}
	return nil
}
func digest(b []byte) string { d := sha256.Sum256(b); return hex.EncodeToString(d[:]) }
func (m *Manager) storageReady() error {
	for _, d := range []string{m.Base, filepath.Join(m.Base, "web"), filepath.Join(m.Base, "cache")} {
		s, err := os.Lstat(d)
		if err != nil || !s.IsDir() || s.Mode()&os.ModeSymlink != 0 || (!m.Test && s.Mode().Perm()&0077 != 0) {
			return fail("insecure_storage")
		}
		if !m.Test && !ownedRoot(s) {
			return fail("insecure_storage")
		}
	}
	return nil
}
func (m *Manager) readState() (State, error) {
	var s State
	b, err := readBounded(m.webPath("state.json"), 4<<20)
	if err != nil || json.Unmarshal(b, &s) != nil || s.Schema != 1 || (s.SelectedID != "Auto" && !stableID.MatchString(s.SelectedID)) {
		return s, fail("not_initialized")
	}
	return s, nil
}
func (m *Manager) writeState(s State) error {
	b, err := json.Marshal(s)
	if err != nil {
		return fail("storage_failed")
	}
	return atomicWrite(m.webPath("state.json"), b, 0600)
}

type journal struct {
	Phase           string `json:"phase"`
	OldProviderHash string `json:"oldProviderHash"`
	OldStateHash    string `json:"oldStateHash"`
	NewProviderHash string `json:"newProviderHash"`
	NewStateHash    string `json:"newStateHash"`
}

func (m *Manager) txPath(n string) string { return m.webPath("txn-" + n) }
func (m *Manager) beginTransaction(provider []byte, s State) error {
	if _, err := os.Lstat(m.txPath("journal.json")); !os.IsNotExist(err) {
		return fail("busy")
	}
	oldProvider, err := readBounded(m.providerPath(), 8<<20)
	if err != nil {
		return fail("storage_failed")
	}
	oldState, err := readBounded(m.webPath("state.json"), 4<<20)
	if err != nil {
		return fail("storage_failed")
	}
	newState, err := json.Marshal(s)
	if err != nil {
		return fail("storage_failed")
	}
	for _, item := range []struct {
		name string
		b    []byte
	}{{"old-provider.yaml", oldProvider}, {"old-state.json", oldState}, {"new-provider.yaml", provider}, {"new-state.json", newState}} {
		if err = atomicWrite(m.txPath(item.name), item.b, 0600); err != nil {
			return err
		}
	}
	j := journal{"PREPARED", digest(oldProvider), digest(oldState), digest(provider), digest(newState)}
	b, _ := json.Marshal(j)
	return atomicWrite(m.txPath("journal.json"), b, 0600)
}
func (m *Manager) commitTransaction() error {
	b, err := readBounded(m.txPath("journal.json"), 4096)
	var j journal
	if err != nil || json.Unmarshal(b, &j) != nil {
		return fail("storage_failed")
	}
	j.Phase = "COMMITTED"
	b, _ = json.Marshal(j)
	return atomicWrite(m.txPath("journal.json"), b, 0600)
}
func (m *Manager) cleanupTransaction() error {
	for _, name := range []string{"old-provider.yaml", "old-state.json", "new-provider.yaml", "new-state.json"} {
		if err := os.Remove(m.txPath(name)); err != nil && !os.IsNotExist(err) {
			return fail("storage_failed")
		}
	}
	if err := os.Remove(m.txPath("journal.json")); err != nil && !os.IsNotExist(err) {
		return fail("storage_failed")
	}
	return syncDir(filepath.Join(m.Base, "web"))
}
func (m *Manager) recoverTransaction() error {
	b, err := readBounded(m.txPath("journal.json"), 4096)
	if os.IsNotExist(err) {
		return nil
	}
	var j journal
	if err != nil || json.Unmarshal(b, &j) != nil {
		return fail("rollback_failed")
	}
	provider, pe := readBounded(m.providerPath(), 8<<20)
	state, se := readBounded(m.webPath("state.json"), 4<<20)
	if j.Phase == "ROLLED_BACK" && pe == nil && se == nil && digest(provider) == j.OldProviderHash && digest(state) == j.OldStateHash {
		return m.cleanupTransaction()
	}
	if j.Phase == "COMMITTED" && pe == nil && se == nil && digest(provider) == j.NewProviderHash && digest(state) == j.NewStateHash {
		return m.cleanupTransaction()
	}
	if j.Phase != "PREPARED" && j.Phase != "COMMITTED" && j.Phase != "ROLLED_BACK" {
		return fail("rollback_failed")
	}
	oldProvider, err := readBounded(m.txPath("old-provider.yaml"), 8<<20)
	if err != nil || digest(oldProvider) != j.OldProviderHash {
		return fail("rollback_failed")
	}
	oldState, err := readBounded(m.txPath("old-state.json"), 4<<20)
	if err != nil || digest(oldState) != j.OldStateHash {
		return fail("rollback_failed")
	}
	if err = atomicWrite(m.providerPath(), oldProvider, 0600); err != nil {
		return fail("rollback_failed")
	}
	if err = atomicWrite(m.webPath("state.json"), oldState, 0600); err != nil {
		return fail("rollback_failed")
	}
	if m.Core != nil {
		ctx, cancel := shortContext()
		defer cancel()
		if snap, e := m.Core.Snapshot(ctx); e == nil && snap.Running {
			if m.Core.Refresh(ctx) != nil {
				return fail("rollback_failed")
			}
			var old State
			if json.Unmarshal(oldState, &old) != nil || m.Core.Select(ctx, old.SelectedID) != nil {
				return fail("rollback_failed")
			}
		}
	}
	j.Phase = "ROLLED_BACK"
	journalBytes, _ := json.Marshal(j)
	if err = atomicWrite(m.txPath("journal.json"), journalBytes, 0600); err != nil {
		return fail("rollback_failed")
	}
	return m.cleanupTransaction()
}
