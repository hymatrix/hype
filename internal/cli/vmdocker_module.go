package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

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
	for _, item := range []struct {
		flag string
		envs []string
	}{
		{flag: "agent-bin", envs: []string{"VMDOCKER_AGENT_BIN"}},
		{flag: "node-url", envs: []string{"VMDOCKER_URL"}},
		{flag: "private-key", envs: []string{"VMDOCKER_PRIVATE_KEY", "HYPE_PRIVATE_KEY", "PRV_KEY"}},
	} {
		if err := hydrateFlagFromEnvs(cmd, item.flag, item.envs...); err != nil {
			return err
		}
	}

	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	envFile := strings.TrimSpace(os.Getenv("VMDOCKER_ENV_FILE"))
	if envFile == "" {
		envFile = filepath.Join(dir, ".env")
	} else if !filepath.IsAbs(envFile) {
		envFile = filepath.Join(dir, envFile)
	}
	values, err := vmdockerpkg.ParseEnvFile(envFile, os.ReadFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for flag, envKey := range map[string]string{
		"agent-bin":   "VMDOCKER_AGENT_BIN",
		"node-url":    "VMDOCKER_URL",
		"private-key": "VMDOCKER_PRIVATE_KEY",
	} {
		current, err := cmd.Flags().GetString(flag)
		if err != nil {
			return err
		}
		if strings.TrimSpace(current) != "" || strings.TrimSpace(values[envKey]) == "" {
			continue
		}
		if err := cmd.Flags().Set(flag, values[envKey]); err != nil {
			return err
		}
	}
	return nil
}
