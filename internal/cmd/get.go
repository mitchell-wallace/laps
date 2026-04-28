package cmd

import (
	"fmt"

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

		var taskID string
		if target == "head" {
			for _, t := range file.Tasks {
				if !t.IsDone {
					taskID = t.ID
					break
				}
			}
			if taskID == "" {
				exit(3, "no head task")
			}
		} else {
			taskID = target
		}

		for _, t := range file.Tasks {
			if t.ID == taskID {
				fmt.Println(t.Title)
				fmt.Println()
				fmt.Println(t.Description)
				return
			}
		}
		exit(3, "task not found")
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
