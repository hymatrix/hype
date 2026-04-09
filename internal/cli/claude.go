package cli

import (
	"errors"
	"fmt"
	"strings"

	goarSchema "github.com/permadao/goar/schema"
	"github.com/spf13/cobra"
)

const maxClaudeCodeFlagsChars = 16384

func newClaudeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Claude runtime workflow commands for vmdocker runtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, err := readOpenclawSharedFlags(cmd)
			if err != nil {
				return err
			}
			prompt, err := cmd.Flags().GetString("prompt")
			if err != nil {
				return err
			}
			pid, err := cmd.Flags().GetString("pid")
			if err != nil {
				return err
			}
			if strings.TrimSpace(prompt) == "" {
				return cmd.Help()
			}
			return runClaudeExec(shared, strings.TrimSpace(pid), prompt)
		},
	}

	cmd.PersistentFlags().StringP("node-url", "u", "http://127.0.0.1:8080", usage_openclaw_node_url)
	cmd.PersistentFlags().StringP("private-key", "k", "", usage_openclaw_private_key)
	cmd.PersistentFlags().Bool("json", false, usage_openclaw_json)

	cmd.Flags().StringP("prompt", "p", "", usage_claude_prompt)
	cmd.Flags().String("pid", "", usage_claude_pid_for_prompt)

	cmd.AddCommand(newClaudeSpawnCmd())
	cmd.AddCommand(newClaudeChatCmd())
	cmd.AddCommand(newClaudeExecCmd())
	return cmd
}

func newClaudeSpawnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn a Claude runtime process",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := hydrateOpenclawPrivateKeyFlag(cmd); err != nil {
				return err
			}
			for _, spec := range []struct {
				flag string
				envs []string
			}{
				{flag: "module-id", envs: []string{"VMDOCKER_MODULE_ID"}},
				{flag: "scheduler", envs: []string{"VMDOCKER_SCHEDULER"}},
				{flag: "runtime-backend", envs: []string{"RUNTIME_BACKEND"}},
				{flag: "api-key", envs: []string{"ANTHROPIC_API_KEY"}},
				{flag: "base-url", envs: []string{"ANTHROPIC_BASE_URL"}},
				{flag: "model", envs: []string{"ANTHROPIC_MODEL", "CLAUDE_MODEL"}},
				{flag: "code-flags", envs: []string{"CLAUDE_CODE_FLAGS"}},
			} {
				if err := hydrateFlagFromEnvs(cmd, spec.flag, spec.envs...); err != nil {
					return err
				}
			}
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "private-key", Prompt: "private-key (-k/--private-key) " + usage_openclaw_private_key + ": "},
				{Name: "module-id", Prompt: "module-id (-m/--module-id) " + usage_claude_module_id + ": "},
				{Name: "scheduler", Prompt: "scheduler (-s/--scheduler) " + usage_claude_scheduler + ": "},
				{Name: "api-key", Prompt: "api-key (--api-key) " + usage_claude_api_key + ": "},
				{Name: "base-url", Prompt: "base-url (--base-url) " + usage_claude_base_url + ": ", Optional: true},
				{Name: "model", Prompt: "model (--model) " + usage_claude_model + ": ", Optional: true},
				{Name: "code-flags", Prompt: "code-flags (--code-flags) " + usage_claude_code_flags + ": ", Optional: true},
				{Name: "runtime-backend", Prompt: "runtime-backend (--runtime-backend) " + usage_claude_runtime_backend + ": ", Optional: true},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, err := readOpenclawSharedFlags(cmd)
			if err != nil {
				return err
			}

			moduleID, _ := cmd.Flags().GetString("module-id")
			scheduler, _ := cmd.Flags().GetString("scheduler")
			apiKey, _ := cmd.Flags().GetString("api-key")
			baseURL, _ := cmd.Flags().GetString("base-url")
			model, _ := cmd.Flags().GetString("model")
			codeFlags, _ := cmd.Flags().GetString("code-flags")
			runtimeBackend, _ := cmd.Flags().GetString("runtime-backend")

			if strings.TrimSpace(moduleID) == "" {
				return errors.New("module-id is required")
			}
			if len(moduleID) > maxIDChars {
				return fmt.Errorf("module-id is too long (max %d)", maxIDChars)
			}
			if strings.TrimSpace(scheduler) == "" {
				return errors.New("scheduler is required")
			}
			if len(scheduler) > maxIDChars {
				return fmt.Errorf("scheduler is too long (max %d)", maxIDChars)
			}
			if strings.TrimSpace(apiKey) == "" {
				return errors.New("api-key is required")
			}
			if len(strings.TrimSpace(apiKey)) > maxAPIKeyChars {
				return fmt.Errorf("api-key is too long (max %d)", maxAPIKeyChars)
			}
			if len(strings.TrimSpace(model)) > maxModelChars {
				return fmt.Errorf("model is too long (max %d)", maxModelChars)
			}
			if len(strings.TrimSpace(codeFlags)) > maxClaudeCodeFlagsChars {
				return fmt.Errorf("code-flags is too long (max %d)", maxClaudeCodeFlagsChars)
			}
			if err := validateRuntimeBackend(runtimeBackend); err != nil {
				return err
			}

			sdkClient, err := newSDK(shared.nodeURL, shared.privateKey)
			if err != nil {
				return err
			}
			defer sdkClient.Close()

			tags := buildClaudeSpawnTags(apiKey, baseURL, model, codeFlags, runtimeBackend)
			res, err := sdkClient.SpawnAndWait(moduleID, scheduler, tags)
			if err != nil {
				return err
			}

			payload := map[string]interface{}{
				"action":      "spawn",
				"pid":         res.Id,
				"response_id": res.Id,
				"message":     res.Message,
			}
			return printOpenclawResult(shared.jsonOut, payload, fmt.Sprintf("spawn ok, pid: %s", res.Id))
		},
	}

	cmd.Flags().StringP("module-id", "m", "", usage_claude_module_id)
	cmd.Flags().StringP("scheduler", "s", "", usage_claude_scheduler)
	cmd.Flags().String("api-key", "", usage_claude_api_key)
	cmd.Flags().String("base-url", "", usage_claude_base_url)
	cmd.Flags().String("model", "", usage_claude_model)
	cmd.Flags().String("code-flags", "", usage_claude_code_flags)
	cmd.Flags().String("runtime-backend", "", usage_claude_runtime_backend)
	_ = cmd.MarkFlagRequired("module-id")
	_ = cmd.MarkFlagRequired("scheduler")
	_ = cmd.MarkFlagRequired("api-key")
	return cmd
}

func newClaudeChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Send a chat message to a Claude runtime process",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := hydrateOpenclawPrivateKeyFlag(cmd); err != nil {
				return err
			}
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "private-key", Prompt: "private-key (-k/--private-key) " + usage_openclaw_private_key + ": "},
				{Name: "pid", Prompt: "pid (-p/--pid) " + usage_openclaw_pid + ": "},
				{Name: "command", Prompt: "command (-c/--command) " + usage_claude_command + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, err := readOpenclawSharedFlags(cmd)
			if err != nil {
				return err
			}
			pid, _ := cmd.Flags().GetString("pid")
			command, _ := cmd.Flags().GetString("command")
			if strings.TrimSpace(pid) == "" {
				return errors.New("pid is required")
			}
			if strings.TrimSpace(command) == "" {
				return errors.New("command is required")
			}

			return runClaudeMessage(shared, "chat", "Chat", strings.TrimSpace(pid), command)
		},
	}

	cmd.Flags().StringP("pid", "p", "", usage_openclaw_pid)
	cmd.Flags().StringP("command", "c", "", usage_claude_command)
	return cmd
}

func newClaudeExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Send a generic Claude prompt to a running Claude process",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := hydrateOpenclawPrivateKeyFlag(cmd); err != nil {
				return err
			}
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "private-key", Prompt: "private-key (-k/--private-key) " + usage_openclaw_private_key + ": "},
				{Name: "pid", Prompt: "pid (--pid) " + usage_claude_pid_for_prompt + ": "},
				{Name: "prompt", Prompt: "prompt (-p/--prompt) " + usage_claude_prompt + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, err := readOpenclawSharedFlags(cmd)
			if err != nil {
				return err
			}
			pid, _ := cmd.Flags().GetString("pid")
			prompt, _ := cmd.Flags().GetString("prompt")
			return runClaudeExec(shared, strings.TrimSpace(pid), prompt)
		},
	}

	cmd.Flags().String("pid", "", usage_claude_pid_for_prompt)
	cmd.Flags().StringP("prompt", "p", "", usage_claude_prompt)
	_ = cmd.MarkFlagRequired("pid")
	_ = cmd.MarkFlagRequired("prompt")
	return cmd
}

func runClaudeExec(shared openclawSharedFlags, pid, prompt string) error {
	if strings.TrimSpace(pid) == "" {
		return errors.New("pid is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return errors.New("prompt is required")
	}
	return runClaudeMessage(shared, "exec", "Execute", pid, prompt)
}

func runClaudeMessage(shared openclawSharedFlags, outputAction, action, pid, command string) error {
	sdkClient, err := newSDK(shared.nodeURL, shared.privateKey)
	if err != nil {
		return err
	}
	defer sdkClient.Close()

	tags := []goarSchema.Tag{
		{Name: "Action", Value: action},
		{Name: "Command", Value: strings.TrimSpace(command)},
	}
	res, err := sdkClient.SendMessageAndWait(strings.TrimSpace(pid), "", tags)
	if err != nil {
		return err
	}
	reply := extractChatReply(res.Message)
	if strings.TrimSpace(reply) == "" {
		reply = res.Message
	}

	return printOpenclawResult(shared.jsonOut, map[string]interface{}{
		"action":      outputAction,
		"pid":         strings.TrimSpace(pid),
		"response_id": res.Id,
		"message":     res.Message,
		"reply":       reply,
	}, fmt.Sprintf("%s reply: %s", outputAction, reply))
}

func buildClaudeSpawnTags(apiKey, baseURL, model, codeFlags, runtimeBackend string) []goarSchema.Tag {
	tags := []goarSchema.Tag{
		{Name: containerEnvTagPrefix + "RUNTIME_TYPE", Value: "claude"},
		{Name: containerEnvTagPrefix + "ANTHROPIC_API_KEY", Value: strings.TrimSpace(apiKey)},
	}
	if value := strings.TrimSpace(baseURL); value != "" {
		tags = append(tags, goarSchema.Tag{Name: containerEnvTagPrefix + "ANTHROPIC_BASE_URL", Value: value})
	}
	if value := strings.TrimSpace(model); value != "" {
		tags = append(tags, goarSchema.Tag{Name: containerEnvTagPrefix + "ANTHROPIC_MODEL", Value: value})
	}
	if value := strings.TrimSpace(codeFlags); value != "" {
		tags = append(tags, goarSchema.Tag{Name: containerEnvTagPrefix + "CLAUDE_CODE_FLAGS", Value: value})
	}
	if value := strings.TrimSpace(runtimeBackend); value != "" {
		tags = append(tags, goarSchema.Tag{Name: "Runtime-Backend", Value: value})
	}
	return tags
}
