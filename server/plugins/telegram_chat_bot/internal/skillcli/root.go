package skillcli

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "telegram_chat_bot",
	Short: "telegram_chat_bot skill CLI",
	Long:  "Utility CLI bundled with the Telegram Q&A plugin skill.",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
