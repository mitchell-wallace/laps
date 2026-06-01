package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var (
	listAll  bool
	listDone bool
)

var listCmd = &cobra.Command{
	Use:   "list [--all | --done]",
	Short: "List tasks",
	Long: `List tasks as a markdown numbered list.

Default shows todo tasks only, head first.
  --all    Include done tasks after todo items (struck through).
  --done   Show only completed tasks, most recent first.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil, args)

		var lines []string
		var jsonTasks []store.Task
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
				if ai == nil && aj == nil {
					return false
				}
				if ai == nil {
					return false
				}
				if aj == nil {
					return true
				}
				return ai.After(*aj)
			})
			for i, d := range done {
				t := file.Tasks[d]
				lines = append(lines, fmt.Sprintf("%d. ~~%s~~", i+1, formatListTask(&t)))
				jsonTasks = append(jsonTasks, t)
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
				lines = append(lines, fmt.Sprintf("%d. %s", num, formatListTask(&t)))
				jsonTasks = append(jsonTasks, t)
				num++
			}
			if listAll {
				for _, idx := range dones {
					t := file.Tasks[idx]
					lines = append(lines, fmt.Sprintf("%d. ~~%s~~", num, formatListTask(&t)))
					jsonTasks = append(jsonTasks, t)
					num++
				}
			}
		}

		output = strings.Join(lines, "\n")
		if jsonOutput {
			if jsonTasks == nil {
				jsonTasks = []store.Task{}
			}
			printJSON(map[string]interface{}{"tasks": jsonTasks})
		} else if output != "" {
			fmt.Println(output)
		}
	},
}

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "include done tasks")
	listCmd.Flags().BoolVar(&listDone, "done", false, "show only done tasks")
	rootCmd.AddCommand(listCmd)
}

func formatListTask(t *store.Task) string {
	text := fmt.Sprintf("%s — %s", t.ID, t.Title)
	if t.Assignee != "" {
		text += fmt.Sprintf(" (assignee: %s)", t.Assignee)
	}
	return text
}
