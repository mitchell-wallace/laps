package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mitchell-wallace/microbeads/internal/store"
	"github.com/spf13/cobra"
)

var (
	addTitle       string
	addDescription string
	addJSON        string
)

var addCmd = &cobra.Command{
	Use:   "add <head|tail|after> [id]",
	Short: "Add a task to the queue",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			exit(1, "add: position required (head, tail, or after <id>)")
		}

		position := args[0]
		if position != "head" && position != "tail" && position != "after" {
			exit(1, "add: position required (head, tail, or after <id>)")
		}

		var afterID string
		if position == "after" {
			if len(args) < 2 {
				exit(1, "add: after requires a task id")
			}
			afterID = args[1]
		}

		if addTitle == "" && addJSON == "" {
			exit(1, "add: --title or --json is required")
		}
		if addTitle != "" && addJSON != "" {
			exit(1, "add: --title and --json are mutually exclusive")
		}

		var title, description string
		if addJSON != "" {
			var payload struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal([]byte(addJSON), &payload); err != nil {
				exit(1, "add: invalid json: %v", err)
			}
			title = payload.Title
			description = payload.Description
		} else {
			title = addTitle
			description = strings.ReplaceAll(addDescription, "\\n", "\n")
		}

		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)

		file := loadFile(path)
		existing := make(map[string]struct{}, len(file.Tasks))
		for _, t := range file.Tasks {
			existing[t.ID] = struct{}{}
		}

		now := time.Now().UTC()
		id, err := store.GenerateID(repoRoot, title, now, description, existing)
		if err != nil {
			exit(2, "add: %v", err)
		}

		task := store.Task{
			ID:          id,
			Title:       title,
			Description: description,
			IsDone:      false,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		switch position {
		case "head":
			file.Tasks = append([]store.Task{task}, file.Tasks...)
		case "tail":
			file.Tasks = append(file.Tasks, task)
		case "after":
			found := false
			for i, t := range file.Tasks {
				if t.ID == afterID {
					file.Tasks = append(file.Tasks[:i+1], append([]store.Task{task}, file.Tasks[i+1:]...)...)
					found = true
					break
				}
			}
			if !found {
				exit(3, "add: task %s not found", afterID)
			}
		}

		if err := store.Save(path, file); err != nil {
			exit(2, "add: %v", err)
		}
		fmt.Println(id)
	},
}

func init() {
	addCmd.Flags().StringVar(&addTitle, "title", "", "task title")
	addCmd.Flags().StringVar(&addDescription, "description", "", "task description")
	addCmd.Flags().StringVar(&addJSON, "json", "", "task as json object")
	rootCmd.AddCommand(addCmd)
}
