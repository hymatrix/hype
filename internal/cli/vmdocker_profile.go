package cli

import (
	"fmt"
	"path/filepath"

	vmdockerpkg "github.com/hymatrix/hype/internal/vmdocker"
	"github.com/spf13/cobra"
)

func newVmdockerProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage VMDocker profile scaffolds",
	}
	cmd.AddCommand(newVmdockerProfileInitCmd())
	return cmd
}

func newVmdockerProfileInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a VMDocker profile scaffold",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "dir", Prompt: "dir (--dir) " + usage_vmdocker_profile_dir + ": "},
				{Name: "from", Prompt: "from (--from) " + usage_vmdocker_profile_from + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			from, _ := cmd.Flags().GetString("from")
			absDir, err := vmdockerpkg.NewManager().InitProfile(dir, from)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "profile directory: %s\n", absDir)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "profile file: %s\n", filepath.Join(absDir, "profile.toml"))
			return nil
		},
	}
	cmd.Flags().String("dir", "", usage_vmdocker_profile_dir)
	cmd.Flags().String("from", "", usage_vmdocker_profile_from)
	_ = cmd.MarkFlagRequired("dir")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}
