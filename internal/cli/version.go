package cli

import (
	"fmt"
	"runtime"

	nodeSchema "github.com/hymatrix/hymx/node/schema"
	"github.com/spf13/cobra"
)

const (
	Version = "v0.0.2"
)

func versionBanner() string {
	return fmt.Sprintf(`
=================================
||            HYPE            ||
=================================

hype: Hymx CLI tool
https://github.com/hymatrix/hype

Version:     %s
HymxVersion: %s
GoVersion:   %s
Compiler:    %s
Platform:    %s/%s
`, Version, nodeSchema.NodeVersion, runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(versionBanner())
		},
	}
}
