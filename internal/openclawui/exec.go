package openclawui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

var errValidation = errors.New("validation failed")

const (
	runtimeBackendDocker  = "docker"
	runtimeBackendSandbox = "sandbox"
)

func runOpenclaw(ctx context.Context, cfg Config, subcmd string, req openclawRequest, store *spawnStore) (runResult, error) {
	if store == nil {
		store = &spawnStore{}
	}

	start := time.Now()
	args, err := buildOpenclawArgs(subcmd, req)
	if err != nil {
		return runResult{}, err
	}

	runCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, cfg.BinaryPath, args...)
	cmd.Env = os.Environ()
	if cfg.WorkingDir != "" {
		cmd.Dir = cfg.WorkingDir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	result := openclawResponse{
		OK:        err == nil,
		Command:   maskSensitive(cfg.BinaryPath, args),
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

	return runResult{response: result, duration: duration, binaryRef: cfg.BinaryPath}, nil
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
			"gatewayToken": req.GatewayToken,
		}); err != nil {
			return nil, err
		}
		runtimeBackend := strings.TrimSpace(req.RuntimeBackend)
		if runtimeBackend != "" && runtimeBackend != runtimeBackendDocker && runtimeBackend != runtimeBackendSandbox {
			return nil, fmt.Errorf("%w: runtimeBackend must be empty, %q, or %q", errValidation, runtimeBackendDocker, runtimeBackendSandbox)
		}
		model := strings.TrimSpace(req.Model)
		provider := strings.TrimSpace(req.Provider)
		apiKey := strings.TrimSpace(req.APIKey)
		normalizedModel, normalizedProvider, err := normalizeModelProvider(model, provider)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errValidation, err)
		}
		model = normalizedModel
		provider = normalizedProvider
		if apiKey != "" && provider == "" {
			return nil, fmt.Errorf("%w: provider is required when apiKey is provided and model has no provider prefix", errValidation)
		}
		args = append(args,
			"--module-id", strings.TrimSpace(req.ModuleID),
			"--scheduler", strings.TrimSpace(req.Scheduler),
			"--model", model,
			"--provider", provider,
			"--api-key", apiKey,
			"--gateway-token", strings.TrimSpace(req.GatewayToken),
		)
		if runtimeBackend != "" {
			args = append(args, "--runtime-backend", runtimeBackend)
		}
		botToken := strings.TrimSpace(req.BotToken)
		if botToken != "" {
			defaultAccount := strings.TrimSpace(req.DefaultAccount)
			if defaultAccount == "" {
				defaultAccount = "main"
			}
			dmPolicy := strings.TrimSpace(req.DMPolicy)
			if dmPolicy == "" {
				dmPolicy = "open"
			}
			allowFrom := strings.TrimSpace(req.AllowFrom)
			if allowFrom == "" {
				allowFrom = "*"
			}
			if err := validateTelegramConfig(dmPolicy, allowFrom); err != nil {
				return nil, err
			}
			args = append(args,
				"--bot-token", botToken,
				"--default-account", defaultAccount,
				"--dm-policy", dmPolicy,
				"--allow-from", allowFrom,
			)
		}
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
		allowFrom := strings.TrimSpace(req.AllowFrom)
		if allowFrom == "" {
			allowFrom = "*"
		}
		if err := validateTelegramConfig(dmPolicy, allowFrom); err != nil {
			return nil, err
		}
		args = append(args,
			"--pid", strings.TrimSpace(req.PID),
			"--bot-token", strings.TrimSpace(req.BotToken),
			"--default-account", defaultAccount,
			"--dm-policy", dmPolicy,
			"--allow-from", allowFrom,
		)
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

func normalizeModelProvider(model, provider string) (string, string, error) {
	model = strings.TrimSpace(model)
	provider = strings.ToLower(strings.TrimSpace(provider))

	prefixedProvider, bareModel := splitModelProvider(model)
	if prefixedProvider == "" {
		return model, provider, nil
	}
	if provider == "" {
		return bareModel, prefixedProvider, nil
	}
	if provider != prefixedProvider {
		return "", "", fmt.Errorf("provider %q conflicts with model prefix %q", provider, prefixedProvider)
	}
	return bareModel, provider, nil
}

func splitModelProvider(model string) (string, string) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", ""
	}
	parts := strings.SplitN(model, "/", 2)
	if len(parts) != 2 {
		return "", model
	}
	return strings.ToLower(strings.TrimSpace(parts[0])), strings.TrimSpace(parts[1])
}

func requireFields(required map[string]string) error {
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: %s is required", errValidation, name)
		}
	}
	return nil
}

func validateTelegramConfig(dmPolicy, allowFrom string) error {
	if strings.EqualFold(strings.TrimSpace(dmPolicy), "open") && !allowFromIncludesWildcard(allowFrom) {
		return fmt.Errorf("%w: dmPolicy=open requires allowFrom to include \"*\"", errValidation)
	}
	return nil
}

func allowFromIncludesWildcard(allowFrom string) bool {
	allowFrom = strings.TrimSpace(allowFrom)
	if allowFrom == "" {
		return false
	}
	if allowFrom == "*" {
		return true
	}
	for _, part := range strings.Split(allowFrom, ",") {
		if strings.TrimSpace(part) == "*" {
			return true
		}
	}
	return strings.Contains(allowFrom, `"*"`)
}

func readHypeVersions(binaryPath, workingDir string) (string, string) {
	cmd := exec.Command(binaryPath, "-v")
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	var hypeVersion string
	var hymxVersion string
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Version:") {
			hypeVersion = strings.TrimSpace(strings.TrimPrefix(trimmed, "Version:"))
		}
		if strings.HasPrefix(trimmed, "HymxVersion:") {
			hymxVersion = strings.TrimSpace(strings.TrimPrefix(trimmed, "HymxVersion:"))
		}
	}
	return hypeVersion, hymxVersion
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
