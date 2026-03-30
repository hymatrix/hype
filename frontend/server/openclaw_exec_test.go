package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOpenclawArgsSpawn(t *testing.T) {
	req := openclawRequest{
		NodeURL:        "http://127.0.0.1:8081",
		PrivateKey:     "0xabc",
		ModuleID:       "mod-1",
		Scheduler:      "scheduler-1",
		Model:          "opencode-go/kimi-k2.5",
		Provider:       "",
		APIKey:         "api-key",
		GatewayToken:   "gateway",
		RuntimeBackend: "sandbox",
		BotToken:       "bot-token",
		DefaultAccount: "main",
		DMPolicy:       "open",
		AllowFrom:      "*",
	}
	args, err := buildOpenclawArgs("spawn", req)
	if err != nil {
		t.Fatalf("build args failed: %v", err)
	}

	has := func(flag, value string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == flag && args[i+1] == value {
				return true
			}
		}
		return false
	}

	if !has("--module-id", "mod-1") || !has("--scheduler", "scheduler-1") || !has("--model", "kimi-k2.5") {
		t.Fatalf("missing required args: %v", args)
	}
	if !has("--runtime-backend", "sandbox") {
		t.Fatalf("missing runtime args: %v", args)
	}
	if !has("--provider", "opencode-go") {
		t.Fatalf("missing normalized provider arg: %v", args)
	}
	if !has("--bot-token", "bot-token") || !has("--allow-from", "*") {
		t.Fatalf("missing telegram follow-up args: %v", args)
	}
}

func TestBuildOpenclawArgsSpawnStripsMatchingModelPrefix(t *testing.T) {
	req := openclawRequest{
		ModuleID:     "mod-1",
		Scheduler:    "scheduler-1",
		Model:        "opencode-go/kimi-k2.5",
		Provider:     "opencode-go",
		GatewayToken: "gateway",
	}
	args, err := buildOpenclawArgs("spawn", req)
	if err != nil {
		t.Fatalf("build args failed: %v", err)
	}

	has := func(flag, value string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == flag && args[i+1] == value {
				return true
			}
		}
		return false
	}

	if !has("--model", "kimi-k2.5") || !has("--provider", "opencode-go") {
		t.Fatalf("expected prefixed model to be canonicalized, got %v", args)
	}
}

func TestBuildOpenclawArgsValidation(t *testing.T) {
	_, err := buildOpenclawArgs("chat", openclawRequest{PID: "pid-1"})
	if err == nil {
		t.Fatal("expected validation error when command is empty")
	}
}

func TestBuildOpenclawArgsSpawnRejectsInvalidRuntimeBackend(t *testing.T) {
	_, err := buildOpenclawArgs("spawn", openclawRequest{
		ModuleID:       "mod-1",
		Scheduler:      "scheduler-1",
		Model:          "plan",
		APIKey:         "api-key",
		GatewayToken:   "gateway",
		RuntimeBackend: "podman",
	})
	if err == nil || !strings.Contains(err.Error(), "runtimeBackend must be empty") {
		t.Fatalf("expected runtimeBackend validation error, got %v", err)
	}
}

func TestBuildOpenclawArgsSpawnRequiresProviderWhenAPIKeyAndPlainModel(t *testing.T) {
	_, err := buildOpenclawArgs("spawn", openclawRequest{
		ModuleID:     "mod-1",
		Scheduler:    "scheduler-1",
		Model:        "plan",
		APIKey:       "api-key",
		GatewayToken: "gateway",
	})
	if err == nil || !strings.Contains(err.Error(), "provider is required when apiKey is provided") {
		t.Fatalf("expected provider validation error, got %v", err)
	}
}

func TestBuildOpenclawArgsSpawnDoesNotIncludeSandboxWorkspace(t *testing.T) {
	args, err := buildOpenclawArgs("spawn", openclawRequest{
		ModuleID:     "mod-1",
		Scheduler:    "scheduler-1",
		GatewayToken: "gateway",
	})
	if err != nil {
		t.Fatalf("build args failed: %v", err)
	}

	has := func(flag, value string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == flag && args[i+1] == value {
				return true
			}
		}
		return false
	}

	if has("--sandbox-workspace", ".") {
		t.Fatalf("did not expect sandbox workspace arg, got %v", args)
	}
}

func TestBuildOpenclawArgsSpawnRejectsConflictingProviderAndModelPrefix(t *testing.T) {
	_, err := buildOpenclawArgs("spawn", openclawRequest{
		ModuleID:     "mod-1",
		Scheduler:    "scheduler-1",
		Model:        "opencode-go/kimi-k2.5",
		Provider:     "zen",
		GatewayToken: "gateway",
	})
	if err == nil || !strings.Contains(err.Error(), "conflicts with model prefix") {
		t.Fatalf("expected provider conflict error, got %v", err)
	}
}

func TestBuildOpenclawArgsConfTGDefaultsAllowFromWildcard(t *testing.T) {
	args, err := buildOpenclawArgs("conf-tg", openclawRequest{
		PID:      "pid-1",
		BotToken: "bot-token",
		DMPolicy: "open",
	})
	if err != nil {
		t.Fatalf("build args failed: %v", err)
	}

	has := func(flag, value string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == flag && args[i+1] == value {
				return true
			}
		}
		return false
	}

	if !has("--allow-from", "*") {
		t.Fatalf("expected default allow-from arg, got %v", args)
	}
}

func TestBuildOpenclawArgsSpawnDefaultsTelegramFollowupValues(t *testing.T) {
	args, err := buildOpenclawArgs("spawn", openclawRequest{
		ModuleID:     "mod-1",
		Scheduler:    "scheduler-1",
		Model:        "zen/plan",
		APIKey:       "api-key",
		GatewayToken: "gateway",
		BotToken:     "bot-token",
	})
	if err != nil {
		t.Fatalf("build args failed: %v", err)
	}

	has := func(flag, value string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == flag && args[i+1] == value {
				return true
			}
		}
		return false
	}

	if !has("--default-account", "main") || !has("--dm-policy", "open") || !has("--allow-from", "*") {
		t.Fatalf("expected default telegram follow-up args, got %v", args)
	}
}

func TestResolveHypeBinaryPrefersBuild(t *testing.T) {
	repoRoot := t.TempDir()
	buildDir := filepath.Join(repoRoot, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	binary := filepath.Join(buildDir, "hype")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	resolved, err := resolveHypeBinary(repoRoot)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if resolved != binary {
		t.Fatalf("expected %s, got %s", binary, resolved)
	}
}

func TestMaskSensitive(t *testing.T) {
	masked := maskSensitive("hype", []string{"openclaw", "spawn", "--private-key", "0xabc", "--api-key", "key", "--model", "x"})
	joined := strings.Join(masked, " ")
	if strings.Contains(joined, "0xabc") || strings.Contains(joined, " key ") {
		t.Fatalf("secret was not masked: %s", joined)
	}
}

func TestExtractSpawnPID(t *testing.T) {
	pid := extractSpawnPID(map[string]any{"pid": "proc-1", "response_id": "fallback"})
	if pid != "proc-1" {
		t.Fatalf("expected pid from pid field, got %q", pid)
	}

	pid = extractSpawnPID(map[string]any{"response_id": "proc-2"})
	if pid != "proc-2" {
		t.Fatalf("expected pid from response_id, got %q", pid)
	}
}

func TestSpawnStoreList(t *testing.T) {
	store := &spawnStore{}
	store.add("p1")
	store.add("p2")
	got := store.list()
	if len(got) != 2 || got[0] != "p1" || got[1] != "p2" {
		t.Fatalf("unexpected list: %#v", got)
	}
}

func TestParseJSONFromMixedOutput(t *testing.T) {
	raw := `t=2026-03-05T17:01:29+0800 lvl=info msg="wallet initialized"
{
  "action": "spawn",
  "pid": "pid-abc",
  "response_id": "pid-abc"
}`

	parsed, ok := parseJSONFromMixedOutput(raw)
	if !ok {
		t.Fatal("expected parser to extract JSON from mixed output")
	}
	if parsed["action"] != "spawn" || parsed["pid"] != "pid-abc" {
		t.Fatalf("unexpected parsed content: %#v", parsed)
	}
}
