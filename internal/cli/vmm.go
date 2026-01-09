package cli

import (
	"fmt"
	"path/filepath"

	"github.com/hymatrix/hype/internal/generator"
	"github.com/hymatrix/hype/internal/generator/schema"
	"github.com/spf13/cobra"
)

func newVmmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vmm",
		Short: "Manage or scaffold a VM module",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "name", Prompt: "name (-n/--name) " + usage_vmm_name + ": "},
				{Name: "format", Prompt: "format (-f/--format) " + usage_vmm_format + ": "},
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
			fmt.Println("vmm name:", name)

			format, err := cmd.Flags().GetString("format")
			if err != nil {
				return err
			}
			if format == "" {
				return fmt.Errorf("format is required")
			}
			fmt.Println("format:", format)

			projectDir, err := filepath.Abs(".")
			if err != nil {
				return err
			}

			if err := generator.GenetrateVmm(schema.Options{
				ProjectDir:   projectDir,
				VmmName:      name,
				ModuleFormat: format,
			}); err != nil {
				return err
			}

			fmt.Println("create vmm success:", name)
			return nil
		},
	}

	cmd.Flags().StringP("name", "n", "", usage_vmm_name)
	cmd.Flags().StringP("format", "f", "", usage_vmm_format)
	return cmd
}
