package cmd

import (
	"fmt"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var claimCmd = &cobra.Command{
	Use:   "claim [head|<id>]",
	Short: "Claim a task for the current session",
	Long: `Claim a task by id, or claim the head task if no argument is given.
Writes the claimed task id to .laps/claim so that 'laps done' knows which
task to complete.

Exit codes for a head claim (no explicit id):
  0   run: a lap was claimed
  10  stop-held: the next stint is held; start nothing new
  11  idle: the queue is empty
  12  finished: every lap is complete`,
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
		defer runAfterHooksDeferredScoped(hookCommandName(cmd), beadsDir, path, ctx.Scope, &task, &output, &exitCode, args)()
		runBeforeHooksScoped(hookCommandName(cmd), beadsDir, path, ctx.Scope, task, args)

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
			if gate := heldGateForContext(ctx); gate != nil {
				warnHeldGate(gate)
				exitState(10)
			}
		}

		selectedFile := fileNameForClaim(beadsDir, path)
		// Read the existing claim before overwriting it so we can tell a same-lap
		// reclaim (a log no-op that preserves claimedAt) from a different-lap
		// replacement (which retires the prior claim). A read error simply yields
		// the zero claim, treated as "no prior claim".
		existing, _ := store.ReadClaim(beadsDir, selectedFile)
		newClaim := store.Claim{Lap: task.ID, File: selectedFile, Scope: ctx.Scope}
		// A same-lap reclaim is exactly the case WriteClaim preserves claimedAt
		// for: same lap and same file. Anything else is a new/replacing claim.
		sameLapReclaim := existing.Lap == newClaim.Lap && existing.File == newClaim.File && normalizeClaimScope(existing) == normalizeClaimScope(newClaim)

		if err := store.WriteClaim(beadsDir, newClaim); err != nil {
			exitCode = 2
			exit(2, "claim: %v", err)
		}

		// Events are appended ONLY after WriteClaim succeeds. A same-lap reclaim
		// emits nothing (no duplicate claimed). Replacing a different claimed lap
		// emits unclaimed(replaced) for the prior lap, then claimed for the new.
		if !sameLapReclaim {
			if existing.Lap != "" {
				// The retired claim may belong to a DIFFERENT task file than the
				// one currently selected, so stamp its own file and denormalize
				// its title/assignee from that file rather than the loaded
				// (selected) file, which need not even contain the prior lap.
				prevTitle, prevAssignee := lapMeta(beadsDir, existing.File, existing.Lap)
				logEvent(beadsDir, &eventlog.Entry{
					Event:    "unclaimed",
					Cmd:      "claim",
					File:     existing.File,
					Scope:    normalizeClaimScope(existing),
					Lap:      existing.Lap,
					Title:    prevTitle,
					Assignee: prevAssignee,
					Detail:   map[string]interface{}{"reason": "replaced"},
				})
			}
			logScopedEvent(beadsDir, ctx, &eventlog.Entry{
				Event:    "claimed",
				Cmd:      "claim",
				Lap:      task.ID,
				Title:    task.Title,
				Assignee: task.Assignee,
			})
		}

		output = task.ID
		if jsonOutput {
			printJSON(map[string]interface{}{"task": task, "claimedId": task.ID})
		} else {
			fmt.Print(formatTaskDetails(task))
			fmt.Printf("\n\n-----\nNot the lap you intended to claim? Undo with 'laps claim undo'\n")
		}
	},
}

var claimUndoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Clear the claimed lap",
	Long:  `Clear the claimed lap stored in .laps/claim. Prints the id and title that was claimed.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()

		claim, err := store.ReadClaim(beadsDir, store.ResolveFile(fileFlag))
		if err != nil {
			exit(2, "read claim: %v", err)
		}
		claimedID := claim.Lap
		if claimedID == "" {
			exit(3, "no claimed lap to clear")
		}

		exitCode := 0
		var output string
		defer runAfterHooksDeferred(hookCommandName(cmd), beadsDir, path, nil, &output, &exitCode, args)()
		runBeforeHooks(hookCommandName(cmd), beadsDir, path, nil, args)

		// Denormalize title/assignee from the CLAIM's own file, which may differ
		// from the currently selected file. Fall back to the id when the lap can
		// no longer be resolved so the event still carries an identity.
		title, assignee := lapMeta(beadsDir, claim.File, claimedID)
		if title == "" {
			title = claimedID
		}

		if err := store.RemoveClaim(beadsDir); err != nil {
			exitCode = 2
			exit(2, "claim undo: %v", err)
		}
		// Append only after RemoveClaim succeeds; a failed remove emits no event.
		// Stamp the claim's own file, not the selected one, so a cross-file undo
		// records where the claim actually lived.
		logEvent(beadsDir, &eventlog.Entry{
			Event:    "unclaimed",
			Cmd:      "claim-undo",
			File:     claim.File,
			Scope:    normalizeClaimScope(claim),
			Lap:      claimedID,
			Title:    title,
			Assignee: assignee,
		})

		output = claimedID
		if jsonOutput {
			printJSON(map[string]interface{}{"unclaimedId": claimedID, "title": title})
		} else {
			fmt.Printf("Claim cleared for %s: %s\n", claimedID, title)
		}
	},
}

func init() {
	addScopeFlags(claimCmd)
	rootCmd.AddCommand(claimCmd)
	claimCmd.AddCommand(claimUndoCmd)
}
