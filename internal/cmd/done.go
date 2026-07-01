package cmd

import (
	"fmt"
	"os"
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
		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path, repoRoot, beadsDir)

		var task *store.Task
		selectedFile := store.ResolveFile(fileFlag)
		var claimFile string
		var hookScope string
		var eventScope string

		if len(args) > 0 {
			id := args[0]
			ctx, err := resolveSelectedContext(path, repoRoot, beadsDir, file)
			if err != nil {
				exit(2, "%v", err)
			}
			path = ctx.Path
			file = ctx.File
			claimFile = fileNameForClaim(beadsDir, path)
			selectedFile = claimFile
			hookScope = ctx.Scope
			eventScope = ctx.Scope
			task = findScopedTask(ctx, id)
			if task == nil {
				exitIfOutOfScope(beadsDir, repoRoot, ctx, id)
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
				ctx, err := resolveActiveContext(path, repoRoot, beadsDir, file)
				if err != nil {
					exit(2, "%v", err)
				}
				task = ctx.Head
				if task == nil {
					exit(3, "no claimed lap and no head task")
				}
				exit(3, "no claimed lap. head task is %s: %s. use 'laps claim' or 'laps done %s'", task.ID, task.Title, task.ID)
			}

			claimFile = claim.File
			hookScope = normalizeClaimScope(claim)
			eventScope = hookScope
			path, err = pathForClaim(beadsDir, claim)
			if err != nil {
				exit(2, "read claim: %v", err)
			}
			file = loadFile(path, repoRoot, beadsDir)
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
		defer runAfterHooksDeferredScoped(hookCommandName(cmd), beadsDir, path, hookScope, &task, &output, &exitCode, args)()
		runBeforeHooksScoped(hookCommandName(cmd), beadsDir, path, hookScope, task, args)

		now := time.Now().UTC()
		task.IsDone = true
		task.CompletedAt = &now
		task.UpdatedAt = now
		drain, shouldDrain, err := prepareStintDrain(beadsDir, repoRoot, path, file)
		if err != nil {
			exitCode = 2
			exit(2, "done: %v", err)
		}
		if err := store.Save(path, file); err != nil {
			exitCode = 2
			exit(2, "done: %v", err)
		}
		if shouldDrain {
			if err := finishStintDrain(drain, now); err != nil {
				exitCode = 2
				exit(2, "done: %v", err)
			}
		}
		logEvent(beadsDir, &eventlog.Entry{
			Event:    "completed",
			Cmd:      "done",
			File:     claimFile,
			Scope:    eventScope,
			Lap:      task.ID,
			Title:    task.Title,
			Assignee: task.Assignee,
		})
		if shouldDrain {
			logEvent(beadsDir, &eventlog.Entry{
				Event: "stint.completed",
				Cmd:   "done",
				File:  store.ResolveFile(""),
				Scope: eventScope,
				Detail: map[string]interface{}{
					"stint": drain.Stint,
					"ref":   drain.RootRef.ID,
				},
			})
			logEvent(beadsDir, &eventlog.Entry{
				Event: "stint.archived",
				Cmd:   "done",
				File:  fileNameForClaim(beadsDir, drain.Dst),
				Scope: eventScope,
				Detail: map[string]interface{}{
					"stint": drain.Stint,
					"from":  fileNameForClaim(beadsDir, drain.Src),
					"to":    fileNameForClaim(beadsDir, drain.Dst),
				},
			})
		}

		// Best-effort: clearing the claim must not block a completed done. When the
		// claim is actually removed, emit a SEPARATE unclaimed event tagged
		// reason "completed", immediately after completed, for log uniformity with
		// the replaced reason. A failed remove emits no unclaimed event.
		if claim, err := store.ReadClaim(beadsDir, selectedFile); err == nil && claim.Lap == task.ID && claim.File == claimFile {
			if err := store.RemoveClaim(beadsDir); err == nil {
				logEvent(beadsDir, &eventlog.Entry{
					Event:    "unclaimed",
					Cmd:      "done",
					File:     claimFile,
					Scope:    eventScope,
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
		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)

		latest, err := findLatestCompletedTask(beadsDir, repoRoot, path)
		if err != nil {
			exit(2, "done undo: %v", err)
		}

		exitCode := 0
		var output string
		hookPath := path
		if latest != nil {
			hookPath = latest.Path
		}
		var latestTask *store.Task
		if latest != nil {
			latestTask = latest.Task
		}
		defer runAfterHooksDeferred(hookCommandName(cmd), beadsDir, hookPath, &latestTask, &output, &exitCode, args)()
		runBeforeHooks(hookCommandName(cmd), beadsDir, hookPath, latestTask, args)

		if latest == nil {
			exitCode = 3
			exit(3, "no completed task to undo")
		}

		age := time.Since(*latest.Task.CompletedAt)
		if age > UndoAgeLimit && !forceUndo {
			exitCode = 3
			exit(3, "last completed task %s (%s) was completed %v ago; use 'laps done undo -y' to force", latest.Task.ID, latest.Task.Title, age.Round(time.Second))
		}

		now := time.Now().UTC()
		if err := reopenLatestCompletion(beadsDir, latest, now); err != nil {
			exitCode = 2
			exit(2, "done undo: %v", err)
		}
		logEvent(beadsDir, &eventlog.Entry{
			Event:    "reopened",
			Cmd:      "done-undo",
			Lap:      latest.Task.ID,
			Title:    latest.Task.Title,
			Assignee: latest.Task.Assignee,
		})

		output = latest.Task.ID
		if jsonOutput {
			printJSON(map[string]interface{}{"task": latest.Task})
		} else {
			fmt.Printf("Done state cleared for %s (%s)\n", latest.Task.Title, latest.Task.ID)
		}
	},
}

type pendingStintDrain struct {
	RootPath string
	RootFile *store.File
	RootRef  *store.Task
	Stint    string
	Src      string
	Dst      string
}

func prepareStintDrain(beadsDir, repoRoot, path string, file *store.File) (*pendingStintDrain, bool, error) {
	stint, ok := store.ActiveStintNameForPath(beadsDir, path)
	if !ok || firstTodo(file) != nil {
		return nil, false, nil
	}

	rootPath := scopedRootPath(beadsDir)
	rootFile := loadFile(rootPath, repoRoot, beadsDir)
	rootRef := findStintRef(rootFile, stint)
	if rootRef == nil {
		return nil, false, nil
	}

	src, dst, err := store.PrepareArchiveStint(beadsDir, stint)
	if err != nil {
		return nil, false, err
	}
	return &pendingStintDrain{
		RootPath: rootPath,
		RootFile: rootFile,
		RootRef:  rootRef,
		Stint:    stint,
		Src:      src,
		Dst:      dst,
	}, true, nil
}

func finishStintDrain(drain *pendingStintDrain, completedAt time.Time) error {
	if err := store.ArchiveStintFile(drain.Src, drain.Dst); err != nil {
		return err
	}
	drain.RootRef.IsDone = true
	drain.RootRef.CompletedAt = &completedAt
	drain.RootRef.UpdatedAt = completedAt
	if err := store.Save(drain.RootPath, drain.RootFile); err != nil {
		drain.RootRef.IsDone = false
		drain.RootRef.CompletedAt = nil
		if rollbackErr := os.Rename(drain.Dst, drain.Src); rollbackErr != nil {
			return fmt.Errorf("save root ref after archive: %v; additionally failed to restore archived stint: %v", err, rollbackErr)
		}
		return err
	}
	return nil
}

func findStintRef(file *store.File, stint string) *store.Task {
	for i := range file.Tasks {
		if file.Tasks[i].Kind == store.KindStint && file.Tasks[i].Ref == stint {
			return &file.Tasks[i]
		}
	}
	return nil
}

type completedTaskCandidate struct {
	Path          string
	File          *store.File
	Task          *store.Task
	ArchivedStint string
}

func findLatestCompletedTask(beadsDir, repoRoot, rootPath string) (*completedTaskCandidate, error) {
	paths, err := store.QueueFilePaths(beadsDir)
	if err != nil {
		return nil, err
	}

	var latest *completedTaskCandidate
	for _, candidatePath := range paths {
		file, err := loadQueueFileForUndo(candidatePath, repoRoot, beadsDir, rootPath)
		if err != nil {
			return nil, err
		}
		archivedStint, _ := store.ArchivedStintNameForPath(beadsDir, candidatePath)
		for i := range file.Tasks {
			task := &file.Tasks[i]
			if task.Kind != store.KindLap || !task.IsDone || task.CompletedAt == nil {
				continue
			}
			if latest == nil || task.CompletedAt.After(*latest.Task.CompletedAt) || task.CompletedAt.Equal(*latest.Task.CompletedAt) && task.ID > latest.Task.ID {
				latest = &completedTaskCandidate{
					Path:          candidatePath,
					File:          file,
					Task:          task,
					ArchivedStint: archivedStint,
				}
			}
		}
	}
	return latest, nil
}

func loadQueueFileForUndo(path, repoRoot, beadsDir, rootPath string) (*store.File, error) {
	if path == rootPath {
		return loadFile(path, repoRoot, beadsDir), nil
	}
	file, err := loadExistingFile(path, repoRoot, beadsDir)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func reopenLatestCompletion(beadsDir string, latest *completedTaskCandidate, now time.Time) error {
	if latest.ArchivedStint != "" {
		rootPath := scopedRootPath(beadsDir)
		rootFile, err := store.Load(rootPath)
		if err != nil {
			return err
		}
		store.Normalize(rootFile)
		rootRef := findStintRef(rootFile, latest.ArchivedStint)
		if rootRef == nil {
			return fmt.Errorf("archived stint %s has no root reference to reopen", latest.ArchivedStint)
		}
		if err := store.RestoreArchivedStint(beadsDir, latest.ArchivedStint); err != nil {
			return err
		}
		rootRef.IsDone = false
		rootRef.CompletedAt = nil
		rootRef.UpdatedAt = now
		if err := store.Save(rootPath, rootFile); err != nil {
			return err
		}
		latest.Path, _ = store.ResolveStintFile(beadsDir, latest.ArchivedStint)
	}

	latest.Task.IsDone = false
	latest.Task.CompletedAt = nil
	latest.Task.UpdatedAt = now
	return store.Save(latest.Path, latest.File)
}

func hookCommandName(cmd *cobra.Command) string {
	if cmd.Parent() != nil && cmd.Parent() != rootCmd {
		return cmd.Parent().Name() + "-" + cmd.Name()
	}
	return cmd.Name()
}

func init() {
	addScopeFlags(doneCmd)
	rootCmd.AddCommand(doneCmd)
	doneCmd.AddCommand(doneUndoCmd)
	doneUndoCmd.Flags().BoolVarP(&forceUndo, "yes", "y", false, "force undo even if completed more than 5 minutes ago")
}
