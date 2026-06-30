package cmd

import (
	"fmt"

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

func init() {
	stintsCmd.AddCommand(stintsNewCmd)
	rootCmd.AddCommand(stintsCmd)
}
