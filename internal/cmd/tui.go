package cmd

import (
	"github.com/mitchell-wallace/laps/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the laps queue TUI",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.Run(tui.Runner{}); err != nil {
			exit(2, "tui: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
