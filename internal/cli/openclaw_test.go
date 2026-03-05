package cli

import (
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func execOpenclaw(t *testing.T, args ...string) error {
	t.Helper()
	cmd := NewRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}

func TestOpenclawPairTgRequiresCode(t *testing.T) {
	err := execOpenclaw(t, "openclaw", "pair-tg", "-p", "pid-1", "-k", "0x1234")
	if err == nil || !strings.Contains(err.Error(), "code is required") {
		t.Fatalf("expected code required error, got: %v", err)
	}
}

func TestOpenclawChatRequiresCommand(t *testing.T) {
	err := execOpenclaw(t, "openclaw", "chat", "-p", "pid-1", "-k", "0x1234")
	if err == nil || !strings.Contains(err.Error(), "command is required") {
		t.Fatalf("expected command required error, got: %v", err)
	}
}

func TestOpenclawConfTgRequiresBotToken(t *testing.T) {
	err := execOpenclaw(t, "openclaw", "conf-tg", "-p", "pid-1", "-k", "0x1234", "--dm-policy", "")
	if err == nil || !strings.Contains(err.Error(), "required flag(s) \"bot-token\" not set") {
		t.Fatalf("expected bot-token required error, got: %v", err)
	}
}

func TestReadOpenclawSharedFlagsUsesEnvPrivateKey(t *testing.T) {
	t.Setenv("HYPE_PRIVATE_KEY", "0xabc")
	cmd := newOpenclawSharedFlagCmd()
	if err := cmd.Flags().Set("node-url", "http://127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}

	shared, err := readOpenclawSharedFlags(cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shared.privateKey != "0xabc" {
		t.Fatalf("expected env private key, got %q", shared.privateKey)
	}
}

func TestReadOpenclawSharedFlagsRequiresPrivateKey(t *testing.T) {
	t.Setenv("HYPE_PRIVATE_KEY", "")
	t.Setenv("PRV_KEY", "")
	cmd := newOpenclawSharedFlagCmd()
	if err := cmd.Flags().Set("node-url", "http://127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}

	_, err := readOpenclawSharedFlags(cmd)
	if err == nil || !strings.Contains(err.Error(), "private-key is required") {
		t.Fatalf("expected private-key is required error, got %v", err)
	}
}

func TestHydrateOpenclawPrivateKeyFlagFromEnv(t *testing.T) {
	t.Setenv("HYPE_PRIVATE_KEY", "0xenv")
	cmd := newOpenclawSharedFlagCmd()
	if err := hydrateOpenclawPrivateKeyFlag(cmd); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	v, err := cmd.Flags().GetString("private-key")
	if err != nil {
		t.Fatalf("unexpected get flag error: %v", err)
	}
	if v != "0xenv" {
		t.Fatalf("expected private-key flag hydrated from env, got %q", v)
	}
}

func TestExtractChatReplyPrefersOutput(t *testing.T) {
	raw := `{"Output":"hello from output","Data":"ignored","Messages":[{"Data":"msg","Tags":[{"name":"Reply","value":"tag-reply"}]}]}`
	got := extractChatReply(raw)
	if got != "hello from output" {
		t.Fatalf("expected output reply, got %q", got)
	}
}

func TestExtractChatReplyFallsBackToReplyTag(t *testing.T) {
	raw := `{"Output":"","Data":"","Messages":[{"Data":"","Tags":[{"name":"Reply","value":"tag-reply"}]}]}`
	got := extractChatReply(raw)
	if got != "tag-reply" {
		t.Fatalf("expected tag reply, got %q", got)
	}
}

func newOpenclawSharedFlagCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("node-url", "http://127.0.0.1:8080", "")
	cmd.Flags().String("private-key", "", "")
	cmd.Flags().Bool("json", false, "")
	return cmd
}
