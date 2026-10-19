package nativebridge

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"u30pro/webmanager/internal/manager"
)

// Uses a fresh test-only RSA key kept entirely in memory. No device key,
// subscription, or fixture private key is written to test output.
func TestAndroidJavaEnvelopeInteroperatesWithExistingManager(t *testing.T) {
	javac, err := exec.LookPath("javac")
	if err != nil {
		t.Skip("JDK compiler unavailable")
	}
	java, err := exec.LookPath("java")
	if err != nil {
		t.Skip("JDK runtime unavailable")
	}
	source := filepath.Join("..", "..", "..", "native-manager", "android")
	classes := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	compile := exec.CommandContext(ctx, javac, "-d", classes,
		filepath.Join(source, "src", "com", "codex", "u30vpn", "CryptoEnvelope.java"),
		filepath.Join(source, "tests", "CryptoEnvelopeTest.java"))
	if compile.Run() != nil {
		t.Fatal("Java crypto fixture compilation failed")
	}
	fixtureBytes, err := exec.CommandContext(ctx, java, "-cp", classes, "com.codex.u30vpn.CryptoEnvelopeTest", "--fixture").Output()
	if err != nil {
		t.Fatal("Java crypto fixture generation failed")
	}
	var fixture struct {
		PrivateKeyPKCS8 string `json:"privateKeyPkcs8"`
		Envelope        string `json:"envelope"`
	}
	if len(fixtureBytes) > 16384 || json.Unmarshal(fixtureBytes, &fixture) != nil {
		t.Fatal("Invalid Java test fixture")
	}
	der, err := base64.StdEncoding.DecodeString(fixture.PrivateKeyPKCS8)
	if err != nil {
		t.Fatal("Invalid test key encoding")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		t.Fatal("Invalid test key")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		t.Fatal("Test key is not RSA")
	}
	requestBytes, _ := json.Marshal(Request{V: 1, Op: "request", Envelope: fixture.Envelope})
	if _, err = ParseRequest(requestBytes); err != nil {
		t.Fatal("Java envelope did not pass bridge boundary")
	}
	req, err := manager.Decrypt(fixture.Envelope, key, time.UnixMilli(1800000000000))
	if err != nil {
		t.Fatal("Java envelope rejected by existing Go manager")
	}
	if req.V != 1 || req.Action != "select" || req.ID != "n-0123456789abcdef" || req.IssuedAt != 1800000000000 || req.Nonce != "AAAAAAAAAAAAAAAAAAAAAA" {
		t.Fatal("Java envelope decrypted to unexpected request")
	}
}
