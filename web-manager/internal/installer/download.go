package installer

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type corePin struct {
	URL              string
	CompressedBytes  int64
	CompressedSHA256 string
	RawBytes         int64
	RawSHA256        string
}

var upstreamCore = corePin{
	"https://github.com/MetaCubeX/mihomo/releases/download/v1.19.32/mihomo-android-arm64-v8-v1.19.32.gz",
	21866463, "42b1f2be8702484d3ae6238daab392622f54bd0d264966700f1f28f52596e326",
	64045272, "bb9afbd48717f92d59e07da35fb694b83d40d6edb63eb9033c2ee4174137d195",
}

func secureHTTPS(u *url.URL) bool {
	return u != nil && u.Scheme == "https" && u.User == nil && (u.Port() == "" || u.Port() == "443") && u.Hostname() != ""
}
func officialRedirect(r *http.Request, via []*http.Request) error {
	if len(via) > 5 || !secureHTTPS(r.URL) {
		return problem("download_failed")
	}
	switch r.URL.Hostname() {
	case "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return nil
	}
	return problem("download_failed")
}
func androidResolver(dial func(context.Context, string, string) (net.Conn, error)) *net.Resolver {
	return &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, _, _ string) (net.Conn, error) { return dial(ctx, "udp4", "223.5.5.5:53") }}
}

// Static Linux Go binaries do not automatically discover Android's APEX trust
// store. Build a real CA pool from both supported Android locations, never use
// InsecureSkipVerify or substitute an HTTP fallback.
func androidClient() (*http.Client, error) {
	roots := x509.NewCertPool()
	count := 0
	for _, dir := range []string{"/apex/com.android.conscrypt/cacerts", "/system/etc/security/cacerts"} {
		entries, e := os.ReadDir(dir)
		if e != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			b, e := readSmall(filepath.Join(dir, entry.Name()), 64<<10)
			if e == nil && roots.AppendCertsFromPEM(b) {
				count++
			}
		}
	}
	if count == 0 {
		return nil, problem("tls_unavailable")
	}
	dnsDialer := &net.Dialer{Timeout: 5 * time.Second}
	dialer := &net.Dialer{Timeout: 20 * time.Second, Resolver: androidResolver(dnsDialer.DialContext)}
	tr := &http.Transport{Proxy: nil, DialContext: dialer.DialContext, TLSHandshakeTimeout: 20 * time.Second, ResponseHeaderTimeout: 30 * time.Second, DisableCompression: true, TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	return &http.Client{Transport: tr, Timeout: 10 * time.Minute, CheckRedirect: officialRedirect}, nil
}

type countedReader struct {
	r io.Reader
	n int64
}

func (c *countedReader) Read(b []byte) (int, error) {
	n, e := c.r.Read(b)
	c.n += int64(n)
	return n, e
}
func downloadPinned(ctx context.Context, client *http.Client, pin corePin, target string) error {
	u, e := url.Parse(pin.URL)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Hostname() == "" || pin.RawBytes < 64 || pin.CompressedBytes < 1 || !validHash(pin.RawSHA256) || !validHash(pin.CompressedSHA256) {
		return problem("download_failed")
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, pin.URL, nil)
	if e != nil {
		return problem("download_failed")
	}
	req.Header.Set("Accept-Encoding", "identity")
	resp, e := client.Do(req)
	if e != nil {
		return problem("download_failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || (resp.ContentLength >= 0 && resp.ContentLength != pin.CompressedBytes) || resp.Header.Get("Content-Encoding") != "" {
		return problem("download_failed")
	}
	// Read no more than either declared size plus one. A proxy or origin cannot
	// cause unbounded memory/disk use, and gzip checksum errors remain fatal.
	compressedHash := sha256.New()
	compressed := &countedReader{r: io.TeeReader(io.LimitReader(resp.Body, pin.CompressedBytes+1), compressedHash)}
	z, e := gzip.NewReader(compressed)
	if e != nil {
		return problem("download_failed")
	}
	defer z.Close()
	f, e := os.CreateTemp(filepath.Dir(target), ".core-download-")
	if e != nil {
		return problem("storage_failed")
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	defer f.Close()
	if f.Chmod(0700) != nil {
		return problem("storage_failed")
	}
	rawHash := sha256.New()
	n, e := io.Copy(io.MultiWriter(f, rawHash), io.LimitReader(z, pin.RawBytes+1))
	if e != nil || n != pin.RawBytes {
		return problem("download_failed")
	}
	// Copy reached EOF on the gzip stream, including CRC validation. Drain any
	// reader look-ahead and account for every compressed byte as well.
	if _, e = io.Copy(io.Discard, compressed); e != nil || compressed.n != pin.CompressedBytes || hex.EncodeToString(compressedHash.Sum(nil)) != pin.CompressedSHA256 || hex.EncodeToString(rawHash.Sum(nil)) != pin.RawSHA256 {
		return problem("download_failed")
	}
	if _, e = f.Seek(0, io.SeekStart); e != nil {
		return problem("storage_failed")
	}
	header := make([]byte, 20)
	if _, e = io.ReadFull(f, header); e != nil || string(header[:4]) != "\x7fELF" || header[4] != 2 || header[5] != 1 || header[18] != 183 || header[19] != 0 {
		return problem("download_failed")
	}
	if f.Sync() != nil || f.Close() != nil {
		return problem("storage_failed")
	}
	if _, e = os.Lstat(target); !os.IsNotExist(e) {
		return problem("existing_installation")
	}
	if os.Rename(tmp, target) != nil {
		return problem("storage_failed")
	}
	return syncDirectory(filepath.Dir(target))
}
