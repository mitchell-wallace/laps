package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var (
	editTitle       string
	editDescription string
	editAssignee    string
)

var editCmd = &cobra.Command{
	Use:   "edit <id> [--title] [--description] [--assignee]",
	Short: "Edit fields of an existing task",
	Long: `Edit fields of an existing task in place, preserving its id and order.

At least one of --title, --description, or --assignee must be provided.
Fields are gated on whether the flag was set, so passing an empty value
clears that field:

  --title "..."        Set a new (non-blank) title.
  --description ""      Clear the description.
  --assignee ""         Clear the assignee.

Editing a completed lap succeeds with a warning and does not reopen it.
Prints the affected task id on success.`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fields := editFields{
			setTitle:       cmd.Flags().Changed("title"),
			setDescription: cmd.Flags().Changed("description"),
			setAssignee:    cmd.Flags().Changed("assignee"),
			title:          editTitle,
			description:    editDescription,
			assignee:       editAssignee,
		}
		if !fields.setTitle && !fields.setDescription && !fields.setAssignee {
			exit(1, "edit: at least one of --title, --description, or --assignee is required")
		}
		if len(args) < 1 {
			exit(1, "edit: a task id is required")
		}
		if len(args) > 1 {
			exit(1, "edit: usage: edit <id> [--title] [--description] [--assignee]")
		}
		runEdit(cmd, args, fields)
	},
}

// editFields captures which fields an edit/assign should change and their
// new values. The set* flags distinguish "clear this field" (set, empty
// value) from "leave it alone" (not set).
type editFields struct {
	setTitle, setDescription, setAssignee bool
	title, description, assignee          string
}

// runEdit applies fields to the task identified by args[0]. It is shared by
// the edit and assign commands; cmd.Name() drives the hook command name and
// error prefixes.
func runEdit(cmd *cobra.Command, args []string, fields editFields) {
	name := cmd.Name()
	path, repoRoot, beadsDir := getStorePath()
	checkDefault(beadsDir)
	file := loadFile(path, repoRoot, beadsDir)
	ctx, err := resolveSelectedContext(path, repoRoot, beadsDir, file)
	if err != nil {
		exit(2, "%v", err)
	}
	path = ctx.Path
	file = ctx.File

	if len(args) < 1 {
		exit(1, "%s: a task id is required", name)
	}
	id := args[0]

	task := findScopedTask(ctx, id)
	if task == nil {
		exitIfOutOfScope(beadsDir, repoRoot, ctx, id)
		exit(1, "%s: task %s not found", name, id)
	}

	exitCode := 0
	var output string
	defer runAfterHooksDeferred(name, beadsDir, path, &task, &output, &exitCode, args)()
	runBeforeHooks(name, beadsDir, path, task, args)

	if task.IsDone {
		fmt.Fprintf(os.Stderr, "laps: lap %s is already complete; editing in place without reopening.\n", task.ID)
	}

	if fields.setTitle {
		if strings.TrimSpace(fields.title) == "" {
			exitCode = 1
			exit(1, "%s: title must not be blank", name)
		}
		task.Title = fields.title
	}
	if fields.setDescription {
		task.Description = strings.ReplaceAll(fields.description, "\\n", "\n")
	}
	if fields.setAssignee {
		task.Assignee = strings.TrimSpace(fields.assignee)
	}

	task.UpdatedAt = time.Now().UTC()
	if err := store.Save(path, file); err != nil {
		exitCode = 2
		exit(2, "%s: %v", name, err)
	}

	var changedFields []string
	if fields.setTitle {
		changedFields = append(changedFields, "title")
	}
	if fields.setDescription {
		changedFields = append(changedFields, "description")
	}
	if fields.setAssignee {
		changedFields = append(changedFields, "assignee")
	}
	logEvent(beadsDir, &eventlog.Entry{
		Event:    "edited",
		Cmd:      name,
		Lap:      task.ID,
		Title:    task.Title,
		Assignee: task.Assignee,
		Detail:   map[string]interface{}{"fields": changedFields},
	})

	output = task.ID
	if jsonOutput {
		printJSON(map[string]interface{}{"task": task})
	} else {
		fmt.Println(task.ID)
	}
}

func init() {
	editCmd.Flags().StringVar(&editTitle, "title", "", "new task title")
	editCmd.Flags().StringVar(&editDescription, "description", "", "new task description (empty clears)")
	editCmd.Flags().StringVar(&editAssignee, "assignee", "", "new task assignee (empty clears)")
	addScopeFlags(editCmd)
	rootCmd.AddCommand(editCmd)
}
