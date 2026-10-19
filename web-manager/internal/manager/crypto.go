package manager

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"os"
	"regexp"
	"time"
)

var stableID = regexp.MustCompile(`^n-[a-f0-9]{16,64}$`)
var jobIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

func strictJSON(raw []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fail("invalid_request")
	}
	return nil
}
func Decrypt(encoded string, key *rsa.PrivateKey, now time.Time) (Request, error) {
	invalid := func() (Request, error) { return Request{}, fail("invalid_request") }
	if len(encoded) == 0 || len(encoded) > MaxEnvelope {
		return invalid()
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return invalid()
	}
	var e Envelope
	if strictJSON(raw, &e) != nil || e.V != 1 {
		return invalid()
	}
	wrapped, err := base64.StdEncoding.DecodeString(e.Key)
	if err != nil || len(wrapped) != key.Size() {
		return invalid()
	}
	iv, err := base64.StdEncoding.DecodeString(e.IV)
	if err != nil || len(iv) != 12 {
		return invalid()
	}
	data, err := base64.StdEncoding.DecodeString(e.Data)
	if err != nil || len(data) < 16 || len(data) > 16384 {
		return invalid()
	}
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, wrapped, []byte(envelopeLabel))
	if err != nil || len(aesKey) != 32 {
		return invalid()
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return invalid()
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return invalid()
	}
	plain, err := gcm.Open(nil, iv, data, []byte(envelopeLabel))
	if err != nil {
		return invalid()
	}
	var r Request
	if strictJSON(plain, &r) != nil || r.V != 1 {
		return invalid()
	}
	delta := now.UnixMilli() - r.IssuedAt
	if delta > 120000 || delta < -30000 {
		return invalid()
	}
	nonce, err := base64.RawURLEncoding.DecodeString(r.Nonce)
	if err != nil || len(nonce) != 16 || base64.RawURLEncoding.EncodeToString(nonce) != r.Nonce {
		return invalid()
	}
	switch r.Action {
	case "set-enabled":
		if r.Enabled == nil || r.ID != "" || r.URL != "" {
			return invalid()
		}
	case "select":
		if r.Enabled != nil || r.URL != "" || (r.ID != "Auto" && !stableID.MatchString(r.ID)) {
			return invalid()
		}
	case "update-subscription":
		if r.Enabled != nil || r.ID != "" || len(r.URL) > 8192 {
			return invalid()
		}
	case "probe-nodes":
		if !probeRequestShape(plain) {
			return invalid()
		}
	case "probe-node":
		if !singleProbeRequestShape(plain) {
			return invalid()
		}
	default:
		return invalid()
	}
	return r, nil
}

// Probe requests have no optional fields. Checking their presence also rejects
// empty/null target fields that typed JSON decoding would otherwise erase.
func probeRequestShape(plain []byte) bool {
	return exactProbeFields(plain, "v", "issuedAt", "nonce", "action")
}
func singleProbeRequestShape(plain []byte) bool {
	if !exactProbeFields(plain, "v", "issuedAt", "nonce", "action", "id") {
		return false
	}
	var request Request
	return json.Unmarshal(plain, &request) == nil && request.V == 1 && request.Action == "probe-node" && stableID.MatchString(request.ID)
}

// A fixed field set rejects both empty/null optional fields and duplicate keys;
// neither can silently disappear during typed JSON decoding.
func exactProbeFields(plain []byte, names ...string) bool {
	allowed := map[string]bool{}
	for _, name := range names {
		allowed[name] = true
	}
	decoder := json.NewDecoder(bytes.NewReader(plain))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return false
	}
	seen := map[string]bool{}
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return false
		}
		name, ok := token.(string)
		if !ok || !allowed[name] || seen[name] {
			return false
		}
		seen[name] = true
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return false
		}
	}
	token, err = decoder.Token()
	if err != nil || token != json.Delim('}') || len(seen) != len(allowed) {
		return false
	}
	_, err = decoder.Token()
	return err == io.EOF
}
func (m *Manager) privateKey() (*rsa.PrivateKey, error) {
	b, err := readBounded(m.webPath("private-key.pem"), 4096)
	if err != nil {
		return nil, fail("not_initialized")
	}
	p, _ := pem.Decode(b)
	if p == nil {
		return nil, fail("not_initialized")
	}
	k, err := x509.ParsePKCS1PrivateKey(p.Bytes)
	if err != nil || k.N.BitLen() != 2048 {
		return nil, fail("not_initialized")
	}
	return k, nil
}
func (m *Manager) publicKey() (string, error) {
	k, err := m.privateKey()
	if err != nil {
		return "", err
	}
	b, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
	if err != nil {
		return "", fail("not_initialized")
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: b})), nil
}
func (m *Manager) createKey() error {
	if _, err := os.Lstat(m.webPath("private-key.pem")); err == nil {
		_, err = m.privateKey()
		return err
	} else if !os.IsNotExist(err) {
		return fail("storage_failed")
	}
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fail("storage_failed")
	}
	return atomicWrite(m.webPath("private-key.pem"), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}), 0600)
}

type nonceRecord struct {
	Nonce string `json:"nonce"`
	At    int64  `json:"at"`
}

func (m *Manager) rememberNonce(nonce string) error {
	var existing []nonceRecord
	b, err := readBounded(m.webPath("nonces.json"), 65536)
	if err == nil {
		if json.Unmarshal(b, &existing) != nil {
			return fail("storage_failed")
		}
	} else if !os.IsNotExist(err) {
		return fail("storage_failed")
	}
	now := m.Now().UnixMilli()
	kept := []nonceRecord{}
	for _, n := range existing {
		if n.At >= now-150000 {
			if n.Nonce == nonce {
				return fail("replayed_request")
			}
			kept = append(kept, n)
		}
	}
	if len(kept) >= 256 {
		return fail("busy")
	}
	kept = append(kept, nonceRecord{nonce, now})
	out, _ := json.Marshal(kept)
	return atomicWrite(m.webPath("nonces.json"), out, 0600)
}
