package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	Version = "v0.0.1"
)

func versionBanner() string {
	return fmt.Sprintf(`
=================================
||            HYPE            ||
=================================

hype: Hymx CLI tool
https://github.com/hymatrix/hype

Version:    %s
GoVersion:  %s
Compiler:   %s
Platform:   %s/%s
`, Version, runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(versionBanner())
	},
}
