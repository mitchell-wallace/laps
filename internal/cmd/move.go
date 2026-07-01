package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <id> <head|tail|after> [target]",
	Short: "Reorder an existing task",
	Long: `Reorder an existing todo task, preserving its id.

Positions:
  head            Move to the front of the queue.
  tail            Move to the end of the queue.
  after <id>      Move to immediately after the specified task id.

Prints the moved task id on success.`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path, repoRoot, beadsDir)
		ctx, err := resolveSelectedContext(path, repoRoot, beadsDir, file)
		if err != nil {
			exit(2, "%v", err)
		}
		path = ctx.Path
		file = ctx.File

		if len(args) < 2 {
			exit(1, "move: usage: move <id> <head|tail|after> [target]")
		}
		moveID := args[0]
		position := args[1]
		if position != "head" && position != "tail" && position != "after" {
			exit(1, "move: position must be head, tail, or after <id>")
		}
		if position != "after" && len(args) != 2 {
			exit(1, "move: usage: move <id> <head|tail|after> [target]")
		}
		if position == "after" && len(args) < 3 {
			exit(1, "move: after requires a target id")
		}
		if position == "after" && len(args) > 3 {
			exit(1, "move: after requires exactly one target id")
		}

		var afterID string
		if position == "after" {
			afterID = args[2]
			if afterID == moveID {
				exit(1, "move: cannot move a lap after itself")
			}
		}

		task := findScopedTask(ctx, moveID)
		if task == nil {
			exitIfOutOfScope(beadsDir, repoRoot, ctx, moveID)
			exit(1, "move: task %s not found", moveID)
		}
		if task.IsDone {
			exit(1, "move: task %s (%s) is already done", task.ID, task.Title)
		}
		if afterID != "" && findScopedTask(ctx, afterID) == nil {
			exitIfOutOfScope(beadsDir, repoRoot, ctx, afterID)
		}

		exitCode := 0
		var output string
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, task, args)

		order, fallbackHead, err := store.ComputeInsertOrder(file, position, afterID)
		if err != nil {
			if errors.Is(err, store.ErrTaskNotFound) {
				exitCode = 3
				exit(3, "move: task %s not found", afterID)
			}
			exitCode = 2
			exit(2, "move: %v", err)
		}
		if fallbackHead {
			fmt.Fprintf(os.Stderr, "laps: lap %s already complete; added to next available spot (head).\n", afterID)
		}

		oldOrder := task.Order
		task.Order = order
		task.UpdatedAt = time.Now().UTC()
		if err := store.Save(path, file); err != nil {
			exitCode = 2
			exit(2, "move: %v", err)
		}

		detail := map[string]interface{}{
			"position": position,
			"from":     oldOrder,
			"to":       task.Order,
		}
		if position == "after" {
			detail["after"] = afterID
		}
		logEvent(beadsDir, &eventlog.Entry{
			Event:    "moved",
			Cmd:      "move",
			Lap:      task.ID,
			Title:    task.Title,
			Assignee: task.Assignee,
			Detail:   detail,
		})

		output = task.ID
		if jsonOutput {
			printJSON(map[string]interface{}{"task": task})
		} else {
			fmt.Println(task.ID)
		}
	},
}

func init() {
	addScopeFlags(moveCmd)
	rootCmd.AddCommand(moveCmd)
}
