package main

import "testing"

func TestCLIRestrictsActionsAndModulePathsBeforeExecution(t *testing.T) {
	for _, args := range [][]string{{}, {"boot"}, {"prepare", "--module", "/tmp/evil"}, {"check", "--runtime", "/tmp"}, {"check", "extra"}, {"boot", "--module", "/data/adb/modules/codex_u30pro_vpn", "extra"}, {"upgrade", "--module", "/data/adb/modules/codex_u30pro_vpn"}} {
		if _, _, _, err := parseArgs(args); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
	for _, args := range [][]string{{"check"}, {"check", "--module", "/data/adb/modules_update/codex_u30pro_vpn"}, {"boot", "--module", "/data/adb/modules/codex_u30pro_vpn"}, {"status", "--module", "/data/adb/modules/codex_u30pro_vpn"}, {"uninstall", "--module", "/data/adb/modules/codex_u30pro_vpn"}} {
		if _, _, _, err := parseArgs(args); err != nil {
			t.Fatalf("rejected %q: %v", args, err)
		}
	}
}
