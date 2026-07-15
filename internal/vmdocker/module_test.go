package vmdocker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestBuildModuleResolvesInputsAndStreams(t *testing.T) {
	checkout := t.TempDir()
	profile := filepath.Join(t.TempDir(), "profile.toml")
	agentBin := filepath.Join(t.TempDir(), "vmdocker-agent")
	if err := os.WriteFile(profile, []byte("[dockerfile]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentBin, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkout, ".env"), []byte("VMDOCKER_URL=http://file\nVMDOCKER_PRIVATE_KEY=file-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotDir string
	var gotEnv, gotArgs []string
	runner := &fakeRunner{runStream: func(dir string, env []string, stdout, stderr io.Writer, name string, args ...string) error {
		gotDir, gotEnv, gotArgs = dir, append([]string(nil), env...), append([]string{name}, args...)
		_, _ = io.WriteString(stdout, "building\n")
		_, _ = io.WriteString(stderr, "progress\n")
		return nil
	}}
	manager := newTestManager(runner)
	var stdout, stderr bytes.Buffer
	manager.SetOutput(&stdout)
	manager.SetErrorOutput(&stderr)
	err := manager.BuildModule(context.Background(), ModuleBuildOptions{
		CheckoutDir: checkout, ProfilePath: profile, AgentBinPath: agentBin,
		NodeURL: "http://flag", PrivateKey: "flag-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != filepath.Join(checkout, "cmd") {
		t.Fatalf("dir = %q", gotDir)
	}
	wantArgs := []string{"go", "run", "./module", "--profile", profile, "--agent-bin", agentBin}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("args = %#v", gotArgs)
	}
	if !hasEnv(gotEnv, "VMDOCKER_URL=http://flag") || !hasEnv(gotEnv, "VMDOCKER_PRIVATE_KEY=flag-key") {
		t.Fatalf("env = %#v", gotEnv)
	}
	if !strings.HasPrefix(stdout.String(), "building\nmodules: ") || stderr.String() != "progress\n" {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestBuildModuleInputPrecedence(t *testing.T) {
	checkout := t.TempDir()
	profile := filepath.Join(t.TempDir(), "profile.toml")
	agentBin := filepath.Join(t.TempDir(), "agent-from-flag")
	if err := os.WriteFile(profile, []byte("[dockerfile]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentBin, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkout, ".env"), []byte(strings.Join([]string{
		"VMDOCKER_URL=http://file",
		"VMDOCKER_PRIVATE_KEY=file-key",
		"VMDOCKER_AGENT_BIN=/file/agent",
	}, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VMDOCKER_URL", "http://env")
	t.Setenv("VMDOCKER_PRIVATE_KEY", "env-key")
	t.Setenv("VMDOCKER_AGENT_BIN", "/env/agent")

	var gotEnv, gotArgs []string
	runner := &fakeRunner{runStream: func(dir string, env []string, stdout, stderr io.Writer, name string, args ...string) error {
		gotEnv, gotArgs = append([]string(nil), env...), append([]string{name}, args...)
		return nil
	}}
	err := newTestManager(runner).BuildModule(context.Background(), ModuleBuildOptions{
		CheckoutDir: checkout, ProfilePath: profile, AgentBinPath: agentBin,
		NodeURL: "http://flag", PrivateKey: "flag-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasEnv(gotEnv, "VMDOCKER_URL=http://flag") || !hasEnv(gotEnv, "VMDOCKER_PRIVATE_KEY=flag-key") {
		t.Fatalf("env = %#v", gotEnv)
	}
	if !slices.Equal(gotArgs, []string{"go", "run", "./module", "--profile", profile, "--agent-bin", agentBin}) {
		t.Fatalf("args = %#v", gotArgs)
	}
}

func TestBuildModuleMovesGeneratedModuleOutputToNodeStore(t *testing.T) {
	checkout := t.TempDir()
	cmdDir := filepath.Join(checkout, "cmd")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(t.TempDir(), "profile.toml")
	agentBin := filepath.Join(t.TempDir(), "vmdocker-agent")
	if err := os.WriteFile(profile, []byte("[dockerfile]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentBin, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{runStream: func(dir string, env []string, stdout, stderr io.Writer, name string, args ...string) error {
		return os.WriteFile(filepath.Join(dir, "mod-test.json"), []byte(`{"ok":true}`), 0o644)
	}}

	err := newTestManager(runner).BuildModule(context.Background(), ModuleBuildOptions{
		CheckoutDir: checkout, ProfilePath: profile, AgentBinPath: agentBin, PrivateKey: "key",
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(cmdDir, "mod", "mod-test.json"))
	if err != nil {
		t.Fatalf("expected moved module output, got %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("unexpected moved data: %s", data)
	}
	if _, err := os.Stat(filepath.Join(cmdDir, "mod-test.json")); !os.IsNotExist(err) {
		t.Fatalf("expected source module output moved, stat err=%v", err)
	}
}

func TestBuildModuleValidation(t *testing.T) {
	checkout := t.TempDir()
	profile := filepath.Join(t.TempDir(), "profile.toml")
	agentBin := filepath.Join(t.TempDir(), "vmdocker-agent")
	if err := os.WriteFile(profile, []byte("[dockerfile]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentBin, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		opts    ModuleBuildOptions
		wantErr string
	}{
		{name: "missing profile", opts: ModuleBuildOptions{CheckoutDir: checkout, AgentBinPath: agentBin, PrivateKey: "key"}, wantErr: "profile is required"},
		{name: "missing adapter", opts: ModuleBuildOptions{CheckoutDir: checkout, ProfilePath: profile, PrivateKey: "key"}, wantErr: "agent-bin is required"},
		{name: "missing private key", opts: ModuleBuildOptions{CheckoutDir: checkout, ProfilePath: profile, AgentBinPath: agentBin}, wantErr: "private-key is required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := newTestManager(&fakeRunner{}).BuildModule(context.Background(), tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestBuildModuleReturnsRunnerFailure(t *testing.T) {
	checkout := t.TempDir()
	profile := filepath.Join(t.TempDir(), "profile.toml")
	agentBin := filepath.Join(t.TempDir(), "vmdocker-agent")
	if err := os.WriteFile(profile, []byte("[dockerfile]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentBin, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{runStream: func(dir string, env []string, stdout, stderr io.Writer, name string, args ...string) error {
		return errors.New("build failed")
	}}
	err := newTestManager(runner).BuildModule(context.Background(), ModuleBuildOptions{
		CheckoutDir: checkout, ProfilePath: profile, AgentBinPath: agentBin, PrivateKey: "key",
	})
	if err == nil || err.Error() != "build failed" {
		t.Fatalf("err = %v", err)
	}
}
