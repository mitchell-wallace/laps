package cmd

import (
	"github.com/spf13/cobra"
)

var assignCmd = &cobra.Command{
	Use:   "assign <id> <role>",
	Short: "Assign a task to a role",
	Long: `Assign a task to a role. Shortcut for 'edit <id> --assignee <role>'.

A blank role clears the assignee. Assigning a completed lap succeeds with a
warning and does not reopen it. Prints the affected task id on success.`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			exit(1, "assign: usage: assign <id> <role>")
		}
		runEdit(cmd, args, editFields{
			setAssignee: true,
			assignee:    args[1],
		})
	},
}

func init() {
	rootCmd.AddCommand(assignCmd)
}
