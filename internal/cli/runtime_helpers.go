package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func hydrateFlagFromEnvs(cmd *cobra.Command, flagName string, envKeys ...string) error {
	value, err := cmd.Flags().GetString(flagName)
	if err != nil {
		return err
	}
	if strings.TrimSpace(value) != "" {
		return nil
	}

	envValue := firstNonEmptyEnv(envKeys...)
	if envValue == "" {
		return nil
	}
	return cmd.Flags().Set(flagName, envValue)
}

func validateRuntimeBackend(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	switch value {
	case openclawRuntimeDocker, openclawRuntimeSandbox:
		return nil
	default:
		return errors.New("runtime-backend must be one of \"docker\" or \"sandbox\"")
	}
}

func writeRuntimeResult(out io.Writer, jsonOut bool, payload map[string]interface{}, fallback string) error {
	if !jsonOut {
		_, err := fmt.Fprintln(out, fallback)
		return err
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(b))
	return err
}
