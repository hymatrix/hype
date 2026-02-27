package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/hymatrix/hype/internal/syncer"
	"github.com/hymatrix/hype/internal/syncer/schema"
	"github.com/spf13/cobra"
)

var exportJsonlCmd = &cobra.Command{
	Use:   "db-export",
	Short: "Export process data from redis into jsonl",
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
