package manager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var forbiddenPrefixes = []netip.Prefix{netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("172.16.0.0/12"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("192.168.0.0/16"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("224.0.0.0/4"), netip.MustParsePrefix("240.0.0.0/4")}

func publicIPv4(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.Is4() || !ip.IsGlobalUnicast() {
		return false
	}
	for _, p := range forbiddenPrefixes {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}
func subscriptionURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 8192 || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") || strings.ContainsAny(u.Hostname(), "%\\\x00\r\n") {
		return nil, fail("invalid_request")
	}
	return u, nil
}
func androidRoots() (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	count := 0
	for _, dir := range []string{"/apex/com.android.conscrypt/cacerts", "/system/etc/security/cacerts"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			b, err := readBounded(filepath.Join(dir, entry.Name()), 1<<20)
			if err == nil && pool.AppendCertsFromPEM(b) {
				count++
			}
		}
	}
	if count == 0 {
		return nil, fail("download_failed")
	}
	return pool, nil
}
func resolver() *net.Resolver {
	return &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 4 * time.Second}).DialContext(ctx, "udp4", "223.5.5.5:53")
	}}
}
func pinnedTransport(ctx context.Context, u *url.URL, roots *x509.CertPool) (*http.Transport, error) {
	var addresses []netip.Addr
	if literal, err := netip.ParseAddr(u.Hostname()); err == nil {
		addresses = []netip.Addr{literal.Unmap()}
	} else {
		resolved, err := resolver().LookupNetIP(ctx, "ip4", u.Hostname())
		if err != nil || len(resolved) == 0 || len(resolved) > 32 {
			return nil, fail("download_failed")
		}
		addresses = resolved
	}
	for _, ip := range addresses {
		if !publicIPv4(ip) {
			return nil, fail("download_failed")
		}
	}
	host := u.Hostname()
	return &http.Transport{Proxy: nil, DisableKeepAlives: true, DisableCompression: true, ResponseHeaderTimeout: 12 * time.Second, TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12, ServerName: host}, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		for _, ip := range addresses {
			conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp4", net.JoinHostPort(ip.String(), "443"))
			if err == nil {
				return conn, nil
			}
			if ctx.Err() != nil {
				break
			}
		}
		return nil, fail("download_failed")
	}}, nil
}
func Download(ctx context.Context, raw string) ([]byte, error) {
	roots, err := androidRoots()
	if err != nil {
		return nil, err
	}
	u, err := subscriptionURL(raw)
	if err != nil {
		return nil, fail("download_failed")
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	for redirects := 0; redirects <= 4; redirects++ {
		transport, err := pinnedTransport(ctx, u, roots)
		if err != nil {
			return nil, fail("download_failed")
		}
		client := &http.Client{Transport: transport, Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fail("download_failed")
		}
		req.Header.Set("User-Agent", "clash.meta/1.19.32")
		req.Header.Set("Accept-Encoding", "identity")
		resp, err := client.Do(req)
		if err != nil {
			transport.CloseIdleConnections()
			return nil, fail("download_failed")
		}
		if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
			location, err := resp.Location()
			resp.Body.Close()
			transport.CloseIdleConnections()
			if err != nil || redirects == 4 {
				return nil, fail("download_failed")
			}
			u, err = subscriptionURL(location.String())
			if err != nil {
				return nil, fail("download_failed")
			}
			continue
		}
		if resp.StatusCode != 200 || resp.ContentLength > 8<<20 {
			resp.Body.Close()
			transport.CloseIdleConnections()
			return nil, fail("download_failed")
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
		resp.Body.Close()
		transport.CloseIdleConnections()
		if err != nil || len(b) == 0 || len(b) > 8<<20 {
			return nil, fail("download_failed")
		}
		return b, nil
	}
	return nil, fail("download_failed")
}
