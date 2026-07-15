package cli

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVmdockerModuleBuildReplPromptsRequiredFlags(t *testing.T) {
	checkout := t.TempDir()
	cmd := newVmdockerModuleBuildCmd()
	var out bytes.Buffer
	cmd.SetContext(withRepl(context.Background(), bufio.NewReader(strings.NewReader("profile.toml\nagent-bin\nkey\n")), &out))
	if err := cmd.Flags().Set("dir", checkout); err != nil {
		t.Fatal(err)
	}
	if err := cmd.PreRunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]string{
		"profile":     "profile.toml",
		"agent-bin":   "agent-bin",
		"private-key": "key",
	} {
		got, err := cmd.Flags().GetString(name)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
	for _, want := range []string{
		"profile (--profile)",
		"agent-bin (--agent-bin)",
		"private-key (-k/--private-key)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("prompt output missing %q:\n%s", want, out.String())
		}
	}
}

func TestVmdockerModuleBuildReplUsesCheckoutEnvBeforePrompting(t *testing.T) {
	checkout := t.TempDir()
	if err := os.WriteFile(filepath.Join(checkout, ".env"), []byte(strings.Join([]string{
		"VMDOCKER_AGENT_BIN=/env/agent",
		"VMDOCKER_PRIVATE_KEY=env-key",
	}, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newVmdockerModuleBuildCmd()
	var out bytes.Buffer
	cmd.SetContext(withRepl(context.Background(), bufio.NewReader(strings.NewReader("profile.toml\n")), &out))
	if err := cmd.Flags().Set("dir", checkout); err != nil {
		t.Fatal(err)
	}
	if err := cmd.PreRunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	profile, _ := cmd.Flags().GetString("profile")
	agentBin, _ := cmd.Flags().GetString("agent-bin")
	privateKey, _ := cmd.Flags().GetString("private-key")
	if profile != "profile.toml" || agentBin != "/env/agent" || privateKey != "env-key" {
		t.Fatalf("profile=%q agent=%q key=%q", profile, agentBin, privateKey)
	}
	if strings.Contains(out.String(), "agent-bin") || strings.Contains(out.String(), "private-key") {
		t.Fatalf("expected env-backed flags not to be prompted:\n%s", out.String())
	}
}
