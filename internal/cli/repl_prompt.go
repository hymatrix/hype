package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

type replModeKey struct{}
type replInKey struct{}
type replOutKey struct{}

type replFlagPrompt struct {
	Name   string
	Prompt string
}

func withRepl(ctx context.Context, in *bufio.Reader, out io.Writer) context.Context {
	ctx = context.WithValue(ctx, replModeKey{}, true)
	ctx = context.WithValue(ctx, replInKey{}, in)
	ctx = context.WithValue(ctx, replOutKey{}, out)
	return ctx
}

func isReplMode(cmd *cobra.Command) bool {
	v := cmd.Context().Value(replModeKey{})
	b, ok := v.(bool)
	return ok && b
}

func replPromptRequiredStringFlags(cmd *cobra.Command, prompts []replFlagPrompt) error {
	if !isReplMode(cmd) {
		return nil
	}

	in, _ := cmd.Context().Value(replInKey{}).(*bufio.Reader)
	out, _ := cmd.Context().Value(replOutKey{}).(io.Writer)
	if in == nil || out == nil {
		return nil
	}

	for _, p := range prompts {
		for {
			v, err := cmd.Flags().GetString(p.Name)
			if err != nil {
				return err
			}
			if strings.TrimSpace(v) != "" {
				break
			}

			s, err := replReadLine(in, out, p.Prompt)
			if err != nil {
				return err
			}
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if err := cmd.Flags().Set(p.Name, s); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func replReadLine(in *bufio.Reader, out io.Writer, prompt string) (string, error) {
	if prompt != "" {
		fmt.Fprint(out, prompt)
	}
	line, err := in.ReadString('\n')
	if err != nil {
		if len(line) > 0 {
			return strings.TrimRight(line, "\r\n"), nil
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
