package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var stintsCmd = &cobra.Command{
	Use:     "stints",
	Aliases: []string{"st"},
	Short:   "Manage stints",
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

func init() {
	stintsCmd.AddCommand(stintsNewCmd)
	stintsCmd.AddCommand(stintsEnqueueCmd)
	rootCmd.AddCommand(stintsCmd)
}
