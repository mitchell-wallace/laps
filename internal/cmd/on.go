package cmd

import (
	"github.com/mitchell-wallace/laps/internal/instructions"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var onCmd = &cobra.Command{
	Use:   "on",
	Short: "Add laps instructions to AGENTS.md (and CLAUDE.md/GEMINI.md if they exist)",
	Long: `Add the <laps-instructions> block to AGENTS.md, creating it if absent.

Also updates CLAUDE.md and GEMINI.md if they already exist.
Idempotent — safe to run multiple times.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()
		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil, args)

		if err := instructions.Enable(); err != nil {
			exitCode = 2
			exit(2, "on: %v", err)
		}
		if jsonOutput {
			printJSON(map[string]interface{}{"status": "enabled"})
		}
	},
}

func init() {
	rootCmd.AddCommand(onCmd)
}
