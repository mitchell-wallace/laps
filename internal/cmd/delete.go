package cmd

import (
	"github.com/mitchell-wallace/microbeads/internal/store"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Long:  `Delete a task by id, regardless of whether it is todo or done.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil)

		found := false
		var tasks []store.Task
		for i := range file.Tasks {
			if file.Tasks[i].ID == id {
				found = true
				task = &file.Tasks[i]
				continue
			}
			tasks = append(tasks, file.Tasks[i])
		}
		if !found {
			exitCode = 3
			exit(3, "task not found")
		}
		file.Tasks = tasks
		if err := store.Save(path, file); err != nil {
			exitCode = 2
			exit(2, "delete: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
