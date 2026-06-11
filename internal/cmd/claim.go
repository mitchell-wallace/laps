package cmd

import (
	"fmt"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var claimCmd = &cobra.Command{
	Use:   "claim [head|<id>]",
	Short: "Claim a task for the current session",
	Long: `Claim a task by id, or claim the head task if no argument is given.
Writes the claimed task id to .laps/claim so that 'laps done' knows which
task to complete.`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		target := "head"
		if len(args) > 0 {
			target = args[0]
		}

		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path)

		var task *store.Task
		if target == "head" {
			for i := range file.Tasks {
				if !file.Tasks[i].IsDone {
					task = &file.Tasks[i]
					break
				}
			}
		} else {
			for i := range file.Tasks {
				if file.Tasks[i].ID == target {
					task = &file.Tasks[i]
					break
				}
			}
		}

		exitCode := 0
		var output string
		defer runAfterHooksDeferred(hookCommandName(cmd), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(hookCommandName(cmd), beadsDir, path, task, args)

		if task == nil {
			exitCode = 3
			if target == "head" {
				exit(3, "no head task")
			}
			exit(3, "task not found")
		}

		if err := store.WriteClaim(beadsDir, task.ID); err != nil {
			exitCode = 2
			exit(2, "claim: %v", err)
		}

		output = task.ID
		if jsonOutput {
			printJSON(map[string]interface{}{"task": task, "claimedId": task.ID})
		} else {
			fmt.Print(formatTaskDetails(task))
			fmt.Printf("\n\n-----\nNot the lap you intended to claim? Undo with 'laps claim undo'\n")
		}
	},
}

var claimUndoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Clear the claimed lap",
	Long:  `Clear the claimed lap stored in .laps/claim. Prints the id and title that was claimed.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()

		claimedID, err := store.ReadClaim(beadsDir)
		if err != nil {
			exit(2, "read claim: %v", err)
		}
		if claimedID == "" {
			exit(3, "no claimed lap to clear")
		}

		exitCode := 0
		var output string
		defer runAfterHooksDeferred(hookCommandName(cmd), beadsDir, path, nil, &output, &exitCode, args)()
		runBeforeHooks(hookCommandName(cmd), beadsDir, path, nil, args)

		title := claimedID
		file, err := store.Load(path)
		if err == nil {
			for i := range file.Tasks {
				if file.Tasks[i].ID == claimedID {
					title = file.Tasks[i].Title
					break
				}
			}
		}

		if err := store.RemoveClaim(beadsDir); err != nil {
			exitCode = 2
			exit(2, "claim undo: %v", err)
		}

		output = claimedID
		if jsonOutput {
			printJSON(map[string]interface{}{"unclaimedId": claimedID, "title": title})
		} else {
			fmt.Printf("Claim cleared for %s: %s\n", claimedID, title)
		}
	},
}

func init() {
	rootCmd.AddCommand(claimCmd)
	claimCmd.AddCommand(claimUndoCmd)
}
