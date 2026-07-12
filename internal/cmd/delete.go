package cmd

import (
	"fmt"
	"os"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var deleteForce bool

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Long:  `Delete a task by id, regardless of whether it is todo or done. A claimed lap requires --force.`,
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			exit(1, "delete: task id required")
		}
		id := args[0]
		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path, repoRoot, beadsDir)
		ctx, err := resolveSelectedContext(path, repoRoot, beadsDir, file)
		if err != nil {
			exit(2, "%v", err)
		}
		path = ctx.Path
		file = ctx.File

		task := findScopedTask(ctx, id)

		exitCode := 0
		var output string
		defer runAfterHooksDeferredScoped(cmd.Name(), beadsDir, path, ctx.Scope, &task, &output, &exitCode, args)()
		runBeforeHooksScoped(cmd.Name(), beadsDir, path, ctx.Scope, task, args)

		if task == nil {
			exitIfOutOfScope(beadsDir, repoRoot, ctx, id)
			exitCode = 3
			exit(3, "task not found")
		}

		claimFile := fileNameForClaim(beadsDir, path)
		claim, err := readClaim(beadsDir, claimFile)
		if err != nil {
			exitCode = 2
			exit(2, "read claim: %v", err)
		}
		deletingClaimed := claim.Lap == task.ID && claim.File == claimFile
		if deletingClaimed && !deleteForce {
			exitCode = 1
			fmt.Fprintf(os.Stderr, "laps: lap %s is currently claimed; use 'laps delete --force %s' to delete it and clear the claim.\n", task.ID, task.ID)
			exit(1, "delete: claimed lap requires --force")
		}

		var tasks []store.Task
		for i := range file.Tasks {
			if file.Tasks[i].ID == id {
				continue
			}
			tasks = append(tasks, file.Tasks[i])
		}
		file.Tasks = tasks
		if err := store.Save(path, file); err != nil {
			exitCode = 2
			exit(2, "delete: %v", err)
		}
		if deletingClaimed {
			if err := store.RemoveClaim(beadsDir); err != nil {
				exitCode = 2
				exit(2, "delete: clear claim: %v", err)
			}
		}
		logScopedEvent(beadsDir, ctx, &eventlog.Entry{
			Event:    "deleted",
			Cmd:      "delete",
			Lap:      task.ID,
			Title:    task.Title,
			Assignee: task.Assignee,
		})
		if jsonOutput {
			printJSON(map[string]interface{}{"deleted": id})
		}
	},
}

func init() {
	deleteCmd.Flags().BoolVar(&deleteForce, "force", false, "delete a claimed lap and clear its claim")
	addScopeFlags(deleteCmd)
	rootCmd.AddCommand(deleteCmd)
}
