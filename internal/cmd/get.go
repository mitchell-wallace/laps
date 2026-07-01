package cmd

import (
	"fmt"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get [head|<id>]",
	Short: "Get a task by id or head",
	Long: `Get a task by id, or read the head task if no argument is given.

Output is title, blank line, description — nothing else.`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		target := "head"
		if len(args) > 0 {
			target = args[0]
		}

		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path, repoRoot, beadsDir)

		var flow *flowResolution
		var ctx *activeContext
		var err error
		if target == "head" {
			flow, err = resolveSelectedFlowStart(path, repoRoot, beadsDir, file, true)
		} else {
			flow, err = resolveSelectedFlowStart(path, repoRoot, beadsDir, file, false)
		}
		if err != nil {
			exit(2, "%v", err)
		}
		ctx = flow.Ctx
		path = ctx.Path

		var task *store.Task
		if target == "head" {
			if flow.State == queueStateLap {
				task = ctx.Head
			}
		} else {
			task = findScopedTask(ctx, target)
			if task == nil {
				exitIfOutOfScope(beadsDir, repoRoot, ctx, target)
			}
		}

		exitCode := 0
		var output string
		restoreCapture := captureExitCode(&exitCode)
		defer restoreCapture()
		defer runAfterHooksDeferredScoped(cmd.Name(), beadsDir, path, ctx.Scope, &task, &output, &exitCode, args)()
		runBeforeHooksScoped(cmd.Name(), beadsDir, path, ctx.Scope, task, args)

		if task == nil {
			if target == "head" {
				if flow.State == queueStateHeld {
					warnHeldGate(flow.Held)
				}
				exitState(exitForQueueState(flow.State))
			}
			exitCode = 3
			exit(3, "task not found")
		}
		if target != "head" {
			warnHeldGate(heldGateForContext(ctx))
		}

		output = formatTaskDetails(task)
		if jsonOutput {
			printJSON(map[string]interface{}{"task": task})
		} else {
			fmt.Println(output)
		}
	},
}

func init() {
	addScopeFlags(getCmd)
	rootCmd.AddCommand(getCmd)
}

func formatTaskDetails(task *store.Task) string {
	if task.Kind == store.KindStint {
		return fmt.Sprintf("%s/ (stint)", task.Ref)
	}
	if task.Assignee == "" {
		return fmt.Sprintf("%s\n\n%s", task.Title, task.Description)
	}
	return fmt.Sprintf("%s\nAssignee: %s\n\n%s", task.Title, task.Assignee, task.Description)
}
