package vmdocker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func (m *Manager) Init(ctx context.Context, dir, envFile string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	m.printf("vmdocker directory: %s\n", absDir)
	binaryPath := filepath.Join(absDir, filepath.FromSlash(NodeBinary))
	if !fileExists(m.stat, binaryPath) {
		return fmt.Errorf("vmdocker binary not found: %s; run `hype vmdocker get` first", binaryPath)
	}

	absEnvFile, err := filepath.Abs(envFile)
	if err != nil {
		return err
	}
	m.printf("env file: %s\n", absEnvFile)
	envMap, err := ParseEnvFile(absEnvFile, m.readFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(envMap["VMDOCKER_PRIVATE_KEY"]) == "" {
		return errors.New("env-file must define VMDOCKER_PRIVATE_KEY")
	}
	envMap["VMDOCKER_URL"] = "http://127.0.0.1:8080"

	configState, err := m.normalizeLocalRedisConfig(absDir)
	if err != nil {
		return err
	}
	m.printf("config: %s\n", configState)

	moduleSyncState, err := m.syncLocalModules(absDir)
	if err != nil {
		return err
	}
	m.printf("modules: %s\n", moduleSyncState)

	binaryState, err := m.syncNodeBinaryToCmd(absDir)
	if err != nil {
		return err
	}
	m.printf("binary: %s\n", binaryState)

	m.printf("redis: starting or reusing container %s\n", RedisContainer)
	redisState, err := m.ensureRedis(ctx)
	if err != nil {
		return err
	}
	m.printf("redis: %s\n", redisState)

	nodeDir := filepath.Join(absDir, "cmd")
	m.printf("node: starting or reusing daemon in dir %s\n", nodeDir)
	nodeState, err := m.ensureNodeStarted(ctx, absDir)
	if err != nil {
		return err
	}
	m.printf("node: %s\n", nodeState)

	healthCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	m.printf("health: waiting for %s\n", HealthURL)
	if err := m.waitForHealth(healthCtx, HealthURL); err != nil {
		logPath := m.latestNodeLog(absDir)
		if logPath != "" {
			return fmt.Errorf("%w; latest log: %s", err, logPath)
		}
		return err
	}
	m.printf("health: passed\n")

	env := make([]string, 0, len(envMap))
	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		env = append(env, fmt.Sprintf("%s=%s", key, envMap[key]))
	}

	examplesDir := filepath.Join(absDir, "examples")
	m.printf("examples init: running\n")
	_, err = m.runner.Output(ctx, examplesDir, env, "go", "run", "./", "init")
	if err != nil {
		return err
	}
	m.printf("examples init: completed\n")
	return err
}

func (m *Manager) ensureRedis(ctx context.Context) (string, error) {
	out, err := m.runner.Output(ctx, "", nil, "docker", "inspect", "-f", "{{.State.Running}}", RedisContainer)
	if err == nil {
		switch strings.TrimSpace(out) {
		case "true":
			if m.isRedisReachable() {
				return "already running", nil
			}
			return m.recreateRedisContainer(ctx, "running container not reachable on host port, recreated")
		case "false":
			_, err := m.runner.Output(ctx, "", nil, "docker", "start", RedisContainer)
			if err != nil {
				return "", err
			}
			if m.isRedisReachable() {
				return "started existing container", nil
			}
			return m.recreateRedisContainer(ctx, "existing container not reachable on host port, recreated")
		}
	}

	if err != nil && !isMissingDockerObjectError(err) {
		return "", err
	}

	if err := m.ensurePortFree(redisAddress); err != nil {
		return "port 6379 already in use, skipping managed redis startup", nil
	}
	_, err = m.runner.Output(ctx, "", nil, "docker", "run", "-d", "--name", RedisContainer, "-p", "6379:6379", redisImage)
	if err != nil {
		if isRedisPortAllocatedError(err) {
			return "port 6379 already in use, skipping managed redis startup", nil
		}
		return "", err
	}
	if m.isRedisReachable() {
		return "started new container", nil
	}
	return "", errors.New("redis container started but host port 127.0.0.1:6379 is not reachable")
}

func isMissingDockerObjectError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such object") || strings.Contains(msg, "no such container")
}

func isRedisPortAllocatedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "bind for 0.0.0.0:6379 failed") || strings.Contains(msg, "port is already allocated")
}

func (m *Manager) ensurePortFree(addr string) error {
	listener, err := m.listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %s is already in use", addr)
	}
	return listener.Close()
}

func (m *Manager) isRedisReachable() bool {
	if m.dial == nil {
		return false
	}
	conn, err := m.dial("tcp", redisAddress, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (m *Manager) recreateRedisContainer(ctx context.Context, state string) (string, error) {
	if _, err := m.runner.Output(ctx, "", nil, "docker", "rm", "-f", RedisContainer); err != nil {
		return "", err
	}
	if err := m.ensurePortFree(redisAddress); err != nil {
		return "", err
	}
	_, err := m.runner.Output(ctx, "", nil, "docker", "run", "-d", "--name", RedisContainer, "-p", "6379:6379", redisImage)
	if err != nil {
		return "", err
	}
	if !m.isRedisReachable() {
		return "", errors.New("redis container recreated but host port 127.0.0.1:6379 is not reachable")
	}
	return state, nil
}

func (m *Manager) ensureNodeStarted(ctx context.Context, dir string) (string, error) {
	cmdDir := filepath.Join(dir, "cmd")
	lockFiles, err := m.glob(filepath.Join(cmdDir, nodeLockGlob))
	if err != nil {
		return "", err
	}
	if len(lockFiles) > 0 {
		alive, err := m.hasLiveNodeProcess(lockFiles)
		if err != nil {
			return "", err
		}
		if alive {
			return "already running", nil
		}
		if err := m.removeStaleLocks(lockFiles); err != nil {
			return "", err
		}
		return m.startNode(ctx, dir, "stale lock removed, started")
	}

	return m.startNode(ctx, dir, "started")
}

func (m *Manager) startNode(ctx context.Context, dir, startedMessage string) (string, error) {
	cmdDir := filepath.Join(dir, "cmd")
	_, err := m.runner.Output(ctx, cmdDir, nil, "./"+nodeCmdBin, "--config", "./config.yaml", "start")
	if err != nil && !strings.Contains(err.Error(), "daemon is already running") {
		return "", err
	}
	if err != nil {
		return "already running", nil
	}
	return startedMessage, nil
}

func (m *Manager) hasLiveNodeProcess(lockFiles []string) (bool, error) {
	for _, lockFile := range lockFiles {
		pid, err := m.readPIDFromLock(lockFile)
		if err != nil {
			return false, err
		}
		if pid <= 0 {
			continue
		}
		if err := m.kill(pid, syscall.Signal(0)); err == nil || errors.Is(err, syscall.EPERM) {
			return true, nil
		} else if !errors.Is(err, syscall.ESRCH) {
			return false, err
		}
	}
	return false, nil
}

func (m *Manager) readPIDFromLock(path string) (int, error) {
	data, err := m.readFile(path)
	if err != nil {
		return 0, err
	}
	pidText := strings.TrimSpace(string(data))
	if pidText == "" {
		return 0, nil
	}
	pid, err := strconv.Atoi(pidText)
	if err != nil {
		return 0, fmt.Errorf("invalid pid in lock file %s: %w", path, err)
	}
	return pid, nil
}

func (m *Manager) syncNodeBinaryToCmd(dir string) (string, error) {
	srcPath := filepath.Join(dir, filepath.FromSlash(NodeBinary))
	info, err := m.stat(srcPath)
	if err != nil {
		return "", err
	}

	data, err := m.readFile(srcPath)
	if err != nil {
		return "", err
	}

	dstPath := filepath.Join(dir, "cmd", nodeCmdBin)
	if err := m.mkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return "", err
	}
	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o755
	}
	if err := m.writeFile(dstPath, data, mode); err != nil {
		return "", err
	}
	if m.chmod != nil {
		if err := m.chmod(dstPath, mode); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("copied %s to %s", srcPath, dstPath), nil
}

func (m *Manager) removeStaleLocks(lockFiles []string) error {
	for _, lockFile := range lockFiles {
		if err := m.remove(lockFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (m *Manager) waitForHealth(ctx context.Context, url string) error {
	for {
		status, err := m.httpGet(url)
		if err == nil && status >= 200 && status < 300 {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("vmdocker health check timed out for %s", url)
		default:
		}
		m.sleep(500 * time.Millisecond)
	}
}

func (m *Manager) latestNodeLog(dir string) string {
	matches, err := m.glob(filepath.Join(dir, "cmd", nodeLogGlob))
	if err != nil || len(matches) == 0 {
		return ""
	}
	slices.Sort(matches)
	return matches[len(matches)-1]
}

func (m *Manager) syncLocalModules(dir string) (string, error) {
	cmdWorkDir := filepath.Join(dir, "cmd")
	cmdDir := filepath.Join(dir, "cmd", "mod")

	names, err := m.generatedModuleJSONNames(cmdWorkDir)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "no generated module files to move", nil
	}

	if err := m.mkdirAll(cmdDir, 0o755); err != nil {
		return "", err
	}
	for _, name := range names {
		srcPath := filepath.Join(cmdWorkDir, name)
		dstPath := filepath.Join(cmdDir, name)
		if err := m.renameFile(srcPath, dstPath); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("moved %d generated module file(s) to cmd/mod", len(names)), nil
}

func (m *Manager) generatedModuleJSONNames(dir string) ([]string, error) {
	entries, err := m.readDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "mod-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}

func (m *Manager) renameFile(srcPath, dstPath string) error {
	if m.rename == nil {
		return os.Rename(srcPath, dstPath)
	}
	return m.rename(srcPath, dstPath)
}

func (m *Manager) normalizeLocalRedisConfig(dir string) (string, error) {
	configPath := filepath.Join(dir, "cmd", "config.yaml")
	data, err := m.readFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "no cmd/config.yaml to normalize", nil
		}
		return "", err
	}

	original := string(data)
	updated := strings.ReplaceAll(original, "redis://@localhost:6379/", "redis://@127.0.0.1:6379/")
	if updated == original {
		return "redis url already normalized", nil
	}
	if err := m.writeFile(configPath, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return "normalized redis url to 127.0.0.1 in cmd/config.yaml", nil
}
