package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

const defaultFileName = "mb.json"

// Task represents a single task record.
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Assignee    string     `json:"assignee,omitempty"`
	IsDone      bool       `json:"isDone"`
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
// .beads/ directory. If a .git directory is encountered first, the walk stops
// and .beads/ is created next to .git. If neither is found up to the
// filesystem root, an error is returned.
func DiscoverRepoRoot() (repoRoot string, beadsDir string, err error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("%w: get working directory: %v", ErrStore, err)
	}

	for {
		beadsPath := filepath.Join(dir, ".beads")
		if info, err := os.Stat(beadsPath); err == nil && info.IsDir() {
			return dir, beadsPath, nil
		}

		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			if mkErr := os.MkdirAll(beadsPath, 0755); mkErr != nil {
				return "", "", fmt.Errorf("%w: create .beads directory: %v", ErrStore, mkErr)
			}
			return dir, beadsPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", "", fmt.Errorf("%w: no .git or .beads directory found in any ancestor", ErrStore)
}

// ResolveFile normalises a user-provided task file name.
// An empty string resolves to the default "mb.json". The .json suffix is
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
func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrEmptyFile
		}
		return nil, fmt.Errorf("%w: read file %s: %w", ErrStore, path, err)
	}

	if len(strings.TrimSpace(string(b))) == 0 {
		return nil, ErrEmptyFile
	}

	var file File
	if err := json.Unmarshal(b, &file); err != nil {
		return nil, fmt.Errorf("%w: parse JSON in %s: %v", ErrStore, path, err)
	}
	return &file, nil
}

// Save marshals and writes a task file, creating parent directories if needed.
func Save(path string, data *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("%w: create directory for %s: %v", ErrStore, path, err)
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: marshal JSON: %v", ErrStore, err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("%w: write file %s: %v", ErrStore, path, err)
	}
	return nil
}

// listCandidates returns the names of other *.json task files in beadsDir.
func listCandidates(beadsDir string) ([]string, error) {
	entries, err := os.ReadDir(beadsDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if name == defaultFileName || name == "mb-hooks.json" || name == "config.json" {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}

// CheckDefaultStore verifies that mb.json exists and contains at least one task.
// If mb.json is missing or empty and other candidate task files exist, it
// returns ErrEmptyState with a hint message.
func CheckDefaultStore(beadsDir string) error {
	path := filepath.Join(beadsDir, defaultFileName)
	data, err := os.ReadFile(path)
	var tasks []Task
	if err == nil {
		var file File
		if uerr := json.Unmarshal(data, &file); uerr == nil {
			tasks = file.Tasks
		}
	}

	if len(tasks) > 0 {
		return nil
	}

	candidates, _ := listCandidates(beadsDir)
	if len(candidates) == 0 {
		return nil
	}

	if err != nil {
		return fmt.Errorf("%w: default store mb.json not found; other task files: %s (use -f)", ErrEmptyState, strings.Join(candidates, ", "))
	}
	return fmt.Errorf("%w: default store mb.json has no tasks; other task files: %s (use -f)", ErrEmptyState, strings.Join(candidates, ", "))
}

// GenerateID creates a task ID according to the spec.
//
// The prefix is derived from the base name of repoRoot: first 4 lowercase
// alphanumeric characters, padded with 'x'. The hash is a SHA-256 hex digest
// of "title|createdAt|description[:200]". If the generated ID collides with
// an entry in existingIDs, the hash slice is extended by one character until
// unique.
func GenerateID(repoRoot string, title string, createdAt time.Time, description string, existingIDs map[string]struct{}) (string, error) {
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
