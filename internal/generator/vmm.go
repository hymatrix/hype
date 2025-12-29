package generator

import (
	"os"
	"path/filepath"

	"github.com/hymatrix/hype/internal/generator/schema"
)

func genVmm(opts *schema.Options) error {
	projectDir := opts.ProjectDir

	data := schema.Options{
		Package:      opts.VmmName,
		ModuleFormat: opts.ModuleFormat,
	}

	// create vmm package
	outDir := filepath.Join(projectDir, opts.VmmName)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := renderTemplateFile("cmd/vmm.go.tmpl", filepath.Join(outDir, opts.VmmName+".go"), data); err != nil {
		return err
	}
	// create vmm schema
	if err := os.MkdirAll(filepath.Join(outDir, "schema"), 0o755); err != nil {
		return err
	}
	if err := renderTemplateFile("cmd/vmmschema.go.tmpl", filepath.Join(outDir, "schema", "schema.go"), data); err != nil {
		return err
	}

	return nil
}
