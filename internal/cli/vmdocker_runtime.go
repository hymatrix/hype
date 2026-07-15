package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/hymatrix/hymx/server/schema"
	vmmSchema "github.com/hymatrix/hymx/vmm/schema"
	goarSchema "github.com/permadao/goar/schema"
	"github.com/spf13/cobra"
)

type vmdockerRuntimeClient interface {
	SpawnAndWait(module, scheduler string, tags []goarSchema.Tag) (*schema.Response, error)
	SendMessageAndWait(target, data string, tags []goarSchema.Tag) (*schema.Response, error)
	Close()
}

type vmdockerRuntimeClientFactory func(nodeURL, privateKey string) (vmdockerRuntimeClient, error)

type vmdockerSpawnOptions struct {
	nodeURL        string
	privateKey     string
	moduleID       string
	scheduler      string
	runtimeType    string
	runtimeBackend string
	env            []string
	jsonOut        bool
}

type vmdockerExportOptions struct {
	nodeURL    string
	privateKey string
	pid        string
	jsonOut    bool
}

var vmdockerEnvKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func newVmdockerRuntimeClient(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
	return newSDK(nodeURL, privateKey)
}

func newVmdockerSpawnCmd() *cobra.Command {
	return newVmdockerSpawnCmdWithClient(newVmdockerRuntimeClient)
}

func newVmdockerExportCmd() *cobra.Command {
	return newVmdockerExportCmdWithClient(newVmdockerRuntimeClient)
}

func newVmdockerSpawnCmdWithClient(factory vmdockerRuntimeClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn a VMDocker process",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			for _, item := range []struct {
				flag string
				envs []string
			}{
				{flag: "module-id", envs: []string{"VMDOCKER_MODULE_ID"}},
				{flag: "scheduler", envs: []string{"VMDOCKER_SCHEDULER"}},
				{flag: "runtime-type", envs: []string{"RUNTIME_TYPE"}},
				{flag: "runtime-backend", envs: []string{"RUNTIME_BACKEND"}},
				{flag: "node-url", envs: []string{"VMDOCKER_URL"}},
				{flag: "private-key", envs: []string{"VMDOCKER_PRIVATE_KEY", "HYPE_PRIVATE_KEY", "PRV_KEY"}},
			} {
				if err := hydrateFlagFromEnvs(cmd, item.flag, item.envs...); err != nil {
					return err
				}
			}
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "module-id", Prompt: "module-id (-m/--module-id) " + usage_vmdocker_module_id + ": "},
				{Name: "scheduler", Prompt: "scheduler (-s/--scheduler) " + usage_vmdocker_scheduler + ": "},
				{Name: "private-key", Prompt: "private-key (-k/--private-key) " + usage_vmdocker_private_key + ": "},
				{Name: "runtime-type", Prompt: "runtime-type (--runtime-type) " + usage_vmdocker_runtime_type + ": ", Optional: true},
				{Name: "runtime-backend", Prompt: "runtime-backend (--runtime-backend) " + usage_vmdocker_runtime_backend + ": ", Optional: true},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleID, _ := cmd.Flags().GetString("module-id")
			scheduler, _ := cmd.Flags().GetString("scheduler")
			runtimeType, _ := cmd.Flags().GetString("runtime-type")
			runtimeBackend, _ := cmd.Flags().GetString("runtime-backend")
			env, _ := cmd.Flags().GetStringArray("env")
			nodeURL, _ := cmd.Flags().GetString("node-url")
			if strings.TrimSpace(nodeURL) == "" {
				nodeURL = "http://127.0.0.1:8080"
			}
			privateKey, _ := cmd.Flags().GetString("private-key")
			jsonOut, _ := cmd.Flags().GetBool("json")
			return runVmdockerSpawn(factory, cmd.OutOrStdout(), vmdockerSpawnOptions{
				nodeURL:        nodeURL,
				privateKey:     privateKey,
				moduleID:       moduleID,
				scheduler:      scheduler,
				runtimeType:    runtimeType,
				runtimeBackend: runtimeBackend,
				env:            env,
				jsonOut:        jsonOut,
			})
		},
	}
	cmd.Flags().StringP("module-id", "m", "", usage_vmdocker_module_id)
	cmd.Flags().StringP("scheduler", "s", "", usage_vmdocker_scheduler)
	cmd.Flags().String("runtime-type", "", usage_vmdocker_runtime_type)
	cmd.Flags().String("runtime-backend", "", usage_vmdocker_runtime_backend)
	cmd.Flags().StringArray("env", nil, usage_vmdocker_env)
	cmd.Flags().StringP("node-url", "u", "", usage_vmdocker_node_url)
	cmd.Flags().StringP("private-key", "k", "", usage_vmdocker_private_key)
	cmd.Flags().Bool("json", false, usage_vmdocker_json)
	return cmd
}

func buildVmdockerSpawnTags(runtimeType, backend string, assignments []string) ([]goarSchema.Tag, error) {
	if err := validateRuntimeBackend(backend); err != nil {
		return nil, err
	}
	tags := make([]goarSchema.Tag, 0, len(assignments)+2)
	if runtimeType = strings.TrimSpace(runtimeType); runtimeType != "" {
		tags = append(tags, goarSchema.Tag{Name: containerEnvTagPrefix + "RUNTIME_TYPE", Value: runtimeType})
	}
	seen := make(map[string]struct{}, len(assignments))
	for _, assignment := range assignments {
		key, value, ok := strings.Cut(assignment, "=")
		key = strings.TrimSpace(key)
		if !ok || !vmdockerEnvKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid environment key in %q", assignment)
		}
		if key == "RUNTIME_TYPE" {
			return nil, errors.New("RUNTIME_TYPE is reserved; use --runtime-type")
		}
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate environment key: %s", key)
		}
		seen[key] = struct{}{}
		tags = append(tags, goarSchema.Tag{Name: containerEnvTagPrefix + key, Value: value})
	}
	if backend = strings.TrimSpace(backend); backend != "" {
		tags = append(tags, goarSchema.Tag{Name: "Runtime-Backend", Value: backend})
	}
	return tags, nil
}

func newVmdockerExportCmdWithClient(factory vmdockerRuntimeClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a VMDocker process as a module",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			for _, item := range []struct {
				flag string
				envs []string
			}{
				{flag: "pid", envs: []string{"VMDOCKER_EXPORT_PID"}},
				{flag: "node-url", envs: []string{"VMDOCKER_URL"}},
				{flag: "private-key", envs: []string{"VMDOCKER_PRIVATE_KEY", "HYPE_PRIVATE_KEY", "PRV_KEY"}},
			} {
				if err := hydrateFlagFromEnvs(cmd, item.flag, item.envs...); err != nil {
					return err
				}
			}
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "pid", Prompt: "pid (-p/--pid) " + usage_vmdocker_pid + ": "},
				{Name: "private-key", Prompt: "private-key (-k/--private-key) " + usage_vmdocker_private_key + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, _ := cmd.Flags().GetString("pid")
			nodeURL, _ := cmd.Flags().GetString("node-url")
			if strings.TrimSpace(nodeURL) == "" {
				nodeURL = "http://127.0.0.1:8080"
			}
			privateKey, _ := cmd.Flags().GetString("private-key")
			jsonOut, _ := cmd.Flags().GetBool("json")
			return runVmdockerExport(factory, cmd.OutOrStdout(), vmdockerExportOptions{
				nodeURL:    nodeURL,
				privateKey: privateKey,
				pid:        pid,
				jsonOut:    jsonOut,
			})
		},
	}
	cmd.Flags().StringP("pid", "p", "", usage_vmdocker_pid)
	cmd.Flags().StringP("node-url", "u", "", usage_vmdocker_node_url)
	cmd.Flags().StringP("private-key", "k", "", usage_vmdocker_private_key)
	cmd.Flags().Bool("json", false, usage_vmdocker_json)
	return cmd
}

func runVmdockerSpawn(factory vmdockerRuntimeClientFactory, out io.Writer, opts vmdockerSpawnOptions) error {
	if strings.TrimSpace(opts.moduleID) == "" {
		return errors.New("module-id is required")
	}
	if len(opts.moduleID) > maxIDChars {
		return fmt.Errorf("module-id is too long (max %d)", maxIDChars)
	}
	if strings.TrimSpace(opts.scheduler) == "" {
		return errors.New("scheduler is required")
	}
	if len(opts.scheduler) > maxIDChars {
		return fmt.Errorf("scheduler is too long (max %d)", maxIDChars)
	}
	if strings.TrimSpace(opts.privateKey) == "" {
		return errors.New("private-key is required")
	}
	tags, err := buildVmdockerSpawnTags(opts.runtimeType, opts.runtimeBackend, opts.env)
	if err != nil {
		return err
	}
	client, err := factory(opts.nodeURL, opts.privateKey)
	if err != nil {
		return err
	}
	defer client.Close()
	res, err := client.SpawnAndWait(opts.moduleID, opts.scheduler, tags)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"action":      "spawn",
		"pid":         res.Id,
		"response_id": res.Id,
		"message":     res.Message,
	}
	return writeRuntimeResult(out, opts.jsonOut, payload, fmt.Sprintf("spawn ok, pid: %s", res.Id))
}

func runVmdockerExport(factory vmdockerRuntimeClientFactory, out io.Writer, opts vmdockerExportOptions) error {
	if strings.TrimSpace(opts.pid) == "" {
		return errors.New("pid is required")
	}
	if len(opts.pid) > maxIDChars {
		return fmt.Errorf("pid is too long (max %d)", maxIDChars)
	}
	if strings.TrimSpace(opts.privateKey) == "" {
		return errors.New("private-key is required")
	}
	client, err := factory(opts.nodeURL, opts.privateKey)
	if err != nil {
		return err
	}
	defer client.Close()
	res, err := client.SendMessageAndWait(opts.pid, "", []goarSchema.Tag{{Name: "Action", Value: "Export"}})
	if err != nil {
		return err
	}
	moduleID, err := decodeVmdockerExportResult(res.Message)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"action":      "export",
		"pid":         opts.pid,
		"module_id":   moduleID,
		"response_id": res.Id,
		"message":     res.Message,
	}
	return writeRuntimeResult(out, opts.jsonOut, payload, fmt.Sprintf("export ok, module id: %s", moduleID))
}

func decodeVmdockerExportResult(message string) (string, error) {
	var result vmmSchema.VmmResult
	if err := json.Unmarshal([]byte(message), &result); err != nil {
		return "", fmt.Errorf("decode export result: %w", err)
	}
	if strings.TrimSpace(result.Error) != "" {
		return "", fmt.Errorf("export failed on node: %s", result.Error)
	}
	moduleID := strings.TrimSpace(result.Data)
	if moduleID == "" {
		return "", errors.New("export returned empty module id")
	}
	return moduleID, nil
}
