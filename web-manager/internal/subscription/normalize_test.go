package subscription

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestStableIDsAndSafeProvider(t *testing.T) {
	a := []byte(`mixed-port: 9999
external-controller: 0.0.0.0:9090
proxies:
  - {name: "香港 <script>", type: VLESS, server: example.test, port: 443, uuid: sample, interface-name: br0, routing-mark: 999}
  - {name: Backup, type: hysteria2, server: backup.test, port: 8443, password: secret}
`)
	r, err := Normalize(a)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Nodes) != 2 || r.Nodes[0].Name != "香港 <script>" || r.Nodes[0].Type != "vless" {
		t.Fatalf("metadata mismatch: %#v", r.Nodes)
	}
	if !strings.HasPrefix(r.Nodes[0].ID, "n-") || len(r.Nodes[0].ID) != 66 {
		t.Fatal("opaque SHA256 identity required")
	}
	for _, p := range r.Proxies {
		if p["interface-name"] != nil || p["routing-mark"] != nil {
			t.Fatal("source routing override imported")
		}
	}
	var provider map[string]any
	if err = json.Unmarshal(r.Provider, &provider); err != nil {
		t.Fatalf("provider must be JSON-compatible YAML: %v", err)
	}
	if len(provider) != 1 || provider["proxies"] == nil || bytes.Contains(r.Provider, []byte("<script>")) {
		t.Fatal("provider leaked source controls or display name")
	}
	b := []byte(`proxies:
  - {password: secret, port: 8443, server: backup.test, type: hysteria2, name: Renamed}
  - {uuid: sample, port: 443, type: vless, server: example.test, name: Changed}
`)
	s, err := Normalize(b)
	if err != nil {
		t.Fatal(err)
	}
	if r.Nodes[0].ID != s.Nodes[1].ID || r.Nodes[1].ID != s.Nodes[0].ID {
		t.Fatal("rename, ordering, or stripped overrides changed identity")
	}
}

func TestDialerReferencesStableAndRemapped(t *testing.T) {
	r, err := Normalize([]byte(`proxies:
 - {name: Entry, type: socks5, server: entry.test, port: 1080}
 - {name: Exit, type: vless, server: exit.test, port: 443, uuid: a, dialer-proxy: Entry}
`))
	if err != nil {
		t.Fatal(err)
	}
	if r.Proxies[1]["dialer-proxy"] != r.Nodes[0].ID {
		t.Fatal("dialer name was not remapped")
	}
	s, err := Normalize([]byte(`proxies:
 - {name: E2, type: vless, server: exit.test, port: 443, uuid: a, dialer-proxy: E1}
 - {name: E1, type: socks5, server: entry.test, port: 1080}
`))
	if err != nil {
		t.Fatal(err)
	}
	if r.Nodes[1].ID != s.Nodes[0].ID {
		t.Fatal("referenced display name affected identity")
	}
	u, err := Normalize([]byte(`proxies:
 - {name: Entry, type: socks5, server: changed.test, port: 1080}
 - {name: Exit, type: vless, server: exit.test, port: 443, uuid: a, dialer-proxy: Entry}
`))
	if err != nil {
		t.Fatal(err)
	}
	if u.Nodes[1].ID == r.Nodes[1].ID {
		t.Fatal("dependency changed without changing outbound identity")
	}
}

func TestInvalidSubscriptionsRejectedWithoutEcho(t *testing.T) {
	cases := map[string]string{
		"empty":                 "",
		"empty nodes":           "proxies: []",
		"wrong shape":           "proxy-providers: {Secret: {url: https://secret.test/token}}",
		"missing name":          "proxies: [{type: vless, server: secret.test}]",
		"empty name":            "proxies: [{name: ' ', type: vless}]",
		"control name":          "proxies: [{name: \"bad\\nname\", type: vless}]",
		"type syntax":           "proxies: [{name: N, type: 'not valid'}]",
		"nonstring name":        "proxies: [{name: 1, type: vless}]",
		"nonstring key":         "proxies: [{name: N, type: vless, 1: bad}]",
		"duplicate key":         "proxies: [{name: N, type: vless, type: trojan}]",
		"duplicate source name": "proxies: [{name: N, type: vless}, {name: N, type: trojan}]",
		"direct":                "proxies: [{name: N, type: direct}]",
		"reject":                "proxies: [{name: N, type: reject}]",
		"reject drop":           "proxies: [{name: N, type: reject-drop}]",
		"dns":                   "proxies: [{name: N, type: dns}]",
		"pass":                  "proxies: [{name: N, type: pass}]",
		"dialer DIRECT":         "proxies: [{name: N, type: vless, dialer-proxy: DIRECT}]",
		"dialer missing":        "proxies: [{name: N, type: vless, dialer-proxy: absent}]",
		"dialer nonstring":      "proxies: [{name: N, type: vless, dialer-proxy: 4}]",
		"dialer self":           "proxies: [{name: N, type: vless, dialer-proxy: N}]",
		"dialer cycle":          "proxies: [{name: N, type: vless, dialer-proxy: M}, {name: M, type: vless, dialer-proxy: N}]",
		"file ca":               "proxies: [{name: N, type: vless, ca: /data/private.pem}]",
		"nested file":           "proxies: [{name: N, type: vless, options: {certificate_file: secret.pem}}]",
		"path":                  "proxies: [{name: N, type: vless, options: {key-path: private.pem}}]",
		"provider ref":          "proxies: [{name: N, type: vless, private-key-file: abc}]",
		"ssh key file":          "proxies: [{name: N, type: ssh, private-key: /data/key}]",
		"certificate file":      "proxies: [{name: N, type: trojan, certificate: /data/cert}]",
		"yaml alias cycle":      "proxies: [&a {name: N, type: vless, options: *a}]",
		"unknown yaml tag":      "proxies: [{name: N, type: vless, token: !custom secret}]",
		"nonfinite":             "proxies: [{name: N, type: vless, token: .nan}]",
		"extra document":        "proxies: [{name: N, type: vless}]\n---\nproxies: [{name: M, type: trojan}]",
		"invalid utf8":          "proxies: [{name: N, type: vless, token: \xff}]",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := Normalize([]byte(input))
			if err == nil {
				t.Fatal("invalid subscription accepted")
			}
			if len(r.Nodes) != 0 || len(r.Provider) != 0 {
				t.Fatal("partial private result returned on failure")
			}
			if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "private.pem") || strings.Contains(err.Error(), "token") {
				t.Fatal("error exposed source data")
			}
		})
	}
}

func TestLimits(t *testing.T) {
	if _, err := Normalize(bytes.Repeat([]byte(" "), 8*1024*1024+1)); err == nil {
		t.Fatal("oversize accepted")
	}
	deep := "proxies: [{name: N, type: vless, options: " + strings.Repeat("[", 34) + "1" + strings.Repeat("]", 34) + "}]"
	if _, err := Normalize([]byte(deep)); err == nil {
		t.Fatal("deep input accepted")
	}
	var many strings.Builder
	many.WriteString("proxies:\n")
	for i := range 4097 {
		fmt.Fprintf(&many, " - {name: 'N%d', type: vless, server: 's%d'}\n", i, i)
	}
	if _, err := Normalize([]byte(many.String())); err == nil {
		t.Fatal("too many nodes accepted")
	}
	items := "proxies: [{name: N, type: vless, values: [" + strings.Repeat("0,", 100001) + "]}]"
	if _, err := Normalize([]byte(items)); err == nil {
		t.Fatal("too many items accepted")
	}
	aliasBomb := "a: &a [1,2,3,4,5,6,7,8,9,10]\nb: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a,*a]\nc: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b,*b]\nd: &d [*c,*c,*c,*c,*c,*c,*c,*c,*c,*c]\nproxies: [{name: N, type: vless, options: [*d,*d,*d,*d,*d,*d,*d,*d,*d,*d]}]"
	if _, err := Normalize([]byte(aliasBomb)); err == nil {
		t.Fatal("alias amplification accepted")
	}
}

func TestInlineCAAndBoundedAliases(t *testing.T) {
	r, err := Normalize([]byte("\ufeffbase: &b {type: trojan, server: example.test, port: 443}\nproxies:\n - <<: *b\n   name: N\n   password: a\n   ca: |\n     -----BEGIN CERTIFICATE-----\n     abc\n     -----END CERTIFICATE-----\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Nodes) != 1 || r.Proxies[0]["server"] != "example.test" || r.Proxies[0]["ca"] == nil {
		t.Fatal("valid inline data lost")
	}
}

func TestAliasStringExpansionBounded(t *testing.T) {
	// Few YAML nodes can still amplify a large anchored string. Limit expanded
	// bytes in addition to structural item count before hashing or serializing.
	large := strings.Repeat("s", 256*1024)
	input := "shared: &s '" + large + "'\nproxies: [{name: N, type: vless, values: [" + strings.Repeat("*s,", 100) + "]}]"
	if _, err := Normalize([]byte(input)); err == nil {
		t.Fatal("expanded scalar byte limit missing")
	}
}

func TestCanonicalJSONAndInlineList(t *testing.T) {
	r, err := Normalize([]byte(`[{name: N, type: wireguard, server: example.test, private-key: keymaterial, options: {count: 1.0, enabled: true, item: null, path: /websocket}}]`))
	if err != nil {
		t.Fatal(err)
	}
	s, err := Normalize([]byte(`{"proxies":[{"name":"Other","type":"wireguard","server":"example.test","private-key":"keymaterial","options":{"path":"/websocket","item":null,"enabled":true,"count":1}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if r.Nodes[0].ID != s.Nodes[0].ID {
		t.Fatal("equivalent JSON/YAML changed identity")
	}
	metadata, err := json.Marshal(r.Nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	var public map[string]any
	if err = json.Unmarshal(metadata, &public); err != nil || len(public) != 3 || public["id"] == nil || public["name"] != "N" || public["type"] != "wireguard" {
		t.Fatal("public metadata contract changed")
	}
}

func TestDuplicateConnectionsDeduplicatedAfterReferenceRemapping(t *testing.T) {
	r, err := Normalize([]byte(`proxies:
 - {name: Z-entry, type: socks5, server: entry.test, port: 1080}
 - {name: A-entry, type: socks5, server: entry.test, port: 1080}
 - {name: Z-exit, type: vless, server: exit.test, port: 443, dialer-proxy: Z-entry}
 - {name: A-exit, type: vless, server: exit.test, port: 443, dialer-proxy: A-entry}
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Nodes) != 2 || len(r.Proxies) != 2 {
		t.Fatal("identical connections were not deduplicated")
	}
	if r.Nodes[0].Name != "A-entry" || r.Nodes[1].Name != "A-exit" {
		t.Fatal("duplicate display name selection is not deterministic")
	}
	if r.Proxies[1]["dialer-proxy"] != r.Nodes[0].ID {
		t.Fatal("deduplicated dialer target broken")
	}
	s, err := Normalize([]byte(`proxies:
 - {name: A-exit, type: vless, server: exit.test, port: 443, dialer-proxy: A-entry}
 - {name: A-entry, type: socks5, server: entry.test, port: 1080}
 - {name: Z-exit, type: vless, server: exit.test, port: 443, dialer-proxy: Z-entry}
 - {name: Z-entry, type: socks5, server: entry.test, port: 1080}
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Nodes) != 2 || r.Nodes[0] != s.Nodes[1] || r.Nodes[1] != s.Nodes[0] {
		t.Fatal("reorder changed deduplicated identities or names")
	}
}

func FuzzNormalize(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`proxies: [{name: N, type: vless, server: example.test, port: 443}]`),
		[]byte(`proxies: [&a {name: N, type: vless, options: *a}]`),
		[]byte("proxies: []"),
		[]byte("{bad YAML"),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		r, err := Normalize(raw)
		if err != nil {
			if len(r.Nodes) != 0 || len(r.Proxies) != 0 || len(r.Provider) != 0 {
				t.Fatal("partial result escaped failure")
			}
			return
		}
		if len(r.Nodes) < 1 || len(r.Nodes) > MaxProxies || len(r.Nodes) != len(r.Proxies) || !json.Valid(r.Provider) {
			t.Fatal("invalid successful result")
		}
		seen := map[string]bool{}
		for i, n := range r.Nodes {
			if seen[n.ID] || !strings.HasPrefix(n.ID, "n-") || len(n.ID) != 66 || r.Proxies[i]["name"] != n.ID {
				t.Fatal("identity invariant failed")
			}
			seen[n.ID] = true
		}
	})
}
