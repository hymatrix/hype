package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hymatrix/hype/internal/generator/schema"
)

func Mount(opts schema.Options) error {
	goModule, err := getGoModule(opts.ProjectDir)
	if err != nil {
		return err
	}
	opts.GoModule = goModule
	return mount(&opts)
}

func MountFromGoPath(opts schema.Options) error {
	return mount(&opts)
}

func mount(opts *schema.Options) error {
	projectDir := opts.ProjectDir
	vmm := opts.VmmName
	goModule := opts.GoModule

	if projectDir == "" || vmm == "" || goModule == "" {
		return fmt.Errorf("invalid mount options")
	}

	mainPath := filepath.Join(projectDir, "cmd", "main.go")
	src, err := os.ReadFile(mainPath)
	if err != nil {
		return err
	}

	content := string(src)
	// imports insertion idempotency check
	hasImportModule := strings.Contains(content, fmt.Sprintf("\"%s/%s\"", goModule, vmm))
	hasImportSchema := strings.Contains(content, fmt.Sprintf("\"%s/%s/schema\"", goModule, vmm))

	updated := content

	if !(hasImportModule && hasImportSchema) {
		// find import block
		importStart := strings.Index(updated, "\nimport (")
		if importStart == -1 {
			return fmt.Errorf("import block not found in %s", mainPath)
		}
		// find the closing parenthesis of the first import block after importStart
		closeIdx := strings.Index(updated[importStart:], ")\n")
		if closeIdx == -1 {
			return fmt.Errorf("import block not closed in %s", mainPath)
		}
		closeIdx += importStart

		// construct new import lines
		fmt.Println("go module: ", goModule)
		fmt.Println("vmm: ", vmm)
		newLines := []string{
			fmt.Sprintf("\t%s \"%s/%s\"\n", vmm, goModule, vmm),
			fmt.Sprintf("\t%sSchema \"%s/%s/schema\"\n", vmm, goModule, vmm),
		}
		insert := strings.Join(newLines, "")

		// insert right before the closing parenthesis of the import block
		var buf bytes.Buffer
		buf.WriteString(updated[:closeIdx])
		buf.WriteString(insert)
		buf.WriteString(updated[closeIdx:])
		updated = buf.String()
	}

	// mount call insertion idempotency check
	mountCall := fmt.Sprintf("\ts.Mount(%sSchema.ModuleFormat, %s.Spawn)\n", vmm, vmm)
	if !strings.Contains(updated, mountCall) {
		// find anchor comment block
		anchor := "FuncForSpawn: the function for spawn your vm"
		pos := strings.Index(updated, anchor)
		if pos == -1 {
			// fallback: insert before s.Run(
			runPos := strings.Index(updated, "\ns.Run(")
			if runPos == -1 {
				return fmt.Errorf("mount anchor not found in %s", mainPath)
			}
			// insert a blank line and mount before s.Run
			updated = updated[:runPos] + mountCall + updated[runPos:]
		} else {
			// insert right after the anchor line
			nl := strings.Index(updated[pos:], "\n")
			if nl == -1 {
				return fmt.Errorf("unexpected content near mount anchor in %s", mainPath)
			}
			insertPoint := pos + nl + 1
			updated = updated[:insertPoint] + mountCall + updated[insertPoint:]
		}
	}

	err = os.WriteFile(mainPath, []byte(updated), 0o644)
	if err != nil {
		return err
	}

	return runCmd(projectDir, "go", "mod", "tidy")
}
