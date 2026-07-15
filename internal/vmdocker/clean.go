package vmdocker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func (m *Manager) Clean(ctx context.Context, dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	m.printf("vmdocker directory: %s\n", absDir)

	nodeState, err := m.stopLocalNode(absDir)
	if err != nil {
		return err
	}
	m.printf("node: %s\n", nodeState)

	fileState, err := m.removeLocalRuntimeFiles(absDir)
	if err != nil {
		return err
	}
	m.printf("files: %s\n", fileState)

	redisState, err := m.removeManagedRedis(ctx)
	if err != nil {
		return err
	}
	m.printf("redis: %s\n", redisState)
	return nil
}

func (m *Manager) stopLocalNode(dir string) (string, error) {
	lockFiles, err := m.glob(filepath.Join(dir, "cmd", nodeLockGlob))
	if err != nil {
		return "", err
	}
	if len(lockFiles) == 0 {
		return "no lock files", nil
	}

	stopped := 0
	removed := 0
	for _, lockFile := range lockFiles {
		pid, err := m.readPIDFromLock(lockFile)
		if err != nil {
			return "", err
		}
		if pid > 0 {
			if err := m.kill(pid, syscall.SIGTERM); err == nil {
				stopped++
			} else if !errors.Is(err, syscall.ESRCH) {
				return "", err
			}
		}
		if err := m.remove(lockFile); err != nil && !os.IsNotExist(err) {
			return "", err
		}
		removed++
	}
	return fmt.Sprintf("sent SIGTERM to %d process(es), removed %d lock file(s)", stopped, removed), nil
}

func (m *Manager) removeLocalRuntimeFiles(dir string) (string, error) {
	removed := 0

	if ok, err := m.removeIfExists(filepath.Join(dir, "cmd", nodeCmdBin)); err != nil {
		return "", err
	} else if ok {
		removed++
	}

	logFiles, err := m.glob(filepath.Join(dir, "cmd", nodeLogGlob))
	if err != nil {
		return "", err
	}
	for _, logFile := range logFiles {
		if ok, err := m.removeIfExists(logFile); err != nil {
			return "", err
		} else if ok {
			removed++
		}
	}

	cmdModDir := filepath.Join(dir, "cmd", "mod")
	entries, err := m.readDir(cmdModDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		return fmt.Sprintf("removed %d generated file(s)", removed), nil
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if ok, err := m.removeIfExists(filepath.Join(cmdModDir, entry.Name())); err != nil {
			return "", err
		} else if ok {
			removed++
		}
	}

	return fmt.Sprintf("removed %d generated file(s)", removed), nil
}

func (m *Manager) removeIfExists(path string) (bool, error) {
	if err := m.remove(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *Manager) removeManagedRedis(ctx context.Context) (string, error) {
	_, err := m.runner.Output(ctx, "", nil, "docker", "rm", "-f", RedisContainer)
	if err != nil {
		if isMissingDockerObjectError(err) {
			return "container not found", nil
		}
		return "", err
	}
	return "removed container " + RedisContainer, nil
}
