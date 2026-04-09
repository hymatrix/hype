package cli

import (
	"io"
	"strings"
	"testing"
)

func execCLICommand(t *testing.T, args ...string) error {
	t.Helper()
	cmd := NewRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}

func TestClaudeExecRequiresPrompt(t *testing.T) {
	err := execCLICommand(t, "claude", "exec", "--pid", "pid-1", "-k", "0x1234")
	if err == nil || !strings.Contains(err.Error(), "required flag(s) \"prompt\" not set") {
		t.Fatalf("expected prompt required error, got: %v", err)
	}
}

func TestClaudeChatRequiresCommand(t *testing.T) {
	err := execCLICommand(t, "claude", "chat", "-p", "pid-1", "-k", "0x1234")
	if err == nil || !strings.Contains(err.Error(), "command is required") {
		t.Fatalf("expected command required error, got: %v", err)
	}
}

func TestClaudeRootPromptRequiresPID(t *testing.T) {
	err := execCLICommand(t, "claude", "-p", "hello", "-k", "0x1234")
	if err == nil || !strings.Contains(err.Error(), "pid is required") {
		t.Fatalf("expected pid required error, got: %v", err)
	}
}

func TestBuildClaudeSpawnTagsIncludesAnthropicEnv(t *testing.T) {
	tags := buildClaudeSpawnTags("anthropic-key", "https://proxy.example", "claude-sonnet-4-5", "--append-system-prompt test", "sandbox")

	if !hasTag(tags, "Container-Env-RUNTIME_TYPE", "claude") {
		t.Fatalf("expected runtime type tag, got %#v", tags)
	}
	if !hasTag(tags, "Container-Env-ANTHROPIC_API_KEY", "anthropic-key") {
		t.Fatalf("expected API key env tag, got %#v", tags)
	}
	if !hasTag(tags, "Container-Env-ANTHROPIC_BASE_URL", "https://proxy.example") {
		t.Fatalf("expected base URL env tag, got %#v", tags)
	}
	if !hasTag(tags, "Container-Env-ANTHROPIC_MODEL", "claude-sonnet-4-5") {
		t.Fatalf("expected model env tag, got %#v", tags)
	}
	if !hasTag(tags, "Container-Env-CLAUDE_CODE_FLAGS", "--append-system-prompt test") {
		t.Fatalf("expected code flags env tag, got %#v", tags)
	}
	if !hasTag(tags, "Runtime-Backend", "sandbox") {
		t.Fatalf("expected runtime backend tag, got %#v", tags)
	}
}

func TestValidateRuntimeBackendRejectsInvalidValue(t *testing.T) {
	err := validateRuntimeBackend("podman")
	if err == nil || !strings.Contains(err.Error(), "runtime-backend must be one of") {
		t.Fatalf("expected runtime-backend validation error, got %v", err)
	}
}

func TestHydrateFlagFromEnvsUsesFirstMatch(t *testing.T) {
	t.Setenv("ANTHROPIC_MODEL", "claude-sonnet")
	cmd := newOpenclawSharedFlagCmd()
	cmd.Flags().String("model", "", "")

	if err := hydrateFlagFromEnvs(cmd, "model", "ANTHROPIC_MODEL", "CLAUDE_MODEL"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	value, err := cmd.Flags().GetString("model")
	if err != nil {
		t.Fatalf("get flag failed: %v", err)
	}
	if value != "claude-sonnet" {
		t.Fatalf("expected env-hydrated model, got %q", value)
	}
}
