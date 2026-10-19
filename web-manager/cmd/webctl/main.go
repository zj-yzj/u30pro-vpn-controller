package main

import (
	"encoding/json"
	"os"

	"u30pro/webmanager/internal/manager"
)

func main() {
	result, code := run(os.Args[1:])
	json.NewEncoder(os.Stdout).Encode(result)
	if code != 0 {
		os.Exit(code)
	}
}
func run(args []string) (any, int) {
	if len(args) == 1 && args[0] == "version" {
		return map[string]any{"ok": true, "version": manager.Version}, 0
	}
	bad := func(err error) (any, int) { return manager.SafeError(manager.ErrorCode(err)), 1 }
	if os.Geteuid() != 0 {
		return manager.SafeError("root_required"), 1
	}
	m := manager.New()
	if len(args) == 0 {
		return manager.SafeError("invalid_request"), 1
	}
	var err error
	switch args[0] {
	case "bootstrap":
		if len(args) != 1 {
			break
		}
		err = m.Bootstrap()
		if err == nil {
			return map[string]any{"ok": true, "installed": true}, 0
		}
		return bad(err)
	case "init":
		if len(args) != 1 {
			break
		}
		err = m.Init()
		if err == nil {
			return map[string]any{"ok": true, "installed": true}, 0
		}
		return bad(err)
	case "status":
		if len(args) != 1 {
			break
		}
		s, e := m.Status()
		if e != nil {
			return bad(e)
		}
		return s, 0
	case "request":
		if len(args) != 2 {
			break
		}
		r, e := m.Queue(args[1])
		if e != nil {
			return bad(e)
		}
		return r, 0
	case "worker":
		if len(args) != 2 {
			break
		}
		err = m.Worker(args[1])
		if err != nil {
			return bad(err)
		}
		return map[string]bool{"ok": true}, 0
	case "boot":
		if len(args) != 1 {
			break
		}
		err = m.Boot()
		if err != nil {
			return bad(err)
		}
		return map[string]bool{"ok": true}, 0
	}
	return manager.SafeError("invalid_request"), 1
}
