package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchell-wallace/laps/internal/hooks"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var version string
var fileFlag string

func init() {
	rootCmd.PersistentFlags().StringVarP(&fileFlag, "file", "f", "", "task file name (without .laps/ path)")
}

var rootCmd = &cobra.Command{
	Use:   "laps",
	Short: "Laps — a minimal task tracker for AI coding agents",
	Long: `Laps (laps) is a minimal, single-binary task tracker for AI coding agents.
Tasks are a flat ordered queue with two states (todo / done). The agent's contract
is simple: read the head, do the work, mark it done.`,
	SilenceUsage: true,
}

type exitError struct {
	code int
}

func (e *exitError) Error() string { return fmt.Sprintf("exit %d", e.code) }

func exit(code int, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "laps: "+format+"\n", args...)
	panic(&exitError{code: code})
}

func Execute(v string) error {
	version = v
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(*exitError); ok {
				os.Exit(ee.code)
			}
			panic(r)
		}
	}()

	// Hook-only command handling
	cmdName, hookArgs, hookFileValue := splitArgs(os.Args[1:])
	if cmdName != "" && !isKnownCommand(cmdName) {
		repoRoot, beadsDir, err := store.DiscoverRepoRoot()
		if err != nil {
			exit(2, "%v", err)
		}
		hf, err := hooks.Load(beadsDir)
		if err != nil {
			exit(2, "%v", err)
		}
		path := filepath.Join(beadsDir, store.ResolveFile(hookFileValue))
		vars := map[string]string{
			"command":   cmdName,
			"file":      path,
			"exit_code": "",
			"output":    "",
			"args":      shellQuoteArgs(hookArgs),
		}
		for i, arg := range hookArgs {
			vars[fmt.Sprintf("%d", i+1)] = arg
		}
		passback, err := hooks.Dispatch(hf, cmdName, "before", vars, repoRoot)
		if err != nil {
			exit(4, "hook: %v", err)
		}
		if passback != "" {
			fmt.Print(passback)
		}
		passback, err = hooks.Dispatch(hf, cmdName, "after", vars, repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "laps: after hook: %v\n", err)
		}
		if passback != "" {
			fmt.Print(passback)
		}
		return nil
	}

	return rootCmd.Execute()
}

func getStorePath() (path, repoRoot, beadsDir string) {
	repoRoot, beadsDir, err := store.DiscoverRepoRoot()
	if err != nil {
		exit(2, "%v", err)
	}
	fileName := store.ResolveFile(fileFlag)
	path = filepath.Join(beadsDir, fileName)
	return path, repoRoot, beadsDir
}

func checkDefault(beadsDir string) {
	if fileFlag != "" {
		return
	}
	if err := store.CheckDefaultStore(beadsDir); err != nil {
		if errors.Is(err, store.ErrEmptyState) {
			exit(3, "%v", err)
		}
		exit(2, "%v", err)
	}
}

func loadFile(path string) *store.File {
	data, err := store.Load(path)
	if err != nil {
		if errors.Is(err, store.ErrEmptyFile) {
			f := &store.File{Version: 1, Tasks: []store.Task{}}
			if err := store.Save(path, f); err != nil {
				exit(2, "initialize file: %v", err)
			}
			return f
		}
		exit(2, "%v", err)
	}
	return data
}
