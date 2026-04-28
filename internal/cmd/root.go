package cmd

import (
	"github.com/spf13/cobra"
)

var version string

var rootCmd = &cobra.Command{
	Use:   "mb",
	Short: "Microbeads — a minimal task tracker for AI coding agents",
	Long: `Microbeads (mb) is a minimal, single-binary task tracker for AI coding agents.
Tasks are a flat ordered queue with two states (todo / done). The agent's contract
is simple: read the head, do the work, mark it done.`,
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute(v string) error {
	version = v
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	return rootCmd.Execute()
}
