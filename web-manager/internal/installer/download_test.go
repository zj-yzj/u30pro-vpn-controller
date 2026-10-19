package installer

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func coreFixture() ([]byte, []byte, corePin) {
	raw := make([]byte, 256)
	copy(raw, []byte{0x7f, 'E', 'L', 'F', 2, 1})
	raw[18] = 183
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	z.Write(raw)
	z.Close()
	gz := b.Bytes()
	r := sha256.Sum256(raw)
	g := sha256.Sum256(gz)
	return raw, gz, corePin{"", int64(len(gz)), hex.EncodeToString(g[:]), int64(len(raw)), hex.EncodeToString(r[:])}
}
func TestDownloadRequiresTLSAndExactCompressedAndRawPins(t *testing.T) {
	for _, kind := range []string{"success", "tls", "compressed-size", "compressed-hash", "raw-size", "raw-hash", "elf", "truncated", "http"} {
		t.Run(kind, func(t *testing.T) {
			raw, gz, pins := coreFixture()
			if kind == "elf" {
				raw[18] = 3
				var b bytes.Buffer
				z := gzip.NewWriter(&b)
				z.Write(raw)
				z.Close()
				gz = b.Bytes()
				pins.CompressedBytes = int64(len(gz))
				pins.CompressedSHA256 = sum(gz)
				pins.RawSHA256 = sum(raw)
			}
			if kind == "truncated" {
				gz = gz[:len(gz)-5]
			}
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(gz) }))
			server.Config.ErrorLog = log.New(io.Discard, "", 0)
			defer server.Close()
			pins.URL = server.URL
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}}
			switch kind {
			case "tls":
				client = &http.Client{}
			case "compressed-size":
				pins.CompressedBytes++
			case "compressed-hash":
				pins.CompressedSHA256 = sum([]byte("bad"))
			case "raw-size":
				pins.RawBytes++
			case "raw-hash":
				pins.RawSHA256 = sum([]byte("bad"))
			case "http":
				pins.URL = "http://example.invalid/core.gz"
			}
			target := filepath.Join(t.TempDir(), "core")
			err := downloadPinned(context.Background(), client, pins, target)
			if kind == "success" {
				if err != nil {
					t.Fatal(err)
				}
				b, _ := os.ReadFile(target)
				if !bytes.Equal(b, raw) {
					t.Fatal("wrong core")
				}
			} else {
				if err == nil {
					t.Fatal("bad download accepted")
				}
				if _, e := os.Stat(target); !os.IsNotExist(e) {
					t.Fatal("failed core published")
				}
			}
		})
	}
}
func TestRedirectRejectsDowngradeUnknownHostAndTooManyHops(t *testing.T) {
	for _, target := range []string{"http://github.com/a", "https://evil.example/a", "https://user:pass@example.invalid/a", "https://github.com:8443/a"} {
		u, _ := url.Parse(target)
		if officialRedirect(&http.Request{URL: u}, nil) == nil {
			t.Fatal(target)
		}
	}
	u, _ := url.Parse("https://release-assets.githubusercontent.com/a")
	if officialRedirect(&http.Request{URL: u}, nil) != nil {
		t.Fatal("official asset host rejected")
	}
	if officialRedirect(&http.Request{URL: u}, make([]*http.Request, 6)) == nil {
		t.Fatal("redirect limit")
	}
}
func TestStaticAndroidResolverUsesExplicitDNSInsteadOfLoopback(t *testing.T) {
	called := ""
	resolver := androidResolver(func(ctx context.Context, network, address string) (net.Conn, error) {
		called = network + " " + address
		return nil, context.Canceled
	})
	_, err := resolver.Dial(context.Background(), "udp", "127.0.0.1:53")
	if err == nil || called != "udp4 223.5.5.5:53" {
		t.Fatalf("unexpected resolver %q %v", called, err)
	}
}
