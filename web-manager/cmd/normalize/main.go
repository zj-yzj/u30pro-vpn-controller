// normalize prepares private installation artifacts without printing node secrets.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"u30pro/webmanager/internal/subscription"
)

func writeNew(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: normalize input provider-output metadata-output validation-output")
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fail("INPUT_UNREADABLE")
	}
	r, err := subscription.Normalize(raw)
	if err != nil {
		// Package errors are fixed categories, never parser excerpts.
		fail("SUBSCRIPTION_INVALID: " + err.Error())
	}
	meta, err := json.Marshal(r.Nodes)
	if err != nil {
		fail("METADATA_INVALID")
	}
	// File providers are not loaded by mihomo -t; inline all nodes in this
	// isolated validation document, with every listener/controller/TUN absent.
	ids := make([]string, len(r.Nodes))
	for i, n := range r.Nodes {
		ids[i] = n.ID
	}
	candidate := map[string]any{
		"port": 0, "socks-port": 0, "mixed-port": 0, "redir-port": 0, "tproxy-port": 0,
		"allow-lan": false, "bind-address": "127.0.0.1", "ipv6": false,
		"mode": "rule", "log-level": "silent", "find-process-mode": "off",
		"proxies":      r.Proxies,
		"proxy-groups": []map[string]any{{"name": "VPN", "type": "select", "proxies": ids}},
		"rules":        []string{"MATCH,VPN"}, "dns": map[string]any{"enable": false}, "tun": map[string]any{"enable": false},
	}
	validation, err := yaml.Marshal(candidate)
	if err != nil {
		fail("VALIDATION_INVALID")
	}
	if writeNew(os.Args[2], r.Provider) != nil || writeNew(os.Args[3], meta) != nil || writeNew(os.Args[4], validation) != nil {
		fail("OUTPUT_WRITE_FAILED")
	}
	fmt.Printf("NORMALIZED nodes=%d\n", len(r.Nodes))
}

func fail(code string) { fmt.Fprintln(os.Stderr, code); os.Exit(1) }
