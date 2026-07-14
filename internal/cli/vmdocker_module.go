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
