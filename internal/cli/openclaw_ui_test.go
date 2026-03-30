package cli

import "testing"

func TestRootIncludesUICmd(t *testing.T) {
	root := NewRootCmd()
	cmd, _, err := root.Find([]string{"ui"})
	if err != nil {
		t.Fatalf("find ui command failed: %v", err)
	}
	if cmd == nil || cmd.Name() != "ui" {
		t.Fatalf("expected root ui command, got %#v", cmd)
	}
}
