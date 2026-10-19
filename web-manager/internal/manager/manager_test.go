package manager

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testManager(t *testing.T) *Manager {
	t.Helper()
	base := t.TempDir()
	for _, d := range []string{"web", "cache"} {
		if err := os.Mkdir(filepath.Join(base, d), 0700); err != nil {
			t.Fatal(err)
		}
	}
	m := NewForTest(base)
	m.Now = func() time.Time { return time.Unix(1800000000, 0) }
	return m
}
func seal(t *testing.T, k *rsa.PrivateKey, r Request) string {
	t.Helper()
	raw, _ := json.Marshal(r)
	key := make([]byte, 32)
	iv := make([]byte, 12)
	rand.Read(key)
	rand.Read(iv)
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	wrapped, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &k.PublicKey, key, []byte(envelopeLabel))
	if err != nil {
		t.Fatal(err)
	}
	e := Envelope{V: 1, Key: base64.StdEncoding.EncodeToString(wrapped), IV: base64.StdEncoding.EncodeToString(iv), Data: base64.StdEncoding.EncodeToString(gcm.Seal(nil, iv, raw, []byte(envelopeLabel)))}
	b, _ := json.Marshal(e)
	return base64.RawURLEncoding.EncodeToString(b)
}
func TestEnvelopeInteroperabilityAndBounds(t *testing.T) {
	m := testManager(t)
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	r := Request{V: 1, IssuedAt: m.Now().UnixMilli(), Nonce: base64.RawURLEncoding.EncodeToString(make([]byte, 16)), Action: "update-subscription", URL: "https://example.com/sub?token=SECRET"}
	encoded := seal(t, k, r)
	got, err := Decrypt(encoded, k, m.Now())
	if err != nil || got.URL != r.URL {
		t.Fatalf("decrypt: %v", err)
	}
	r.IssuedAt -= 121000
	if _, err = Decrypt(seal(t, k, r), k, m.Now()); err == nil {
		t.Fatal("old envelope accepted")
	}
	r.IssuedAt = m.Now().UnixMilli()
	r.Action = "shell"
	if _, err = Decrypt(seal(t, k, r), k, m.Now()); err == nil {
		t.Fatal("action accepted")
	}
	if _, err = Decrypt(strings.Repeat("A", MaxEnvelope+1), k, m.Now()); err == nil {
		t.Fatal("oversized envelope")
	}
	if err = m.rememberNonce(got.Nonce); err != nil {
		t.Fatal(err)
	}
	if err = m.rememberNonce(got.Nonce); err == nil {
		t.Fatal("replay accepted")
	}
}
func TestTransactionRollbackAndRecovery(t *testing.T) {
	m := testManager(t)
	old := []byte("proxies: old\n")
	next := []byte("proxies: new\n")
	s := State{Schema: 1, DesiredEnabled: true, SelectedID: "Auto"}
	if err := m.writeState(s); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(m.providerPath(), old, 0600); err != nil {
		t.Fatal(err)
	}
	ns := s
	ns.SubscriptionURL = "https://example.com/SECRET"
	if err := m.beginTransaction(next, ns); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(m.providerPath(), next, 0600); err != nil {
		t.Fatal(err)
	}
	if err := m.writeState(ns); err != nil {
		t.Fatal(err)
	}
	if err := m.recoverTransaction(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(m.providerPath())
	if string(b) != string(old) {
		t.Fatal("provider not restored")
	}
	got, _ := m.readState()
	if got.SubscriptionURL != "" {
		t.Fatal("state not restored")
	}
	if err := m.beginTransaction(next, ns); err != nil {
		t.Fatal(err)
	}
	atomicWrite(m.providerPath(), next, 0600)
	m.writeState(ns)
	if err := m.commitTransaction(); err != nil {
		t.Fatal(err)
	}
	if err := m.recoverTransaction(); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(m.providerPath())
	if string(b) != string(next) {
		t.Fatal("committed provider reverted")
	}
}
func TestSafeErrorDoesNotContainInput(t *testing.T) {
	b, _ := json.Marshal(SafeError("download_failed"))
	if strings.Contains(string(b), "SECRET") || !strings.Contains(string(b), "download_failed") {
		t.Fatal("unsafe error")
	}
}
