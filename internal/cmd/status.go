package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

// statusClaim is the claim slice of the status snapshot. claimedAt is a nullable
// RFC3339 UTC timestamp (null when no lap is claimed or the claim is a legacy
// bare-id with no recorded time); ageSeconds is the integer seconds elapsed
// since claimedAt (null whenever claimedAt is null). valid is false for a
// dangling claim (the claimed lap was deleted, completed, or belongs to another
// file); such claims are surfaced, never silently cleared.
type statusClaim struct {
	Valid      bool    `json:"valid"`
	Lap        string  `json:"lap"`
	File       string  `json:"file"`
	ClaimedAt  *string `json:"claimedAt"`
	AgeSeconds *int64  `json:"ageSeconds"`
}

// statusAssignee is one row of the todo-lap assignee breakdown.
type statusAssignee struct {
	Assignee string `json:"assignee"`
	Todo     int    `json:"todo"`
}

// statusCounts holds the todo/done/total lap counts.
type statusCounts struct {
	Todo  int `json:"todo"`
	Done  int `json:"done"`
	Total int `json:"total"`
}

// statusHead is the identity of the head (first todo) lap, or null when none.
type statusHead struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Assignee string `json:"assignee,omitempty"`
}

type statusStint struct {
	Name     string `json:"name"`
	Scope    string `json:"scope"`
	File     string `json:"file"`
	Todo     int    `json:"todo"`
	Done     int    `json:"done"`
	Total    int    `json:"total"`
	Queued   bool   `json:"queued"`
	Archived bool   `json:"archived"`
	Active   bool   `json:"active"`
}

// statusSnapshot is the stable JSON shape consumed by Rally. file is the
// selected task file identity; state is the queue taxonomy
// (active|ready|empty|complete).
type statusSnapshot struct {
	File        string           `json:"file"`
	State       string           `json:"state"`
	Counts      statusCounts     `json:"counts"`
	Head        *statusHead      `json:"head"`
	Claim       statusClaim      `json:"claim"`
	Assignees   []statusAssignee `json:"assignees"`
	ActiveStint *statusStint     `json:"activeStint"`
	Stints      []statusStint    `json:"stints"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show a snapshot of the lap queue",
	Long: `Report a snapshot of the current task file: todo/done counts, the head
(next todo) lap, the active (claimed) lap, the assignee breakdown of todo laps,
and the queue state.

Queue state is one of:
  active    a valid todo lap is claimed (work in progress)
  ready     todo laps exist and nothing valid is claimed
  empty     no laps exist
  complete  laps exist and all are done

A claim that points at a deleted, completed, or wrong-file lap yields a degraded
snapshot with claim.valid=false; it is reported, never silently cleared.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)
		file := loadFile(path, repoRoot, beadsDir)

		exitCode := 0
		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil, args)

		selectedFile := store.ResolveFile(fileFlag)
		activeCtx, err := resolveActiveContext(scopedRootPath(beadsDir), repoRoot, beadsDir, loadFile(scopedRootPath(beadsDir), repoRoot, beadsDir))
		if err != nil {
			exitCode = 2
			exit(2, "%v", err)
		}

		// A malformed claim is a real error, not a hidden-healthy snapshot: surface
		// it on the normal (non-zero) error path. A missing/empty claim reads back
		// as the zero Claim with a nil error.
		claim, err := store.ReadClaim(beadsDir, selectedFile)
		if err != nil {
			exitCode = 2
			exit(2, "read claim: %v", err)
		}

		// Counts, head, and the todo assignee breakdown over canonical order.
		var counts statusCounts
		var head *statusHead
		assigneeTodos := make(map[string]int)
		for i := range file.Tasks {
			t := &file.Tasks[i]
			counts.Total++
			if t.IsDone {
				counts.Done++
				continue
			}
			counts.Todo++
			if head == nil {
				head = &statusHead{ID: t.ID, Title: t.Title, Assignee: t.Assignee}
			}
			role := t.Assignee
			if role == "" {
				role = "unassigned"
			}
			assigneeTodos[role]++
		}

		// A claim is valid only when it names a lap in THIS file that is still a
		// todo. A different-file claim, a deleted/pruned lap, or a completed lap is
		// a dangling claim: surfaced as valid=false without auto-clearing.
		claimValid := false
		if claim.Lap != "" && claim.File == selectedFile {
			for i := range file.Tasks {
				if file.Tasks[i].ID == claim.Lap {
					claimValid = !file.Tasks[i].IsDone
					break
				}
			}
		}

		sc := statusClaim{
			Valid: claimValid,
			Lap:   claim.Lap,
			File:  claim.File,
		}
		if claim.ClaimedAt != nil {
			ts := claim.ClaimedAt.UTC().Format(time.RFC3339)
			sc.ClaimedAt = &ts
			age := int64(time.Since(*claim.ClaimedAt).Seconds())
			sc.AgeSeconds = &age
		}

		state := "ready"
		switch {
		case claimValid:
			state = "active"
		case counts.Total == 0:
			state = "empty"
		case counts.Todo == 0:
			state = "complete"
		}

		assignees := make([]statusAssignee, 0, len(assigneeTodos))
		for role, n := range assigneeTodos {
			assignees = append(assignees, statusAssignee{Assignee: role, Todo: n})
		}
		sort.Slice(assignees, func(i, j int) bool {
			return assignees[i].Assignee < assignees[j].Assignee
		})
		stints, activeStint, err := statusStints(beadsDir, repoRoot, activeCtx)
		if err != nil {
			exitCode = 2
			exit(2, "status: %v", err)
		}

		snapshot := statusSnapshot{
			File:        selectedFile,
			State:       state,
			Counts:      counts,
			Head:        head,
			Claim:       sc,
			Assignees:   assignees,
			ActiveStint: activeStint,
			Stints:      stints,
		}

		output = formatStatusHuman(&snapshot)
		if jsonOutput {
			printJSON(snapshot)
		} else {
			fmt.Println(output)
		}
	},
}

func formatStatusHuman(s *statusSnapshot) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("File: %s", s.File))
	lines = append(lines, fmt.Sprintf("State: %s", s.State))
	lines = append(lines, fmt.Sprintf("Laps: %d todo, %d done (%d total)", s.Counts.Todo, s.Counts.Done, s.Counts.Total))

	if s.Head != nil {
		lines = append(lines, fmt.Sprintf("Head: %s — %s", s.Head.ID, s.Head.Title))
	} else {
		lines = append(lines, "Head: none")
	}
	if s.ActiveStint != nil {
		lines = append(lines, fmt.Sprintf("Active stint: %s (%d todo, %d done, %d total)", s.ActiveStint.Scope, s.ActiveStint.Todo, s.ActiveStint.Done, s.ActiveStint.Total))
	}

	switch s.Claim.Lap {
	case "":
		lines = append(lines, "Claim: none")
	default:
		validity := "invalid"
		if s.Claim.Valid {
			validity = "valid"
		}
		claimLine := fmt.Sprintf("Claim: %s (%s)", s.Claim.Lap, validity)
		if s.Claim.ClaimedAt != nil {
			claimLine += fmt.Sprintf(", claimed %s", *s.Claim.ClaimedAt)
			if s.Claim.AgeSeconds != nil {
				claimLine += fmt.Sprintf(" (age %ds)", *s.Claim.AgeSeconds)
			}
		}
		lines = append(lines, claimLine)
	}

	if len(s.Assignees) > 0 {
		lines = append(lines, "Todo by assignee:")
		for _, a := range s.Assignees {
			lines = append(lines, fmt.Sprintf("- %s: %d", a.Assignee, a.Todo))
		}
	}

	return strings.Join(lines, "\n")
}

func statusStints(beadsDir, repoRoot string, activeCtx *activeContext) ([]statusStint, *statusStint, error) {
	rootFile, err := loadExistingFile(scopedRootPath(beadsDir), repoRoot, beadsDir)
	if err != nil {
		if !errors.Is(err, store.ErrEmptyFile) {
			return nil, nil, err
		}
		rootFile = &store.File{Version: store.CurrentVersion, Tasks: []store.Task{}}
	}
	queued := queuedStintNames(rootFile)
	activeScope := ""
	activeName := ""
	if activeCtx != nil && activeCtx.Scope != "" && activeCtx.Scope != "root" {
		activeScope = activeCtx.Scope
		parts := strings.Split(activeScope, "/")
		activeName = parts[len(parts)-1]
	}

	var stints []statusStint
	var active *statusStint
	if err := walkStintFiles(beadsDir, func(path, name string, archived bool) error {
		file, err := loadExistingFile(path, repoRoot, beadsDir)
		if err != nil {
			return err
		}
		var todo, done int
		for i := range file.Tasks {
			if file.Tasks[i].Kind != store.KindLap {
				continue
			}
			if file.Tasks[i].IsDone {
				done++
			} else {
				todo++
			}
		}
		scope := name
		isActive := !archived && name == activeName
		if isActive && activeScope != "" {
			scope = activeScope
		}
		row := statusStint{
			Name:     name,
			Scope:    scope,
			File:     fileNameForClaim(beadsDir, path),
			Todo:     todo,
			Done:     done,
			Total:    todo + done,
			Queued:   queued[name],
			Archived: archived,
			Active:   isActive,
		}
		stints = append(stints, row)
		if isActive {
			copy := row
			active = &copy
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}
	sort.Slice(stints, func(i, j int) bool {
		if stints[i].Name != stints[j].Name {
			return stints[i].Name < stints[j].Name
		}
		return !stints[i].Archived && stints[j].Archived
	})
	return stints, active, nil
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
