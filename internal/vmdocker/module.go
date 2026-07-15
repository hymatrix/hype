package vmdocker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ModuleBuildOptions struct {
	CheckoutDir  string
	ProfilePath  string
	AgentBinPath string
	NodeURL      string
	PrivateKey   string
}

func (m *Manager) BuildModule(ctx context.Context, opts ModuleBuildOptions) error {
	checkout, err := filepath.Abs(opts.CheckoutDir)
	if err != nil {
		return err
	}
	if info, err := m.stat(checkout); err != nil || !info.IsDir() {
		return fmt.Errorf("vmdockerv2 checkout not found: %s", checkout)
	}
	envValues, err := m.readBuildEnv(checkout)
	if err != nil {
		return err
	}
	agentBin := firstValue(opts.AgentBinPath, os.Getenv("VMDOCKER_AGENT_BIN"), envValues["VMDOCKER_AGENT_BIN"])
	nodeURL := firstValue(opts.NodeURL, os.Getenv("VMDOCKER_URL"), envValues["VMDOCKER_URL"], "http://127.0.0.1:8080")
	privateKey := firstValue(opts.PrivateKey, os.Getenv("VMDOCKER_PRIVATE_KEY"), os.Getenv("HYPE_PRIVATE_KEY"), os.Getenv("PRV_KEY"), envValues["VMDOCKER_PRIVATE_KEY"])
	if privateKey == "" {
		return errors.New("private-key is required")
	}
	profile, err := absoluteRegularFile(opts.ProfilePath, m.stat, "profile")
	if err != nil {
		return err
	}
	agentBin, err = absoluteRegularFile(agentBin, m.stat, "agent-bin")
	if err != nil {
		return err
	}
	stdout, stderr := m.out, m.errOut
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	cmdDir := filepath.Join(checkout, "cmd")
	if err := m.runner.Run(ctx, cmdDir, []string{
		"VMDOCKER_URL=" + nodeURL,
		"VMDOCKER_PRIVATE_KEY=" + privateKey,
	}, stdout, stderr, "go", "run", "./module", "--profile", profile, "--agent-bin", agentBin); err != nil {
		return err
	}
	moduleSyncState, err := m.syncLocalModules(checkout)
	if err != nil {
		return err
	}
	m.printf("modules: %s\n", moduleSyncState)
	return nil
}

func (m *Manager) readBuildEnv(checkout string) (map[string]string, error) {
	path := strings.TrimSpace(os.Getenv("VMDOCKER_ENV_FILE"))
	if path == "" {
		path = filepath.Join(checkout, ".env")
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(checkout, path)
	}
	values, err := ParseEnvFile(path, m.readFile)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	return values, err
}

func firstValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func absoluteRegularFile(path string, stat func(string) (os.FileInfo, error), label string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := stat(absPath)
	if err != nil {
		return "", fmt.Errorf("%s not found: %s: %w", label, absPath, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file: %s", label, absPath)
	}
	return absPath, nil
}
