package cli

import "testing"

func TestVmdockerV2Defaults(t *testing.T) {
	root := NewRootCmd()
	get, _, err := root.Find([]string{"vmdocker", "get"})
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := get.Flags().GetString("ref")
	dir, _ := get.Flags().GetString("dir")
	if ref != "main" || dir != "./vmdockerv2" {
		t.Fatalf("ref=%q dir=%q", ref, dir)
	}
	if get.Flags().Lookup("version") != nil {
		t.Fatal("--version must be removed")
	}
	initCmd, _, err := root.Find([]string{"vmdocker", "init"})
	if err != nil {
		t.Fatal(err)
	}
	initDir, _ := initCmd.Flags().GetString("dir")
	if initDir != "./vmdockerv2" {
		t.Fatalf("init dir = %q", initDir)
	}
	if initCmd.Flags().Lookup("env-file") == nil {
		t.Fatal("--env-file must be defined")
	}
	if initCmd.Flags().Lookup("private-key") != nil {
		t.Fatal("--private-key must not be defined for init")
	}
	cleanCmd, _, err := root.Find([]string{"vmdocker", "clean"})
	if err != nil {
		t.Fatal(err)
	}
	cleanDir, _ := cleanCmd.Flags().GetString("dir")
	if cleanDir != "./vmdockerv2" {
		t.Fatalf("clean dir = %q", cleanDir)
	}
}
