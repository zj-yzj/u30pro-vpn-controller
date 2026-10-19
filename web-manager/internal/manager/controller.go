package manager

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

func shortContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 25*time.Second)
}

type UnixController struct{ Socket string }

func (c *UnixController) call(ctx context.Context, method, path string, payload any, out any) error {
	_, err := c.callStatus(ctx, method, path, payload, out, 8<<20)
	return err
}

// Keep the HTTP status for the manual probe: 503/504 are completed negative
// measurements, while transport, oversized or malformed responses are failures.
func (c *UnixController) callStatus(ctx context.Context, method, path string, payload any, out any, limit int64) (int, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return 0, fail("apply_failed")
		}
		body = bytes.NewReader(b)
	}
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "unix", c.Socket)
	}, DisableKeepAlives: true, ResponseHeaderTimeout: 8 * time.Second}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, err := http.NewRequestWithContext(ctx, method, "http://mihomo"+path, body)
	if err != nil {
		return 0, fail("apply_failed")
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, fail("apply_failed")
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return 0, fail("apply_failed")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.StatusCode, fail("apply_failed")
	}
	if out != nil && json.Unmarshal(raw, out) != nil {
		return 0, fail("apply_failed")
	}
	return response.StatusCode, nil
}

type nativeGroup struct {
	Now string   `json:"now"`
	All []string `json:"all"`
}

const (
	manualProbeURL          = "https://www.gstatic.com/generate_204"
	manualProbeTotalTimeout = 120 * time.Second
	manualProbeConcurrency  = 4
	manualProbeNodeTimeout  = "5000"
	singleProbeTotalTimeout = 12 * time.Second
)

type nativeProbeState struct {
	Alive   *bool `json:"alive"`
	History []struct {
		Delay uint16 `json:"delay"`
	} `json:"history"`
}
type nativeProviderProxy struct {
	Name string `json:"name"`
	nativeProbeState
	Extra map[string]nativeProbeState `json:"extra"`
}
type nativeProvider struct {
	Proxies []nativeProviderProxy `json:"proxies"`
}

func (p nativeProviderProxy) node() NativeNode {
	state := p.nativeProbeState
	if measured, exists := p.Extra[manualProbeURL]; exists {
		state = measured
	}
	n := NativeNode{Alive: state.Alive}
	if len(state.History) > 0 {
		delay := state.History[len(state.History)-1].Delay
		n.Delay = &delay
	}
	return n
}

func (c *UnixController) Snapshot(ctx context.Context) (NativeSnapshot, error) {
	s := NativeSnapshot{Nodes: map[string]NativeNode{}}
	var vpn nativeGroup
	if err := c.call(ctx, http.MethodGet, "/proxies/VPN", nil, &vpn); err != nil {
		return s, err
	}
	s.Running = true
	s.Selected = vpn.Now
	s.Active = vpn.Now
	if vpn.Now == "Auto" {
		var auto nativeGroup
		if err := c.call(ctx, http.MethodGet, "/proxies/Auto", nil, &auto); err != nil {
			return NativeSnapshot{}, err
		}
		s.Active = auto.Now
	}
	var provider nativeProvider
	if err := c.call(ctx, http.MethodGet, "/providers/proxies/Subscription", nil, &provider); err != nil {
		return NativeSnapshot{}, err
	}
	for _, p := range provider.Proxies {
		s.Nodes[p.Name] = p.node()
	}
	return s, nil
}

// Probe is deliberately never called by Snapshot. It has no caller-controlled
// target, provider or URL, and changes only the inner Auto selector after a full
// successful round. The outer VPN selector and persisted settings are untouched.
func (c *UnixController) Probe(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, manualProbeTotalTimeout)
	defer cancel()
	var provider nativeProvider
	if _, err := c.callStatus(ctx, http.MethodGet, "/providers/proxies/Subscription", nil, &provider, 4<<20); err != nil {
		return fail("probe_failed")
	}
	names, ok := probeProviderNames(provider)
	if !ok {
		return fail("probe_failed")
	}
	type result struct {
		id       string
		delay    uint16
		negative bool
		err      error
	}
	work := make(chan string)
	results := make(chan result, len(names))
	var workers sync.WaitGroup
	for i := 0; i < manualProbeConcurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case id, open := <-work:
					if !open || ctx.Err() != nil {
						return
					}
					var answer struct {
						Delay *uint16 `json:"delay"`
					}
					query := url.Values{"url": {manualProbeURL}, "timeout": {manualProbeNodeTimeout}, "expected": {"204"}}
					path := "/providers/proxies/Subscription/" + url.PathEscape(id) + "/healthcheck?" + query.Encode()
					status, err := c.callStatus(ctx, http.MethodGet, path, nil, &answer, 4096)
					r := result{id: id, err: err}
					if status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout {
						r.negative = true
						r.err = nil
					} else if err == nil && status == http.StatusOK && answer.Delay != nil && *answer.Delay > 0 {
						r.delay = *answer.Delay
					} else {
						r.err = fail("probe_failed")
					}
					results <- r
				}
			}
		}()
	}
	go func() {
		defer close(work)
		for _, id := range names {
			select {
			case <-ctx.Done():
				return
			case work <- id:
			}
		}
	}()
	go func() { workers.Wait(); close(results) }()
	current := map[string]uint16{}
	completed := 0
	protocolFailure := false
	for r := range results {
		completed++
		if r.err != nil {
			protocolFailure = true
			cancel()
		}
		if !r.negative && r.err == nil {
			current[r.id] = r.delay
		}
	}
	if ctx.Err() != nil || protocolFailure || completed != len(names) {
		return fail("probe_failed")
	}
	if len(current) == 0 {
		return nil
	}
	// URLTest's top-level alive/history can describe a successful HTTP response
	// even when expected=204 was not satisfied. Its fixed-URL extra state records
	// that distinction, so verify it before accepting this round's response delay.
	var measured nativeProvider
	if _, err := c.callStatus(ctx, http.MethodGet, "/providers/proxies/Subscription", nil, &measured, 4<<20); err != nil {
		return fail("probe_failed")
	}
	measuredNames, ok := probeProviderNames(measured)
	if !ok || len(names) != len(measuredNames) {
		return fail("probe_failed")
	}
	for i := range names {
		if names[i] != measuredNames[i] {
			return fail("probe_failed")
		}
	}
	best := ""
	var bestDelay uint16
	for _, p := range measured.Proxies {
		delay, tested := current[p.Name]
		if !tested {
			continue
		}
		extra, exists := p.Extra[manualProbeURL]
		if !exists || extra.Alive == nil || len(extra.History) == 0 {
			return fail("probe_failed")
		}
		if !*extra.Alive {
			continue
		}
		if extra.History[len(extra.History)-1].Delay != delay {
			return fail("probe_failed")
		}
		if best == "" || delay < bestDelay || (delay == bestDelay && p.Name < best) {
			best = p.Name
			bestDelay = delay
		}
	}
	if best == "" {
		return nil
	}
	if _, err := c.callStatus(ctx, http.MethodPut, "/proxies/Auto", map[string]string{"name": best}, nil, 4096); err != nil {
		return fail("probe_failed")
	}
	return nil
}

// ProbeNode never enumerates or tests another node and never updates a selector.
// The manager checks saved/live membership first; the provider-scoped Mihomo
// route independently refuses names no longer present before running URLTest.
func (c *UnixController) ProbeNode(ctx context.Context, id string) error {
	if !stableID.MatchString(id) {
		return fail("invalid_node")
	}
	ctx, cancel := context.WithTimeout(ctx, singleProbeTotalTimeout)
	defer cancel()
	query := url.Values{"url": {manualProbeURL}, "timeout": {manualProbeNodeTimeout}, "expected": {"204"}}
	path := "/providers/proxies/Subscription/" + url.PathEscape(id) + "/healthcheck?" + query.Encode()
	var answer struct {
		Delay *uint16 `json:"delay"`
	}
	status, err := c.callStatus(ctx, http.MethodGet, path, nil, &answer, 4096)
	if status == http.StatusNotFound {
		return fail("invalid_node")
	}
	if status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout {
		return nil
	}
	if err != nil || status != http.StatusOK || answer.Delay == nil || *answer.Delay == 0 {
		return fail("probe_failed")
	}
	return nil
}
func probeProviderNames(provider nativeProvider) ([]string, bool) {
	if len(provider.Proxies) == 0 || len(provider.Proxies) > 4096 {
		return nil, false
	}
	names := make([]string, 0, len(provider.Proxies))
	seen := map[string]bool{}
	for _, p := range provider.Proxies {
		if !stableID.MatchString(p.Name) || seen[p.Name] {
			return nil, false
		}
		seen[p.Name] = true
		names = append(names, p.Name)
	}
	sort.Strings(names)
	return names, true
}
func (c *UnixController) Refresh(ctx context.Context) error {
	return c.call(ctx, http.MethodPut, "/providers/proxies/Subscription", nil, nil)
}
func (c *UnixController) Select(ctx context.Context, id string) error {
	if id != "Auto" && !stableID.MatchString(id) {
		return fail("invalid_node")
	}
	return c.call(ctx, http.MethodPut, "/proxies/VPN", map[string]string{"name": id}, nil)
}
func (c *UnixController) Health(ctx context.Context) error {
	roots, rootErr := androidRoots()
	if rootErr != nil {
		return fail("apply_failed")
	}
	proxy, _ := url.Parse("http://127.0.0.1:17890")
	transport := &http.Transport{Proxy: http.ProxyURL(proxy), DisableKeepAlives: true, ResponseHeaderTimeout: 12 * time.Second, TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 18 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.gstatic.com/generate_204", nil)
	resp, err := client.Do(req)
	if err != nil {
		return fail("apply_failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return fail("apply_failed")
	}
	return nil
}
func (m *Manager) control(ctx context.Context, action string) error {
	switch action {
	case "start", "stop", "status":
	default:
		return fail("invalid_request")
	}
	command := exec.CommandContext(ctx, "/system/bin/sh", filepath.Join(m.Base, "control.sh"), action)
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if command.Run() != nil {
		return fail("control_failed")
	}
	return nil
}
func (m *Manager) validate(ctx context.Context, proxies []map[string]any) error {
	directory, err := os.MkdirTemp(m.webPath(""), "validate-")
	if err != nil {
		return fail("storage_failed")
	}
	defer os.RemoveAll(directory)
	if os.Chmod(directory, 0700) != nil || os.Mkdir(filepath.Join(directory, "cache"), 0700) != nil {
		return fail("storage_failed")
	}
	names := []string{}
	for _, p := range proxies {
		id, ok := p["name"].(string)
		if !ok {
			return fail("invalid_subscription")
		}
		names = append(names, id)
	}
	cfg := map[string]any{"port": 0, "socks-port": 0, "mixed-port": 0, "redir-port": 0, "tproxy-port": 0, "allow-lan": false, "ipv6": false, "mode": "rule", "find-process-mode": "off", "log-level": "silent", "external-controller": "", "external-controller-unix": "", "dns": map[string]any{"enable": false}, "tun": map[string]any{"enable": false}, "proxies": proxies, "proxy-groups": []any{map[string]any{"name": "VPN", "type": "select", "proxies": names, "empty-fallback": "REJECT"}}, "rules": []string{"MATCH,VPN"}}
	b, err := json.Marshal(cfg)
	if err != nil {
		return fail("invalid_subscription")
	}
	path := filepath.Join(directory, "config.json")
	if atomicWrite(path, b, 0600) != nil {
		return fail("storage_failed")
	}
	cmd := exec.CommandContext(ctx, filepath.Join(m.Base, "mihomo"), "-t", "-d", filepath.Join(directory, "cache"), "-f", path)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if cmd.Run() != nil {
		return fail("invalid_subscription")
	}
	return nil
}
