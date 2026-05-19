package utils

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func ExecScript(ctx context.Context, script string, out io.Writer) error {
	script = strings.TrimSpace(script)
	if script == "" {
		return nil
	}
	if out == nil {
		out = io.Discard
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Stdout = out
	cmd.Stderr = out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script failed: %w", err)
	}
	return nil
}
