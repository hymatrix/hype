package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hymatrix/hymx/server/schema"
	goarSchema "github.com/permadao/goar/schema"
)

type fakeVmdockerRuntimeClient struct {
	spawnModule    string
	spawnScheduler string
	spawnTags      []goarSchema.Tag
	spawnResponse  *schema.Response
	spawnErr       error

	messageTarget   string
	messageTags     []goarSchema.Tag
	messageResponse *schema.Response
	messageErr      error
}

func (f *fakeVmdockerRuntimeClient) SpawnAndWait(module, scheduler string, tags []goarSchema.Tag) (*schema.Response, error) {
	f.spawnModule, f.spawnScheduler = module, scheduler
	f.spawnTags = append([]goarSchema.Tag(nil), tags...)
	return f.spawnResponse, f.spawnErr
}

func (f *fakeVmdockerRuntimeClient) SendMessageAndWait(target, data string, tags []goarSchema.Tag) (*schema.Response, error) {
	f.messageTarget = target
	f.messageTags = append([]goarSchema.Tag(nil), tags...)
	return f.messageResponse, f.messageErr
}

func (f *fakeVmdockerRuntimeClient) Close() {}

func TestBuildVmdockerSpawnTags(t *testing.T) {
	tags, err := buildVmdockerSpawnTags("claude", "docker", []string{"TOKEN=a=b", "EMPTY="})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []goarSchema.Tag{
		{Name: "Container-Env-RUNTIME_TYPE", Value: "claude"},
		{Name: "Container-Env-TOKEN", Value: "a=b"},
		{Name: "Container-Env-EMPTY", Value: ""},
		{Name: "Runtime-Backend", Value: "docker"},
	} {
		if !hasTag(tags, want.Name, want.Value) {
			t.Fatalf("missing %#v in %#v", want, tags)
		}
	}
}

func TestBuildVmdockerSpawnTagsRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		env     []string
		wantErr string
	}{
		{name: "bad key", env: []string{"BAD-KEY=x"}, wantErr: "invalid environment key"},
		{name: "duplicate", env: []string{"TOKEN=a", "TOKEN=b"}, wantErr: "duplicate environment key"},
		{name: "reserved", env: []string{"RUNTIME_TYPE=claude"}, wantErr: "RUNTIME_TYPE is reserved"},
		{name: "bad backend", backend: "podman", wantErr: "runtime-backend must be one of"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildVmdockerSpawnTags("claude", tc.backend, tc.env)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestVmdockerSpawnCommand(t *testing.T) {
	fake := &fakeVmdockerRuntimeClient{spawnResponse: &schema.Response{Id: "pid-1", Message: "ok"}}
	cmd := newVmdockerSpawnCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
		return fake, nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--module-id", "mod-1", "--scheduler", "scheduler-1", "--private-key", "key", "--runtime-type", "claude", "--env", "TOKEN=a=b"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if fake.spawnModule != "mod-1" || fake.spawnScheduler != "scheduler-1" {
		t.Fatalf("module=%q scheduler=%q", fake.spawnModule, fake.spawnScheduler)
	}
	if !hasTag(fake.spawnTags, "Container-Env-TOKEN", "a=b") {
		t.Fatalf("tags = %#v", fake.spawnTags)
	}
	if !strings.Contains(out.String(), "spawn ok, pid: pid-1") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestVmdockerSpawnReplPromptsRequiredFlags(t *testing.T) {
	cmd := newVmdockerSpawnCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
		t.Fatalf("unexpected client creation")
		return nil, nil
	})
	var out bytes.Buffer
	cmd.SetContext(withRepl(context.Background(), bufio.NewReader(strings.NewReader("mod-1\nscheduler-1\nkey\nclaude\ndocker\n")), &out))
	if err := cmd.PreRunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]string{
		"module-id":       "mod-1",
		"scheduler":       "scheduler-1",
		"private-key":     "key",
		"runtime-type":    "claude",
		"runtime-backend": "docker",
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
		"module-id (-m/--module-id)",
		"scheduler (-s/--scheduler)",
		"private-key (-k/--private-key)",
		"runtime-type (--runtime-type)",
		"runtime-backend (--runtime-backend)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("prompt output missing %q:\n%s", want, out.String())
		}
	}
}

func TestVmdockerSpawnJSONHydratesEnvWithoutEchoingValues(t *testing.T) {
	t.Setenv("VMDOCKER_MODULE_ID", "mod-env")
	t.Setenv("VMDOCKER_SCHEDULER", "scheduler-env")
	t.Setenv("RUNTIME_TYPE", "claude")
	t.Setenv("RUNTIME_BACKEND", "sandbox")
	t.Setenv("VMDOCKER_URL", "http://node-env")
	t.Setenv("VMDOCKER_PRIVATE_KEY", "key-env")
	fake := &fakeVmdockerRuntimeClient{spawnResponse: &schema.Response{Id: "pid-1"}}
	var gotNodeURL, gotPrivateKey string
	cmd := newVmdockerSpawnCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
		gotNodeURL, gotPrivateKey = nodeURL, privateKey
		return fake, nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--json", "--env", "SECRET=do-not-print"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if gotNodeURL != "http://node-env" || gotPrivateKey != "key-env" {
		t.Fatalf("node=%q key=%q", gotNodeURL, gotPrivateKey)
	}
	if fake.spawnModule != "mod-env" || fake.spawnScheduler != "scheduler-env" {
		t.Fatalf("module=%q scheduler=%q", fake.spawnModule, fake.spawnScheduler)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["action"] != "spawn" || payload["pid"] != "pid-1" {
		t.Fatalf("payload = %#v", payload)
	}
	if strings.Contains(out.String(), "do-not-print") {
		t.Fatalf("secret leaked: %s", out.String())
	}
}

func TestDecodeVmdockerExportResult(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
		wantErr string
	}{
		{"success", `{"Data":"mod-2","Error":""}`, "mod-2", ""},
		{"vmm error", `{"Data":"","Error":"export failed"}`, "", "export failed"},
		{"malformed", `{`, "", "decode export result"},
		{"empty id", `{"Data":"   ","Error":""}`, "", "empty module id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeVmdockerExportResult(tc.message)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			if tc.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestVmdockerExportCommand(t *testing.T) {
	fake := &fakeVmdockerRuntimeClient{messageResponse: &schema.Response{
		Id: "response-1", Message: `{"Data":"mod-2","Error":""}`,
	}}
	cmd := newVmdockerExportCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
		return fake, nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--pid", "pid-1", "--private-key", "key"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if fake.messageTarget != "pid-1" {
		t.Fatalf("target = %q", fake.messageTarget)
	}
	if len(fake.messageTags) != 1 || fake.messageTags[0] != (goarSchema.Tag{Name: "Action", Value: "Export"}) {
		t.Fatalf("tags = %#v", fake.messageTags)
	}
	if !strings.Contains(out.String(), "export ok, module id: mod-2") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestVmdockerExportReplPromptsRequiredFlags(t *testing.T) {
	cmd := newVmdockerExportCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
		t.Fatalf("unexpected client creation")
		return nil, nil
	})
	var out bytes.Buffer
	cmd.SetContext(withRepl(context.Background(), bufio.NewReader(strings.NewReader("pid-1\nkey\n")), &out))
	if err := cmd.PreRunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	pid, _ := cmd.Flags().GetString("pid")
	privateKey, _ := cmd.Flags().GetString("private-key")
	if pid != "pid-1" || privateKey != "key" {
		t.Fatalf("pid=%q key=%q", pid, privateKey)
	}
	for _, want := range []string{
		"pid (-p/--pid)",
		"private-key (-k/--private-key)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("prompt output missing %q:\n%s", want, out.String())
		}
	}
}

func TestVmdockerExportCommandDoesNotPrintSuccessOnNodeError(t *testing.T) {
	fake := &fakeVmdockerRuntimeClient{messageResponse: &schema.Response{Message: `{"Error":"export failed"}`}}
	cmd := newVmdockerExportCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
		return fake, nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--pid", "pid-1", "--private-key", "key"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "export failed") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(out.String(), "export ok") {
		t.Fatalf("unexpected success output: %q", out.String())
	}
}

func TestVmdockerExportJSON(t *testing.T) {
	fake := &fakeVmdockerRuntimeClient{messageResponse: &schema.Response{Id: "response-1", Message: `{"Data":"mod-2"}`}}
	cmd := newVmdockerExportCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
		return fake, nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--pid", "pid-1", "--private-key", "do-not-print", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["module_id"] != "mod-2" {
		t.Fatalf("payload = %#v", payload)
	}
	if strings.Contains(out.String(), "do-not-print") {
		t.Fatalf("private key leaked: %s", out.String())
	}
}
