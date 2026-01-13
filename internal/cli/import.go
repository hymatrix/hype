package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hymatrix/hype/internal/syncer"
	"github.com/spf13/cobra"
)

func newDBImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db-import",
		Short: "Import a json data file into redis",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "redis-url", Prompt: "redis-url (-r/--redis-url) " + usage_db_import_redis_url + ": "},
				{Name: "file", Prompt: "file (-f/--file) " + usage_db_import_file + ": "},
			})
		},
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
			if !strings.HasSuffix(jsonFile, ".jsonl") {
				return errors.New("file must be .jsonl")
			}
			if err := syncer.ImportFromJSONL(redisURL, jsonFile, force); err != nil {
				fmt.Fprintln(os.Stderr, "db-import failed:", err)
				return err
			}
			fmt.Println("db-import succeeded (jsonl)")
			return nil
		},
	}

	cmd.Flags().StringP("redis-url", "r", "", usage_db_import_redis_url)
	cmd.Flags().StringP("file", "f", "", usage_db_import_file)
	cmd.Flags().BoolP("force", "F", false, usage_db_import_force)
	return cmd
}
