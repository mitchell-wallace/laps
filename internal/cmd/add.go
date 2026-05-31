package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var (
	addTitle       string
	addDescription string
	addAssignee    string
	addJSON        string
	addStdin       bool
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
  --title "..." --stdin [--assignee "..."]   Read description from stdin.

Prints the new task's id on success.`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)

		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil, args)

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
		if !flagMode && addJSON == "" && !addStdin {
			exitCode = 1
			exit(1, "add: --title or --json or --stdin is required")
		}
		if addJSON != "" && (flagMode || addStdin) {
			exitCode = 1
			exit(1, "add: --json is mutually exclusive with --title, --description, --assignee, and --stdin")
		}
		if addStdin && cmd.Flags().Changed("description") {
			exitCode = 1
			exit(1, "add: --stdin is mutually exclusive with --description")
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
		} else if addStdin {
			fi, err := os.Stdin.Stat()
			if err != nil {
				exitCode = 1
				exit(1, "add: cannot check stdin: %v", err)
			}
			if fi.Mode()&os.ModeCharDevice != 0 {
				exitCode = 1
				exit(1, "add: --stdin requires piped input, not a terminal")
			}
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				exitCode = 1
				exit(1, "add: reading stdin: %v", err)
			}
			title = addTitle
			description = strings.TrimRight(string(data), "\n")
			assignee = strings.TrimSpace(addAssignee)
		} else {
			title = addTitle
			description = strings.ReplaceAll(addDescription, "\\n", "\n")
			assignee = strings.TrimSpace(addAssignee)
		}

		if strings.TrimSpace(title) == "" {
			exitCode = 1
			exit(1, "add: title is required")
		}

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

		order, fallbackHead, err := store.ComputeInsertOrder(file, position, afterID)
		if err != nil {
			if errors.Is(err, store.ErrTaskNotFound) {
				exitCode = 3
				exit(3, "add: task %s not found", afterID)
			}
			exitCode = 2
			exit(2, "add: %v", err)
		}
		if fallbackHead {
			fmt.Fprintf(os.Stderr, "laps: lap %s already complete; added to next available spot (head).\n", afterID)
		}

		t := store.Task{
			ID:          id,
			Title:       title,
			Description: description,
			Assignee:    assignee,
			IsDone:      false,
			Order:       order,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		task = &t

		file.Tasks = append(file.Tasks, *task)

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
	addCmd.Flags().BoolVar(&addStdin, "stdin", false, "read description from stdin")
	rootCmd.AddCommand(addCmd)
}
