package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hymatrix/hype/internal/syncer"
	"github.com/spf13/cobra"
)

var syncJsonCmd = &cobra.Command{
	Use:   "sync-json",
	Short: "Sync a json data file into redis",
	RunE: func(cmd *cobra.Command, args []string) error {
		redisURL, err := cmd.Flags().GetString("redis-url")
		if err != nil {
			return err
		}
		jsonFile, err := cmd.Flags().GetString("file")
		if err != nil {
			return err
		}
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return err
		}
		if redisURL == "" || jsonFile == "" {
			return errors.New("redis-url and file are required")
		}
		if strings.HasSuffix(jsonFile, ".jsonl") {
			if err := syncer.ImportFromJSONL(redisURL, jsonFile, force); err != nil {
				fmt.Fprintln(os.Stderr, "sync-json failed:", err)
				return err
			}
			fmt.Println("sync-json succeeded (jsonl)")
			return nil
		}
		if err := syncer.ImportFromJSON(redisURL, jsonFile, force); err != nil {
			fmt.Fprintln(os.Stderr, "sync-json failed:", err)
			return err
		}
		fmt.Println("sync-json succeeded")
		return nil
	},
}
