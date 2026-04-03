package vmdocker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	Output(ctx context.Context, dir string, env []string, name string, args ...string) (string, error)
}

type ExecCommandRunner struct{}

func (ExecCommandRunner) Output(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		if output == "" {
			return "", fmt.Errorf("%s %v failed: %w", name, args, err)
		}
		return output, fmt.Errorf("%s %v failed: %w: %s", name, args, err, output)
	}
	return output, nil
}
