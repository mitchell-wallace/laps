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
	addAssignee    string
	addJSON        string
)

var addCmd = &cobra.Command{
	Use:   "add <head|tail|after> [id]",
	Short: "Add a task to the queue",
	Long: `Add a task to the queue.

Positions:
  head          Insert at the front of the queue.
  tail          Append to the end of the queue.
  after <id>    Insert immediately after the specified task id.

Input modes (mutually exclusive):
  --title "..." [--description "..."] [--assignee "..."]   Provide task fields.
  --json '{"title":"...","description":"...","assignee":"..."}'   Provide task as JSON.

Prints the new task's id on success.`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		path, _, beadsDir := getStorePath()
		checkDefault(beadsDir)

		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil)

		if len(args) == 0 {
			exitCode = 1
			exit(1, "add: position required (head, tail, or after <id>)")
		}

		position := args[0]
		if position != "head" && position != "tail" && position != "after" {
			exitCode = 1
			exit(1, "add: position required (head, tail, or after <id>)")
		}

		var afterID string
		if position == "after" {
			if len(args) < 2 {
				exitCode = 1
				exit(1, "add: after requires a task id")
			}
			afterID = args[1]
		}

		flagMode := cmd.Flags().Changed("title") || cmd.Flags().Changed("description") || cmd.Flags().Changed("assignee")
		if !flagMode && addJSON == "" {
			exitCode = 1
			exit(1, "add: --title or --json is required")
		}
		if flagMode && addJSON != "" {
			exitCode = 1
			exit(1, "add: --json is mutually exclusive with --title, --description, and --assignee")
		}

		var title, description, assignee string
		if addJSON != "" {
			var payload struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Assignee    string `json:"assignee"`
			}
			if err := json.Unmarshal([]byte(addJSON), &payload); err != nil {
				exitCode = 1
				exit(1, "add: invalid json: %v", err)
			}
			title = payload.Title
			description = payload.Description
			assignee = strings.TrimSpace(payload.Assignee)
		} else {
			title = addTitle
			description = strings.ReplaceAll(addDescription, "\\n", "\n")
			assignee = strings.TrimSpace(addAssignee)
		}

		if strings.TrimSpace(title) == "" {
			exitCode = 1
			exit(1, "add: title is required")
		}

		_, repoRoot, _ := getStorePath()
		file := loadFile(path)
		existing := make(map[string]struct{}, len(file.Tasks))
		for _, t := range file.Tasks {
			existing[t.ID] = struct{}{}
		}

		now := time.Now().UTC()
		id, err := store.GenerateID(repoRoot, title, now, description, existing)
		if err != nil {
			exitCode = 2
			exit(2, "add: %v", err)
		}

		t := store.Task{
			ID:          id,
			Title:       title,
			Description: description,
			Assignee:    assignee,
			IsDone:      false,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		task = &t

		switch position {
		case "head":
			file.Tasks = append([]store.Task{*task}, file.Tasks...)
		case "tail":
			file.Tasks = append(file.Tasks, *task)
		case "after":
			found := false
			for i, t := range file.Tasks {
				if t.ID == afterID {
					file.Tasks = append(file.Tasks[:i+1], append([]store.Task{*task}, file.Tasks[i+1:]...)...)
					found = true
					break
				}
			}
			if !found {
				exitCode = 3
				exit(3, "add: task %s not found", afterID)
			}
		}

		if err := store.Save(path, file); err != nil {
			exitCode = 2
			exit(2, "add: %v", err)
		}
		output = id
		fmt.Println(id)
	},
}

func init() {
	addCmd.Flags().StringVar(&addTitle, "title", "", "task title")
	addCmd.Flags().StringVar(&addDescription, "description", "", "task description")
	addCmd.Flags().StringVar(&addAssignee, "assignee", "", "task assignee")
	addCmd.Flags().StringVar(&addJSON, "json", "", "task as json object")
	rootCmd.AddCommand(addCmd)
}
