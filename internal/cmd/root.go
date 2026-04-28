package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchell-wallace/microbeads/internal/store"
	"github.com/spf13/cobra"
)

var version string
var fileFlag string

func init() {
	rootCmd.PersistentFlags().StringVarP(&fileFlag, "file", "f", "", "task file name (without .beads/ path)")
}

var rootCmd = &cobra.Command{
	Use:   "mb",
	Short: "Microbeads — a minimal task tracker for AI coding agents",
	Long: `Microbeads (mb) is a minimal, single-binary task tracker for AI coding agents.
Tasks are a flat ordered queue with two states (todo / done). The agent's contract
is simple: read the head, do the work, mark it done.`,
	SilenceUsage: true,
}

type exitError struct {
	code int
}

func (e *exitError) Error() string { return fmt.Sprintf("exit %d", e.code) }

func exit(code int, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "mb: "+format+"\n", args...)
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
	return rootCmd.Execute()
}

func getStorePath() (path string, repoRoot string, beadsDir string) {
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
		if errors.Is(err, os.ErrNotExist) {
			return &store.File{Version: 1}
		}
		exit(2, "%v", err)
	}
	return data
}
