package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mitchell-wallace/laps/internal/eventlog"
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
  --json '{"title":"...","description":"...","assignee":"..."}'   Provide one task as JSON.
  --json '[{"title":"..."},{"title":"..."}]'   Provide multiple tasks as JSON.
  --json -   Read one task or an array of tasks as JSON from stdin.
  --title "..." --stdin [--assignee "..."]   Read description from stdin.

Prints each new task id on success.`,
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

		var payloads []addPayload
		isBatch := false
		if addJSON != "" {
			jsonInput := []byte(addJSON)
			if addJSON == "-" {
				data, err := readPipedStdin("--json -")
				if err != nil {
					exitCode = 1
					exit(1, "add: %v", err)
				}
				jsonInput = data
			}
			var err error
			payloads, isBatch, err = parseAddJSON(jsonInput)
			if err != nil {
				exitCode = 1
				exit(1, "add: invalid json: %v", err)
			}
		} else if addStdin {
			data, err := readPipedStdin("--stdin")
			if err != nil {
				exitCode = 1
				exit(1, "add: %v", err)
			}
			payloads = []addPayload{{
				Title:       addTitle,
				Description: strings.TrimRight(string(data), "\n"),
				Assignee:    strings.TrimSpace(addAssignee),
			}}
		} else {
			payloads = []addPayload{{
				Title:       addTitle,
				Description: strings.ReplaceAll(addDescription, "\\n", "\n"),
				Assignee:    strings.TrimSpace(addAssignee),
			}}
		}

		for i := range payloads {
			payloads[i].Assignee = strings.TrimSpace(payloads[i].Assignee)
			if strings.TrimSpace(payloads[i].Title) == "" {
				exitCode = 1
				if isBatch {
					exit(1, "add: task %d title is required", i+1)
				}
				exit(1, "add: title is required")
			}
		}

		file := loadFile(path, repoRoot, beadsDir)
		ctx, err := resolveSelectedContext(path, repoRoot, beadsDir, file)
		if err != nil {
			exitCode = 2
			exit(2, "%v", err)
		}
		path = ctx.Path
		file = ctx.File
		if afterID != "" && findScopedTask(ctx, afterID) == nil {
			exitIfOutOfScope(beadsDir, repoRoot, ctx, afterID)
		}
		scopePrefix := store.RepoPrefix(repoRoot)
		if file.Prefix != "" {
			scopePrefix = file.Prefix
		}
		existing := make(map[string]struct{}, len(file.Tasks))
		for _, t := range file.Tasks {
			existing[t.ID] = struct{}{}
		}

		tasks := make([]store.Task, len(payloads))
		indices := insertionIndices(position, len(payloads))
		currentAfterID := afterID
		for _, payloadIndex := range indices {
			payload := payloads[payloadIndex]
			now := time.Now().UTC()
			id, err := store.GenerateID(scopePrefix, payload.Title, now, payload.Description, existing)
			if err != nil {
				exitCode = 2
				exit(2, "add: %v", err)
			}
			existing[id] = struct{}{}

			insertAfterID := currentAfterID
			order, fallbackHead, err := store.ComputeInsertOrder(file, position, insertAfterID)
			if err != nil {
				if errors.Is(err, store.ErrTaskNotFound) {
					exitCode = 3
					exit(3, "add: task %s not found", insertAfterID)
				}
				exitCode = 2
				exit(2, "add: %v", err)
			}
			if fallbackHead {
				fmt.Fprintf(os.Stderr, "laps: lap %s already complete; added to next available spot (head).\n", insertAfterID)
			}

			t := store.Task{
				ID:          id,
				Title:       payload.Title,
				Description: payload.Description,
				Assignee:    payload.Assignee,
				IsDone:      false,
				Order:       order,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			tasks[payloadIndex] = t
			file.Tasks = append(file.Tasks, t)
			if position == "after" {
				currentAfterID = id
			}
		}

		if err := store.Save(path, file); err != nil {
			exitCode = 2
			exit(2, "add: %v", err)
		}

		for i := range tasks {
			logEvent(beadsDir, &eventlog.Entry{
				Event:    "created",
				Cmd:      "add",
				Lap:      tasks[i].ID,
				Title:    tasks[i].Title,
				Assignee: tasks[i].Assignee,
				Detail:   map[string]interface{}{"position": position},
			})
		}

		ids := make([]string, len(tasks))
		for i := range tasks {
			ids[i] = tasks[i].ID
		}
		output = strings.Join(ids, "\n")
		if len(tasks) == 1 {
			task = &tasks[0]
		}
		if jsonOutput {
			if isBatch {
				printJSON(map[string]interface{}{"tasks": tasks})
			} else {
				printJSON(map[string]interface{}{"task": &tasks[0]})
			}
		} else {
			fmt.Println(output)
		}
	},
}

type addPayload struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Assignee    string `json:"assignee"`
}

func parseAddJSON(data []byte) (payloads []addPayload, isBatch bool, err error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, false, errors.New("empty input")
	}
	switch trimmed[0] {
	case '{':
		var payload addPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, false, err
		}
		return []addPayload{payload}, false, nil
	case '[':
		if err := json.Unmarshal(data, &payloads); err != nil {
			return nil, true, err
		}
		if len(payloads) == 0 {
			return nil, true, errors.New("task array must not be empty")
		}
		return payloads, true, nil
	default:
		return nil, false, errors.New("expected object or array")
	}
}

func readPipedStdin(flag string) ([]byte, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot check stdin: %v", err)
	}
	if fi.Mode()&os.ModeCharDevice != 0 {
		return nil, fmt.Errorf("%s requires piped input, not a terminal", flag)
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %v", err)
	}
	return data, nil
}

func insertionIndices(position string, count int) []int {
	indices := make([]int, count)
	for i := range indices {
		if position == "head" {
			indices[i] = count - 1 - i
		} else {
			indices[i] = i
		}
	}
	return indices
}

func init() {
	addCmd.Flags().StringVar(&addTitle, "title", "", "task title")
	addCmd.Flags().StringVar(&addDescription, "description", "", "task description")
	addCmd.Flags().StringVar(&addAssignee, "assignee", "", "task assignee")
	addCmd.Flags().StringVar(&addJSON, "json", "", "task as json object")
	addCmd.Flags().BoolVar(&addStdin, "stdin", false, "read description from stdin")
	addScopeFlags(addCmd)
	rootCmd.AddCommand(addCmd)
}
