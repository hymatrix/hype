package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "hype",
	Short:   "Hymx project scaffolding and management CLI",
	Long:    "Command-line tool to generate Hymx project structure, manage modules, and run the sample project.",
	Version: Version,
}

func init() {
	// new
	newCmd.Flags().StringP("out", "o", ".", "Output base directory for the generated project")
	newCmd.Flags().StringP("module", "m", "", "Go module name (e.g. github.com/hymatrix/hype)")
	rootCmd.AddCommand(newCmd)
	// download
	getCmd.Flags().StringP("package", "p", "", "Go module path of the VMM package")
	rootCmd.AddCommand(getCmd)
	// vmm
	vmmCmd.Flags().StringP("name", "n", "", "Name of the vmm")
	vmmCmd.Flags().StringP("format", "f", "", "Module format of the vmm")
	rootCmd.AddCommand(vmmCmd)
	// mount
	mountCmd.Flags().StringP("name", "n", "", "Name of the vmm")
	rootCmd.AddCommand(mountCmd)
	// module
	moduleCmd.Flags().StringP("name", "n", "", "Name of the module")
	moduleCmd.Flags().StringP("node-url", "u", "http://127.0.0.1:8080", "Node URL")
	moduleCmd.Flags().StringP("private-key", "k", "", "Ethereum ECDSA secp256k1 private key hex (0x-prefixed)")
	rootCmd.AddCommand(moduleCmd)
	// run
	rootCmd.AddCommand(runCmd)
	// db-import
	dbImportCmd.Flags().StringP("redis-url", "r", "", "Redis URL")
	dbImportCmd.Flags().StringP("file", "f", "", "JSON file path")
	dbImportCmd.Flags().BoolP("force", "F", false, "Override if data exists")
	rootCmd.AddCommand(dbImportCmd)
	// db-export
	exportJsonlCmd.Flags().StringP("redis-url", "r", "", "Redis URL")
	exportJsonlCmd.Flags().StringP("pid", "p", "", "Process id")
	exportJsonlCmd.Flags().StringP("out", "o", "", "Output jsonl file path (supports .gz)")
	exportJsonlCmd.Flags().Int64("progress-every", 1000, "Print progress every N lines")
	rootCmd.AddCommand(exportJsonlCmd)
	// version
	rootCmd.AddCommand(versionCmd)
	rootCmd.SetVersionTemplate(versionBanner())
}

func Execute() error { return rootCmd.Execute() }
