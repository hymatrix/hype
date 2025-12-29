package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func checkGoEnv(projectDir string) error {
	if projectDir == "" {
		return fmt.Errorf("projectDir is empty")
	}
	goMod := filepath.Join(projectDir, "go.mod")
	if _, err := os.Stat(goMod); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("go.mod not found in %s", projectDir)
		}
		return err
	}
	return nil
}

func getGoModule(projectDir string) (string, error) {
	if err := checkGoEnv(projectDir); err != nil {
		return "", err
	}

	goMod := filepath.Join(projectDir, "go.mod")
	lines, err := readLines(goMod)
	if err != nil {
		return "", err
	}
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "module ")), nil
		}
	}
	return "", fmt.Errorf("module not declared in %s", goMod)
}
