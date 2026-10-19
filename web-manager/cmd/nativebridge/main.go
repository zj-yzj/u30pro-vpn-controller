package main

import (
	"encoding/json"
	"fmt"
	"os"

	"u30pro/webmanager/internal/nativebridge"
)

func run(args []string) ([]byte, int) {
	if len(args) == 1 && args[0] == "version" {
		out, _ := json.Marshal(map[string]any{"ok": true, "version": nativebridge.Version})
		return out, 0
	}
	if os.Geteuid() != 0 {
		return nativebridge.SafeError("root_required"), 1
	}
	if len(args) != 1 {
		return nativebridge.SafeError("invalid_request"), 1
	}
	var err error
	switch args[0] {
	case "start":
		err = nativebridge.Start()
	case "serve":
		err = nativebridge.Serve()
	case "stop":
		err = nativebridge.Stop()
	default:
		return nativebridge.SafeError("invalid_request"), 1
	}
	if err != nil {
		return nativebridge.SafeError("unavailable"), 1
	}
	return []byte(`{"ok":true}`), 0
}
func main() {
	out, code := run(os.Args[1:])
	fmt.Println(string(out))
	if code != 0 {
		os.Exit(code)
	}
}
