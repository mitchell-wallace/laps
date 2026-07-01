package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Show count and breakdown of laps",
	Long:  `Show the count of completed and total laps, along with a breakdown of complete and incomplete laps per role (assignee).`,
	Run: func(cmd *cobra.Command, args []string) {
		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path, repoRoot, beadsDir)

		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil, args)

		var doneCount, totalCount int
		type roleStats struct {
			complete   int
			incomplete int
		}
		stats := make(map[string]*roleStats)

		for _, t := range file.Tasks {
			role := t.Assignee
			if role == "" {
				role = "unassigned"
			}
			s, exists := stats[role]
			if !exists {
				s = &roleStats{}
				stats[role] = s
			}
			if t.IsDone {
				doneCount++
				s.complete++
			} else {
				s.incomplete++
			}
			totalCount++
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("Laps done: %d out of %d", doneCount, totalCount))
		if totalCount > 0 {
			lines = append(lines, "")
			lines = append(lines, "Breakdown by role:")

			var roles []string
			for r := range stats {
				roles = append(roles, r)
			}
			sort.Strings(roles)

			for _, r := range roles {
				s := stats[r]
				lines = append(lines, fmt.Sprintf("- %s: %d complete, %d incomplete", r, s.complete, s.incomplete))
			}
		} else {
			lines = append(lines, "")
			lines = append(lines, "No tasks found.")
		}

		output = strings.Join(lines, "\n")
		if jsonOutput {
			type roleBreakdown struct {
				Assignee   string `json:"assignee"`
				Complete   int    `json:"complete"`
				Incomplete int    `json:"incomplete"`
			}
			var breakdown []roleBreakdown
			var roles []string
			for r := range stats {
				roles = append(roles, r)
			}
			sort.Strings(roles)
			for _, r := range roles {
				s := stats[r]
				breakdown = append(breakdown, roleBreakdown{
					Assignee:   r,
					Complete:   s.complete,
					Incomplete: s.incomplete,
				})
			}
			if breakdown == nil {
				breakdown = []roleBreakdown{}
			}
			printJSON(map[string]interface{}{
				"done":      doneCount,
				"total":     totalCount,
				"breakdown": breakdown,
			})
		} else {
			fmt.Println(output)
		}
	},
}

func init() {
	addScopeFlags(countCmd)
	rootCmd.AddCommand(countCmd)
}
