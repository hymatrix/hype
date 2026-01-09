package cli

import (
	"errors"
	"path/filepath"

	"github.com/hymatrix/hype/internal/generator"
	genSchema "github.com/hymatrix/hype/internal/generator/schema"
	"github.com/spf13/cobra"
)

// helpers moved to internal/generator/get.go

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a VMM package via go tooling and mount it",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "package", Prompt: "package (-p/--package) " + usage_get_package + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg, err := cmd.Flags().GetString("package")
			if err != nil {
				return err
			}
			if pkg == "" {
				return errors.New("package is required")
			}
			projectDir, err := filepath.Abs(".")
			if err != nil {
				return err
			}

			return generator.Get(genSchema.Options{
				ProjectDir: projectDir,
				Package:    pkg,
			})
		},
	}

	cmd.Flags().StringP("package", "p", "", usage_get_package)
	return cmd
}
