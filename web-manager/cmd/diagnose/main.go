//go:build diagnostic

package main

import (
	"encoding/json"
	"os"
	"u30pro/webmanager/internal/manager"
)

func main() {
	if os.Geteuid() != 0 {
		json.NewEncoder(os.Stdout).Encode(manager.SafeError("root_required"))
		os.Exit(1)
	}
	if len(os.Args) != 1 {
		json.NewEncoder(os.Stdout).Encode(manager.SafeError("invalid_request"))
		os.Exit(1)
	}
	report, err := manager.New().Diagnose()
	if err != nil {
		json.NewEncoder(os.Stdout).Encode(manager.SafeError(manager.ErrorCode(err)))
		os.Exit(1)
	}
	json.NewEncoder(os.Stdout).Encode(report)
}
