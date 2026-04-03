package openclawui

import "time"

const (
	defaultNodeURL = "http://127.0.0.1:8080"
	defaultListen  = "127.0.0.1:7788"
)

const defaultCommandTimeout = 90 * time.Second

type Config struct {
	Listen     string
	BinaryPath string
	WorkingDir string
	Timeout    time.Duration
}

type healthResponse struct {
	OK          bool   `json:"ok"`
	HypeBinary  string `json:"hypeBinary"`
	Listen      string `json:"listen"`
	WorkingDir  string `json:"workingDir"`
	VmdockerDir string `json:"vmdockerDir"`
	HypeVersion string `json:"hypeVersion,omitempty"`
	HymxVersion string `json:"hymxVersion,omitempty"`
}

type envLoadRequest struct {
	Path string `json:"path"`
}

type envLoadResponse struct {
	OK       bool   `json:"ok"`
	FileName string `json:"fileName,omitempty"`
	Path     string `json:"path,omitempty"`
	Content  string `json:"content,omitempty"`
	Error    string `json:"error,omitempty"`
}

type openclawRequest struct {
	NodeURL        string `json:"nodeUrl"`
	PrivateKey     string `json:"privateKey"`
	ModuleID       string `json:"moduleId"`
	Scheduler      string `json:"scheduler"`
	Model          string `json:"model"`
	Provider       string `json:"provider"`
	APIKey         string `json:"apiKey"`
	GatewayToken   string `json:"gatewayToken"`
	RuntimeBackend string `json:"runtimeBackend"`
	PID            string `json:"pid"`
	BotToken       string `json:"botToken"`
	DefaultAccount string `json:"defaultAccount"`
	DMPolicy       string `json:"dmPolicy"`
	AllowFrom      string `json:"allowFrom"`
	Code           string `json:"code"`
	Channel        string `json:"channel"`
	Command        string `json:"command"`
	Dir            string `json:"dir"`
	Version        string `json:"version"`
	EnvFileName    string `json:"envFileName"`
	EnvFileContent string `json:"envFileContent"`
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
	pids []string
}

func (s *spawnStore) add(pid string) {
	if pid == "" {
		return
	}
	s.pids = append(s.pids, pid)
}

func (s *spawnStore) list() []string {
	out := make([]string, len(s.pids))
	copy(out, s.pids)
	return out
}
