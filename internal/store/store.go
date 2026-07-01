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
const CurrentVersion = 3

const (
	// KindLap is the default queue-entry kind. Missing kind values are treated
	// as laps for backward compatibility with schema v1/v2 files.
	KindLap = "lap"
	// KindStint identifies a queue entry that references a stint file.
	KindStint = "stint"

	stintsDirName       = "stints"
	stintArchiveDirName = "archive"
	stintFileSuffix     = ".laps.json"
)

// orderStep is the gap between adjacent todo order keys. New head/tail laps step
// by this amount and "after" inserts take the midpoint of the surrounding gap.
// When a gap is exhausted, todos are renumbered back to multiples of orderStep.
const orderStep = 1 << 16

// Task represents a single task record.
type Task struct {
	Kind        string     `json:"kind,omitempty"`
	ID          string     `json:"id"`
	Ref         string     `json:"ref,omitempty"`
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
	Prefix  string `json:"prefix,omitempty"`
	// Held flags a non-archived stint as held. It folds into schema v3 and
	// defaults to false when absent. It has no effect on the root queue.
	Held  bool   `json:"held,omitempty"`
	Tasks []Task `json:"tasks"`
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

// StintsDir returns the directory containing active stint files.
func StintsDir(beadsDir string) string {
	return filepath.Join(beadsDir, stintsDirName)
}

// StintArchiveDir returns the directory containing archived stint files.
func StintArchiveDir(beadsDir string) string {
	return filepath.Join(StintsDir(beadsDir), stintArchiveDirName)
}

// ValidateStintName rejects names that are paths or otherwise unsafe as file
// names below .laps/stints/.
func ValidateStintName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: stint name cannot be blank", ErrStore)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("%w: invalid stint name %q", ErrStore, name)
	}
	if strings.ContainsRune(name, os.PathSeparator) || filepath.Base(name) != name {
		return fmt.Errorf("%w: stint name must be a file name, not a path: %q", ErrStore, name)
	}
	return nil
}

// ResolveStintFile returns the active stint file path for name.
func ResolveStintFile(beadsDir, name string) (string, error) {
	if err := ValidateStintName(name); err != nil {
		return "", err
	}
	return filepath.Join(StintsDir(beadsDir), name+stintFileSuffix), nil
}

// ResolveArchivedStintFile returns the archived stint file path for name.
func ResolveArchivedStintFile(beadsDir, name string) (string, error) {
	if err := ValidateStintName(name); err != nil {
		return "", err
	}
	return filepath.Join(StintArchiveDir(beadsDir), name+stintFileSuffix), nil
}

// ActiveStintNameForPath returns the stint name when path identifies an active
// stint file directly under .laps/stints/.
func ActiveStintNameForPath(beadsDir, path string) (string, bool) {
	rel, err := filepath.Rel(StintsDir(beadsDir), path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", false
	}
	if filepath.Dir(rel) != "." || !strings.HasSuffix(rel, stintFileSuffix) {
		return "", false
	}
	name := strings.TrimSuffix(rel, stintFileSuffix)
	if err := ValidateStintName(name); err != nil {
		return "", false
	}
	return name, true
}

// CheckStintNameAvailable rejects a name that already exists as either an
// active or archived stint file.
func CheckStintNameAvailable(beadsDir, name string) error {
	active, err := ResolveStintFile(beadsDir, name)
	if err != nil {
		return err
	}
	if err := rejectExistingFile(active, "active stint"); err != nil {
		return err
	}

	archived, err := ResolveArchivedStintFile(beadsDir, name)
	if err != nil {
		return err
	}
	if err := rejectExistingFile(archived, "archived stint"); err != nil {
		return err
	}
	return nil
}

func rejectExistingFile(path, label string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w: %s file already exists: %s", ErrStore, label, path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: stat %s: %v", ErrStore, path, err)
	}
	return nil
}

// ArchiveStint moves an active stint file into the archive directory without
// overwriting an existing archived file.
func ArchiveStint(beadsDir, name string) error {
	src, dst, err := PrepareArchiveStint(beadsDir, name)
	if err != nil {
		return err
	}
	return ArchiveStintFile(src, dst)
}

// PrepareArchiveStint validates the active and archived paths, creates and
// checks the archive directory, and returns the paths to use for the final
// no-overwrite rename.
func PrepareArchiveStint(beadsDir, name string) (src, dst string, err error) {
	src, err = ResolveStintFile(beadsDir, name)
	if err != nil {
		return "", "", err
	}
	dst, err = ResolveArchivedStintFile(beadsDir, name)
	if err != nil {
		return "", "", err
	}
	if err := prepareMoveTarget(dst, "archived stint", "stint archive directory"); err != nil {
		return "", "", err
	}
	return src, dst, nil
}

// ArchiveStintFile moves src to dst, creating dst's parent directory and
// refusing to overwrite an existing file.
func ArchiveStintFile(src, dst string) error {
	if err := prepareMoveTarget(dst, "archived stint", "stint archive directory"); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("%w: archive stint %s to %s: %v", ErrStore, src, dst, err)
	}
	return nil
}

// RestoreArchivedStint moves an archived stint back to the active stints
// directory without overwriting an existing active file.
func RestoreArchivedStint(beadsDir, name string) error {
	src, err := ResolveArchivedStintFile(beadsDir, name)
	if err != nil {
		return err
	}
	dst, err := ResolveStintFile(beadsDir, name)
	if err != nil {
		return err
	}
	if err := prepareMoveTarget(dst, "active stint", "stints directory"); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("%w: restore archived stint %s to %s: %v", ErrStore, src, dst, err)
	}
	return nil
}

func prepareMoveTarget(dst, collisionLabel, dirLabel string) error {
	if err := rejectExistingFile(dst, collisionLabel); err != nil {
		return err
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("%w: create %s: %v", ErrStore, dirLabel, err)
	}
	if err := checkDirWritable(dir); err != nil {
		return fmt.Errorf("%w: %s is not writable: %v", ErrStore, dirLabel, err)
	}
	return nil
}

func checkDirWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".laps-write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	closeErr := f.Close()
	removeErr := os.Remove(name)
	if closeErr != nil {
		return closeErr
	}
	return removeErr
}

// ArchivedStintNameForPath returns the stint name when path identifies an
// archived stint file directly under .laps/stints/archive/.
func ArchivedStintNameForPath(beadsDir, path string) (string, bool) {
	rel, err := filepath.Rel(StintArchiveDir(beadsDir), path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", false
	}
	if filepath.Dir(rel) != "." || !strings.HasSuffix(rel, stintFileSuffix) {
		return "", false
	}
	name := strings.TrimSuffix(rel, stintFileSuffix)
	if err := ValidateStintName(name); err != nil {
		return "", false
	}
	return name, true
}

// QueueFilePaths returns the root queue plus all active and archived stint
// files in deterministic path order.
func QueueFilePaths(beadsDir string) ([]string, error) {
	paths := []string{filepath.Join(beadsDir, defaultFileName)}
	stintsDir := StintsDir(beadsDir)
	if _, err := os.Stat(stintsDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return paths, nil
		}
		return nil, fmt.Errorf("%w: stat %s: %v", ErrStore, stintsDir, err)
	}
	err := filepath.WalkDir(stintsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("%w: scan queue file %s: %v", ErrStore, path, err)
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), stintFileSuffix) {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths[1:])
	return paths, nil
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

	var envelope struct {
		Version *int `json:"version"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return nil, fmt.Errorf("%w: file %s exists but is not a valid laps task file: %v", ErrStore, path, err)
	}

	if envelope.Version == nil {
		return nil, fmt.Errorf("%w: file %s exists but is not a valid laps task file (missing version)", ErrStore, path)
	}
	if *envelope.Version > CurrentVersion {
		return nil, fmt.Errorf("file %s was written by a newer version of laps (schema version %d); please update laps", path, *envelope.Version)
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var raw struct {
		Version *int   `json:"version"`
		Prefix  string `json:"prefix,omitempty"`
		Held    bool   `json:"held,omitempty"`
		Tasks   []Task `json:"tasks"`
	}
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: file %s exists but is not a valid laps task file: %v", ErrStore, path, err)
	}
	if raw.Tasks == nil {
		return nil, fmt.Errorf("%w: file %s exists but is not a valid laps task file (missing tasks)", ErrStore, path)
	}
	if err := validatePrefix(raw.Prefix); err != nil {
		return nil, fmt.Errorf("%w: file %s exists but is not a valid laps task file: %v", ErrStore, path, err)
	}
	if err := normalizeTaskKinds(raw.Tasks); err != nil {
		return nil, fmt.Errorf("%w: file %s exists but is not a valid laps task file: %v", ErrStore, path, err)
	}

	return &File{Version: *raw.Version, Prefix: raw.Prefix, Held: raw.Held, Tasks: raw.Tasks}, nil
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
	if err := validatePrefix(data.Prefix); err != nil {
		return fmt.Errorf("%w: invalid task file: %v", ErrStore, err)
	}
	if err := normalizeTaskKinds(data.Tasks); err != nil {
		return fmt.Errorf("%w: invalid task file: %v", ErrStore, err)
	}
	// Apply the canonical ordering invariant on every write: done laps above
	// todo laps, done by completedAt (oldest first), todos by their order key.
	Normalize(data)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: marshal JSON: %v", ErrStore, err)
	}
	b = append(b, '\n')

	if err := SafeWriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("%w: %v", ErrStore, err)
	}
	return nil
}

// SafeWriteFile writes data to a temporary file, calls Sync to ensure it is committed to disk,
// closes the file, and then renames it atomically to path. It also attempts to sync the parent
// directory of the destination file.
func SafeWriteFile(path string, data []byte, perm os.FileMode) error {
	tmpPath := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	var success bool
	defer func() {
		if !success {
			_ = f.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	success = true

	// Sync parent directory to persist the rename. Ignore errors as some filesystems
	// or operating systems (like Windows) do not support directory syncing.
	if df, err := os.Open(filepath.Dir(path)); err == nil {
		_ = df.Sync()
		_ = df.Close()
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

func normalizeTaskKinds(tasks []Task) error {
	for i := range tasks {
		switch tasks[i].Kind {
		case "":
			tasks[i].Kind = KindLap
		case KindLap, KindStint:
		default:
			return fmt.Errorf("task %q has invalid kind %q", tasks[i].ID, tasks[i].Kind)
		}
	}
	return nil
}

// Migrate upgrades an older file in place to CurrentVersion, returning whether
// any change was made. The version 1 -> 2 step assigns explicit integer order
// keys to every entry in current array order, preserving today's todo ordering.
// The version 2 -> 3 step stamps missing kinds as laps.
func Migrate(f *File) bool {
	if f.Version >= CurrentVersion {
		return false
	}
	changed := false
	if f.Version < 2 {
		for i := range f.Tasks {
			f.Tasks[i].Order = (i + 1) * orderStep
		}
		f.Version = 2
		changed = true
	}
	if f.Version < 3 {
		for i := range f.Tasks {
			if f.Tasks[i].Kind == "" {
				f.Tasks[i].Kind = KindLap
			}
		}
		f.Version = 3
		changed = true
	}
	return changed
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

// FindTask returns a pointer to the lap with the given id, or nil.
func FindTask(f *File, id string) *Task {
	return findTask(f, id)
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

// GenerateID creates a task ID using the caller's containing-scope prefix.
//
// scopePrefix is normalized to the first 4 lowercase alphanumeric characters,
// padded with 'x'. The hash is a SHA-256 hex digest of
// "title|createdAt|description[:200]". If the generated ID collides with an
// entry in existingIDs, the hash slice is extended by one character until
// unique.
func GenerateID(scopePrefix, title string, createdAt time.Time, description string, existingIDs map[string]struct{}) (string, error) {
	prefix := normalizePrefix(scopePrefix)
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

// RepoPrefix returns the task-id prefix for root laps in repoRoot.
func RepoPrefix(repoRoot string) string {
	return normalizePrefix(filepath.Base(repoRoot))
}

// AllocateStintPrefix returns a deterministic unused 4-character prefix for a
// new stint name. It checks the repository prefix and prefixes recorded in all
// active and archived stint files.
func AllocateStintPrefix(beadsDir, repoRoot, name string) (string, error) {
	if err := ValidateStintName(name); err != nil {
		return "", err
	}

	used, err := ExistingPrefixes(beadsDir, repoRoot)
	if err != nil {
		return "", err
	}

	candidates := stintPrefixCandidates(name)
	for _, candidate := range candidates {
		if _, exists := used[candidate]; !exists {
			return candidate, nil
		}
	}

	base := normalizePrefix(name)
	alphabet := "0123456789abcdefghijklmnopqrstuvwxyz"
	for _, r := range alphabet {
		candidate := base[:3] + string(r)
		if _, exists := used[candidate]; !exists {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("%w: could not allocate unique prefix for stint %q", ErrStore, name)
}

// ExistingPrefixes returns the reserved id prefixes: the root repository prefix
// and every prefix recorded in active or archived stint files.
func ExistingPrefixes(beadsDir, repoRoot string) (map[string]struct{}, error) {
	used := map[string]struct{}{
		RepoPrefix(repoRoot): {},
	}

	owners, err := StintPrefixMap(beadsDir)
	if err != nil {
		return nil, err
	}
	for prefix := range owners {
		used[prefix] = struct{}{}
	}

	return used, nil
}

// StintPrefixMap returns an id-prefix to stint-name map for all active and
// archived stint files that carry prefix metadata.
func StintPrefixMap(beadsDir string) (map[string]string, error) {
	owners := map[string]string{}
	stintsDir := StintsDir(beadsDir)
	if _, err := os.Stat(stintsDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return owners, nil
		}
		return nil, fmt.Errorf("%w: stat %s: %v", ErrStore, stintsDir, err)
	}

	err := filepath.WalkDir(stintsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("%w: read stint prefix %s: %v", ErrStore, path, err)
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), stintFileSuffix) {
			return nil
		}
		data, err := Load(path)
		if err != nil {
			return fmt.Errorf("%w: read stint prefix %s: %v", ErrStore, path, err)
		}
		if data.Prefix == "" {
			return nil
		}
		if err := validatePrefix(data.Prefix); err != nil {
			return fmt.Errorf("%w: read stint prefix %s: %v", ErrStore, path, err)
		}
		name := strings.TrimSuffix(d.Name(), stintFileSuffix)
		owners[data.Prefix] = name
		return nil
	})
	if err != nil {
		return nil, err
	}
	return owners, nil
}

func validatePrefix(prefix string) error {
	if prefix == "" {
		return nil
	}
	if len(prefix) != 4 {
		return fmt.Errorf("prefix %q must be 4 lowercase alphanumeric characters", prefix)
	}
	for _, r := range prefix {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return fmt.Errorf("prefix %q must be 4 lowercase alphanumeric characters", prefix)
		}
	}
	return nil
}

func stintPrefixCandidates(name string) []string {
	base := normalizePrefix(name)
	chars := prefixChars(name)
	for len(chars) < 4 {
		chars = append(chars, 'x')
	}

	candidates := map[string]struct{}{base: {}}
	for start := 0; start+4 <= len(chars); start++ {
		candidates[string(chars[start:start+4])] = struct{}{}
	}

	limit := len(chars)
	if limit > 16 {
		limit = 16
	}
	for i := 0; i < limit; i++ {
		for j := 0; j < limit; j++ {
			if j == i {
				continue
			}
			for k := 0; k < limit; k++ {
				if k == i || k == j {
					continue
				}
				for l := 0; l < limit; l++ {
					if l == i || l == j || l == k {
						continue
					}
					candidates[string([]rune{chars[i], chars[j], chars[k], chars[l]})] = struct{}{}
				}
			}
		}
	}

	out := make([]string, 0, len(candidates))
	for candidate := range candidates {
		out = append(out, candidate)
	}
	sort.Strings(out)

	if out[0] == base {
		return out
	}
	for i, candidate := range out {
		if candidate == base {
			copy(out[1:i+1], out[:i])
			out[0] = base
			break
		}
	}
	return out
}

func prefixChars(name string) []rune {
	var out []rune
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out = append(out, r)
		}
	}
	return out
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
