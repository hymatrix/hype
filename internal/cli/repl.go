package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

func newReplCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repl",
		Short: "Start interactive mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReplLoop(
				bufio.NewReader(cmd.InOrStdin()),
				cmd.OutOrStdout(),
				cmd.ErrOrStderr(),
			)
		},
	}
}

func runReplLoop(in *bufio.Reader, out io.Writer, errOut io.Writer) error {
	fmt.Fprint(out, versionBanner())
	fmt.Fprintln(out, `Type "help" for commands, "exit" to quit.`)

	for {
		line, err := replReadLine(in, out, "hype> ")
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(out)
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}
		if line == "help" || line == "?" {
			root := NewRootCmd()
			root.SetArgs([]string{"--help"})
			root.SetOut(out)
			root.SetErr(errOut)
			_ = root.Execute()
			continue
		}
		if strings.HasPrefix(line, "!") {
			shellLine := strings.TrimSpace(strings.TrimPrefix(line, "!"))
			if shellLine == "" {
				continue
			}
			c := exec.Command("bash", "-lc", shellLine)
			c.Stdin = os.Stdin
			c.Stdout = out
			c.Stderr = errOut
			if err := c.Run(); err != nil {
				fmt.Fprintln(errOut, err)
			}
			continue
		}

		cmdArgs, err := splitArgs(line)
		if err != nil {
			fmt.Fprintln(errOut, err)
			continue
		}
		if len(cmdArgs) == 0 {
			continue
		}
		if cmdArgs[0] == "hype" {
			cmdArgs = cmdArgs[1:]
		}
		if len(cmdArgs) == 0 {
			continue
		}
		if cmdArgs[0] == "repl" {
			fmt.Fprintln(out, "already in repl")
			continue
		}

		root := NewRootCmd()
		root.SetArgs(cmdArgs)
		root.SetContext(withRepl(context.Background(), in, out))
		root.SetOut(out)
		root.SetErr(errOut)
		if err := root.Execute(); err != nil {
			fmt.Fprintln(errOut, err)
		}
	}
}

func splitArgs(line string) ([]string, error) {
	var args []string
	var b strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	prevWasSpace := true

	flush := func() {
		if b.Len() == 0 {
			return
		}
		args = append(args, b.String())
		b.Reset()
	}

	for _, r := range line {
		if escaped {
			b.WriteRune(r)
			escaped = false
			prevWasSpace = false
			continue
		}

		if !inSingle && r == '\\' {
			escaped = true
			prevWasSpace = false
			continue
		}

		if !inDouble && r == '\'' {
			inSingle = !inSingle
			prevWasSpace = false
			continue
		}
		if !inSingle && r == '"' {
			inDouble = !inDouble
			prevWasSpace = false
			continue
		}

		if !inSingle && !inDouble {
			if r == '#' && prevWasSpace {
				break
			}
			if unicode.IsSpace(r) {
				flush()
				prevWasSpace = true
				continue
			}
		}

		b.WriteRune(r)
		prevWasSpace = false
	}

	if escaped {
		return nil, fmt.Errorf("unfinished escape")
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()
	return args, nil
}
