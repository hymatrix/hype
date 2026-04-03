package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hymatrix/hype/internal/openclawui"
	"github.com/spf13/cobra"
)

func newOpenclawUICmd() *cobra.Command {
	var listen string
	var timeoutMS int

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Start the embedded Openclaw Web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			if timeoutMS <= 0 {
				return fmt.Errorf("timeout-ms must be greater than 0")
			}

			binaryPath, err := os.Executable()
			if err != nil {
				return err
			}
			workingDir, err := defaultOpenclawUIWorkingDir()
			if err != nil {
				return err
			}

			cfg, err := openclawui.ResolveConfig(openclawui.Config{
				Listen:     listen,
				BinaryPath: binaryPath,
				WorkingDir: workingDir,
				Timeout:    time.Duration(timeoutMS) * time.Millisecond,
			})
			if err != nil {
				return err
			}

			baseCtx := cmd.Context()
			if baseCtx == nil {
				baseCtx = context.Background()
			}
			ctx, stop := signal.NotifyContext(baseCtx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			fmt.Fprintf(cmd.OutOrStdout(), "Openclaw UI listening on http://%s\n", cfg.Listen)
			fmt.Fprintf(cmd.OutOrStdout(), "Using hype binary: %s\n", cfg.BinaryPath)
			return openclawui.RunContext(ctx, cfg)
		},
	}

	cmd.Flags().StringVar(&listen, "listen", defaultOpenclawUIListen(), usage_openclaw_ui_listen)
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", defaultOpenclawUITimeoutMS(), usage_openclaw_ui_timeout_ms)
	return cmd
}

func defaultOpenclawUIListen() string {
	if v := strings.TrimSpace(os.Getenv("OPENCLAW_WEBUI_LISTEN")); v != "" {
		return v
	}
	return "127.0.0.1:7788"
}

func defaultOpenclawUITimeoutMS() int {
	v := strings.TrimSpace(os.Getenv("OPENCLAW_WEBUI_TIMEOUT_MS"))
	if v == "" {
		return 90000
	}
	timeoutMS, err := strconv.Atoi(v)
	if err != nil || timeoutMS <= 0 {
		return 90000
	}
	return timeoutMS
}

func defaultOpenclawUIWorkingDir() (string, error) {
	return os.Getwd()
}
