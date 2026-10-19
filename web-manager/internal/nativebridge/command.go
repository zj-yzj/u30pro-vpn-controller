package nativebridge

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"time"
)

type boundedOutput struct {
	buf      []byte
	limit    int
	overflow bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > b.limit-len(b.buf) {
		b.overflow = true
		return 0, errors.New("output_limit")
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}
func RunCommand(ctx context.Context, path string, args []string, limit int) ([]byte, error) {
	if limit < 1 || limit > MaxResponse {
		return nil, errUnavailable
	}
	allowed := path == WebctlPath || path == CmdPath || path == PMPath || path == DumpsysPath
	if !allowed {
		return nil, errDenied
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.WaitDelay = time.Second
	cmd.Env = []string{"PATH=/system/bin:/system/xbin", "HOME=/", "LANG=C", "LC_ALL=C"}
	cmd.Stdin = nil
	cmd.Stderr = io.Discard
	output := &boundedOutput{limit: limit}
	cmd.Stdout = output
	err := cmd.Run()
	if output.overflow {
		return nil, errUnavailable
	}
	return output.buf, err
}
