package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ErrStore is a generic store error.
var ErrStore = errors.New("store error")

// ErrEmptyState indicates the default store is missing or empty while other
// candidate task files exist.
var ErrEmptyState = errors.New("empty state")

// ErrEmptyFile indicates the file does not exist or is empty.
var ErrEmptyFile = errors.New("empty file")

// ErrTaskNotFound indicates a referenced task id does not exist.
var ErrTaskNotFound = errors.New("task not found")

const defaultFileName = "laps.json"

// CurrentVersion is the latest on-disk schema version. Files with a lower
// version are migrated automatically; files with a higher version are rejected.
const CurrentVersion = 2

// orderStep is the gap between adjacent todo order keys. New head/tail laps step
// by this amount and "after" inserts take the midpoint of the surrounding gap.
// When a gap is exhausted, todos are renumbered back to multiples of orderStep.
const orderStep = 1 << 16

// Task represents a single task record.
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Assignee    string     `json:"assignee,omitempty"`
	IsDone      bool       `json:"isDone"`
	Order       int        `json:"order"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt"`
}

// File is the top-level envelope stored on disk.
type File struct {
	Version int    `json:"version"`
	Tasks   []Task `json:"tasks"`
}

// DiscoverRepoRoot walks up from the current working directory looking for a
// .laps/ directory. If a .git directory is encountered first, the walk stops
// and .laps/ is created next to .git. If neither is found up to the
// filesystem root, an error is returned.
func DiscoverRepoRoot() (repoRoot, beadsDir string, err error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("%w: get working directory: %v", ErrStore, err)
	}

	for {
		beadsPath := filepath.Join(dir, ".laps")
		if info, err := os.Stat(beadsPath); err == nil {
			if info.IsDir() {
				return dir, beadsPath, nil
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "", fmt.Errorf("%w: stat %s: %v", ErrStore, beadsPath, err)
		}

		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			if mkErr := os.MkdirAll(beadsPath, 0o755); mkErr != nil {
				return "", "", fmt.Errorf("%w: create .laps directory: %v", ErrStore, mkErr)
			}
			return dir, beadsPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", "", fmt.Errorf("%w: no .git or .laps directory found in any ancestor", ErrStore)
}

// ResolveFile normalises a user-provided task file name.
// An empty string resolves to the default "laps.json". The .json suffix is
// appended only when not already present.
func ResolveFile(f string) string {
	if f == "" {
		return defaultFileName
	}
	if strings.HasSuffix(f, ".json") {
		return f
	}
	return f + ".json"
}

// Load reads and unmarshals a task file.
// It validates that the file contains only the expected fields and structure.
// Files containing only {} or whitespace are treated as empty.
func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrEmptyFile
		}
		return nil, fmt.Errorf("%w: read file %s: %w", ErrStore, path, err)
	}

	content := strings.TrimSpace(string(b))
	if len(content) == 0 || content == "{}" {
		return nil, ErrEmptyFile
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var raw struct {
		Version *int   `json:"version"`
		Tasks   []Task `json:"tasks"`
	}
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: file %s exists but is not a valid laps task file: %v", ErrStore, path, err)
	}

	if raw.Version == nil {
		return nil, fmt.Errorf("%w: file %s exists but is not a valid laps task file (missing version)", ErrStore, path)
	}
	if raw.Tasks == nil {
		return nil, fmt.Errorf("%w: file %s exists but is not a valid laps task file (missing tasks)", ErrStore, path)
	}

	return &File{Version: *raw.Version, Tasks: raw.Tasks}, nil
}

// Save marshals and writes a task file, creating parent directories if needed.
func Save(path string, data *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%w: create directory for %s: %v", ErrStore, path, err)
	}
	// Normalise nil slices so we write [] instead of null.
	if data.Tasks == nil {
		data.Tasks = []Task{}
	}
	// Apply the canonical ordering invariant on every write: done laps above
	// todo laps, done by completedAt (oldest first), todos by their order key.
	Normalize(data)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: marshal JSON: %v", ErrStore, err)
	}
	b = append(b, '\n')

	tmpPath := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return fmt.Errorf("%w: write temp file: %v", ErrStore, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		if rmErr := os.Remove(tmpPath); rmErr != nil {
			return fmt.Errorf("%w: rename temp file to %s: %v (cleanup: %v)", ErrStore, path, err, rmErr)
		}
		return fmt.Errorf("%w: rename temp file to %s: %v", ErrStore, path, err)
	}
	return nil
}

// Normalize rewrites f.Tasks into canonical order without depending on the
// current array order: all done laps first (sorted by completedAt ascending,
// nil first, tie-broken by order then id), then all todo laps (sorted by order,
// tie-broken by id). It builds a fresh slice from copies, leaving the original
// backing array untouched so callers holding a &f.Tasks[i] pointer stay valid.
func Normalize(f *File) {
	done := make([]Task, 0, len(f.Tasks))
	todo := make([]Task, 0, len(f.Tasks))
	for i := range f.Tasks {
		if f.Tasks[i].IsDone {
			done = append(done, f.Tasks[i])
		} else {
			todo = append(todo, f.Tasks[i])
		}
	}
	sort.SliceStable(done, func(i, j int) bool { return doneLess(&done[i], &done[j]) })
	sort.SliceStable(todo, func(i, j int) bool {
		if todo[i].Order != todo[j].Order {
			return todo[i].Order < todo[j].Order
		}
		return todo[i].ID < todo[j].ID
	})
	result := make([]Task, 0, len(done)+len(todo))
	result = append(result, done...)
	result = append(result, todo...)
	f.Tasks = result
}

// doneLess orders two done laps: oldest completedAt first, nil completedAt first,
// tie-broken deterministically by order then id.
func doneLess(a, b *Task) bool {
	switch {
	case a.CompletedAt == nil && b.CompletedAt == nil:
	case a.CompletedAt == nil:
		return true
	case b.CompletedAt == nil:
		return false
	case !a.CompletedAt.Equal(*b.CompletedAt):
		return a.CompletedAt.Before(*b.CompletedAt)
	}
	if a.Order != b.Order {
		return a.Order < b.Order
	}
	return a.ID < b.ID
}

// Migrate upgrades an older file in place to CurrentVersion, returning whether
// any change was made. The version 1 -> 2 step assigns explicit integer order
// keys to every lap in current array order, preserving today's todo ordering.
func Migrate(f *File) bool {
	if f.Version >= CurrentVersion {
		return false
	}
	for i := range f.Tasks {
		f.Tasks[i].Order = (i + 1) * orderStep
	}
	f.Version = CurrentVersion
	return true
}

// minTodoOrder returns the smallest order key among todo laps.
func minTodoOrder(f *File) (int, bool) {
	best, found := 0, false
	for i := range f.Tasks {
		if f.Tasks[i].IsDone {
			continue
		}
		if !found || f.Tasks[i].Order < best {
			best, found = f.Tasks[i].Order, true
		}
	}
	return best, found
}

// maxTodoOrder returns the largest order key among todo laps.
func maxTodoOrder(f *File) (int, bool) {
	best, found := 0, false
	for i := range f.Tasks {
		if f.Tasks[i].IsDone {
			continue
		}
		if !found || f.Tasks[i].Order > best {
			best, found = f.Tasks[i].Order, true
		}
	}
	return best, found
}

// nextTodoOrder returns the smallest todo order key strictly greater than after.
func nextTodoOrder(f *File, after int) (int, bool) {
	next, found := 0, false
	for i := range f.Tasks {
		if f.Tasks[i].IsDone || f.Tasks[i].Order <= after {
			continue
		}
		if !found || f.Tasks[i].Order < next {
			next, found = f.Tasks[i].Order, true
		}
	}
	return next, found
}

// renumberTodos reassigns todo order keys to evenly spaced multiples of
// orderStep, preserving their current relative order. Used when an "after"
// insert finds no integer gap between two adjacent todos.
func renumberTodos(f *File) {
	idx := make([]int, 0, len(f.Tasks))
	for i := range f.Tasks {
		if !f.Tasks[i].IsDone {
			idx = append(idx, i)
		}
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ta, tb := f.Tasks[idx[a]], f.Tasks[idx[b]]
		if ta.Order != tb.Order {
			return ta.Order < tb.Order
		}
		return ta.ID < tb.ID
	})
	for rank, i := range idx {
		f.Tasks[i].Order = (rank + 1) * orderStep
	}
}

// ComputeInsertOrder returns the order key for a new todo lap inserted at the
// given position ("head", "tail", or "after"). For "after", afterID names an
// existing lap; if it names a done lap, fallbackHead is true and a head order is
// returned. It may renumber todos in place when an adjacent gap is exhausted.
// Returns ErrTaskNotFound when afterID names no existing lap.
func ComputeInsertOrder(f *File, position, afterID string) (order int, fallbackHead bool, err error) {
	switch position {
	case "tail":
		if hi, ok := maxTodoOrder(f); ok {
			return hi + orderStep, false, nil
		}
		return orderStep, false, nil
	case "head":
		if lo, ok := minTodoOrder(f); ok {
			return lo - orderStep, false, nil
		}
		return orderStep, false, nil
	case "after":
		target := findTask(f, afterID)
		if target == nil {
			return 0, false, ErrTaskNotFound
		}
		if target.IsDone {
			order, _, err = ComputeInsertOrder(f, "head", "")
			return order, true, err
		}
		next, hasNext := nextTodoOrder(f, target.Order)
		if hasNext && next-target.Order <= 1 {
			renumberTodos(f)
			target = findTask(f, afterID)
			next, hasNext = nextTodoOrder(f, target.Order)
		}
		if !hasNext {
			return target.Order + orderStep, false, nil
		}
		return target.Order + (next-target.Order)/2, false, nil
	default:
		return 0, false, fmt.Errorf("%w: invalid position %q", ErrStore, position)
	}
}

// findTask returns a pointer to the lap with the given id, or nil.
func findTask(f *File, id string) *Task {
	for i := range f.Tasks {
		if f.Tasks[i].ID == id {
			return &f.Tasks[i]
		}
	}
	return nil
}

// CheckDefaultStore verifies that laps.json is a valid laps task file if it exists.
// A missing or empty laps.json is allowed (it will be initialised on demand).
func CheckDefaultStore(beadsDir string) error {
	path := filepath.Join(beadsDir, defaultFileName)
	_, err := Load(path)
	if err == nil || errors.Is(err, ErrEmptyFile) {
		return nil
	}
	return err
}

// GenerateID creates a task ID according to the spec.
//
// The prefix is derived from the base name of repoRoot: first 4 lowercase
// alphanumeric characters, padded with 'x'. The hash is a SHA-256 hex digest
// of "title|createdAt|description[:200]". If the generated ID collides with
// an entry in existingIDs, the hash slice is extended by one character until
// unique.
func GenerateID(repoRoot, title string, createdAt time.Time, description string, existingIDs map[string]struct{}) (string, error) {
	prefix := normalizePrefix(filepath.Base(repoRoot))
	input := title + "|" + createdAt.Format(time.RFC3339) + "|" + truncate(description, 200)
	sum := sha256.Sum256([]byte(input))
	hexStr := hex.EncodeToString(sum[:])

	for length := 4; length <= len(hexStr); length++ {
		id := prefix + "-" + hexStr[:length]
		if _, exists := existingIDs[id]; !exists {
			return id, nil
		}
	}

	return "", errors.New("could not generate unique ID")
}

func normalizePrefix(name string) string {
	var out []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			out = append(out, r)
			if len(out) >= 4 {
				break
			}
		}
	}
	s := strings.ToLower(string(out))
	for len(s) < 4 {
		s += "x"
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
