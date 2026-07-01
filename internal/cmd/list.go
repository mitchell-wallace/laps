package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var (
	listAll     bool
	listDone    bool
	listOneline bool
	listTree    bool
)

var listCmd = &cobra.Command{
	Use:     "list [--all | --done]",
	Aliases: []string{"ls"},
	Short:   "List tasks",
	Long: `List tasks as a markdown numbered list.

Default shows todo tasks only, head first.
  --all    Include done tasks after todo items (struck through).
  --done   Show only completed tasks, most recent first.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path, repoRoot, beadsDir)
		hookScope := "root"
		if listTree {
			path = scopedRootPath(beadsDir)
			file = loadFile(path, repoRoot, beadsDir)
		} else {
			ctx, err := resolveSelectedContext(path, repoRoot, beadsDir, file)
			if err != nil {
				exit(2, "%v", err)
			}
			path = ctx.Path
			file = ctx.File
			hookScope = ctx.Scope
		}

		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferredScoped(cmd.Name(), beadsDir, path, hookScope, &task, &output, &exitCode, args)()
		runBeforeHooksScoped(cmd.Name(), beadsDir, path, hookScope, nil, args)

		claim, err := store.ReadClaim(beadsDir, fileNameForClaim(beadsDir, path))
		if err != nil {
			exit(2, "read claim: %v", err)
		}
		activeID := claim.Lap

		var lines []string
		var jsonTasks []store.Task
		if listTree {
			lines, jsonTasks, err = renderTreeList(beadsDir, repoRoot, path, file, activeID)
			if err != nil {
				exit(2, "list tree: %v", err)
			}
		} else if listDone {
			var done []int
			for i, t := range file.Tasks {
				if t.IsDone {
					done = append(done, i)
				}
			}
			sort.Slice(done, func(i, j int) bool {
				ai := file.Tasks[done[i]].CompletedAt
				aj := file.Tasks[done[j]].CompletedAt
				if ai == nil && aj == nil {
					return false
				}
				if ai == nil {
					return false
				}
				if aj == nil {
					return true
				}
				return ai.After(*aj)
			})
			for i, d := range done {
				t := file.Tasks[d]
				lines = append(lines, formatListEntryWithContext(&t, i+1, true, activeID, beadsDir, repoRoot))
				jsonTasks = append(jsonTasks, t)
			}
		} else {
			var todos []int
			var dones []int
			for i, t := range file.Tasks {
				if t.IsDone {
					dones = append(dones, i)
				} else {
					todos = append(todos, i)
				}
			}
			num := 1
			for _, idx := range todos {
				t := file.Tasks[idx]
				lines = append(lines, formatListEntryWithContext(&t, num, false, activeID, beadsDir, repoRoot))
				jsonTasks = append(jsonTasks, t)
				num++
			}
			if listAll {
				for _, idx := range dones {
					t := file.Tasks[idx]
					lines = append(lines, formatListEntryWithContext(&t, num, true, activeID, beadsDir, repoRoot))
					jsonTasks = append(jsonTasks, t)
					num++
				}
			}
		}

		output = strings.Join(lines, "\n")
		if jsonOutput {
			if jsonTasks == nil {
				jsonTasks = []store.Task{}
			}
			printJSON(map[string]interface{}{"tasks": jsonTasks})
		} else if output != "" {
			fmt.Println(output)
		}
	},
}

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "include done tasks")
	listCmd.Flags().BoolVar(&listDone, "done", false, "show only done tasks")
	listCmd.Flags().BoolVar(&listOneline, "oneline", false, "render each lap on a single line (prior format)")
	listCmd.Flags().BoolVar(&listTree, "tree", false, "render recursive stint overview")
	addScopeFlags(listCmd)
	rootCmd.AddCommand(listCmd)
}

func formatListEntryWithContext(t *store.Task, num int, done bool, activeID, beadsDir, repoRoot string) string {
	if t.Kind == store.KindStint {
		return formatStintListEntry(t, num, done, beadsDir, repoRoot)
	}
	if listOneline {
		if done {
			return fmt.Sprintf("%d. ~~%s~~", num, formatListTask(t))
		}
		return fmt.Sprintf("%d. %s", num, formatListTask(t))
	}

	prefix := fmt.Sprintf("%d. ", num)
	marker := ""
	if activeID != "" && t.ID == activeID {
		marker = "> "
	}
	title := t.Title
	if done {
		title = fmt.Sprintf("~~%s~~", title)
	}
	line1 := prefix + marker + title

	assignee := t.Assignee
	if assignee == "" {
		assignee = "—"
	}
	state := "todo"
	if t.IsDone {
		state = "done"
	}
	line2 := strings.Repeat(" ", len(prefix)) + fmt.Sprintf("%s · %s · %s", t.ID, assignee, state)

	return line1 + "\n" + line2
}

func formatStintListEntry(t *store.Task, num int, done bool, beadsDir, repoRoot string) string {
	name := t.Ref
	if name == "" {
		name = t.Title
	}
	count := "?"
	if beadsDir != "" {
		if n, err := stintLapCount(beadsDir, repoRoot, name); err == nil {
			count = fmt.Sprintf("%d", n)
		}
	}
	text := fmt.Sprintf("%s/ (stint - %s laps)", name, count)
	if done {
		text = fmt.Sprintf("~~%s~~", text)
	}
	return fmt.Sprintf("%d. %s", num, text)
}

func formatListTask(t *store.Task) string {
	text := fmt.Sprintf("%s — %s", t.ID, t.Title)
	if t.Assignee != "" {
		text += fmt.Sprintf(" (assignee: %s)", t.Assignee)
	}
	return text
}

func stintLapCount(beadsDir, repoRoot, name string) (int, error) {
	path, _, err := existingStintPath(beadsDir, name)
	if err != nil {
		return 0, err
	}
	if path == "" {
		return 0, store.ErrEmptyFile
	}
	file, err := loadExistingFile(path, repoRoot, beadsDir)
	if err != nil {
		return 0, err
	}
	return len(file.Tasks), nil
}

func renderTreeList(beadsDir, repoRoot, path string, file *store.File, activeID string) ([]string, []store.Task, error) {
	var lines []string
	var tasks []store.Task
	visited := make(map[string]struct{})
	if err := appendTreeList(&lines, &tasks, beadsDir, repoRoot, path, file, activeID, "", visited); err != nil {
		return nil, nil, err
	}
	return lines, tasks, nil
}

func appendTreeList(lines *[]string, tasks *[]store.Task, beadsDir, repoRoot, path string, file *store.File, activeID, indent string, visited map[string]struct{}) error {
	indexes := listIndexes(file)
	for num, idx := range indexes {
		t := file.Tasks[idx]
		*tasks = append(*tasks, t)
		entry := formatListEntryWithContext(&t, num+1, t.IsDone, activeID, beadsDir, repoRoot)
		if indent != "" {
			entry = indent + strings.ReplaceAll(entry, "\n", "\n"+indent)
		}
		*lines = append(*lines, entry)
		if t.Kind != store.KindStint || t.Ref == "" {
			continue
		}
		childPath, _, err := existingStintPath(beadsDir, t.Ref)
		if err != nil {
			return err
		}
		if childPath == "" {
			*lines = append(*lines, indent+"  (missing stint file)")
			continue
		}
		childIdentity := childPath
		if _, seen := visited[childIdentity]; seen {
			*lines = append(*lines, indent+"  (cycle detected)")
			continue
		}
		visited[childIdentity] = struct{}{}
		childFile, err := loadExistingFile(childPath, repoRoot, beadsDir)
		if err != nil {
			return err
		}
		if err := appendTreeList(lines, tasks, beadsDir, repoRoot, childPath, childFile, activeID, indent+"  ", visited); err != nil {
			return err
		}
		delete(visited, childIdentity)
	}
	return nil
}

func listIndexes(file *store.File) []int {
	var indexes []int
	if listDone {
		for i := range file.Tasks {
			if file.Tasks[i].IsDone {
				indexes = append(indexes, i)
			}
		}
		sort.Slice(indexes, func(i, j int) bool {
			ai := file.Tasks[indexes[i]].CompletedAt
			aj := file.Tasks[indexes[j]].CompletedAt
			if ai == nil && aj == nil {
				return false
			}
			if ai == nil {
				return false
			}
			if aj == nil {
				return true
			}
			return ai.After(*aj)
		})
		return indexes
	}
	for i := range file.Tasks {
		if !file.Tasks[i].IsDone {
			indexes = append(indexes, i)
		}
	}
	if listAll {
		for i := range file.Tasks {
			if file.Tasks[i].IsDone {
				indexes = append(indexes, i)
			}
		}
	}
	return indexes
}
