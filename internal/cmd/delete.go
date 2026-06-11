package cmd

import (
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Long:  `Delete a task by id, regardless of whether it is todo or done.`,
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			exit(1, "delete: task id required")
		}
		id := args[0]
		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		var task *store.Task
		for i := range file.Tasks {
			if file.Tasks[i].ID == id {
				task = &file.Tasks[i]
				break
			}
		}

		exitCode := 0
		var output string
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, task, args)

		if task == nil {
			exitCode = 3
			exit(3, "task not found")
		}

		var tasks []store.Task
		for i := range file.Tasks {
			if file.Tasks[i].ID == id {
				continue
			}
			tasks = append(tasks, file.Tasks[i])
		}
		file.Tasks = tasks
		if err := store.Save(path, file); err != nil {
			exitCode = 2
			exit(2, "delete: %v", err)
		}
		if jsonOutput {
			printJSON(map[string]interface{}{"deleted": id})
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
