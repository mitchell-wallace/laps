package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var (
	listAll     bool
	listDone    bool
	listOneline bool
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

		activeID, err := store.ReadClaim(beadsDir)
		if err != nil {
			exit(2, "read claim: %v", err)
		}

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
				lines = append(lines, formatListEntry(&t, i+1, true, activeID))
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
				lines = append(lines, formatListEntry(&t, num, false, activeID))
				jsonTasks = append(jsonTasks, t)
				num++
			}
			if listAll {
				for _, idx := range dones {
					t := file.Tasks[idx]
					lines = append(lines, formatListEntry(&t, num, true, activeID))
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
	listCmd.Flags().BoolVar(&listOneline, "oneline", false, "render each lap on a single line (prior format)")
	rootCmd.AddCommand(listCmd)
}

// formatListEntry renders one lap for `list`. With --oneline it reuses the prior
// single-line shape (whole-line strike when done); otherwise it renders the
// two-line default: line 1 holds the position, active marker, and title (struck
// when done); line 2 holds the id, assignee (em dash when unset), and state.
func formatListEntry(t *store.Task, num int, done bool, activeID string) string {
	if listOneline {
		if done {
			return fmt.Sprintf("%d. ~~%s~~", num, formatListTask(t))
		}
		return fmt.Sprintf("%d. %s", num, formatListTask(t))
	}

	prefix := fmt.Sprintf("%d. ", num)
	marker := ""
	if activeID != "" && t.ID == activeID {
		marker = "> "
	}
	title := t.Title
	if done {
		title = fmt.Sprintf("~~%s~~", title)
	}
	line1 := prefix + marker + title

	assignee := t.Assignee
	if assignee == "" {
		assignee = "—"
	}
	state := "todo"
	if t.IsDone {
		state = "done"
	}
	line2 := strings.Repeat(" ", len(prefix)) + fmt.Sprintf("%s · %s · %s", t.ID, assignee, state)

	return line1 + "\n" + line2
}

func formatListTask(t *store.Task) string {
	text := fmt.Sprintf("%s — %s", t.ID, t.Title)
	if t.Assignee != "" {
		text += fmt.Sprintf(" (assignee: %s)", t.Assignee)
	}
	return text
}
