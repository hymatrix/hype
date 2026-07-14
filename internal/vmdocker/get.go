package vmdocker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *Manager) Get(ctx context.Context, dir, ref string) (string, string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = DefaultRef
	}

	info, err := m.stat(absDir)
	switch {
	case err == nil && !info.IsDir():
		return "", "", fmt.Errorf("vmdocker dir is not a directory: %s", absDir)
	case os.IsNotExist(err):
		if err := m.mkdirAll(filepath.Dir(absDir), 0o755); err != nil {
			return "", "", err
		}
		if _, err := m.runner.Output(ctx, "", nil, "git", "clone", RepoURL, absDir); err != nil {
			return "", "", err
		}
	case err != nil:
		return "", "", err
	}

	binaryPath := filepath.Join(absDir, filepath.FromSlash(NodeBinary))
	shouldBuild, err := m.checkoutRef(ctx, absDir, ref, binaryPath)
	if err != nil {
		return "", "", err
	}
	if shouldBuild {
		if err := m.mkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
			return "", "", err
		}
		if _, err := m.runner.Output(ctx, absDir, nil, "go", "mod", "tidy"); err != nil {
			return "", "", err
		}
		if _, err := m.runner.Output(ctx, absDir, nil, "go", "build", "-o", "./build/hymx-node", "./cmd"); err != nil {
			return "", "", err
		}
	}
	return ref, binaryPath, nil
}

func (m *Manager) checkoutRef(ctx context.Context, dir, ref, binaryPath string) (bool, error) {
	remote, err := m.runner.Output(ctx, dir, nil, "git", "remote", "get-url", "origin")
	if err != nil {
		return false, fmt.Errorf("existing dir is not a vmdockerv2 git repository: %w", err)
	}
	if !isVmdockerRemote(remote) {
		return false, fmt.Errorf("existing dir is not a vmdockerv2 git repository: %s", dir)
	}

	if _, err := m.runner.Output(ctx, dir, nil, "git", "fetch", "--force", "origin", ref); err != nil {
		return false, err
	}
	target, err := m.runner.Output(ctx, dir, nil, "git", "rev-parse", "FETCH_HEAD")
	if err != nil {
		return false, err
	}
	current, err := m.runner.Output(ctx, dir, nil, "git", "rev-parse", "HEAD")
	if err != nil {
		return false, err
	}
	target = strings.TrimSpace(target)
	if target == strings.TrimSpace(current) {
		return !fileExists(m.stat, binaryPath), nil
	}
	status, err := m.runner.Output(ctx, dir, nil, "git", "status", "--short", "--untracked-files=no")
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(status) != "" {
		return false, errors.New("vmdockerv2 checkout has tracked changes; refusing to switch ref")
	}
	if _, err := m.runner.Output(ctx, dir, nil, "git", "checkout", "--detach", target); err != nil {
		return false, err
	}
	return true, nil
}

func isVmdockerRemote(remote string) bool {
	remote = strings.TrimSpace(strings.TrimSuffix(remote, "/"))
	return remote == RepoURL || remote == repoURLNoSuffix || remote == repoURLSSH
}

func fileExists(stat func(string) (os.FileInfo, error), path string) bool {
	info, err := stat(path)
	return err == nil && !info.IsDir()
}
