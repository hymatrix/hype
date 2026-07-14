package vmdocker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

type fakeRunner struct {
	calls []string
	run   func(dir string, env []string, name string, args ...string) (string, error)
}

func (f *fakeRunner) Output(_ context.Context, dir string, env []string, name string, args ...string) (string, error) {
	call := strings.TrimSpace(strings.Join(append([]string{name}, args...), " "))
	f.calls = append(f.calls, call)
	if f.run == nil {
		return "", nil
	}
	return f.run(dir, env, name, args...)
}

type fakeListener struct {
	closed bool
}

func (l *fakeListener) Accept() (net.Conn, error) { return nil, errors.New("not implemented") }
func (l *fakeListener) Close() error {
	l.closed = true
	return nil
}
func (l *fakeListener) Addr() net.Addr { return fakeAddr("tcp") }

type fakeAddr string

func (a fakeAddr) Network() string { return string(a) }
func (a fakeAddr) String() string  { return "127.0.0.1:6379" }

type fakeConn struct{}

func (c *fakeConn) Read(b []byte) (int, error)       { return 0, nil }
func (c *fakeConn) Write(b []byte) (int, error)      { return len(b), nil }
func (c *fakeConn) Close() error                     { return nil }
func (c *fakeConn) LocalAddr() net.Addr              { return fakeAddr("tcp") }
func (c *fakeConn) RemoteAddr() net.Addr             { return fakeAddr("tcp") }
func (c *fakeConn) SetDeadline(time.Time) error      { return nil }
func (c *fakeConn) SetReadDeadline(time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(time.Time) error { return nil }

func newTestManager(runner *fakeRunner) *Manager {
	return &Manager{
		runner:    runner,
		httpGet:   func(string) (int, error) { return 200, nil },
		stat:      os.Stat,
		readDir:   os.ReadDir,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
		sleep:     func(time.Duration) {},
		listen:    net.Listen,
		glob:      filepath.Glob,
	}
}

func TestParseEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := strings.Join([]string{
		"# comment",
		`VMDOCKER_PRIVATE_KEY="0xabc"`,
		"OPENCLAW_PROVIDER=zen",
		"INVALID_LINE",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	values, err := ParseEnvFile(path, os.ReadFile)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if values["VMDOCKER_PRIVATE_KEY"] != "0xabc" {
		t.Fatalf("expected private key to be parsed, got %#v", values)
	}
	if values["OPENCLAW_PROVIDER"] != "zen" {
		t.Fatalf("expected provider to be parsed, got %#v", values)
	}
}

func TestParseEnvContent(t *testing.T) {
	values, err := ParseEnvContent(strings.Join([]string{
		"# comment",
		"VMDOCKER_PRIVATE_KEY='0xdef'",
		"OPENCLAW_MODEL=plan",
	}, "\n"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if values["VMDOCKER_PRIVATE_KEY"] != "0xdef" {
		t.Fatalf("expected private key to be parsed, got %#v", values)
	}
	if values["OPENCLAW_MODEL"] != "plan" {
		t.Fatalf("expected model to be parsed, got %#v", values)
	}
}

func TestGetRejectsNonRepoDirectory(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{
		run: func(dir string, env []string, name string, args ...string) (string, error) {
			if name == "git" && len(args) >= 3 && args[0] == "remote" {
				return "https://example.com/not-vmdocker.git", nil
			}
			return "", nil
		},
	}
	manager := newTestManager(runner)

	if _, _, err := manager.Get(context.Background(), dir, "feature/profile"); err == nil || !strings.Contains(err.Error(), "not a vmdockerv2 git repository") {
		t.Fatalf("expected repo mismatch error, got %v", err)
	}
}

func TestGetDefaultsToMainAndBuildsV2(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "vmdockerv2")
	runner := &fakeRunner{
		run: func(dir string, env []string, name string, args ...string) (string, error) {
			switch strings.Join(append([]string{name}, args...), " ") {
			case "git remote get-url origin":
				return RepoURL, nil
			case "git rev-parse FETCH_HEAD":
				return "new-commit", nil
			case "git rev-parse HEAD":
				return "old-commit", nil
			case "git status --short --untracked-files=no":
				return "", nil
			}
			return "", nil
		},
	}
	manager := newTestManager(runner)
	ref, binaryPath, err := manager.Get(context.Background(), dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if ref != "main" {
		t.Fatalf("ref = %q", ref)
	}
	if binaryPath != filepath.Join(dir, "build", "hymx-node") {
		t.Fatalf("binary = %q", binaryPath)
	}
	for _, want := range []string{
		"git clone https://github.com/cryptowizard0/vmdockerv2.git " + dir,
		"git fetch --force origin main",
		"git checkout --detach new-commit",
		"go mod tidy",
		"go build -o ./build/hymx-node ./cmd",
	} {
		if !containsCall(runner.calls, want) {
			t.Fatalf("missing %q in %#v", want, runner.calls)
		}
	}
}

func TestGetRefusesDirtyCheckoutBeforeSwitch(t *testing.T) {
	runner := &fakeRunner{
		run: func(dir string, env []string, name string, args ...string) (string, error) {
			switch strings.Join(append([]string{name}, args...), " ") {
			case "git remote get-url origin":
				return RepoURL, nil
			case "git rev-parse FETCH_HEAD":
				return "new-commit", nil
			case "git rev-parse HEAD":
				return "old-commit", nil
			case "git status --short --untracked-files=no":
				return " M go.mod", nil
			}
			return "", nil
		},
	}
	_, _, err := newTestManager(runner).Get(context.Background(), t.TempDir(), "feature/profile")
	if err == nil || !strings.Contains(err.Error(), "tracked changes") {
		t.Fatalf("expected tracked changes error, got %v", err)
	}
}

func TestInitRequiresPrivateKeyInEnvFile(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "hymx-node"), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(dir, "local.env")
	if err := os.WriteFile(envFile, []byte("OPENCLAW_PROVIDER=zen\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	manager := &Manager{
		runner:    &fakeRunner{},
		httpGet:   func(string) (int, error) { return 200, nil },
		stat:      os.Stat,
		readDir:   os.ReadDir,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
		sleep:     func(time.Duration) {},
		listen:    net.Listen,
		glob:      filepath.Glob,
	}

	err := manager.Init(context.Background(), dir, envFile)
	if err == nil || !strings.Contains(err.Error(), "VMDOCKER_PRIVATE_KEY") {
		t.Fatalf("expected private key validation error, got %v", err)
	}
}

func TestInitOrchestratesRedisNodeAndExamples(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "hymx-node"), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(dir, "local.env")
	if err := os.WriteFile(envFile, []byte("VMDOCKER_PRIVATE_KEY=0xabc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	listener := &fakeListener{}
	statuses := []int{503, 200}
	expectedCmdDir := filepath.Join(dir, "cmd")
	expectedExamplesDir := filepath.Join(dir, "examples")
	runner := &fakeRunner{
		run: func(dir string, env []string, name string, args ...string) (string, error) {
			call := strings.Join(append([]string{name}, args...), " ")
			switch call {
			case "docker inspect -f {{.State.Running}} hype-vmdocker-redis":
				return "", fmt.Errorf("No such object: %s", RedisContainer)
			case "docker run -d --name hype-vmdocker-redis -p 6379:6379 redis:latest":
				return "container-id", nil
			case "./hymx-node --config ./config.yaml start":
				if dir != expectedCmdDir {
					return "", fmt.Errorf("unexpected node dir: %s", dir)
				}
				return "started", nil
			case "go run ./ init":
				if dir != expectedExamplesDir {
					return "", fmt.Errorf("unexpected examples dir: %s", dir)
				}
				if !hasEnv(env, "VMDOCKER_PRIVATE_KEY=0xabc") {
					return "", errors.New("missing VMDOCKER_PRIVATE_KEY env")
				}
				if !hasEnv(env, "VMDOCKER_URL=http://127.0.0.1:8080") {
					return "", errors.New("missing VMDOCKER_URL env")
				}
				return "ok", nil
			default:
				return "", nil
			}
		},
	}

	manager := &Manager{
		runner: runner,
		httpGet: func(string) (int, error) {
			status := statuses[0]
			if len(statuses) > 1 {
				statuses = statuses[1:]
			}
			if status >= 400 {
				return status, errors.New("not ready")
			}
			return status, nil
		},
		stat:      os.Stat,
		readDir:   os.ReadDir,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
		sleep:     func(time.Duration) {},
		listen: func(network, address string) (net.Listener, error) {
			if address != redisAddress {
				t.Fatalf("unexpected redis address %s", address)
			}
			return listener, nil
		},
		dial: func(network, address string, timeout time.Duration) (net.Conn, error) {
			if address != redisAddress {
				t.Fatalf("unexpected redis dial address %s", address)
			}
			return &fakeConn{}, nil
		},
		glob: func(pattern string) ([]string, error) {
			switch filepath.Base(pattern) {
			case nodeLockGlob:
				return nil, nil
			case nodeLogGlob:
				return nil, nil
			default:
				return filepath.Glob(pattern)
			}
		},
	}

	if err := manager.Init(context.Background(), dir, envFile); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	copiedBinary := filepath.Join(dir, "cmd", nodeCmdBin)
	data, err := os.ReadFile(copiedBinary)
	if err != nil {
		t.Fatalf("expected copied node binary, got %v", err)
	}
	if string(data) != "bin" {
		t.Fatalf("unexpected copied node binary content: %s", data)
	}
	if !listener.closed {
		t.Fatal("expected redis port probe listener to be closed")
	}
	expectedCalls := []string{
		"docker inspect -f {{.State.Running}} hype-vmdocker-redis",
		"docker run -d --name hype-vmdocker-redis -p 6379:6379 redis:latest",
		"./hymx-node --config ./config.yaml start",
	}
	for _, want := range expectedCalls {
		if !containsCall(runner.calls, want) {
			t.Fatalf("expected call %q in %#v", want, runner.calls)
		}
	}
}

func TestInitAcceptsLowercaseMissingDockerObjectError(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "hymx-node"), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(dir, "local.env")
	if err := os.WriteFile(envFile, []byte("VMDOCKER_PRIVATE_KEY=0xabc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	listener := &fakeListener{}
	runner := &fakeRunner{
		run: func(dir string, env []string, name string, args ...string) (string, error) {
			call := strings.Join(append([]string{name}, args...), " ")
			switch call {
			case "docker inspect -f {{.State.Running}} hype-vmdocker-redis":
				return "", errors.New("docker [inspect -f {{.State.Running}} hype-vmdocker-redis] failed: exit status 1: error: no such object: hype-vmdocker-redis")
			case "docker run -d --name hype-vmdocker-redis -p 6379:6379 redis:latest":
				return "container-id", nil
			case "./hymx-node --config ./config.yaml start":
				return "started", nil
			case "go run ./ init":
				return "ok", nil
			default:
				return "", nil
			}
		},
	}

	manager := &Manager{
		runner:    runner,
		httpGet:   func(string) (int, error) { return 200, nil },
		stat:      os.Stat,
		readDir:   os.ReadDir,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
		sleep:     func(time.Duration) {},
		listen:    func(network, address string) (net.Listener, error) { return listener, nil },
		dial: func(network, address string, timeout time.Duration) (net.Conn, error) {
			if address != redisAddress {
				t.Fatalf("unexpected redis dial address %s", address)
			}
			return &fakeConn{}, nil
		},
		glob: func(pattern string) ([]string, error) {
			switch filepath.Base(pattern) {
			case nodeLockGlob, nodeLogGlob:
				return nil, nil
			default:
				return filepath.Glob(pattern)
			}
		},
	}

	if err := manager.Init(context.Background(), dir, envFile); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !containsCall(runner.calls, "docker run -d --name hype-vmdocker-redis -p 6379:6379 redis:latest") {
		t.Fatalf("expected redis docker run, got %#v", runner.calls)
	}
}

func TestInitSkipsManagedRedisWhenPortAlreadyInUse(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "hymx-node"), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(dir, "local.env")
	if err := os.WriteFile(envFile, []byte("VMDOCKER_PRIVATE_KEY=0xabc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		run: func(dir string, env []string, name string, args ...string) (string, error) {
			call := strings.Join(append([]string{name}, args...), " ")
			switch call {
			case "docker inspect -f {{.State.Running}} hype-vmdocker-redis":
				return "", fmt.Errorf("error: no such object: %s", RedisContainer)
			case "./hymx-node --config ./config.yaml start":
				return "started", nil
			case "go run ./ init":
				return "ok", nil
			default:
				return "", nil
			}
		},
	}

	manager := &Manager{
		runner:    runner,
		httpGet:   func(string) (int, error) { return 200, nil },
		stat:      os.Stat,
		readDir:   os.ReadDir,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
		sleep:     func(time.Duration) {},
		listen: func(network, address string) (net.Listener, error) {
			return nil, errors.New("bind: address already in use")
		},
		glob: func(pattern string) ([]string, error) {
			switch filepath.Base(pattern) {
			case nodeLockGlob, nodeLogGlob:
				return nil, nil
			default:
				return filepath.Glob(pattern)
			}
		},
	}

	if err := manager.Init(context.Background(), dir, envFile); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if containsCall(runner.calls, "docker run -d --name hype-vmdocker-redis -p 6379:6379 redis:latest") {
		t.Fatalf("did not expect managed redis docker run, got %#v", runner.calls)
	}
}

func TestInitRemovesStaleLockAndRestartsNode(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "hymx-node"), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	lockFile := filepath.Join(dir, "cmd", "hymx-v0.1.0.lock")
	if err := os.WriteFile(lockFile, []byte("69462"), 0o644); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(dir, "local.env")
	if err := os.WriteFile(envFile, []byte("VMDOCKER_PRIVATE_KEY=0xabc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		run: func(dir string, env []string, name string, args ...string) (string, error) {
			call := strings.Join(append([]string{name}, args...), " ")
			switch call {
			case "docker inspect -f {{.State.Running}} hype-vmdocker-redis":
				return "true", nil
			case "./hymx-node --config ./config.yaml start":
				return "started", nil
			case "go run ./ init":
				return "ok", nil
			default:
				return "", nil
			}
		},
	}

	removed := make([]string, 0, 1)
	manager := &Manager{
		runner:    runner,
		httpGet:   func(string) (int, error) { return 200, nil },
		stat:      os.Stat,
		readDir:   os.ReadDir,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
		remove: func(path string) error {
			removed = append(removed, path)
			return os.Remove(path)
		},
		sleep:  func(time.Duration) {},
		listen: net.Listen,
		dial: func(network, address string, timeout time.Duration) (net.Conn, error) {
			if address != redisAddress {
				t.Fatalf("unexpected redis dial address %s", address)
			}
			return &fakeConn{}, nil
		},
		glob: func(pattern string) ([]string, error) {
			switch filepath.Base(pattern) {
			case nodeLockGlob:
				return []string{lockFile}, nil
			case nodeLogGlob:
				return nil, nil
			default:
				return filepath.Glob(pattern)
			}
		},
		kill: func(pid int, sig syscall.Signal) error {
			if pid != 69462 || sig != syscall.Signal(0) {
				t.Fatalf("unexpected pid probe %d %v", pid, sig)
			}
			return syscall.ESRCH
		},
	}

	if err := manager.Init(context.Background(), dir, envFile); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(removed) != 1 || removed[0] != lockFile {
		t.Fatalf("expected stale lock removal, got %#v", removed)
	}
	if !containsCall(runner.calls, "./hymx-node --config ./config.yaml start") {
		t.Fatalf("expected node start after stale lock removal, got %#v", runner.calls)
	}
}

func TestEnsureRedisRecreatesUnreachableRunningContainer(t *testing.T) {
	runner := &fakeRunner{
		run: func(dir string, env []string, name string, args ...string) (string, error) {
			call := strings.Join(append([]string{name}, args...), " ")
			switch call {
			case "docker inspect -f {{.State.Running}} hype-vmdocker-redis":
				return "true", nil
			case "docker rm -f hype-vmdocker-redis":
				return "removed", nil
			case "docker run -d --name hype-vmdocker-redis -p 6379:6379 redis:latest":
				return "container-id", nil
			default:
				return "", nil
			}
		},
	}

	dialCount := 0
	manager := &Manager{
		runner: runner,
		listen: func(network, address string) (net.Listener, error) {
			if address != redisAddress {
				t.Fatalf("unexpected redis listen address %s", address)
			}
			return &fakeListener{}, nil
		},
		dial: func(network, address string, timeout time.Duration) (net.Conn, error) {
			if address != redisAddress {
				t.Fatalf("unexpected redis dial address %s", address)
			}
			dialCount++
			if dialCount == 1 {
				return nil, errors.New("connect: connection refused")
			}
			return &fakeConn{}, nil
		},
	}

	state, err := manager.ensureRedis(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if state != "running container not reachable on host port, recreated" {
		t.Fatalf("unexpected redis state: %s", state)
	}
	expectedCalls := []string{
		"docker inspect -f {{.State.Running}} hype-vmdocker-redis",
		"docker rm -f hype-vmdocker-redis",
		"docker run -d --name hype-vmdocker-redis -p 6379:6379 redis:latest",
	}
	for _, want := range expectedCalls {
		if !containsCall(runner.calls, want) {
			t.Fatalf("expected call %q in %#v", want, runner.calls)
		}
	}
}

func TestWaitForHealthTimesOut(t *testing.T) {
	manager := &Manager{
		httpGet:   func(string) (int, error) { return 0, errors.New("not ready") },
		sleep:     func(time.Duration) {},
		runner:    &fakeRunner{},
		stat:      os.Stat,
		readDir:   os.ReadDir,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
		listen:    net.Listen,
		glob:      filepath.Glob,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := manager.waitForHealth(ctx, HealthURL)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func hasEnv(env []string, expected string) bool {
	for _, item := range env {
		if item == expected {
			return true
		}
	}
	return false
}

func containsCall(calls []string, expected string) bool {
	for _, call := range calls {
		if call == expected {
			return true
		}
	}
	return false
}

func callIndex(calls []string, expected string) int {
	for i, call := range calls {
		if call == expected {
			return i
		}
	}
	return -1
}

func TestSyncLocalModulesCopiesCmdModToMod(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "cmd", "mod")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "mod-test.json"), []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	manager := &Manager{
		readDir:   os.ReadDir,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
	}

	state, err := manager.syncLocalModules(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(state, "synced 1 module file") {
		t.Fatalf("unexpected sync state: %s", state)
	}
	data, err := os.ReadFile(filepath.Join(dir, "mod", "mod-test.json"))
	if err != nil {
		t.Fatalf("expected copied module file, got %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("unexpected copied data: %s", data)
	}
}

func TestNormalizeLocalRedisConfigRewritesLocalhost(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "cmd")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(cmdDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("redisURL: redis://@localhost:6379/0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	manager := &Manager{
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
	}

	state, err := manager.normalizeLocalRedisConfig(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(state, "normalized redis url") {
		t.Fatalf("unexpected state: %s", state)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "redis://@127.0.0.1:6379/0") {
		t.Fatalf("expected normalized redis url, got %s", data)
	}
}
