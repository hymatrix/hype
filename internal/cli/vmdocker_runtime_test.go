package cli

import (
	"bytes"
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
