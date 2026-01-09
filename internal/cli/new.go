package cli

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/hymatrix/hype/internal/generator"
	genSchema "github.com/hymatrix/hype/internal/generator/schema"

	"github.com/spf13/cobra"
)

func newNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new Golang project",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "module", Prompt: "module (-m/--module) " + usage_new_module + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			module, err := cmd.Flags().GetString("module")
			if err != nil {
				return err
			}
			if module == "" {
				return fmt.Errorf("module name is required")
			}

			outPath, err := cmd.Flags().GetString("out")
			if err != nil {
				return err
			}
			base := outPath
			if base == "" {
				base = "."
			}

			projectName := path.Base(module)

			projectDir := filepath.Join(base, projectName)
			absProjectDir, err := filepath.Abs(projectDir)
			if err != nil {
				return err
			}
			projectDir = absProjectDir

			fmt.Println("Project directory:", projectDir)
			fmt.Printf("Go Module: %s\n", module)

			pkg := filepath.Base(projectDir)

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

	cmd.Flags().StringP("out", "o", ".", usage_new_out)
	cmd.Flags().StringP("module", "m", "", usage_new_module)
	return cmd
}
