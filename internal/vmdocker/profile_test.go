package vmdocker

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitProfileCreatesTestagentScaffold(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agent")
	absDir, err := NewManager().InitProfile(dir, "registry.example/base:1")
	if err != nil {
		t.Fatal(err)
	}
	if absDir != dir {
		t.Fatalf("absDir = %q, want %q", absDir, dir)
	}
	profile, err := os.ReadFile(filepath.Join(dir, "profile.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`FROM = "registry.example/base:1"`,
		`tools = []`,
		`RUN = []`,
		`# CMD = ["openclaw", "gateway", "--serve"]`,
		`public = ["~/skills/*", "~/persona/*", "~/.hermes/plugin/*"]`,
		`# Declarative recipe for a vmdockerv2 agent module.`,
	} {
		if !strings.Contains(string(profile), want) {
			t.Fatalf("profile missing %q:\n%s", want, profile)
		}
	}
	for _, tc := range []struct {
		path string
		want string
	}{
		{filepath.Join(dir, "skills", "soul.md"), "MY-SOUL\n"},
		{filepath.Join(dir, "persona", "style.md"), "terse, precise\n"},
		{filepath.Join(dir, "bin", ".keep"), ""},
	} {
		got, err := os.ReadFile(tc.path)
		if err != nil || string(got) != tc.want {
			t.Fatalf("%s = %q, %v; want %q", tc.path, got, err, tc.want)
		}
	}
}

func TestInitProfileRefusesNonEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := NewManager().InitProfile(dir, "example/base:latest")
	if err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("expected non-empty error, got %v", err)
	}
}

func TestInitProfileRequiresFrom(t *testing.T) {
	_, err := NewManager().InitProfile(filepath.Join(t.TempDir(), "agent"), "  ")
	if err == nil || err.Error() != "from is required" {
		t.Fatalf("err = %v", err)
	}
}

func TestInitProfileRejectsFileTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent")
	if err := os.WriteFile(path, []byte("occupied"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := NewManager().InitProfile(path, "example/base:latest")
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("err = %v", err)
	}
}

func TestInitProfileCleansNewDirectoryOnWriteFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agent")
	manager := newFailingProfileManager(2)

	if _, err := manager.InitProfile(dir, "registry.example/base:1"); err == nil {
		t.Fatal("expected write failure")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected failed new profile directory removed, stat err=%v", err)
	}
}

func TestInitProfileLeavesExistingEmptyDirectoryReusableOnWriteFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manager := newFailingProfileManager(2)

	if _, err := manager.InitProfile(dir, "registry.example/base:1"); err == nil {
		t.Fatal("expected write failure")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("expected existing profile directory preserved, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected existing profile directory cleaned, got %d entries", len(entries))
	}

	if _, err := NewManager().InitProfile(dir, "registry.example/base:1"); err != nil {
		t.Fatalf("expected cleaned directory to be reusable, got %v", err)
	}
}

func newFailingProfileManager(failOnWrite int) *Manager {
	writes := 0
	return &Manager{
		stat:      os.Stat,
		readDir:   os.ReadDir,
		mkdirAll:  os.MkdirAll,
		remove:    os.Remove,
		removeAll: os.RemoveAll,
		writeFile: func(path string, data []byte, mode os.FileMode) error {
			writes++
			if writes == failOnWrite {
				return errors.New("write failed")
			}
			return os.WriteFile(path, data, mode)
		},
	}
}
