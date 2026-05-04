package cmd

import (
	"github.com/mitchell-wallace/laps/internal/instructions"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var offCmd = &cobra.Command{
	Use:   "off",
	Short: "Remove laps instructions from AGENTS.md, CLAUDE.md, and GEMINI.md",
	Long:  `Remove the <laps-instructions> block from AGENTS.md, CLAUDE.md, and GEMINI.md. Leaves the rest of each file untouched.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()
		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil, args)

		if err := instructions.Disable(); err != nil {
			exitCode = 2
			exit(2, "off: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(offCmd)
}
