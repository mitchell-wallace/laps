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

		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil, args)

		var taskID string
		if target == "head" {
			for _, t := range file.Tasks {
				if !t.IsDone {
					taskID = t.ID
					break
				}
			}
			if taskID == "" {
				exitCode = 3
				exit(3, "no head task")
			}
		} else {
			taskID = target
		}

		for i := range file.Tasks {
			if file.Tasks[i].ID == taskID {
				task = &file.Tasks[i]
				output = formatTaskDetails(task)
				if jsonOutput {
					printJSON(map[string]interface{}{"task": task})
				} else {
					fmt.Println(output)
				}
				return
			}
		}
		exitCode = 3
		exit(3, "task not found")
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func formatTaskDetails(task *store.Task) string {
	if task.Assignee == "" {
		return fmt.Sprintf("%s\n\n%s", task.Title, task.Description)
	}
	return fmt.Sprintf("%s\nAssignee: %s\n\n%s", task.Title, task.Assignee, task.Description)
}
