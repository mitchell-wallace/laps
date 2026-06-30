package cmd

import (
	"fmt"
	"time"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

const UndoAgeLimit = 5 * time.Minute

var forceUndo bool

var doneCmd = &cobra.Command{
	Use:   "done [<id>]",
	Short: "Complete the claimed or specified task",
	Long: `Complete the claimed task. If no task is claimed and no id is given,
prints a hint with the head task id and title.  If an id is given, completes
that task regardless of the claim state.

When a claimed task is completed, .laps/claim is cleared.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		var task *store.Task
		selectedFile := store.ResolveFile(fileFlag)

		if len(args) > 0 {
			id := args[0]
			for i := range file.Tasks {
				if file.Tasks[i].ID == id {
					task = &file.Tasks[i]
					break
				}
			}
			if task == nil {
				exit(3, "task not found")
			}
			if task.IsDone {
				exit(3, "task %s (%s) is already done", task.ID, task.Title)
			}
		} else {
			claim, err := store.ReadClaim(beadsDir, selectedFile)
			if err != nil {
				exit(2, "read claim: %v", err)
			}
			claimedID := claim.Lap

			if claimedID == "" {
				for i := range file.Tasks {
					if !file.Tasks[i].IsDone {
						task = &file.Tasks[i]
						break
					}
				}
				if task == nil {
					exit(3, "no claimed lap and no head task")
				}
				exit(3, "no claimed lap. head task is %s: %s. use 'laps claim' or 'laps done %s'", task.ID, task.Title, task.ID)
			}

			for i := range file.Tasks {
				if file.Tasks[i].ID == claimedID {
					task = &file.Tasks[i]
					break
				}
			}
			if task == nil {
				exit(3, "claimed lap %s not found", claimedID)
			}
			if task.IsDone {
				exit(3, "claimed lap %s (%s) is already done", task.ID, task.Title)
			}
		}

		exitCode := 0
		var output string
		defer runAfterHooksDeferred(hookCommandName(cmd), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(hookCommandName(cmd), beadsDir, path, task, args)

		now := time.Now().UTC()
		task.IsDone = true
		task.CompletedAt = &now
		task.UpdatedAt = now
		if err := store.Save(path, file); err != nil {
			exitCode = 2
			exit(2, "done: %v", err)
		}
		logEvent(beadsDir, &eventlog.Entry{
			Event:    "completed",
			Cmd:      "done",
			Lap:      task.ID,
			Title:    task.Title,
			Assignee: task.Assignee,
		})

		// Best-effort: clearing the claim must not block a completed done. When the
		// claim is actually removed, emit a SEPARATE unclaimed event tagged
		// reason "completed", immediately after completed, for log uniformity with
		// the replaced reason. A failed remove emits no unclaimed event.
		if claim, err := store.ReadClaim(beadsDir, selectedFile); err == nil && claim.Lap == task.ID && claim.File == selectedFile {
			if err := store.RemoveClaim(beadsDir); err == nil {
				logEvent(beadsDir, &eventlog.Entry{
					Event:    "unclaimed",
					Cmd:      "done",
					Lap:      task.ID,
					Title:    task.Title,
					Assignee: task.Assignee,
					Detail:   map[string]interface{}{"reason": "completed"},
				})
			}
		}

		output = task.ID
		if jsonOutput {
			printJSON(map[string]interface{}{"task": task})
		} else {
			fmt.Printf("%s\n\n-----\nNot the lap you intended to mark done? Undo with 'laps done undo'\n", task.Title)
		}
	},
}

var doneUndoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo the most recent completion",
	Long: `Re-open the most recently completed lap. If it was completed more than
5 minutes ago the command fails unless --yes (-y) is passed.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		var latest *store.Task
		for i := range file.Tasks {
			if file.Tasks[i].IsDone && file.Tasks[i].CompletedAt != nil {
				if latest == nil || file.Tasks[i].CompletedAt.After(*latest.CompletedAt) {
					latest = &file.Tasks[i]
				}
			}
		}

		exitCode := 0
		var output string
		defer runAfterHooksDeferred(hookCommandName(cmd), beadsDir, path, &latest, &output, &exitCode, args)()
		runBeforeHooks(hookCommandName(cmd), beadsDir, path, latest, args)

		if latest == nil {
			exitCode = 3
			exit(3, "no completed task to undo")
		}

		age := time.Since(*latest.CompletedAt)
		if age > UndoAgeLimit && !forceUndo {
			exitCode = 3
			exit(3, "last completed task %s (%s) was completed %v ago; use 'laps done undo -y' to force", latest.ID, latest.Title, age.Round(time.Second))
		}

		latest.IsDone = false
		latest.CompletedAt = nil
		latest.UpdatedAt = time.Now().UTC()
		if err := store.Save(path, file); err != nil {
			exitCode = 2
			exit(2, "done undo: %v", err)
		}
		logEvent(beadsDir, &eventlog.Entry{
			Event:    "reopened",
			Cmd:      "done-undo",
			Lap:      latest.ID,
			Title:    latest.Title,
			Assignee: latest.Assignee,
		})

		output = latest.ID
		if jsonOutput {
			printJSON(map[string]interface{}{"task": latest})
		} else {
			fmt.Printf("Done state cleared for %s (%s)\n", latest.Title, latest.ID)
		}
	},
}

func hookCommandName(cmd *cobra.Command) string {
	if cmd.Parent() != nil && cmd.Parent() != rootCmd {
		return cmd.Parent().Name() + "-" + cmd.Name()
	}
	return cmd.Name()
}

func init() {
	rootCmd.AddCommand(doneCmd)
	doneCmd.AddCommand(doneUndoCmd)
	doneUndoCmd.Flags().BoolVarP(&forceUndo, "yes", "y", false, "force undo even if completed more than 5 minutes ago")
}
