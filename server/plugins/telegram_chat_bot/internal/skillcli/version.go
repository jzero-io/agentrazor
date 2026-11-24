package skillcli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version string
	Commit  string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print telegram_chat_bot skill CLI version",
	Run: func(*cobra.Command, []string) {
		if Version == "" {
			Version = "unknown"
		}
		if Commit == "" {
			Commit = "unknown"
		}
		fmt.Printf("telegram_chat_bot skill CLI %s %s/%s\nGit commit %s\n", Version, runtime.GOOS, runtime.GOARCH, Commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
