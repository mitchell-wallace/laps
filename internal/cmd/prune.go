package cmd

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune [N]",
	Short: "Remove old done tasks, keeping the N most recent",
	Long: `Remove old done tasks, keeping the N most recent. Default N is 20.

Use prune 0 to remove all done tasks. Todo tasks are never touched.
Prints the number of tasks removed.`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		n := 20
		if len(args) > 0 {
			parsed, err := strconv.Atoi(args[0])
			if err != nil || parsed < 0 {
				exit(1, "prune: N must be a non-negative integer")
			}
			n = parsed
		}

		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path, repoRoot, beadsDir)
		ctx, err := resolveSelectedContext(path, repoRoot, beadsDir, file)
		if err != nil {
			exit(2, "%v", err)
		}
		path = ctx.Path
		file = ctx.File

		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferredScoped(cmd.Name(), beadsDir, path, ctx.Scope, &task, &output, &exitCode, args)()
		runBeforeHooksScoped(cmd.Name(), beadsDir, path, ctx.Scope, nil, args)

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

		var keepDone []store.Task
		for i := 0; i < len(done) && i < n; i++ {
			keepDone = append(keepDone, file.Tasks[done[i]])
		}

		var pruned []store.Task
		for i := n; i < len(done); i++ {
			pruned = append(pruned, file.Tasks[done[i]])
		}
		removed := len(pruned)
		var result []store.Task
		result = append(result, todo...)
		result = append(result, keepDone...)
		file.Tasks = result

		if err := store.Save(path, file); err != nil {
			exitCode = 2
			exit(2, "prune: %v", err)
		}
		for i := range pruned {
			logScopedEvent(beadsDir, ctx, &eventlog.Entry{
				Event:    "pruned",
				Cmd:      "prune",
				Lap:      pruned[i].ID,
				Title:    pruned[i].Title,
				Assignee: pruned[i].Assignee,
			})
		}
		output = fmt.Sprintf("%d", removed)
		if jsonOutput {
			printJSON(map[string]interface{}{"removed": removed})
		} else {
			fmt.Println(removed)
		}
	},
}

func init() {
	addScopeFlags(pruneCmd)
	rootCmd.AddCommand(pruneCmd)
}
