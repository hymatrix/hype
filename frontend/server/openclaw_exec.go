package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultCommandTimeout = 90 * time.Second

var errValidation = errors.New("validation failed")

func runOpenclaw(ctx context.Context, repoRoot, subcmd string, req openclawRequest, store *spawnStore) (runResult, error) {
	if store == nil {
		store = &spawnStore{}
	}

	start := time.Now()
	binaryPath, err := resolveHypeBinary(repoRoot)
	if err != nil {
		return runResult{}, err
	}

	args, err := buildOpenclawArgs(subcmd, req)
	if err != nil {
		return runResult{}, err
	}

	timeout := commandTimeoutFromEnv()
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, binaryPath, args...)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	result := openclawResponse{
		OK:        err == nil,
		Command:   maskSensitive(binaryPath, args),
		StdoutRaw: strings.TrimSpace(stdout.String()),
		StderrRaw: strings.TrimSpace(stderr.String()),
	}

	if result.StdoutRaw != "" {
		if parsed, ok := parseJSONFromMixedOutput(result.StdoutRaw); ok {
			result.ParsedJSON = parsed
			if subcmd == "spawn" && err == nil {
				pid := extractSpawnPID(parsed)
				if pid != "" {
					result.SpawnPID = pid
					store.add(pid)
				}
			}
		}
	}

	if err != nil {
		result.Error = err.Error()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	} else {
		result.ExitCode = 0
	}

	duration := time.Since(start)
	result.DurationMS = duration.Milliseconds()
	result.SpawnedPIDs = store.list()

	return runResult{response: result, duration: duration, binaryRef: binaryPath}, nil
}

func extractSpawnPID(parsed map[string]any) string {
	if parsed == nil {
		return ""
	}
	if pid, ok := parsed["pid"].(string); ok && strings.TrimSpace(pid) != "" {
		return strings.TrimSpace(pid)
	}
	if id, ok := parsed["response_id"].(string); ok && strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	return ""
}

func parseJSONFromMixedOutput(raw string) (map[string]any, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}

	var direct map[string]any
	if json.Unmarshal([]byte(raw), &direct) == nil {
		return direct, true
	}

	// Handle noisy stdout (e.g. logs before/after JSON payload).
	left := strings.Index(raw, "{")
	right := strings.LastIndex(raw, "}")
	if left >= 0 && right > left {
		var bounded map[string]any
		if json.Unmarshal([]byte(raw[left:right+1]), &bounded) == nil {
			return bounded, true
		}
	}

	for i := 0; i < len(raw); i++ {
		if raw[i] != '{' {
			continue
		}
		for j := len(raw) - 1; j > i; j-- {
			if raw[j] != '}' {
				continue
			}
			var candidate map[string]any
			if json.Unmarshal([]byte(raw[i:j+1]), &candidate) == nil {
				return candidate, true
			}
		}
	}
	return nil, false
}

func buildOpenclawArgs(subcmd string, req openclawRequest) ([]string, error) {
	nodeURL := strings.TrimSpace(req.NodeURL)
	if nodeURL == "" {
		nodeURL = defaultNodeURL
	}

	args := []string{"openclaw", subcmd, "--json", "--node-url", nodeURL}
	if key := strings.TrimSpace(req.PrivateKey); key != "" {
		args = append(args, "--private-key", key)
	}

	switch subcmd {
	case "spawn":
		if err := requireFields(map[string]string{
			"moduleId":     req.ModuleID,
			"scheduler":    req.Scheduler,
			"model":        req.Model,
			"apiKey":       req.APIKey,
			"gatewayToken": req.GatewayToken,
		}); err != nil {
			return nil, err
		}
		timeout := strings.TrimSpace(req.TimeoutMS)
		if timeout == "" {
			timeout = "180000"
		}
		if n, err := strconv.Atoi(timeout); err != nil || n < 1000 || n > 3600000 {
			return nil, fmt.Errorf("%w: timeoutMs must be in [1000, 3600000]", errValidation)
		}
		args = append(args,
			"--module-id", strings.TrimSpace(req.ModuleID),
			"--scheduler", strings.TrimSpace(req.Scheduler),
			"--model", strings.TrimSpace(req.Model),
			"--timeout-ms", timeout,
			"--api-key", strings.TrimSpace(req.APIKey),
			"--gateway-token", strings.TrimSpace(req.GatewayToken),
		)
	case "conf-tg":
		if err := requireFields(map[string]string{
			"pid":      req.PID,
			"botToken": req.BotToken,
		}); err != nil {
			return nil, err
		}
		defaultAccount := strings.TrimSpace(req.DefaultAccount)
		if defaultAccount == "" {
			defaultAccount = "main"
		}
		dmPolicy := strings.TrimSpace(req.DMPolicy)
		if dmPolicy == "" {
			dmPolicy = "pairing"
		}
		args = append(args,
			"--pid", strings.TrimSpace(req.PID),
			"--bot-token", strings.TrimSpace(req.BotToken),
			"--default-account", defaultAccount,
			"--dm-policy", dmPolicy,
		)
		if allowFrom := strings.TrimSpace(req.AllowFrom); allowFrom != "" {
			args = append(args, "--allow-from", allowFrom)
		}
	case "pair-tg":
		if err := requireFields(map[string]string{
			"pid":  req.PID,
			"code": req.Code,
		}); err != nil {
			return nil, err
		}
		channel := strings.TrimSpace(req.Channel)
		if channel == "" {
			channel = "telegram"
		}
		dmPolicy := strings.TrimSpace(req.DMPolicy)
		if dmPolicy == "" {
			dmPolicy = "pairing"
		}
		args = append(args,
			"--pid", strings.TrimSpace(req.PID),
			"--code", strings.TrimSpace(req.Code),
			"--channel", channel,
			"--dm-policy", dmPolicy,
		)
	case "chat":
		if err := requireFields(map[string]string{
			"pid":     req.PID,
			"command": req.Command,
		}); err != nil {
			return nil, err
		}
		args = append(args,
			"--pid", strings.TrimSpace(req.PID),
			"--command", strings.TrimSpace(req.Command),
		)
	default:
		return nil, fmt.Errorf("%w: unsupported command %q", errValidation, subcmd)
	}

	return args, nil
}

func requireFields(required map[string]string) error {
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: %s is required", errValidation, name)
		}
	}
	return nil
}

func resolveHypeBinary(repoRoot string) (string, error) {
	preferred := filepath.Join(repoRoot, "build", "hype")
	if info, err := os.Stat(preferred); err == nil && !info.IsDir() {
		return preferred, nil
	}
	pathBinary, err := exec.LookPath("hype")
	if err != nil {
		return "", errors.New("cannot find hype binary, expected build/hype or hype in PATH")
	}
	return pathBinary, nil
}

func commandTimeoutFromEnv() time.Duration {
	v := strings.TrimSpace(os.Getenv("OPENCLAW_WEBUI_TIMEOUT_MS"))
	if v == "" {
		return defaultCommandTimeout
	}
	ms, err := strconv.Atoi(v)
	if err != nil || ms <= 0 {
		return defaultCommandTimeout
	}
	return time.Duration(ms) * time.Millisecond
}

func maskSensitive(binary string, args []string) []string {
	masked := make([]string, 0, len(args)+1)
	masked = append(masked, binary)
	secretFlags := map[string]struct{}{
		"--private-key":   {},
		"--api-key":       {},
		"--gateway-token": {},
		"--bot-token":     {},
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		masked = append(masked, arg)
		if _, ok := secretFlags[arg]; ok && i+1 < len(args) {
			masked = append(masked, "***")
			i++
		}
	}
	return masked
}
