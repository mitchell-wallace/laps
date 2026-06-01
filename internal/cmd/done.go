package cmd

import (
	"fmt"
	"time"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done",
	Short: "Complete the head task",
	Long: `Complete the head task. Sets it to done and prints the task id.

If there is no head task, exits non-zero with "no head task".`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil, args)

		for i := range file.Tasks {
			if !file.Tasks[i].IsDone {
				now := time.Now().UTC()
				file.Tasks[i].IsDone = true
				file.Tasks[i].CompletedAt = &now
				file.Tasks[i].UpdatedAt = now
			task = &file.Tasks[i]
			if err := store.Save(path, file); err != nil {
				exitCode = 2
				exit(2, "done: %v", err)
			}
			output = task.ID
			if jsonOutput {
				printJSON(map[string]interface{}{"task": task})
			} else {
				fmt.Println(task.ID)
			}
			return
			}
		}
		exitCode = 3
		exit(3, "no head task")
	},
}

func init() {
	rootCmd.AddCommand(doneCmd)
}
