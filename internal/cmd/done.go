package cmd

import (
	"fmt"
	"time"

	"github.com/mitchell-wallace/microbeads/internal/store"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done",
	Short: "Complete the head task",
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		for i, t := range file.Tasks {
			if !t.IsDone {
				now := time.Now().UTC()
				file.Tasks[i].IsDone = true
				file.Tasks[i].CompletedAt = &now
				file.Tasks[i].UpdatedAt = now
				if err := store.Save(path, file); err != nil {
					exit(2, "done: %v", err)
				}
				fmt.Println(t.ID)
				return
			}
		}
		exit(3, "no head task")
	},
}

func init() {
	rootCmd.AddCommand(doneCmd)
}
