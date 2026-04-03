package openclawui

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
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

func TestBuildOpenclawArgsValidation(t *testing.T) {
	_, err := buildOpenclawArgs("chat", openclawRequest{PID: "pid-1"})
	if err == nil {
		t.Fatal("expected validation error when command is empty")
	}
}

func TestMaskSensitive(t *testing.T) {
	masked := maskSensitive("hype", []string{"openclaw", "spawn", "--private-key", "0xabc", "--api-key", "key", "--model", "x"})
	joined := strings.Join(masked, " ")
	if strings.Contains(joined, "0xabc") || strings.Contains(joined, " key ") {
		t.Fatalf("secret was not masked: %s", joined)
	}
}

func TestResolveConfigUsesCurrentExecutable(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve executable failed: %v", err)
	}

	cfg, err := ResolveConfig(Config{})
	if err != nil {
		t.Fatalf("resolve config failed: %v", err)
	}
	if cfg.BinaryPath != executable {
		t.Fatalf("expected binary path %q, got %q", executable, cfg.BinaryPath)
	}
	if cfg.Listen != defaultListen {
		t.Fatalf("expected default listen %q, got %q", defaultListen, cfg.Listen)
	}
	if cfg.Timeout != defaultCommandTimeout {
		t.Fatalf("expected default timeout %s, got %s", defaultCommandTimeout, cfg.Timeout)
	}
}

func TestNewHandlerServesEmbeddedIndexAndAssets(t *testing.T) {
	handler, err := NewHandler(Config{
		BinaryPath: "/bin/echo",
		WorkingDir: t.TempDir(),
		Listen:     defaultListen,
		Timeout:    defaultCommandTimeout,
	})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootResp := httptest.NewRecorder()
	handler.ServeHTTP(rootResp, rootReq)
	if rootResp.Code != http.StatusOK {
		t.Fatalf("expected root 200, got %d", rootResp.Code)
	}
	body := rootResp.Body.String()
	if !strings.Contains(body, "<title>HypeUI</title>") {
		t.Fatalf("expected embedded index html, got %q", body)
	}

	assetPath := regexp.MustCompile(`/assets/[^"]+`).FindString(body)
	if assetPath == "" {
		t.Fatalf("expected asset path in index html, got %q", body)
	}

	assetReq := httptest.NewRequest(http.MethodGet, assetPath, nil)
	assetResp := httptest.NewRecorder()
	handler.ServeHTTP(assetResp, assetReq)
	if assetResp.Code != http.StatusOK {
		t.Fatalf("expected asset 200, got %d", assetResp.Code)
	}
	if assetResp.Body.Len() == 0 {
		t.Fatal("expected embedded asset body")
	}
}

func TestHealthRouteUsesConfiguredBinary(t *testing.T) {
	workingDir := t.TempDir()
	binaryPath := writeFakeHypeBinary(t, workingDir)

	handler, err := NewHandler(Config{
		BinaryPath: binaryPath,
		WorkingDir: workingDir,
		Listen:     "127.0.0.1:7788",
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d", resp.Code)
	}

	body := resp.Body.String()
	if !strings.Contains(body, `"hypeBinary":"`+binaryPath+`"`) {
		t.Fatalf("expected binary path in response, got %s", body)
	}
	if !strings.Contains(body, `"workingDir":"`+workingDir+`"`) {
		t.Fatalf("expected working dir in response, got %s", body)
	}
	if !strings.Contains(body, `"hypeVersion":"v0.0.6"`) || !strings.Contains(body, `"hymxVersion":"v0.4.8"`) {
		t.Fatalf("expected versions in response, got %s", body)
	}
}

func TestChatRouteExecutesCurrentBinary(t *testing.T) {
	workingDir := t.TempDir()
	binaryPath := writeFakeHypeBinary(t, workingDir)

	handler, err := NewHandler(Config{
		BinaryPath: binaryPath,
		WorkingDir: workingDir,
		Listen:     defaultListen,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/openclaw/chat", strings.NewReader(`{"pid":"pid-1","command":"status"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected chat 200, got %d: %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	if !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"response_id":"msg-1"`) {
		t.Fatalf("expected parsed chat response, got %s", body)
	}
}

func TestVmdockerGetRouteExecutesCurrentBinary(t *testing.T) {
	workingDir := t.TempDir()
	binaryPath := writeFakeHypeBinary(t, workingDir)

	handler, err := NewHandler(Config{
		BinaryPath: binaryPath,
		WorkingDir: workingDir,
		Listen:     defaultListen,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/vmdocker/get", strings.NewReader(`{"dir":"./sandbox/vmdocker","version":"v0.0.1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected vmdocker get 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body openclawResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if !body.OK || !strings.Contains(body.StdoutRaw, "vmdocker get ok") {
		t.Fatalf("unexpected response: %#v", body)
	}
	if !strings.Contains(strings.Join(body.Command, " "), "vmdocker get --dir ./sandbox/vmdocker --version v0.0.1") {
		t.Fatalf("unexpected command: %#v", body.Command)
	}
}

func TestEnvLoadRouteReadsFile(t *testing.T) {
	workingDir := t.TempDir()
	binaryPath := writeFakeHypeBinary(t, workingDir)
	envPath := filepath.Join(workingDir, ".env")
	if err := os.WriteFile(envPath, []byte("VMDOCKER_PRIVATE_KEY=0xabc\n"), 0o644); err != nil {
		t.Fatalf("write env file failed: %v", err)
	}

	handler, err := NewHandler(Config{
		BinaryPath: binaryPath,
		WorkingDir: workingDir,
		Listen:     defaultListen,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/env/load", strings.NewReader(`{"path":".env"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected env load 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body envLoadResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if !body.OK || body.FileName != ".env" || !strings.Contains(body.Content, "VMDOCKER_PRIVATE_KEY=0xabc") {
		t.Fatalf("unexpected env load response: %#v", body)
	}
}

func TestVmdockerInitRouteWritesTempEnvFileAndCleansUp(t *testing.T) {
	workingDir := t.TempDir()
	binaryPath := writeFakeHypeBinary(t, workingDir)

	handler, err := NewHandler(Config{
		BinaryPath: binaryPath,
		WorkingDir: workingDir,
		Listen:     defaultListen,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/vmdocker/init", strings.NewReader(`{"dir":"./sandbox/vmdocker","envFileName":"local.env","envFileContent":"VMDOCKER_PRIVATE_KEY=0xabc\nOPENCLAW_PROVIDER=zen\n"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected vmdocker init 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body openclawResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if !body.OK {
		t.Fatalf("expected ok response, got %#v", body)
	}
	if !strings.Contains(body.StdoutRaw, "VMDOCKER_PRIVATE_KEY=0xabc") || !strings.Contains(body.StdoutRaw, "OPENCLAW_PROVIDER=zen") {
		t.Fatalf("expected env content in stdout, got %q", body.StdoutRaw)
	}
	var tempPath string
	for _, line := range strings.Split(body.StdoutRaw, "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "TEMP_ENV="); ok {
			tempPath = after
			break
		}
	}
	if tempPath == "" {
		t.Fatalf("expected temp env path in stdout, got %q", body.StdoutRaw)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("expected temp env file to be cleaned up, stat err=%v", err)
	}
}

func TestBuildVmdockerArgsValidation(t *testing.T) {
	_, cleanup, err := buildVmdockerArgs("init", openclawRequest{EnvFileContent: "OPENCLAW_PROVIDER=zen"})
	if cleanup != nil {
		cleanup()
	}
	if err == nil || !strings.Contains(err.Error(), "VMDOCKER_PRIVATE_KEY") {
		t.Fatalf("expected missing private key validation error, got %v", err)
	}
}

func writeFakeHypeBinary(t *testing.T, dir string) string {
	t.Helper()

	script := `#!/bin/sh
if [ "$1" = "-v" ]; then
  cat <<'EOF'
=================================
||            HYPE            ||
=================================
Version:     v0.0.6
HymxVersion: v0.4.8
EOF
  exit 0
fi

if [ "$1" = "openclaw" ] && [ "$2" = "chat" ]; then
  cat <<'EOF'
{"action":"chat","response_id":"msg-1","message":"ok"}
EOF
  exit 0
fi

if [ "$1" = "vmdocker" ] && [ "$2" = "get" ]; then
  echo "vmdocker get ok"
  exit 0
fi

if [ "$1" = "vmdocker" ] && [ "$2" = "init" ]; then
  env_file=""
  prev=""
  for arg in "$@"; do
    if [ "$prev" = "--env-file" ]; then
      env_file="$arg"
      break
    fi
    prev="$arg"
  done
  if [ ! -f "$env_file" ]; then
    echo "env file missing" >&2
    exit 1
  fi
  cat "$env_file"
  printf '\nTEMP_ENV=%s\n' "$env_file"
  exit 0
fi

echo '{}'`
	binaryPath := filepath.Join(dir, "hype")
	if err := os.WriteFile(binaryPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake binary failed: %v", err)
	}
	return binaryPath
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

func TestAssetRouteBodyIsReadable(t *testing.T) {
	handler, err := NewHandler(Config{
		BinaryPath: "/bin/echo",
		WorkingDir: t.TempDir(),
		Listen:     defaultListen,
		Timeout:    defaultCommandTimeout,
	})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	assetPath := regexp.MustCompile(`/assets/[^"]+`).FindString(resp.Body.String())
	if assetPath == "" {
		t.Fatal("expected asset path")
	}

	assetReq := httptest.NewRequest(http.MethodGet, assetPath, nil)
	assetResp := httptest.NewRecorder()
	handler.ServeHTTP(assetResp, assetReq)
	body, err := io.ReadAll(assetResp.Result().Body)
	if err != nil {
		t.Fatalf("read asset body failed: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty asset body")
	}
}

func TestRunContextStopsServerOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, Config{
			BinaryPath: "/bin/echo",
			WorkingDir: t.TempDir(),
			Listen:     "127.0.0.1:0",
			Timeout:    time.Second,
		})
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected clean shutdown, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected RunContext to stop after cancellation")
	}
}

func TestSuggestedVmdockerDirPrefersSiblingRepo(t *testing.T) {
	workspaceDir := t.TempDir()
	hypeDir := filepath.Join(workspaceDir, "hype")
	vmdockerDir := filepath.Join(workspaceDir, "vmdocker")
	if err := os.MkdirAll(hypeDir, 0o755); err != nil {
		t.Fatalf("mkdir hype failed: %v", err)
	}
	if err := os.MkdirAll(vmdockerDir, 0o755); err != nil {
		t.Fatalf("mkdir vmdocker failed: %v", err)
	}

	got := suggestedVmdockerDir(hypeDir)
	if got != vmdockerDir {
		t.Fatalf("expected sibling vmdocker dir %q, got %q", vmdockerDir, got)
	}
}
