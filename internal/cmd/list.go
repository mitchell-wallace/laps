package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var (
	listAll  bool
	listDone bool
)

var listCmd = &cobra.Command{
	Use:   "list [--all | --done]",
	Short: "List tasks",
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		if listDone {
			var done []struct {
				idx int
			}
			for i, t := range file.Tasks {
				if t.IsDone {
					done = append(done, struct{ idx int }{i})
				}
			}
			sort.Slice(done, func(i, j int) bool {
				ai := file.Tasks[done[i].idx].CompletedAt
				aj := file.Tasks[done[j].idx].CompletedAt
				if ai == nil || aj == nil {
					return false
				}
				return ai.After(*aj)
			})
			for i, d := range done {
				t := file.Tasks[d.idx]
				fmt.Printf("%d. ~~%s — %s~~\n", i+1, t.ID, t.Title)
			}
			return
		}

		var todos []int
		var dones []int
		for i, t := range file.Tasks {
			if t.IsDone {
				dones = append(dones, i)
			} else {
				todos = append(todos, i)
			}
		}

		num := 1
		for _, idx := range todos {
			t := file.Tasks[idx]
			fmt.Printf("%d. %s — %s\n", num, t.ID, t.Title)
			num++
		}

		if listAll {
			for _, idx := range dones {
				t := file.Tasks[idx]
				fmt.Printf("%d. ~~%s — %s~~\n", num, t.ID, t.Title)
				num++
			}
		}
	},
}

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "include done tasks")
	listCmd.Flags().BoolVar(&listDone, "done", false, "show only done tasks")
	rootCmd.AddCommand(listCmd)
}
