package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hymatrix/hype/internal/syncer"
	"github.com/hymatrix/hype/internal/syncer/schema"
	"github.com/spf13/cobra"
)

func newExportJSONLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db-export",
		Short: "Export process data from redis into jsonl",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "redis-url", Prompt: "redis-url (-r/--redis-url) " + usage_db_export_redis_url + ": "},
				{Name: "pid", Prompt: "pid (-p/--pid) " + usage_db_export_pid + ": "},
				{Name: "out", Prompt: "out (-o/--out) " + usage_db_export_out + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			redisURL, err := cmd.Flags().GetString("redis-url")
			if err != nil {
				return err
			}
			pid, err := cmd.Flags().GetString("pid")
			if err != nil {
				return err
			}
			outFile, err := cmd.Flags().GetString("out")
			if err != nil {
				return err
			}
			if outFile != "" && !strings.HasSuffix(outFile, ".jsonl") && !strings.HasSuffix(outFile, ".jsonl.gz") {
				outFile = outFile + ".jsonl"
			}
			progressEvery, err := cmd.Flags().GetInt64("progress-every")
			if err != nil {
				return err
			}
			if redisURL == "" || pid == "" || outFile == "" {
				return errors.New("redis-url, pid and out are required")
			}

			start := time.Now()
			var lastPrint time.Time
			progress := func(done, total int64) {
				now := time.Now()
				if !lastPrint.IsZero() && now.Sub(lastPrint) < 200*time.Millisecond && done != total {
					return
				}
				lastPrint = now
				percent := float64(done) * 100 / float64(total)
				fmt.Fprintf(os.Stderr, "\rexporting %s: %d/%d (%.2f%%)", pid, done, total, percent)
				if done == total {
					fmt.Fprintf(os.Stderr, " in %s\n", time.Since(start).Truncate(time.Millisecond))
				}
			}

			if err := syncer.ExportToJSONL(redisURL, pid, outFile, &schema.ExportOptions{
				ProgressEvery: progressEvery,
				Progress:      progress,
			}); err != nil {
				fmt.Fprintln(os.Stderr, "db-export failed:", err)
				return err
			}
			fmt.Println("db-export succeeded:", outFile)
			return nil
		},
	}

	cmd.Flags().StringP("redis-url", "r", "", usage_db_export_redis_url)
	cmd.Flags().StringP("pid", "p", "", usage_db_export_pid)
	cmd.Flags().StringP("out", "o", "", usage_db_export_out)
	cmd.Flags().Int64("progress-every", 1000, usage_db_export_progress_every)
	return cmd
}
