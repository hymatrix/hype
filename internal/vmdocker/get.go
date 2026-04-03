package vmdocker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *Manager) Get(ctx context.Context, dir, version string) (string, string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}

	tag, err := m.resolveVersion(ctx, version)
	if err != nil {
		return "", "", err
	}

	binaryPath := filepath.Join(absDir, filepath.FromSlash(NodeBinary))
	shouldBuild := false
	info, err := m.stat(absDir)
	switch {
	case err == nil && !info.IsDir():
		return "", "", fmt.Errorf("vmdocker dir is not a directory: %s", absDir)
	case err == nil:
		shouldBuild, err = m.ensureExistingRepo(ctx, absDir, tag, binaryPath)
		if err != nil {
			return "", "", err
		}
	case os.IsNotExist(err):
		if err := os.MkdirAll(filepath.Dir(absDir), 0o755); err != nil {
			return "", "", err
		}
		if _, err := m.runner.Output(ctx, "", nil, "git", "clone", "--branch", tag, "--depth", "1", RepoURL, absDir); err != nil {
			return "", "", err
		}
		shouldBuild = true
	default:
		return "", "", err
	}

	if shouldBuild || !fileExists(m.stat, binaryPath) {
		if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
			return "", "", err
		}
		if _, err := m.runner.Output(ctx, absDir, nil, "go", "mod", "tidy"); err != nil {
			return "", "", err
		}
		if _, err := m.runner.Output(ctx, absDir, nil, "go", "build", "-o", "./build/hymx-node", "./cmd"); err != nil {
			return "", "", err
		}
	}

	return tag, binaryPath, nil
}

func (m *Manager) resolveVersion(ctx context.Context, version string) (string, error) {
	if version != "" {
		if !IsSemverTag(version) {
			return "", fmt.Errorf("invalid vmdocker version: %s", version)
		}
		return version, nil
	}

	out, err := m.runner.Output(ctx, "", nil, "git", "ls-remote", "--tags", "--refs", RepoURL)
	if err != nil {
		return "", err
	}
	return LatestSemverTag(out)
}

func (m *Manager) ensureExistingRepo(ctx context.Context, dir, tag, binaryPath string) (bool, error) {
	remote, err := m.runner.Output(ctx, dir, nil, "git", "remote", "get-url", "origin")
	if err != nil {
		return false, fmt.Errorf("existing dir is not a vmdocker git repository: %w", err)
	}
	if !isVmdockerRemote(remote) {
		return false, fmt.Errorf("existing dir is not a vmdocker git repository: %s", dir)
	}

	currentTag, tagErr := m.runner.Output(ctx, dir, nil, "git", "describe", "--tags", "--exact-match")
	if tagErr == nil && strings.TrimSpace(currentTag) == tag && fileExists(m.stat, binaryPath) {
		return false, nil
	}

	if _, err := m.runner.Output(ctx, dir, nil, "git", "fetch", "--tags", "--force", "origin"); err != nil {
		return false, err
	}
	if _, err := m.runner.Output(ctx, dir, nil, "git", "checkout", "--detach", tag); err != nil {
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
