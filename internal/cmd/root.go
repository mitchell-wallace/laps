package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	SilenceUsage:               true,
	SuggestionsMinimumDistance: 2,
}

type exitError struct {
	code int
}

func (e *exitError) Error() string { return fmt.Sprintf("exit %d", e.code) }

var capturedExitCode *int

func captureExitCode(exitCode *int) func() {
	previous := capturedExitCode
	capturedExitCode = exitCode
	return func() {
		capturedExitCode = previous
	}
}

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

func exitState(code int) {
	if capturedExitCode != nil {
		*capturedExitCode = code
	}
	if jsonOutput {
		printJSON(map[string]interface{}{
			"state":    queueStateForExitCode(code),
			"exitCode": code,
		})
	}
	panic(&exitError{code: code})
}

func queueStateForExitCode(code int) string {
	switch code {
	case 10:
		return string(queueStateHeld)
	case 11:
		return string(queueStateEmpty)
	case 12:
		return string(queueStateComplete)
	default:
		return "unknown"
	}
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

	// Hook-only command handling. Cobra's registered command tree is the
	// authoritative built-in registry; custom commands must be declared by a
	// hook before they bypass Cobra's unknown-command handling.
	cmdName, hookArgs, hookFileValue := splitArgs(os.Args[1:])
	if cmdName != "" && !builtinNames()[cmdName] {
		repoRoot, beadsDir, err := store.DiscoverRepoRoot()
		if err != nil {
			exit(2, "%v", err)
		}
		hf, declared, err := hooksDeclare(beadsDir, cmdName)
		if err != nil {
			exit(2, "%v", err)
		}
		if !declared {
			if jsonOutput {
				exit(1, "%s", unknownCommandMessage(cmdName))
			}
			return rootCmd.Execute()
		}
		if scopeFlagName, ok := scopeFlagInArgs(os.Args[1:]); ok {
			return fmt.Errorf("unknown flag: %s", scopeFlagName)
		}
		path := filepath.Join(beadsDir, store.ResolveFile(hookFileValue))
		vars := map[string]string{
			"command":   cmdName,
			"file":      path,
			"scope":     "root",
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

func builtinNames() map[string]bool {
	names := map[string]bool{
		"help":       true,
		"completion": true,
	}
	for _, command := range rootCmd.Commands() {
		names[command.Name()] = true
		for _, alias := range command.Aliases {
			names[alias] = true
		}
	}
	return names
}

func hooksDeclare(beadsDir, name string) (*hooks.File, bool, error) {
	hf, err := hooks.Load(beadsDir)
	if err != nil {
		return nil, false, err
	}
	for _, hook := range hf.Hooks {
		if hook.Command == name {
			return hf, true, nil
		}
	}
	return hf, false, nil
}

func unknownCommandMessage(name string) string {
	message := fmt.Sprintf("unknown command %q for %q", name, rootCmd.CommandPath())
	if suggestions := rootCmd.SuggestionsFor(name); len(suggestions) > 0 {
		message += "\n\nDid you mean this?\n\t" + suggestions[0]
	}
	return message
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

func isEmptyOrMissingFile(err error) bool {
	return errors.Is(err, store.ErrEmptyFile) || errors.Is(err, store.ErrFileNotFound)
}

func loadFile(path, repoRoot, beadsDir string) *store.File {
	return loadFileWithPolicy(path, repoRoot, beadsDir, false)
}

func loadFileCreating(path, repoRoot, beadsDir string) *store.File {
	return loadFileWithPolicy(path, repoRoot, beadsDir, true)
}

func loadFileWithPolicy(path, repoRoot, beadsDir string, createMissing bool) *store.File {
	data, err := store.Load(path)
	if err != nil {
		missing := errors.Is(err, store.ErrFileNotFound)
		defaultPath := filepath.Join(beadsDir, store.ResolveFile(""))
		if missing && !createMissing && filepath.Clean(path) != filepath.Clean(defaultPath) {
			name, relErr := filepath.Rel(beadsDir, path)
			if relErr != nil {
				name = path
			}
			hint := suggestTarget(beadsDir, fileFlag)
			exit(3, "task file %s not found%s", filepath.ToSlash(name), hint)
		}
		if missing || errors.Is(err, store.ErrEmptyFile) {
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

func suggestTarget(beadsDir, target string) string {
	cleaned := filepath.ToSlash(filepath.Clean(target))
	if cleaned == "." || cleaned == "" {
		return ""
	}
	if strings.HasPrefix(cleaned, "stints/") {
		name := strings.TrimSuffix(filepath.Base(cleaned), ".json")
		name = strings.TrimSuffix(name, ".laps")
		if name != "" {
			return fmt.Sprintf(" (did you mean --stint %s?)", name)
		}
	}
	if !strings.Contains(cleaned, "/") {
		name := strings.TrimSuffix(cleaned, ".json")
		name = strings.TrimSuffix(name, ".laps")
		path, err := store.ResolveStintFile(beadsDir, name)
		if err == nil {
			if _, statErr := os.Stat(path); statErr == nil {
				return fmt.Sprintf(" (did you mean --stint %s?)", name)
			}
		}
	}
	return ""
}

func printJSON(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		exit(2, "json marshal: %v", err)
	}
	fmt.Println(string(b))
}
