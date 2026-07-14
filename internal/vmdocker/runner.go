package vmdocker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	Output(ctx context.Context, dir string, env []string, name string, args ...string) (string, error)
	Run(ctx context.Context, dir string, env []string, stdout, stderr io.Writer, name string, args ...string) error
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

func (ExecCommandRunner) Run(ctx context.Context, dir string, env []string, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v failed: %w", name, args, err)
	}
	return nil
}
