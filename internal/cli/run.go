package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run generated project (cd cmd && go run ./)",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, err := filepath.Abs(".")
		if err != nil {
			return err
		}
		cmdDir := filepath.Join(projectDir, "cmd")
		if _, err := os.Stat(cmdDir); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("cmd directory not found in %s", projectDir)
			}
			return err
		}
		c := exec.Command("go", "run", "./")
		c.Dir = cmdDir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}
