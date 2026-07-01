package cmd

import (
	"fmt"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get [head|<id>]",
	Short: "Get a task by id or head",
	Long: `Get a task by id, or read the head task if no argument is given.

Output is title, blank line, description — nothing else.`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		target := "head"
		if len(args) > 0 {
			target = args[0]
		}

		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path, repoRoot, beadsDir)

		ctx, err := resolveSelectedContext(path, repoRoot, beadsDir, file)
		if err != nil {
			exit(2, "%v", err)
		}
		path = ctx.Path
		file = ctx.File

		var task *store.Task
		if target == "head" {
			task = ctx.Head
		} else {
			task = findScopedTask(ctx, target)
			if task == nil {
				exitIfOutOfScope(beadsDir, repoRoot, ctx, target)
			}
		}

		exitCode := 0
		var output string
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, task, args)

		if task == nil {
			exitCode = 3
			if target == "head" {
				exit(3, "no head task")
			}
			exit(3, "task not found")
		}

		output = formatTaskDetails(task)
		if jsonOutput {
			printJSON(map[string]interface{}{"task": task})
		} else {
			fmt.Println(output)
		}
	},
}

func init() {
	addScopeFlags(getCmd)
	rootCmd.AddCommand(getCmd)
}

func formatTaskDetails(task *store.Task) string {
	if task.Assignee == "" {
		return fmt.Sprintf("%s\n\n%s", task.Title, task.Description)
	}
	return fmt.Sprintf("%s\nAssignee: %s\n\n%s", task.Title, task.Assignee, task.Description)
}
