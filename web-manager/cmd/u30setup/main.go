package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"u30pro/webmanager/internal/installer"
)

const usage = "u30setup check|prepare|boot|status|uninstall [--module /data/adb/modules[/_update]/codex_u30pro_vpn]\ncheck may omit --module; all other actions require it."

func parseArgs(args []string) (action, module string, help bool, err error) {
	invalid := func() (string, string, bool, error) { return "", "", false, errors.New("invalid_request") }
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		return "", "", true, nil
	}
	if len(args) != 1 && len(args) != 3 {
		return invalid()
	}
	action = args[0]
	switch action {
	case "check", "prepare", "boot", "status", "uninstall":
	default:
		return invalid()
	}
	if len(args) == 3 {
		if args[1] != "--module" || !installer.ValidModuleDir(args[2]) {
			return invalid()
		}
		module = args[2]
	}
	if action != "check" && module == "" {
		return invalid()
	}
	return action, module, false, nil
}
func main() {
	action, module, help, err := parseArgs(os.Args[1:])
	if help {
		fmt.Println(usage)
		return
	}
	if err != nil {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": false, "version": installer.Version, "complete": false, "error": map[string]string{"code": "invalid_request", "message": "invalid_request"}})
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	engine := installer.New()
	var r installer.Result
	switch action {
	case "check":
		r, err = engine.Check(ctx, module)
	case "prepare":
		r, err = engine.Prepare(ctx, module)
	case "boot":
		r, err = engine.Boot(ctx, module)
	case "status":
		r, err = engine.Status(ctx, module)
	case "uninstall":
		r, err = engine.Uninstall(ctx, module)
	}
	if err != nil {
		json.NewEncoder(os.Stdout).Encode(installer.ErrorResult(err, r))
		if r.Deferred {
			os.Exit(2)
		}
		os.Exit(1)
	}
	json.NewEncoder(os.Stdout).Encode(r)
}
