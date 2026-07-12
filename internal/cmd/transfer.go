package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var transferCmd = &cobra.Command{
	Use:   "transfer <root|stint> <task-id>...",
	Short: "Move tasks between the root queue and stints",
	Long: `Move one or more tasks atomically from the selected queue to another queue.

The source uses the normal scope flags: --root, --stint, or --active. With no
scope flag, the deepest active queue is selected. The target is "root" or the
name of an existing active stint. Task IDs and every task field are preserved.
Claimed tasks cannot be transferred.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		targetName := args[0]
		ids := args[1:]
		repoRoot, beadsDir, err := store.DiscoverRepoRoot()
		if err != nil {
			exit(2, "%v", err)
		}
		if err := store.CheckDefaultStore(beadsDir); err != nil {
			exit(2, "%v", err)
		}

		sourceCtx := transferSourceContext(beadsDir, repoRoot)
		targetCtx := transferTargetContext(beadsDir, repoRoot, targetName)
		if filepath.Clean(sourceCtx.Path) == filepath.Clean(targetCtx.Path) {
			exit(1, "transfer: source and target are both %s", targetCtx.Scope)
		}

		seen := make(map[string]struct{}, len(ids))
		movedByID := make(map[string]store.Task, len(ids))
		for _, id := range ids {
			if _, duplicate := seen[id]; duplicate {
				exit(1, "transfer: task %s was named more than once", id)
			}
			seen[id] = struct{}{}
			task := store.FindTask(sourceCtx.File, id)
			if task == nil {
				exitIfOutOfScope(beadsDir, repoRoot, sourceCtx, id)
				exit(3, "transfer: task %s not found in %s", id, sourceCtx.Scope)
			}
			if store.FindTask(targetCtx.File, id) != nil {
				exit(1, "transfer: target %s already contains task %s", targetCtx.Scope, id)
			}
			movedByID[id] = *task
		}

		claim, err := store.ReadClaim(beadsDir, fileNameForClaim(beadsDir, sourceCtx.Path))
		if err != nil {
			exit(2, "transfer: read claim: %v", err)
		}
		if _, movingClaimed := seen[claim.Lap]; movingClaimed && claimTargetsContext(beadsDir, claim, sourceCtx) {
			exit(1, "transfer: task %s is currently claimed; clear the claim before transferring it", claim.Lap)
		}

		moved := make([]store.Task, 0, len(ids))
		remaining := make([]store.Task, 0, len(sourceCtx.File.Tasks)-len(ids))
		for _, task := range sourceCtx.File.Tasks {
			if _, ok := seen[task.ID]; ok {
				moved = append(moved, task)
				continue
			}
			remaining = append(remaining, task)
		}
		sourceCtx.File.Tasks = remaining
		targetCtx.File.Tasks = append(targetCtx.File.Tasks, moved...)

		first := &moved[0]
		exitCode := 0
		output := strings.Join(ids, "\n")
		defer runAfterHooksDeferredScoped(cmd.Name(), beadsDir, sourceCtx.Path, sourceCtx.Scope, &first, &output, &exitCode, args)()
		runBeforeHooksScoped(cmd.Name(), beadsDir, sourceCtx.Path, sourceCtx.Scope, first, args)

		if err := store.SaveFilesAtomically(map[string]*store.File{
			sourceCtx.Path: sourceCtx.File,
			targetCtx.Path: targetCtx.File,
		}); err != nil {
			exitCode = 2
			exit(2, "transfer: %v", err)
		}

		for _, id := range ids {
			task := movedByID[id]
			logEvent(beadsDir, &eventlog.Entry{
				Event:    "transferred",
				Cmd:      "transfer",
				File:     fileNameForClaim(beadsDir, targetCtx.Path),
				Scope:    targetCtx.Scope,
				Lap:      task.ID,
				Title:    task.Title,
				Assignee: task.Assignee,
				Detail: map[string]interface{}{
					"from":     sourceCtx.Scope,
					"to":       targetCtx.Scope,
					"fromFile": fileNameForClaim(beadsDir, sourceCtx.Path),
					"toFile":   fileNameForClaim(beadsDir, targetCtx.Path),
				},
			})
		}

		if jsonOutput {
			printJSON(map[string]interface{}{"from": sourceCtx.Scope, "to": targetCtx.Scope, "tasks": moved})
			return
		}
		fmt.Println(output)
	},
}

func transferSourceContext(beadsDir, repoRoot string) *activeContext {
	if fileFlag != "" {
		exit(1, "transfer: --file is not supported; use --root or --stint for the source")
	}
	if scopeStint != "" {
		path, archived, err := existingStintPath(beadsDir, scopeStint)
		if err != nil {
			exit(2, "transfer: %v", err)
		}
		if path == "" {
			exit(3, "transfer: source stint %s not found", scopeStint)
		}
		if archived {
			exit(3, "transfer: source stint %s is archived", scopeStint)
		}
		file := loadFile(path, repoRoot, beadsDir)
		return &activeContext{Path: path, Scope: scopeStint, File: file, Head: firstTodo(file)}
	}

	rootPath := scopedRootPath(beadsDir)
	rootFile := loadFile(rootPath, repoRoot, beadsDir)
	if scopeRoot {
		return &activeContext{Path: rootPath, Scope: "root", File: rootFile, Head: firstTodo(rootFile)}
	}
	ctx, err := resolveSelectedContext(rootPath, repoRoot, beadsDir, rootFile)
	if err != nil {
		exit(2, "transfer: %v", err)
	}
	return ctx
}

func transferTargetContext(beadsDir, repoRoot, target string) *activeContext {
	if target == "root" {
		path := scopedRootPath(beadsDir)
		file := loadFile(path, repoRoot, beadsDir)
		return &activeContext{Path: path, Scope: "root", File: file, Head: firstTodo(file)}
	}
	path, archived, err := existingStintPath(beadsDir, target)
	if err != nil {
		exit(2, "transfer: %v", err)
	}
	if path == "" {
		exit(3, "transfer: target stint %s not found", target)
	}
	if archived {
		exit(3, "transfer: target stint %s is archived", target)
	}
	file := loadFile(path, repoRoot, beadsDir)
	return &activeContext{Path: path, Scope: target, File: file, Head: firstTodo(file)}
}

func claimTargetsContext(beadsDir string, claim store.Claim, ctx *activeContext) bool {
	claimPath, err := pathForClaim(beadsDir, claim)
	return err == nil && filepath.Clean(claimPath) == filepath.Clean(ctx.Path)
}

func init() {
	addScopeFlags(transferCmd)
	rootCmd.AddCommand(transferCmd)
}
