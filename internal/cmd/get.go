package cmd

import (
	"fmt"

	"github.com/mitchell-wallace/microbeads/internal/store"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get [head|<id>]",
	Short: "Get a task by id or head",
	Args:  cobra.MaximumNArgs(1),
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
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil)

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
				output = fmt.Sprintf("%s\n\n%s", task.Title, task.Description)
				fmt.Println(task.Title)
				fmt.Println()
				fmt.Println(task.Description)
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
