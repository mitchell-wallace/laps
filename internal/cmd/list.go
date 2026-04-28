package cmd

import (
	"fmt"
	"sort"
	"strings"

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

		exitCode := 0
		var output string
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, nil, &output, &exitCode)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil)

		var lines []string
		if listDone {
			var done []int
			for i, t := range file.Tasks {
				if t.IsDone {
					done = append(done, i)
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
			for i, d := range done {
				t := file.Tasks[d]
				lines = append(lines, fmt.Sprintf("%d. ~~%s — %s~~", i+1, t.ID, t.Title))
			}
		} else {
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
				lines = append(lines, fmt.Sprintf("%d. %s — %s", num, t.ID, t.Title))
				num++
			}
			if listAll {
				for _, idx := range dones {
					t := file.Tasks[idx]
					lines = append(lines, fmt.Sprintf("%d. ~~%s — %s~~", num, t.ID, t.Title))
					num++
				}
			}
		}

		output = strings.Join(lines, "\n")
		if output != "" {
			fmt.Println(output)
		}
	},
}

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "include done tasks")
	listCmd.Flags().BoolVar(&listDone, "done", false, "show only done tasks")
	rootCmd.AddCommand(listCmd)
}
