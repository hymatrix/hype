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

var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Generate and mount a module by name",
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
		// get format
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

		// get node url, default to local node
		nodeUrl, err := cmd.Flags().GetString("node-url")
		if err != nil {
			return err
		}
		if nodeUrl == "" {
			nodeUrl = "http://127.0.0.1:8080"
		}

		// get private key, 0x prefixed
		privateKey, err := cmd.Flags().GetString("private-key")
		if err != nil {
			return err
		}
		if err := generator.GenAndSaveModule(genSchema.Options{
			ProjectDir:   projectDir,
			VmmName:      name,
			ModuleFormat: format,
			NodeUrl:      nodeUrl,
			PrivateKey:   privateKey,
		}); err != nil {
			return err
		}
		return nil
	},
}
