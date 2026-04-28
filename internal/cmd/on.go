package cmd

import (
	"github.com/mitchell-wallace/microbeads/internal/instructions"
	"github.com/spf13/cobra"
)

var onCmd = &cobra.Command{
	Use:   "on",
	Short: "Add mb instructions to AGENTS.md (and CLAUDE.md/GEMINI.md if they exist)",
	Run: func(cmd *cobra.Command, args []string) {
		if err := instructions.Enable(); err != nil {
			exit(2, "on: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(onCmd)
}
