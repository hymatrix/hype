package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hymatrix/hype/internal/generator/schema"
)

func GenerateProject(opts schema.Options) error {
	// pkg := opts.Package
	projectDir := opts.ProjectDir
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return err
	}

	// generate frameworks
	if err := genFrameworks(opts); err != nil {
		return err
	}

	// go init & tidy
	goModule := opts.GoModule
	if goModule == "" {
		return fmt.Errorf("go module cannot be empty")
	}

	if err := runGoInitAndTidy(filepath.Join(projectDir), goModule); err != nil {
		return err
	}

	return nil
}

func GenetrateVmm(opts schema.Options) error {
	// get go module
	goModule, err := getGoModule(opts.ProjectDir)
	if err != nil {
		return err
	}
	opts.GoModule = goModule
	if err := genVmm(&opts); err != nil {
		return err
	}
	return mount(&opts)
}
