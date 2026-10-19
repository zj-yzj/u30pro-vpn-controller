//go:build diagnostic

package manager

// This temporary, fixed-target diagnostic returns classifications only. It must
// never return raw errors, request URLs, provider bytes or connection settings.
import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"u30pro/webmanager/internal/subscription"
)

type DiagnosticSnapshot struct {
	OK              bool   `json:"ok"`
	Running         bool   `json:"running"`
	NodeCount       int    `json:"nodeCount"`
	SelectedID      string `json:"selectedId"`
	ActiveID        string `json:"activeId"`
	ExpectedMissing int    `json:"expectedMissing"`
	Unexpected      int    `json:"unexpected"`
}
type DiagnosticHealth struct {
	OK                 bool   `json:"ok"`
	Category           string `json:"category"`
	ErrorType          string `json:"errorType"`
	Status             int    `json:"status"`
	RootCount          int    `json:"rootCount"`
	ElapsedMS          int64  `json:"elapsedMs"`
	ProxyConnectStatus int    `json:"proxyConnectStatus"`
	TLSStarted         bool   `json:"tlsStarted"`
	TLSCompleted       bool   `json:"tlsCompleted"`
}
type DiagnosticReport struct {
	OK                   bool               `json:"ok"`
	ExpectedNodeCount    int                `json:"expectedNodeCount"`
	ProviderNodeCount    int                `json:"providerNodeCount"`
	ProviderMatchesState bool               `json:"providerMatchesState"`
	DesiredEnabled       bool               `json:"desiredEnabled"`
	SavedSelectedID      string             `json:"savedSelectedId"`
	Before               DiagnosticSnapshot `json:"before"`
	RefreshOK            bool               `json:"refreshOk"`
	After                DiagnosticSnapshot `json:"after"`
	CoreHealthOK         bool               `json:"coreHealthOk"`
	DetailedHealth       DiagnosticHealth   `json:"detailedHealth"`
	AfterHealth          DiagnosticSnapshot `json:"afterHealth"`
}

func safeDiagnosticID(id string) string {
	if id == "Auto" || stableID.MatchString(id) {
		return id
	}
	return ""
}
func diagnosticSnapshot(ctx context.Context, core Controller, expected map[string]bool) DiagnosticSnapshot {
	snapshot, err := core.Snapshot(ctx)
	out := DiagnosticSnapshot{OK: err == nil, Running: err == nil && snapshot.Running, NodeCount: len(snapshot.Nodes), SelectedID: safeDiagnosticID(snapshot.Selected), ActiveID: safeDiagnosticID(snapshot.Active)}
	if err != nil {
		return out
	}
	for id := range expected {
		if _, ok := snapshot.Nodes[id]; !ok {
			out.ExpectedMissing++
		}
	}
	for id := range snapshot.Nodes {
		if !expected[id] {
			out.Unexpected++
		}
	}
	return out
}
func diagnosticError(err error) (string, string) {
	if err == nil {
		return "ok", ""
	}
	typ := fmt.Sprintf("%T", err)
	var verify *tls.CertificateVerificationError
	var unknown x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalid x509.CertificateInvalidError
	var dns *net.DNSError
	var op *net.OpError
	var ne net.Error
	switch {
	case errors.As(err, &unknown):
		return "tls_unknown_ca", typ
	case errors.As(err, &hostname):
		return "tls_hostname", typ
	case errors.As(err, &invalid):
		return "tls_certificate", typ
	case errors.As(err, &verify):
		return "tls_verification", typ
	case errors.As(err, &dns):
		return "dns", typ
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout", typ
	case errors.As(err, &ne) && ne.Timeout():
		return "timeout", typ
	case errors.As(err, &op):
		return "connect", typ
	default:
		return "request", typ
	}
}
func detailedProxyHealth(ctx context.Context) DiagnosticHealth {
	start := time.Now()
	out := DiagnosticHealth{}
	roots, err := androidRoots()
	if err != nil {
		out.Category = "tls_roots"
		out.ErrorType = "roots_unavailable"
		return out
	}
	out.RootCount = len(roots.Subjects())
	proxy, _ := url.Parse("http://127.0.0.1:17890")
	transport := &http.Transport{Proxy: http.ProxyURL(proxy), DisableKeepAlives: true, ResponseHeaderTimeout: 12 * time.Second, TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	transport.OnProxyConnectResponse = func(_ context.Context, _ *url.URL, _ *http.Request, response *http.Response) error {
		out.ProxyConnectStatus = response.StatusCode
		return nil
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 18 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.gstatic.com/generate_204", nil)
	request = request.WithContext(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{TLSHandshakeStart: func() { out.TLSStarted = true }, TLSHandshakeDone: func(_ tls.ConnectionState, err error) { out.TLSCompleted = err == nil }}))
	response, err := client.Do(request)
	out.ElapsedMS = time.Since(start).Milliseconds()
	if err != nil {
		out.Category, out.ErrorType = diagnosticError(err)
		if out.ProxyConnectStatus != 0 && out.ProxyConnectStatus != 200 {
			out.Category = "status"
		}
		return out
	}
	defer response.Body.Close()
	out.Status = response.StatusCode
	if response.StatusCode != 204 {
		out.Category = "status"
		return out
	}
	out.OK = true
	out.Category = "ok"
	return out
}
func (m *Manager) Diagnose() (DiagnosticReport, error) {
	out := DiagnosticReport{}
	if err := m.storageReady(); err != nil {
		return out, err
	}
	unlock, err := acquireLock(m.webPath("manager.lock"), false)
	if err != nil {
		return out, err
	}
	defer unlock()
	state, err := m.readState()
	if err != nil {
		return out, err
	}
	out.DesiredEnabled = state.DesiredEnabled
	out.SavedSelectedID = safeDiagnosticID(state.SelectedID)
	expected := map[string]bool{}
	for _, n := range state.Nodes {
		expected[n.ID] = true
	}
	out.ExpectedNodeCount = len(expected)
	provider, err := readBounded(m.providerPath(), 8<<20)
	if err == nil {
		if parsed, e := subscription.Normalize(provider); e == nil {
			out.ProviderNodeCount = len(parsed.Nodes)
			out.ProviderMatchesState = len(parsed.Nodes) == len(expected)
			for _, n := range parsed.Nodes {
				if !expected[n.ID] {
					out.ProviderMatchesState = false
				}
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 85*time.Second)
	defer cancel()
	out.Before = diagnosticSnapshot(ctx, m.Core, expected)
	out.RefreshOK = m.Core.Refresh(ctx) == nil
	out.After = diagnosticSnapshot(ctx, m.Core, expected)
	out.CoreHealthOK = m.Core.Health(ctx) == nil
	out.DetailedHealth = detailedProxyHealth(ctx)
	out.AfterHealth = diagnosticSnapshot(ctx, m.Core, expected)
	out.OK = true
	return out, nil
}
