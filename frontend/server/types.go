package main

import (
	"sync"
	"time"
)

const (
	defaultNodeURL = "http://127.0.0.1:8080"
	defaultListen  = "127.0.0.1:7788"
)

type healthResponse struct {
	OK          bool   `json:"ok"`
	HypeBinary  string `json:"hypeBinary"`
	Listen      string `json:"listen"`
	HypeVersion string `json:"hypeVersion,omitempty"`
	HymxVersion string `json:"hymxVersion,omitempty"`
}

type openclawRequest struct {
	NodeURL        string `json:"nodeUrl"`
	PrivateKey     string `json:"privateKey"`
	ModuleID       string `json:"moduleId"`
	Scheduler      string `json:"scheduler"`
	Model          string `json:"model"`
	TimeoutMS      string `json:"timeoutMs"`
	APIKey         string `json:"apiKey"`
	GatewayToken   string `json:"gatewayToken"`
	PID            string `json:"pid"`
	BotToken       string `json:"botToken"`
	DefaultAccount string `json:"defaultAccount"`
	DMPolicy       string `json:"dmPolicy"`
	AllowFrom      string `json:"allowFrom"`
	Code           string `json:"code"`
	Channel        string `json:"channel"`
	Command        string `json:"command"`
}

type openclawResponse struct {
	OK          bool           `json:"ok"`
	ExitCode    int            `json:"exitCode"`
	Command     []string       `json:"command"`
	StdoutRaw   string         `json:"stdoutRaw"`
	StderrRaw   string         `json:"stderrRaw"`
	ParsedJSON  map[string]any `json:"parsedJson,omitempty"`
	SpawnPID    string         `json:"spawnPid,omitempty"`
	SpawnedPIDs []string       `json:"spawnedPids,omitempty"`
	Error       string         `json:"error,omitempty"`
	DurationMS  int64          `json:"durationMs"`
}

type runResult struct {
	response  openclawResponse
	duration  time.Duration
	binaryRef string
}

type spawnStore struct {
	mu   sync.Mutex
	pids []string
}

func (s *spawnStore) add(pid string) {
	if pid == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pids = append(s.pids, pid)
}

func (s *spawnStore) list() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.pids))
	copy(out, s.pids)
	return out
}
