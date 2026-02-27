package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/hymatrix/hype/internal/generator"
	genSchema "github.com/hymatrix/hype/internal/generator/schema"
	"github.com/spf13/cobra"
)

func newMountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mount",
		Short: "Mount a VM module into cmd/main.go",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "name", Prompt: "name (-n/--name) " + usage_mount_name + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := cmd.Flags().GetString("name")
			if err != nil {
				return err
			}
			if name == "" {
				return fmt.Errorf("name is required")
			}

			projectDir, err := filepath.Abs(".")
			if err != nil {
				return err
			}
			schemaPath := filepath.Join(projectDir, name, "schema", "schema.go")
			b, err := os.ReadFile(schemaPath)
			if err != nil {
				return err
			}
			re := regexp.MustCompile(`ModuleFormat\s*=\s*"(.*?)"`)
			m := re.FindStringSubmatch(string(b))
			if len(m) < 2 {
				return fmt.Errorf("ModuleFormat not found in %s", schemaPath)
			}
			format := m[1]
			if err := generator.GenetrateVmm(genSchema.Options{
				ProjectDir:   projectDir,
				VmmName:      name,
				ModuleFormat: format,
			}); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringP("name", "n", "", usage_mount_name)
	return cmd
}
