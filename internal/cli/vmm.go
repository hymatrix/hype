package cli

import (
	"fmt"
	"path/filepath"

	"github.com/hymatrix/hype/internal/generator"
	"github.com/hymatrix/hype/internal/generator/schema"
	"github.com/spf13/cobra"
)

var vmmCmd = &cobra.Command{
	Use:   "vmm",
	Short: "Manage or scaffold a VM module",
	RunE: func(cmd *cobra.Command, args []string) error {
		// --name / -n
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("name is required")
		}
		fmt.Println("vmm name:", name)

		// --format / -f
		format, err := cmd.Flags().GetString("format")
		if err != nil {
			return err
		}
		if format == "" {
			return fmt.Errorf("format is required")
		}
		fmt.Println("format:", format)

		// get project directory
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
