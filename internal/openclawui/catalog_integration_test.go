package openclawui_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hymatrix/hype/internal/cli"
	"github.com/hymatrix/hype/internal/openclawui"
)

type testCatalogResponse struct {
	OK    bool              `json:"ok"`
	Roots []testCatalogRoot `json:"roots"`
}

type testCatalogRoot struct {
	Name     string               `json:"name"`
	Commands []testCatalogCommand `json:"commands"`
}

type testCatalogCommand struct {
	Name           string             `json:"name"`
	Path           []string           `json:"path"`
	Supported      bool               `json:"supported"`
	DisabledReason string             `json:"disabledReason"`
	Fields         []testCatalogField `json:"fields"`
}

type testCatalogField struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Required bool     `json:"required"`
	EnvKeys  []string `json:"envKeys"`
}

type testRunResponse struct {
	OK      bool     `json:"ok"`
	Command []string `json:"command"`
	Error   string   `json:"error"`
}

func TestCatalogRouteListsExpectedCommands(t *testing.T) {
	handler := newHandlerWithRealCatalog(t)

	req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body testCatalogResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal catalog response failed: %v", err)
	}
	if !body.OK {
		t.Fatalf("expected ok response, got %#v", body)
	}

	expectedRoots := []string{"new", "get", "vmm", "mount", "module", "run", "openclaw", "ui", "version", "vmdocker", "repl", "db-import", "db-export"}
	for _, name := range expectedRoots {
		if !hasRoot(body.Roots, name) {
			t.Fatalf("expected root %q in catalog", name)
		}
	}
	if hasRoot(body.Roots, "help") || hasRoot(body.Roots, "completion") {
		t.Fatalf("catalog must not expose help/completion: %#v", body.Roots)
	}

	uiCommand := findCommand(body.Roots, []string{"ui"})
	if uiCommand == nil || uiCommand.Supported || uiCommand.DisabledReason == "" {
		t.Fatalf("expected ui command to be disabled, got %#v", uiCommand)
	}
	replCommand := findCommand(body.Roots, []string{"repl"})
	if replCommand == nil || replCommand.Supported || replCommand.DisabledReason == "" {
		t.Fatalf("expected repl command to be disabled, got %#v", replCommand)
	}
}

func TestCatalogRouteIncludesRequiredAndEnvMappedFields(t *testing.T) {
	handler := newHandlerWithRealCatalog(t)

	req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body testCatalogResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal catalog response failed: %v", err)
	}

	dbImport := findCommand(body.Roots, []string{"db-import"})
	if dbImport == nil {
		t.Fatal("expected db-import catalog entry")
	}
	if field := findField(dbImport.Fields, "file"); field == nil || !field.Required || field.Kind != "path" {
		t.Fatalf("expected db-import.file to be required path field, got %#v", field)
	}
	if field := findField(dbImport.Fields, "redis-url"); field == nil || !field.Required || !contains(field.EnvKeys, "REDIS_URL") {
		t.Fatalf("expected db-import.redis-url to be required with REDIS_URL alias, got %#v", field)
	}

	spawn := findCommand(body.Roots, []string{"openclaw", "spawn"})
	if spawn == nil {
		t.Fatal("expected openclaw spawn catalog entry")
	}
	if field := findField(spawn.Fields, "gateway-token"); field == nil || !field.Required || !contains(field.EnvKeys, "OPENCLAW_GATEWAY_TOKEN") {
		t.Fatalf("expected openclaw spawn.gateway-token to be required with env alias, got %#v", field)
	}
	if field := findField(spawn.Fields, "api-key"); field == nil || !contains(field.EnvKeys, "OPENCLAW_API_KEY") {
		t.Fatalf("expected openclaw spawn.api-key to expose env alias, got %#v", field)
	}

	confTG := findCommand(body.Roots, []string{"openclaw", "conf-tg"})
	if confTG == nil {
		t.Fatal("expected openclaw conf-tg catalog entry")
	}
	if field := findField(confTG.Fields, "bot-token"); field == nil || !field.Required || !contains(field.EnvKeys, "OPENCLAW_TELEGRAM_BOT_TOKEN") {
		t.Fatalf("expected openclaw conf-tg.bot-token to be required with env alias, got %#v", field)
	}

	chat := findCommand(body.Roots, []string{"openclaw", "chat"})
	if chat == nil {
		t.Fatal("expected openclaw chat catalog entry")
	}
	if field := findField(chat.Fields, "command"); field == nil || !field.Required || field.Kind != "multiline" {
		t.Fatalf("expected openclaw chat.command to be required multiline field, got %#v", field)
	}
}

func TestRunRouteBuildsExpectedCommands(t *testing.T) {
	handler := newHandlerWithRealCatalog(t)

	testCases := []struct {
		name      string
		body      string
		fragments []string
	}{
		{
			name: "new",
			body: `{"path":["new"],"values":{"module":"github.com/acme/demo","out":"./output"}}`,
			fragments: []string{
				`hype new `,
				`--module github.com/acme/demo`,
				`--out ./output`,
			},
		},
		{
			name: "module",
			body: `{"path":["module"],"values":{"name":"sample","node-url":"http://127.0.0.1:8080","private-key":"0xabc"}}`,
			fragments: []string{
				`hype module `,
				`--name sample`,
				`--node-url http://127.0.0.1:8080`,
				`--private-key ***`,
			},
		},
		{
			name: "db-import",
			body: `{"path":["db-import"],"values":{"redis-url":"redis://localhost:6379/0","file":"./data.jsonl","force":true}}`,
			fragments: []string{
				`hype db-import `,
				`--redis-url redis://localhost:6379/0`,
				`--file ./data.jsonl`,
				`--force true`,
			},
		},
		{
			name: "db-export",
			body: `{"path":["db-export"],"values":{"redis-url":"redis://localhost:6379/0","pid":"proc-1","out":"./out.jsonl","progress-every":"50"}}`,
			fragments: []string{
				`hype db-export `,
				`--redis-url redis://localhost:6379/0`,
				`--pid proc-1`,
				`--out ./out.jsonl`,
				`--progress-every 50`,
			},
		},
		{
			name: "run",
			body: `{"path":["run"],"values":{"mode":"rebuild"}}`,
			fragments: []string{
				`hype run `,
				`--mode rebuild`,
			},
		},
		{
			name: "version",
			body: `{"path":["version"],"values":{}}`,
			fragments: []string{
				`hype version`,
			},
		},
		{
			name: "openclaw spawn",
			body: `{"path":["openclaw","spawn"],"values":{"node-url":"http://127.0.0.1:8080","private-key":"0xabc","module-id":"mod-1","scheduler":"sched-1","gateway-token":"gw"}}`,
			fragments: []string{
				`hype openclaw spawn `,
				`--json`,
				`--node-url http://127.0.0.1:8080`,
				`--private-key ***`,
				`--module-id mod-1`,
				`--scheduler sched-1`,
				`--gateway-token ***`,
			},
		},
		{
			name: "vmdocker get",
			body: `{"path":["vmdocker","get"],"values":{"dir":"./vmdocker","version":"v0.0.1"}}`,
			fragments: []string{
				`hype vmdocker get `,
				`--dir ./vmdocker`,
				`--version v0.0.1`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/run", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
			}

			var body testRunResponse
			if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal run response failed: %v", err)
			}
			if !body.OK {
				t.Fatalf("expected ok response, got %#v", body)
			}
			command := strings.Join(body.Command, " ")
			for _, fragment := range tc.fragments {
				if !strings.Contains(command, fragment) {
					t.Fatalf("expected command to contain %q, got %q", fragment, command)
				}
			}
		})
	}
}

func newHandlerWithRealCatalog(t *testing.T) http.Handler {
	t.Helper()

	openclawui.SetCommandCatalogFactory(cli.NewRootCmd)
	binaryPath := writeEchoBinary(t, t.TempDir())

	handler, err := openclawui.NewHandler(openclawui.Config{
		BinaryPath: binaryPath,
		WorkingDir: t.TempDir(),
		Listen:     "127.0.0.1:7788",
	})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}
	return handler
}

func writeEchoBinary(t *testing.T, dir string) string {
	t.Helper()

	script := `#!/bin/sh
printf '%s\n' "$@"
`
	binaryPath := filepath.Join(dir, "hype")
	if err := os.WriteFile(binaryPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake binary failed: %v", err)
	}
	return binaryPath
}

func hasRoot(roots []testCatalogRoot, name string) bool {
	for _, root := range roots {
		if root.Name == name {
			return true
		}
	}
	return false
}

func findCommand(roots []testCatalogRoot, path []string) *testCatalogCommand {
	for _, root := range roots {
		for _, command := range root.Commands {
			if len(command.Path) != len(path) {
				continue
			}
			match := true
			for i := range path {
				if command.Path[i] != path[i] {
					match = false
					break
				}
			}
			if match {
				commandCopy := command
				return &commandCopy
			}
		}
	}
	return nil
}

func findField(fields []testCatalogField, name string) *testCatalogField {
	for _, field := range fields {
		if field.Name == name {
			fieldCopy := field
			return &fieldCopy
		}
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
