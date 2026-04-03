package cli

import (
	"bufio"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "hype",
		Short:   "Hymx project scaffolding and management CLI",
		Long:    "Command-line tool to generate Hymx project structure, manage modules, and run the sample project.",
		Version: Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReplLoop(
				bufio.NewReader(cmd.InOrStdin()),
				cmd.OutOrStdout(),
				cmd.ErrOrStderr(),
			)
		},
	}

	rootCmd.AddCommand(newNewCmd())
	rootCmd.AddCommand(newGetCmd())
	rootCmd.AddCommand(newVmmCmd())
	rootCmd.AddCommand(newMountCmd())
	rootCmd.AddCommand(newModuleCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newVmdockerCmd())
	rootCmd.AddCommand(newDBImportCmd())
	rootCmd.AddCommand(newExportJSONLCmd())
	rootCmd.AddCommand(newReplCmd())
	rootCmd.AddCommand(newOpenclawCmd())
	rootCmd.AddCommand(newOpenclawUICmd())
	rootCmd.AddCommand(newVersionCmd())

	rootCmd.SetVersionTemplate(versionBanner())
	return rootCmd
}

func Execute() error { return NewRootCmd().Execute() }
