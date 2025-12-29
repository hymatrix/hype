package cli

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/hymatrix/hype/internal/generator"
	genSchema "github.com/hymatrix/hype/internal/generator/schema"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new Golang project",
	RunE: func(cmd *cobra.Command, args []string) error {
		// --module / -m
		module, err := cmd.Flags().GetString("module")
		if err != nil {
			return err
		}
		if module == "" {
			return fmt.Errorf("module name is required")
		}

		// --out / -o
		outPath, err := cmd.Flags().GetString("out")
		if err != nil {
			return err
		}
		base := outPath
		if base == "" {
			base = "."
		}

		projectName := path.Base(module)

		// get project directory, absolute path
		projectDir := filepath.Join(base, projectName)
		absProjectDir, err := filepath.Abs(projectDir)
		if err != nil {
			return err
		}
		projectDir = absProjectDir

		fmt.Println("Project directory:", projectDir)
		fmt.Printf("Go Module: %s\n", module)

		// package name
		pkg := filepath.Base(projectDir)

		// generate project
		if err := generator.GenerateProject(genSchema.Options{
			Package:    pkg,
			ProjectDir: projectDir,
			GoModule:   module,
		}); err != nil {
			return err
		}

		fmt.Println("Project generation completed")
		return nil
	},
}
