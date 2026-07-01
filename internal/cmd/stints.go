package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var stintsRmForce bool

var stintsCmd = &cobra.Command{
	Use:     "stints",
	Aliases: []string{"st"},
	Short:   "Manage stints",
}

var stintsLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List stint files",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		repoRoot, beadsDir, err := store.DiscoverRepoRoot()
		if err != nil {
			exit(2, "%v", err)
		}
		stints, err := collectStintSummaries(beadsDir, repoRoot)
		if err != nil {
			exit(2, "stints ls: %v", err)
		}

		if jsonOutput {
			printJSON(map[string]interface{}{"stints": stints})
			return
		}
		for _, stint := range stints {
			fmt.Printf("%s\tlaps=%d\tqueued=%t\tarchived=%t\n", stint.Name, stint.Laps, stint.Queued, stint.Archived)
		}
	},
}

var stintsNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create an empty stint file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		repoRoot, beadsDir, err := store.DiscoverRepoRoot()
		if err != nil {
			exit(2, "%v", err)
		}
		if err := store.CheckStintNameAvailable(beadsDir, name); err != nil {
			exit(2, "stints new: %v", err)
		}
		prefix, err := store.AllocateStintPrefix(beadsDir, repoRoot, name)
		if err != nil {
			exit(2, "stints new: %v", err)
		}
		path, err := store.ResolveStintFile(beadsDir, name)
		if err != nil {
			exit(2, "stints new: %v", err)
		}
		file := &store.File{Version: store.CurrentVersion, Prefix: prefix, Tasks: []store.Task{}}
		if err := store.Save(path, file); err != nil {
			exit(2, "stints new: %v", err)
		}

		if jsonOutput {
			printJSON(map[string]interface{}{"name": name, "prefix": prefix, "path": path})
			return
		}
		fmt.Println(prefix)
	},
}

var stintsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a stint queue",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		repoRoot, beadsDir, err := store.DiscoverRepoRoot()
		if err != nil {
			exit(2, "%v", err)
		}
		path, archived, err := existingStintPath(beadsDir, name)
		if err != nil {
			exit(2, "stints show: %v", err)
		}
		if path == "" {
			exit(3, "stints show: stint %s not found", name)
		}
		file := loadFile(path, repoRoot, beadsDir)
		if jsonOutput {
			printJSON(map[string]interface{}{"name": name, "archived": archived, "tasks": file.Tasks})
			return
		}
		if archived {
			fmt.Printf("%s/ (archived)\n", name)
		} else {
			fmt.Printf("%s/\n", name)
		}
		for i := range file.Tasks {
			fmt.Println(formatListEntryWithContext(&file.Tasks[i], i+1, file.Tasks[i].IsDone, "", beadsDir, repoRoot))
		}
	},
}

var stintsRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a stint file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		repoRoot, beadsDir, err := store.DiscoverRepoRoot()
		if err != nil {
			exit(2, "%v", err)
		}
		if err := removeStint(beadsDir, repoRoot, name, stintsRmForce); err != nil {
			var refusal *stintRemoveRefusal
			if errors.As(err, &refusal) {
				exit(3, "stints rm: %v", err)
			}
			exit(2, "stints rm: %v", err)
		}
		if jsonOutput {
			printJSON(map[string]interface{}{"removed": name, "force": stintsRmForce})
			return
		}
		fmt.Println(name)
	},
}

type stintSummary struct {
	Name     string `json:"name"`
	Laps     int    `json:"laps"`
	Queued   bool   `json:"queued"`
	Archived bool   `json:"archived"`
}

type stintRemoveRefusal struct {
	reasons []string
}

func (e *stintRemoveRefusal) Error() string {
	return fmt.Sprintf("%s is protected (%s); use --force to remove it", e.reasons[0], strings.Join(e.reasons[1:], ", "))
}

var stintsEnqueueCmd = &cobra.Command{
	Use:   "enqueue <name> [head|tail|after <id>]",
	Short: "Enqueue a stint in the root queue",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		position := "tail"
		afterID := ""
		if len(args) > 1 {
			position = args[1]
		}
		switch position {
		case "head", "tail":
			if len(args) > 2 {
				exit(1, "stints enqueue: %s does not take an id", position)
			}
		case "after":
			if len(args) != 3 {
				exit(1, "stints enqueue: after requires a task id")
			}
			afterID = args[2]
		default:
			exit(1, "stints enqueue: position must be head, tail, or after <id>")
		}

		repoRoot, beadsDir, err := store.DiscoverRepoRoot()
		if err != nil {
			exit(2, "%v", err)
		}
		if err := store.CheckDefaultStore(beadsDir); err != nil {
			exit(2, "%v", err)
		}
		stintPath, err := store.ResolveStintFile(beadsDir, name)
		if err != nil {
			exit(2, "stints enqueue: %v", err)
		}
		if _, err := os.Stat(stintPath); err != nil {
			if os.IsNotExist(err) {
				exit(3, "stints enqueue: stint %s not found", name)
			}
			exit(2, "stints enqueue: %v", err)
		}

		rootPath := scopedRootPath(beadsDir)
		rootFile := loadFile(rootPath, repoRoot, beadsDir)
		if position == "after" && store.FindTask(rootFile, afterID) == nil {
			ctx := &activeContext{Path: rootPath, Scope: "root", File: rootFile, Head: firstTodo(rootFile)}
			exitIfOutOfScope(beadsDir, repoRoot, ctx, afterID)
		}

		order, fallbackHead, err := store.ComputeInsertOrder(rootFile, position, afterID)
		if err != nil {
			if errors.Is(err, store.ErrTaskNotFound) {
				exit(3, "stints enqueue: task %s not found", afterID)
			}
			exit(2, "stints enqueue: %v", err)
		}
		if fallbackHead {
			fmt.Fprintf(os.Stderr, "laps: lap %s already complete; enqueued at next available spot (head).\n", afterID)
		}

		now := time.Now().UTC()
		existing := make(map[string]struct{}, len(rootFile.Tasks))
		for _, task := range rootFile.Tasks {
			existing[task.ID] = struct{}{}
		}
		id, err := store.GenerateID(store.RepoPrefix(repoRoot), "stint:"+name, now, "", existing)
		if err != nil {
			exit(2, "stints enqueue: %v", err)
		}
		ref := store.Task{
			Kind:      store.KindStint,
			ID:        id,
			Ref:       name,
			Title:     name,
			IsDone:    false,
			Order:     order,
			CreatedAt: now,
			UpdatedAt: now,
		}
		rootFile.Tasks = append(rootFile.Tasks, ref)
		if err := store.Save(rootPath, rootFile); err != nil {
			exit(2, "stints enqueue: %v", err)
		}
		logEvent(beadsDir, &eventlog.Entry{
			Event: "stint.enqueued",
			Cmd:   "stints-enqueue",
			File:  store.ResolveFile(""),
			Scope: "root",
			Detail: map[string]interface{}{
				"stint":    name,
				"ref":      ref.ID,
				"position": position,
			},
		})

		if jsonOutput {
			printJSON(map[string]interface{}{"stint": name, "ref": ref})
			return
		}
		fmt.Println(id)
	},
}

func scopedRootPath(beadsDir string) string {
	return filepath.Join(beadsDir, store.ResolveFile(""))
}

func collectStintSummaries(beadsDir, repoRoot string) ([]stintSummary, error) {
	rootFile, err := loadExistingFile(scopedRootPath(beadsDir), repoRoot, beadsDir)
	if err != nil {
		if !errors.Is(err, store.ErrEmptyFile) {
			return nil, err
		}
		rootFile = &store.File{Version: store.CurrentVersion, Tasks: []store.Task{}}
	}
	queued := queuedStintNames(rootFile)

	var stints []stintSummary
	if err := walkStintFiles(beadsDir, func(path, name string, archived bool) error {
		file, err := loadExistingFile(path, repoRoot, beadsDir)
		if err != nil {
			return err
		}
		stints = append(stints, stintSummary{
			Name:     name,
			Laps:     len(file.Tasks),
			Queued:   queued[name],
			Archived: archived,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Slice(stints, func(i, j int) bool {
		if stints[i].Name != stints[j].Name {
			return stints[i].Name < stints[j].Name
		}
		return !stints[i].Archived && stints[j].Archived
	})
	return stints, nil
}

func walkStintFiles(beadsDir string, visit func(path, name string, archived bool) error) error {
	stintsDir := store.StintsDir(beadsDir)
	if _, err := os.Stat(stintsDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return filepath.WalkDir(stintsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".laps.json") {
			return nil
		}
		if name, ok := store.ActiveStintNameForPath(beadsDir, path); ok {
			return visit(path, name, false)
		}
		if name, ok := store.ArchivedStintNameForPath(beadsDir, path); ok {
			return visit(path, name, true)
		}
		return nil
	})
}

func queuedStintNames(rootFile *store.File) map[string]bool {
	queued := make(map[string]bool)
	for i := range rootFile.Tasks {
		if rootFile.Tasks[i].Kind == store.KindStint && rootFile.Tasks[i].Ref != "" {
			queued[rootFile.Tasks[i].Ref] = true
		}
	}
	return queued
}

func existingStintPath(beadsDir, name string) (path string, archived bool, err error) {
	activePath, err := store.ResolveStintFile(beadsDir, name)
	if err != nil {
		return "", false, err
	}
	if _, err := os.Stat(activePath); err == nil {
		return activePath, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}

	archivedPath, err := store.ResolveArchivedStintFile(beadsDir, name)
	if err != nil {
		return "", false, err
	}
	if _, err := os.Stat(archivedPath); err == nil {
		return archivedPath, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	return "", false, nil
}

func removeStint(beadsDir, repoRoot, name string, force bool) error {
	activePath, err := store.ResolveStintFile(beadsDir, name)
	if err != nil {
		return err
	}
	archivedPath, err := store.ResolveArchivedStintFile(beadsDir, name)
	if err != nil {
		return err
	}
	activeExists, err := fileExists(activePath)
	if err != nil {
		return err
	}
	archivedExists, err := fileExists(archivedPath)
	if err != nil {
		return err
	}
	if !activeExists && !archivedExists {
		return &stintRemoveRefusal{reasons: []string{name, "not found"}}
	}

	rootPath := scopedRootPath(beadsDir)
	rootFile := loadFile(rootPath, repoRoot, beadsDir)
	matchingRefs := findStintRefs(rootFile, name)
	claimMatches, err := claimMatchesStint(beadsDir, name, activePath)
	if err != nil {
		return err
	}
	if activeExists && !force {
		var reasons []string
		if hasTodoRef(matchingRefs) {
			reasons = append(reasons, "queued")
		}
		if isActiveStint(rootFile, name) {
			reasons = append(reasons, "active")
		}
		if claimMatches {
			reasons = append(reasons, "claimed")
		}
		if len(reasons) > 0 {
			return &stintRemoveRefusal{reasons: append([]string{name}, reasons...)}
		}
	}

	if activeExists {
		if err := os.Remove(activePath); err != nil {
			return err
		}
	}
	if archivedExists {
		if err := os.Remove(archivedPath); err != nil {
			return err
		}
	}
	if len(matchingRefs) > 0 && (force || archivedExists) {
		rootFile.Tasks = removeStintRefs(rootFile.Tasks, name)
		if err := store.Save(rootPath, rootFile); err != nil {
			return err
		}
	}
	if force && claimMatches {
		if err := store.RemoveClaim(beadsDir); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}

func findStintRefs(file *store.File, name string) []*store.Task {
	var refs []*store.Task
	for i := range file.Tasks {
		if file.Tasks[i].Kind == store.KindStint && file.Tasks[i].Ref == name {
			refs = append(refs, &file.Tasks[i])
		}
	}
	return refs
}

func hasTodoRef(refs []*store.Task) bool {
	for _, ref := range refs {
		if !ref.IsDone {
			return true
		}
	}
	return false
}

func isActiveStint(rootFile *store.File, name string) bool {
	head := firstTodo(rootFile)
	return head != nil && head.Kind == store.KindStint && head.Ref == name
}

func removeStintRefs(tasks []store.Task, name string) []store.Task {
	filtered := tasks[:0]
	for i := range tasks {
		if tasks[i].Kind == store.KindStint && tasks[i].Ref == name {
			continue
		}
		filtered = append(filtered, tasks[i])
	}
	return filtered
}

func claimMatchesStint(beadsDir, name, activePath string) (bool, error) {
	claim, err := store.ReadClaim(beadsDir, store.ResolveFile(""))
	if err != nil {
		return false, err
	}
	if claim.IsZero() {
		return false, nil
	}
	if normalizeClaimScope(claim) == name {
		return true, nil
	}
	if strings.HasSuffix(normalizeClaimScope(claim), "/"+name) {
		return true, nil
	}
	claimPath, err := pathForClaim(beadsDir, claim)
	if err == nil && filepath.Clean(claimPath) == filepath.Clean(activePath) {
		return true, nil
	}
	return false, nil
}

func init() {
	stintsCmd.AddCommand(stintsLsCmd)
	stintsCmd.AddCommand(stintsNewCmd)
	stintsCmd.AddCommand(stintsEnqueueCmd)
	stintsCmd.AddCommand(stintsShowCmd)
	stintsRmCmd.Flags().BoolVar(&stintsRmForce, "force", false, "remove a queued or claimed stint and clear matching state")
	stintsCmd.AddCommand(stintsRmCmd)
	rootCmd.AddCommand(stintsCmd)
}
