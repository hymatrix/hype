package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVmdockerProfileInitCreatesScaffold(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agent")
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{
		"vmdocker", "profile", "init",
		"--dir", dir,
		"--from", "registry.example/base:1",
	})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		filepath.Join(dir, "profile.toml"),
		filepath.Join(dir, "skills", "soul.md"),
		filepath.Join(dir, "persona", "style.md"),
		filepath.Join(dir, "bin", ".keep"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	for _, want := range []string{
		"profile directory: " + dir,
		"profile file: " + filepath.Join(dir, "profile.toml"),
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}
