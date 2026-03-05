package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/everFinance/goether"
	"github.com/hymatrix/hymx/sdk"
	hymxVmmSchema "github.com/hymatrix/hymx/vmm/schema"
	"github.com/permadao/goar"
	goarSchema "github.com/permadao/goar/schema"
	"github.com/spf13/cobra"
)

const (
	containerEnvTagPrefix  = "Container-Env-"
	defaultOpenclawTimeout = "180000"
	minOpenclawTimeoutMs   = 1000
	maxOpenclawTimeoutMs   = 3600000
	maxReplyOutputChars    = 4000
	maxIDChars             = 256
	maxModelChars          = 128
	maxTokenChars          = 8192
	maxAPIKeyChars         = 4096
)

type openclawSharedFlags struct {
	nodeURL    string
	privateKey string
	jsonOut    bool
}

func newOpenclawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openclaw",
		Short: "Openclaw workflow commands for vmdocker runtime",
	}

	cmd.PersistentFlags().StringP("node-url", "u", "http://127.0.0.1:8080", usage_openclaw_node_url)
	cmd.PersistentFlags().StringP("private-key", "k", "", usage_openclaw_private_key)
	cmd.PersistentFlags().Bool("json", false, usage_openclaw_json)

	cmd.AddCommand(newOpenclawSpawnCmd())
	cmd.AddCommand(newOpenclawConfTgCmd())
	cmd.AddCommand(newOpenclawPairTgCmd())
	cmd.AddCommand(newOpenclawChatCmd())
	return cmd
}

func newOpenclawSpawnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn an openclaw process",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := hydrateOpenclawPrivateKeyFlag(cmd); err != nil {
				return err
			}
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "private-key", Prompt: "private-key (-k/--private-key) " + usage_openclaw_private_key + ": "},
				{Name: "module-id", Prompt: "module-id (-m/--module-id) " + usage_openclaw_module_id + ": "},
				{Name: "scheduler", Prompt: "scheduler (-s/--scheduler) " + usage_openclaw_scheduler + ": "},
				{Name: "model", Prompt: "model (--model) " + usage_openclaw_model + ": "},
				{Name: "timeout-ms", Prompt: "timeout-ms (--timeout-ms) " + usage_openclaw_timeout_ms + " (default " + defaultOpenclawTimeout + "): ", Optional: true},
				{Name: "api-key", Prompt: "api-key (--api-key) " + usage_openclaw_api_key + ": "},
				{Name: "gateway-token", Prompt: "gateway-token (--gateway-token) " + usage_openclaw_gateway_token + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, err := readOpenclawSharedFlags(cmd)
			if err != nil {
				return err
			}

			moduleID, err := cmd.Flags().GetString("module-id")
			if err != nil {
				return err
			}
			scheduler, err := cmd.Flags().GetString("scheduler")
			if err != nil {
				return err
			}
			model, err := cmd.Flags().GetString("model")
			if err != nil {
				return err
			}
			timeoutMs, err := cmd.Flags().GetString("timeout-ms")
			if err != nil {
				return err
			}
			apiKey, err := cmd.Flags().GetString("api-key")
			if err != nil {
				return err
			}
			gatewayToken, err := cmd.Flags().GetString("gateway-token")
			if err != nil {
				return err
			}
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
			if strings.TrimSpace(model) == "" {
				return errors.New("model is required")
			}
			if len(model) > maxModelChars {
				return fmt.Errorf("model is too long (max %d)", maxModelChars)
			}
			if strings.TrimSpace(timeoutMs) == "" {
				timeoutMs = defaultOpenclawTimeout
			}
			timeout, err := strconv.Atoi(timeoutMs)
			if err != nil {
				return fmt.Errorf("timeout-ms must be integer milliseconds: %w", err)
			}
			if timeout < minOpenclawTimeoutMs || timeout > maxOpenclawTimeoutMs {
				return fmt.Errorf("timeout-ms out of range [%d, %d]", minOpenclawTimeoutMs, maxOpenclawTimeoutMs)
			}
			if strings.TrimSpace(apiKey) == "" {
				return errors.New("api-key is required")
			}
			if len(apiKey) > maxAPIKeyChars {
				return fmt.Errorf("api-key is too long (max %d)", maxAPIKeyChars)
			}
			if strings.TrimSpace(gatewayToken) == "" {
				return errors.New("gateway-token is required")
			}
			if len(gatewayToken) > maxTokenChars {
				return fmt.Errorf("gateway-token is too long (max %d)", maxTokenChars)
			}

			sdkClient, err := newSDK(shared.nodeURL, shared.privateKey)
			if err != nil {
				return err
			}
			defer sdkClient.Close()

			tags := []goarSchema.Tag{{Name: "model", Value: model}}
			if strings.TrimSpace(apiKey) != "" {
				tags = append(tags, goarSchema.Tag{Name: "apiKey", Value: apiKey})
			}
			if strings.TrimSpace(gatewayToken) != "" {
				tags = append(tags, goarSchema.Tag{Name: containerEnvTagPrefix + "OPENCLAW_GATEWAY_TOKEN", Value: gatewayToken})
			}
			tags = append(tags, goarSchema.Tag{Name: containerEnvTagPrefix + "OPENCLAW_TIMEOUT_MS", Value: timeoutMs})

			res, err := sdkClient.SpawnAndWait(moduleID, scheduler, tags)
			if err != nil {
				return err
			}

			return printOpenclawResult(shared.jsonOut, map[string]interface{}{
				"action":      "spawn",
				"pid":         res.Id,
				"response_id": res.Id,
				"message":     res.Message,
			}, fmt.Sprintf("spawn ok, pid: %s", res.Id))
		},
	}

	cmd.Flags().StringP("module-id", "m", "", usage_openclaw_module_id)
	cmd.Flags().StringP("scheduler", "s", "", usage_openclaw_scheduler)
	cmd.Flags().String("model", "", usage_openclaw_model)
	cmd.Flags().String("timeout-ms", defaultOpenclawTimeout, usage_openclaw_timeout_ms)
	cmd.Flags().String("api-key", "", usage_openclaw_api_key)
	cmd.Flags().String("gateway-token", "", usage_openclaw_gateway_token)

	_ = cmd.MarkFlagRequired("module-id")
	_ = cmd.MarkFlagRequired("scheduler")
	_ = cmd.MarkFlagRequired("model")
	_ = cmd.MarkFlagRequired("api-key")
	_ = cmd.MarkFlagRequired("gateway-token")
	return cmd
}

func newOpenclawConfTgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conf-tg",
		Short: "Configure Telegram settings for an openclaw process",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := hydrateOpenclawPrivateKeyFlag(cmd); err != nil {
				return err
			}
			if err := replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "private-key", Prompt: "private-key (-k/--private-key) " + usage_openclaw_private_key + ": "},
				{Name: "pid", Prompt: "pid (-p/--pid) " + usage_openclaw_pid + ": "},
				{Name: "bot-token", Prompt: "bot-token (--bot-token) " + usage_openclaw_bot_token + ": "},
			}); err != nil {
				return err
			}
			if err := replPromptStringFlagWithDefault(cmd, "default-account (--default-account)", "default-account", "main"); err != nil {
				return err
			}
			if err := replPromptStringFlagWithDefault(cmd, "dm-policy (--dm-policy)", "dm-policy", "pairing"); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, err := readOpenclawSharedFlags(cmd)
			if err != nil {
				return err
			}
			pid, err := cmd.Flags().GetString("pid")
			if err != nil {
				return err
			}
			if strings.TrimSpace(pid) == "" {
				return errors.New("pid is required")
			}

			botToken, _ := cmd.Flags().GetString("bot-token")
			defaultAccount, _ := cmd.Flags().GetString("default-account")
			dmPolicy, _ := cmd.Flags().GetString("dm-policy")
			allowFrom, _ := cmd.Flags().GetString("allow-from")

			if strings.TrimSpace(botToken) == "" {
				return errors.New("bot-token is required")
			}

			tags := []goarSchema.Tag{{Name: "Action", Value: "ConfigureTelegram"}}
			patchCount := 0
			if strings.TrimSpace(botToken) != "" {
				tags = append(tags, goarSchema.Tag{Name: "botToken", Value: botToken})
				patchCount++
			}
			if strings.TrimSpace(defaultAccount) != "" {
				tags = append(tags, goarSchema.Tag{Name: "defaultAccount", Value: defaultAccount})
				patchCount++
			}
			if strings.TrimSpace(dmPolicy) != "" {
				tags = append(tags, goarSchema.Tag{Name: "dmPolicy", Value: dmPolicy})
				patchCount++
			}
			if strings.TrimSpace(allowFrom) != "" {
				tags = append(tags, goarSchema.Tag{Name: "allowFrom", Value: allowFrom})
				patchCount++
			}
			if patchCount == 0 {
				return errors.New("at least one telegram patch field is required")
			}

			sdkClient, err := newSDK(shared.nodeURL, shared.privateKey)
			if err != nil {
				return err
			}
			defer sdkClient.Close()

			res, err := sdkClient.SendMessageAndWait(pid, "", tags)
			if err != nil {
				return err
			}

			return printOpenclawResult(shared.jsonOut, map[string]interface{}{
				"action":      "conf-tg",
				"pid":         pid,
				"response_id": res.Id,
				"message":     res.Message,
			}, fmt.Sprintf("conf-tg ok, pid: %s", pid))
		},
	}

	cmd.Flags().StringP("pid", "p", "", usage_openclaw_pid)
	cmd.Flags().String("bot-token", "", usage_openclaw_bot_token)
	cmd.Flags().String("default-account", "main", usage_openclaw_default_account)
	cmd.Flags().String("dm-policy", "pairing", usage_openclaw_dm_policy)
	cmd.Flags().String("allow-from", "", usage_openclaw_allow_from)
	_ = cmd.MarkFlagRequired("bot-token")
	return cmd
}

func newOpenclawPairTgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pair-tg",
		Short: "Approve Telegram pairing for an openclaw process",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := hydrateOpenclawPrivateKeyFlag(cmd); err != nil {
				return err
			}
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "private-key", Prompt: "private-key (-k/--private-key) " + usage_openclaw_private_key + ": "},
				{Name: "pid", Prompt: "pid (-p/--pid) " + usage_openclaw_pid + ": "},
				{Name: "code", Prompt: "code (-c/--code) " + usage_openclaw_code + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, err := readOpenclawSharedFlags(cmd)
			if err != nil {
				return err
			}
			pid, err := cmd.Flags().GetString("pid")
			if err != nil {
				return err
			}
			code, err := cmd.Flags().GetString("code")
			if err != nil {
				return err
			}
			channel, err := cmd.Flags().GetString("channel")
			if err != nil {
				return err
			}
			dmPolicy, err := cmd.Flags().GetString("dm-policy")
			if err != nil {
				return err
			}

			if strings.TrimSpace(pid) == "" {
				return errors.New("pid is required")
			}
			if strings.TrimSpace(code) == "" {
				return errors.New("code is required")
			}

			sdkClient, err := newSDK(shared.nodeURL, shared.privateKey)
			if err != nil {
				return err
			}
			defer sdkClient.Close()

			tags := []goarSchema.Tag{
				{Name: "Action", Value: "ApproveTelegramPairing"},
				{Name: "code", Value: code},
				{Name: "channel", Value: channel},
				{Name: "dmPolicy", Value: dmPolicy},
			}

			res, err := sdkClient.SendMessageAndWait(pid, "", tags)
			if err != nil {
				return err
			}
			return printOpenclawResult(shared.jsonOut, map[string]interface{}{
				"action":      "pair-tg",
				"pid":         pid,
				"response_id": res.Id,
				"message":     res.Message,
			}, fmt.Sprintf("pair-tg ok, pid: %s", pid))
		},
	}

	cmd.Flags().StringP("pid", "p", "", usage_openclaw_pid)
	cmd.Flags().StringP("code", "c", "", usage_openclaw_code)
	cmd.Flags().String("channel", "telegram", usage_openclaw_channel)
	cmd.Flags().String("dm-policy", "pairing", usage_openclaw_dm_policy)
	return cmd
}

func newOpenclawChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Send a chat command to an openclaw process",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := hydrateOpenclawPrivateKeyFlag(cmd); err != nil {
				return err
			}
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "private-key", Prompt: "private-key (-k/--private-key) " + usage_openclaw_private_key + ": "},
				{Name: "pid", Prompt: "pid (-p/--pid) " + usage_openclaw_pid + ": "},
				{Name: "command", Prompt: "command (-c/--command) " + usage_openclaw_command + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, err := readOpenclawSharedFlags(cmd)
			if err != nil {
				return err
			}
			pid, err := cmd.Flags().GetString("pid")
			if err != nil {
				return err
			}
			command, err := cmd.Flags().GetString("command")
			if err != nil {
				return err
			}
			if strings.TrimSpace(pid) == "" {
				return errors.New("pid is required")
			}
			if strings.TrimSpace(command) == "" {
				return errors.New("command is required")
			}

			sdkClient, err := newSDK(shared.nodeURL, shared.privateKey)
			if err != nil {
				return err
			}
			defer sdkClient.Close()

			tags := []goarSchema.Tag{
				{Name: "Action", Value: "Chat"},
				{Name: "Command", Value: command},
			}
			res, err := sdkClient.SendMessageAndWait(pid, "", tags)
			if err != nil {
				return err
			}
			reply := extractChatReply(res.Message)
			if strings.TrimSpace(reply) == "" {
				reply = res.Message
			}

			return printOpenclawResult(shared.jsonOut, map[string]interface{}{
				"action":      "chat",
				"pid":         pid,
				"response_id": res.Id,
				"message":     res.Message,
				"reply":       reply,
			}, fmt.Sprintf("chat reply: %s", reply))
		},
	}

	cmd.Flags().StringP("pid", "p", "", usage_openclaw_pid)
	cmd.Flags().StringP("command", "c", "", usage_openclaw_command)
	return cmd
}

func readOpenclawSharedFlags(cmd *cobra.Command) (openclawSharedFlags, error) {
	nodeURL, err := cmd.Flags().GetString("node-url")
	if err != nil {
		return openclawSharedFlags{}, err
	}
	privateKey, err := cmd.Flags().GetString("private-key")
	if err != nil {
		return openclawSharedFlags{}, err
	}
	jsonOut, err := cmd.Flags().GetBool("json")
	if err != nil {
		return openclawSharedFlags{}, err
	}

	if strings.TrimSpace(privateKey) == "" {
		privateKey = strings.TrimSpace(os.Getenv("HYPE_PRIVATE_KEY"))
	}
	if strings.TrimSpace(privateKey) == "" {
		privateKey = strings.TrimSpace(os.Getenv("PRV_KEY"))
	}
	if strings.TrimSpace(privateKey) == "" {
		return openclawSharedFlags{}, errors.New("private-key is required")
	}

	return openclawSharedFlags{
		nodeURL:    nodeURL,
		privateKey: privateKey,
		jsonOut:    jsonOut,
	}, nil
}

func hydrateOpenclawPrivateKeyFlag(cmd *cobra.Command) error {
	v, err := cmd.Flags().GetString("private-key")
	if err != nil {
		return err
	}
	if strings.TrimSpace(v) != "" {
		return nil
	}
	if env := strings.TrimSpace(os.Getenv("HYPE_PRIVATE_KEY")); env != "" {
		return cmd.Flags().Set("private-key", env)
	}
	if env := strings.TrimSpace(os.Getenv("PRV_KEY")); env != "" {
		return cmd.Flags().Set("private-key", env)
	}
	return nil
}

func replPromptStringFlagWithDefault(cmd *cobra.Command, label, promptKey, defaultValue string) error {
	if !isReplMode(cmd) {
		return nil
	}
	flag := cmd.Flags().Lookup(promptKey)
	if flag == nil {
		return nil
	}
	if flag.Changed {
		return nil
	}

	in, _ := cmd.Context().Value(replInKey{}).(*bufio.Reader)
	out, _ := cmd.Context().Value(replOutKey{}).(io.Writer)
	if in == nil || out == nil {
		return nil
	}

	value, err := replReadLine(in, out, fmt.Sprintf("%s (default %s): ", label, defaultValue))
	if err != nil {
		return err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultValue
	}
	return cmd.Flags().Set(promptKey, value)
}

func newSDK(nodeURL, privateKey string) (*sdk.SDK, error) {
	signer, err := goether.NewSigner(privateKey)
	if err != nil {
		return nil, fmt.Errorf("newSDK: init signer failed: %w", err)
	}
	bundler, err := goar.NewBundler(signer)
	if err != nil {
		return nil, fmt.Errorf("newSDK: init bundler failed: %w", err)
	}
	return sdk.NewFromBundler(nodeURL, bundler), nil
}

func printOpenclawResult(jsonOut bool, payload map[string]interface{}, fallback string) error {
	if !jsonOut {
		fmt.Println(fallback)
		return nil
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func extractChatReply(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var result hymxVmmSchema.VmmResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return ""
	}

	if v, ok := result.Output.(string); ok && strings.TrimSpace(v) != "" {
		return limitText(strings.TrimSpace(v), maxReplyOutputChars)
	}
	if strings.TrimSpace(result.Data) != "" {
		return limitText(strings.TrimSpace(result.Data), maxReplyOutputChars)
	}
	for _, msg := range result.Messages {
		if msg == nil {
			continue
		}
		for _, t := range msg.Tags {
			if strings.EqualFold(strings.TrimSpace(t.Name), "Reply") && strings.TrimSpace(t.Value) != "" {
				return limitText(strings.TrimSpace(t.Value), maxReplyOutputChars)
			}
		}
		if strings.TrimSpace(msg.Data) != "" {
			return limitText(strings.TrimSpace(msg.Data), maxReplyOutputChars)
		}
	}
	return ""
}

func limitText(s string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxChars {
		return s
	}
	return string(r[:maxChars]) + "...(truncated)"
}
