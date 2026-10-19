package manager

import (
	"bytes"
	"os"

	"u30pro/webmanager/internal/subscription"
)

const emptyProvider = "proxies: []\n"

// Bootstrap initializes a clean installation without enabling VPN or requiring
// subscription data. The installer must establish ownership of m.Base first.
// Writes are ordered key -> provider -> state, so an interrupted initial write
// can resume without replacing an identity or an existing configuration.
func (m *Manager) Bootstrap() error {
	if err := m.storageReady(); err != nil {
		return err
	}
	if _, _, err := m.bootstrapFile(m.webPath("manager.lock"), 4096); err != nil {
		return err
	}
	unlock, err := acquireLock(m.webPath("manager.lock"), false)
	if err != nil {
		return err
	}
	defer unlock()

	keyExists, _, err := m.bootstrapFile(m.webPath("private-key.pem"), 4096)
	if err != nil {
		return err
	}
	providerExists, provider, err := m.bootstrapFile(m.providerPath(), 8<<20)
	if err != nil {
		return err
	}
	stateExists, _, err := m.bootstrapFile(m.webPath("state.json"), 4<<20)
	if err != nil {
		return err
	}
	if stateExists {
		// A committed state is never repaired by making a new identity or by
		// substituting an empty provider. Those combinations need review.
		if !keyExists || !providerExists || len(provider) == 0 {
			return fail("not_initialized")
		}
		key, keyErr := m.privateKey()
		if keyErr != nil || key.Validate() != nil {
			return fail("not_initialized")
		}
		_, err = m.readState()
		return err
	}
	// A provider cannot precede the key in this operation. With no state,
	// accept only the exact empty bytes this bootstrap operation creates.
	if providerExists && (!keyExists || !bytes.Equal(provider, []byte(emptyProvider))) {
		return fail("not_initialized")
	}
	if keyExists {
		key, keyErr := m.privateKey()
		if keyErr != nil || key.Validate() != nil {
			return fail("not_initialized")
		}
	} else if err = m.createKey(); err != nil {
		return err
	}
	if !providerExists {
		if err = atomicWrite(m.providerPath(), []byte(emptyProvider), 0600); err != nil {
			return err
		}
	}
	return m.writeState(State{Schema: 1, DesiredEnabled: false, SelectedID: "Auto", Nodes: []subscription.Node{}})
}

func (m *Manager) bootstrapFile(path string, limit int64) (bool, []byte, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fail("storage_failed")
	}
	if !info.Mode().IsRegular() || (!m.Test && (info.Mode().Perm() != 0600 || !ownedRoot(info))) {
		return false, nil, fail("insecure_storage")
	}
	b, err := readBounded(path, limit)
	if err != nil {
		return false, nil, fail("storage_failed")
	}
	return true, b, nil
}
