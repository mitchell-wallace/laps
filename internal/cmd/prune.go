package cmd

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/mitchell-wallace/microbeads/internal/store"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune [N]",
	Short: "Remove old done tasks, keeping the N most recent",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		n := 20
		if len(args) > 0 {
			parsed, err := strconv.Atoi(args[0])
			if err != nil || parsed < 0 {
				exit(1, "prune: N must be a non-negative integer")
			}
			n = parsed
		}

		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		var done []int
		var todo []store.Task
		for i, t := range file.Tasks {
			if t.IsDone {
				done = append(done, i)
			} else {
				todo = append(todo, t)
			}
		}

		sort.Slice(done, func(i, j int) bool {
			ai := file.Tasks[done[i]].CompletedAt
			aj := file.Tasks[done[j]].CompletedAt
			if ai == nil || aj == nil {
				return false
			}
			return ai.After(*aj)
		})

		var keepDone []store.Task
		for i := 0; i < len(done) && i < n; i++ {
			keepDone = append(keepDone, file.Tasks[done[i]])
		}

		removed := len(done) - len(keepDone)

		// Rebuild tasks preserving todo order, then append kept done in original order
		// Actually, spec says prune retains N most recent done by completedAt descending.
		// The remaining tasks should be: todos in original order, then kept done.
		// But we want to keep the done tasks in their original array positions? No,
		// spec says "retains the N most recently completed done tasks".
		// Let's keep todos in place, and for done, keep only the N most recent.
		// The simplest: todo tasks in original order, then kept done sorted by completedAt desc.
		var result []store.Task
		result = append(result, todo...)
		result = append(result, keepDone...)
		file.Tasks = result

		if err := store.Save(path, file); err != nil {
			exit(2, "prune: %v", err)
		}
		fmt.Println(removed)
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}
