package cmd

import (
	"encoding/json"
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
var jsonOutput bool
var rootFlagsInitialized bool

func init() {
	ensureRootFlags()
}

func ensureRootFlags() {
	if rootFlagsInitialized {
		return
	}
	rootCmd.PersistentFlags().StringVarP(&fileFlag, "file", "f", "", "task file name (without .laps/ path)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json-output", false, "emit structured JSON output")
	rootFlagsInitialized = true
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
	msg := fmt.Sprintf(format, args...)
	if jsonOutput {
		b, _ := json.Marshal(map[string]interface{}{
			"error":    msg,
			"exitCode": code,
		})
		fmt.Fprintln(os.Stderr, string(b))
	} else {
		fmt.Fprintf(os.Stderr, "laps: %s\n", msg)
	}
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

	// Detect --json-output early for hook-only commands
	if isJSONOutput(os.Args[1:]) {
		jsonOutput = true
	}

	// Intercept --version before Cobra so we can respect jsonOutput.
	// Cobra's built-in --version handler prints plain text unconditionally.
	for _, a := range os.Args[1:] {
		if a == "--version" || a == "-v" {
			if jsonOutput {
				printJSON(map[string]interface{}{"version": version})
			} else {
				fmt.Println(version)
			}
			return nil
		}
	}

	// Hook-only command handling
	cmdName, hookArgs, hookFileValue := splitArgs(os.Args[1:])
	if cmdName != "" && !isKnownCommand(cmdName) {
		if scopeFlagName, ok := scopeFlagInArgs(os.Args[1:]); ok {
			return fmt.Errorf("unknown flag: %s", scopeFlagName)
		}
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
		if passback != "" && !jsonOutput {
			fmt.Print(passback)
		}
		passback, err = hooks.Dispatch(hf, cmdName, "after", vars, repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "laps: after hook: %v\n", err)
		}
		if passback != "" && !jsonOutput {
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
	path = scopedStorePath(beadsDir)
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

func loadFile(path, repoRoot, beadsDir string) *store.File {
	data, err := store.Load(path)
	if err != nil {
		if errors.Is(err, store.ErrEmptyFile) {
			f := &store.File{Version: store.CurrentVersion, Tasks: []store.Task{}}
			if name, ok := store.ActiveStintNameForPath(beadsDir, path); ok {
				prefix, err := store.AllocateStintPrefix(beadsDir, repoRoot, name)
				if err != nil {
					exit(2, "initialize file: %v", err)
				}
				f.Prefix = prefix
			}
			if err := store.Save(path, f); err != nil {
				exit(2, "initialize file: %v", err)
			}
			return f
		}
		exit(2, "%v", err)
	}
	if name, ok := store.ActiveStintNameForPath(beadsDir, path); ok && data.Prefix == "" {
		if len(data.Tasks) > 0 {
			exit(2, "file %s is a stint file but is missing prefix metadata; recreate it with `laps stints new %s` or add a 4-character prefix", path, name)
		}
		prefix, err := store.AllocateStintPrefix(beadsDir, repoRoot, name)
		if err != nil {
			exit(2, "initialize file: %v", err)
		}
		data.Prefix = prefix
		if err := store.Save(path, data); err != nil {
			exit(2, "initialize file: %v", err)
		}
	}
	if data.Version > store.CurrentVersion {
		exit(2, "file %s was written by a newer version of laps (schema version %d); please update laps", path, data.Version)
	}
	if store.Migrate(data) {
		if err := store.Save(path, data); err != nil {
			exit(2, "migrate file: %v", err)
		}
	}
	// Present canonical order to every command regardless of the on-disk array
	// order, so head selection never depends on how the file happens to be laid
	// out (e.g. after a hand edit or merge).
	store.Normalize(data)
	return data
}

func printJSON(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		exit(2, "json marshal: %v", err)
	}
	fmt.Println(string(b))
}
