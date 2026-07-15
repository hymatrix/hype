package cli

import (
	vmdockerpkg "github.com/hymatrix/hype/internal/vmdocker"
	"github.com/spf13/cobra"
)

func newVmdockerModuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module",
		Short: "Manage VMDocker modules",
	}
	cmd.AddCommand(newVmdockerModuleBuildCmd())
	return cmd
}

func newVmdockerModuleBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a VMDocker module from a profile",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := hydrateVmdockerModuleBuildFlags(cmd); err != nil {
				return err
			}
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "profile", Prompt: "profile (--profile) " + usage_vmdocker_profile + ": "},
				{Name: "agent-bin", Prompt: "agent-bin (--agent-bin) " + usage_vmdocker_agent_bin + ": "},
				{Name: "private-key", Prompt: "private-key (-k/--private-key) " + usage_vmdocker_private_key + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			profile, _ := cmd.Flags().GetString("profile")
			agentBin, _ := cmd.Flags().GetString("agent-bin")
			nodeURL, _ := cmd.Flags().GetString("node-url")
			privateKey, _ := cmd.Flags().GetString("private-key")

			manager := vmdockerpkg.NewManager()
			manager.SetOutput(cmd.OutOrStdout())
			manager.SetErrorOutput(cmd.ErrOrStderr())
			return manager.BuildModule(commandContext(cmd), vmdockerpkg.ModuleBuildOptions{
				CheckoutDir:  dir,
				ProfilePath:  profile,
				AgentBinPath: agentBin,
				NodeURL:      nodeURL,
				PrivateKey:   privateKey,
			})
		},
	}
	cmd.Flags().String("dir", "./vmdockerv2", usage_vmdocker_checkout_dir)
	cmd.Flags().String("profile", "", usage_vmdocker_profile)
	cmd.Flags().String("agent-bin", "", usage_vmdocker_agent_bin)
	cmd.Flags().String("node-url", "", usage_vmdocker_node_url)
	cmd.Flags().String("private-key", "", usage_vmdocker_private_key)
	_ = cmd.MarkFlagRequired("profile")
	return cmd
}

func hydrateVmdockerModuleBuildFlags(cmd *cobra.Command) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	profile, err := cmd.Flags().GetString("profile")
	if err != nil {
		return err
	}
	agentBin, err := cmd.Flags().GetString("agent-bin")
	if err != nil {
		return err
	}
	nodeURL, err := cmd.Flags().GetString("node-url")
	if err != nil {
		return err
	}
	privateKey, err := cmd.Flags().GetString("private-key")
	if err != nil {
		return err
	}

	resolved, err := vmdockerpkg.NewManager().ResolveModuleBuildOptions(vmdockerpkg.ModuleBuildOptions{
		CheckoutDir:  dir,
		ProfilePath:  profile,
		AgentBinPath: agentBin,
		NodeURL:      nodeURL,
		PrivateKey:   privateKey,
	})
	if err != nil {
		return err
	}
	for flag, value := range map[string]string{
		"agent-bin":   resolved.AgentBinPath,
		"node-url":    resolved.NodeURL,
		"private-key": resolved.PrivateKey,
	} {
		if value == "" {
			continue
		}
		if err := cmd.Flags().Set(flag, value); err != nil {
			return err
		}
	}
	return nil
}
