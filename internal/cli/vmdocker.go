package cli

import (
	"context"
	"errors"
	"path/filepath"

	vmdockerpkg "github.com/hymatrix/hype/internal/vmdocker"
	"github.com/spf13/cobra"
)

func newVmdockerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vmdocker",
		Short: "Manage local VMDocker runtime workflows",
	}

	cmd.AddCommand(newVmdockerGetCmd())
	cmd.AddCommand(newVmdockerInitCmd())
	cmd.AddCommand(newVmdockerProfileCmd())
	return cmd
}

func newVmdockerGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Fetch a VMDocker ref and build hymx-node",
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := cmd.Flags().GetString("ref")
			if err != nil {
				return err
			}
			dir, err := cmd.Flags().GetString("dir")
			if err != nil {
				return err
			}

			manager := vmdockerpkg.NewManager()
			ctx := commandContext(cmd)

			resolvedRef, binaryPath, err := manager.Get(ctx, dir, ref)
			if err != nil {
				return err
			}

			absDir := filepath.Dir(filepath.Dir(binaryPath))
			out := cmd.OutOrStdout()
			_, _ = out.Write([]byte("vmdocker directory: " + absDir + "\n"))
			_, _ = out.Write([]byte("vmdocker ref: " + resolvedRef + "\n"))
			_, _ = out.Write([]byte("vmdocker binary: " + binaryPath + "\n"))
			return nil
		},
	}

	cmd.Flags().String("ref", vmdockerpkg.DefaultRef, usage_vmdocker_ref)
	cmd.Flags().String("dir", "./vmdockerv2", usage_vmdocker_dir)
	return cmd
}

func newVmdockerInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Start local VMDocker services and run examples init",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "env-file", Prompt: "env-file (--env-file) " + usage_vmdocker_env_file + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := cmd.Flags().GetString("dir")
			if err != nil {
				return err
			}
			envFile, err := cmd.Flags().GetString("env-file")
			if err != nil {
				return err
			}
			if envFile == "" {
				return errors.New("env-file is required")
			}

			manager := vmdockerpkg.NewManager()
			manager.SetOutput(cmd.OutOrStdout())
			ctx := commandContext(cmd)
			return manager.Init(ctx, dir, envFile)
		},
	}

	cmd.Flags().String("dir", "./vmdockerv2", usage_vmdocker_dir)
	cmd.Flags().String("env-file", "", usage_vmdocker_env_file)
	return cmd
}

func commandContext(cmd *cobra.Command) context.Context {
	if cmd.Context() != nil {
		return cmd.Context()
	}
	return context.Background()
}
