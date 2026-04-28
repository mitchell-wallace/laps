package cmd

import (
	"github.com/mitchell-wallace/microbeads/internal/store"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		found := false
		var tasks []store.Task
		for _, t := range file.Tasks {
			if t.ID == id {
				found = true
				continue
			}
			tasks = append(tasks, t)
		}
		if !found {
			exit(3, "task not found")
		}
		file.Tasks = tasks
		if err := store.Save(path, file); err != nil {
			exit(2, "delete: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
