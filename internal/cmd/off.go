package cmd

import (
	"github.com/mitchell-wallace/microbeads/internal/instructions"
	"github.com/spf13/cobra"
)

var offCmd = &cobra.Command{
	Use:   "off",
	Short: "Remove mb instructions from AGENTS.md, CLAUDE.md, and GEMINI.md",
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()
		exitCode := 0
		var output string
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, nil, &output, &exitCode)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil)

		if err := instructions.Disable(); err != nil {
			exitCode = 2
			exit(2, "off: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(offCmd)
}
