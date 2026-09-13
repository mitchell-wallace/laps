// Package dispatchguard prevents a linked worktree from silently reusing
// another dispatch's .laps state after the worktree changes branches.
package dispatchguard

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	metadataName    = "dispatch.json"
	metadataVersion = 1
)

// Result describes any state transition performed by Ensure.
type Result struct {
	Branch    string
	Archived  string
	Protected bool
}

// StaleError identifies state which must not be consumed without --reset.
type StaleError struct {
	CurrentBranch  string
	RecordedBranch string
	Reason         string
}

func (e *StaleError) Error() string {
	if e.RecordedBranch == "" {
		return fmt.Sprintf("stale .laps dispatch on branch %q: %s; rerun with --reset to archive it", e.CurrentBranch, e.Reason)
	}
	return fmt.Sprintf("stale .laps dispatch: recorded branch %q does not match current branch %q; rerun with --reset to archive it", e.RecordedBranch, e.CurrentBranch)
}

type metadata struct {
	Version int    `json:"version"`
	Branch  string `json:"branch"`
}

// Ensure binds a linked worktree's .laps directory to its current branch.
// Primary worktrees remain repo-global and retain their historical behavior.
func Ensure(repoRoot, lapsDir string, reset bool, now time.Time) (Result, error) {
	linked, branch, err := linkedWorktreeBranch(repoRoot)
	if err != nil {
		return Result{}, err
	}
	if !linked {
		return Result{}, nil
	}

	result := Result{Branch: branch, Protected: true}
	recorded, exists, err := readMetadata(lapsDir)
	if err != nil {
		return result, err
	}

	var stale *StaleError
	switch {
	case exists && recorded.Branch != branch:
		stale = &StaleError{CurrentBranch: branch, RecordedBranch: recorded.Branch}
	case !exists:
		hasState, stateErr := hasDispatchState(lapsDir)
		if stateErr != nil {
			return result, stateErr
		}
		if hasState {
			stale = &StaleError{
				CurrentBranch: branch,
				Reason:        "non-empty state has no branch binding (likely inherited or left by an earlier worktree dispatch)",
			}
		}
	}

	if stale != nil {
		if !reset {
			return result, stale
		}
		archive, archiveErr := archiveState(lapsDir, now)
		if archiveErr != nil {
			return result, archiveErr
		}
		result.Archived = archive
	}

	if stale != nil || !exists {
		if err := writeMetadata(lapsDir, metadata{Version: metadataVersion, Branch: branch}); err != nil {
			return result, err
		}
	}
	return result, nil
}

func linkedWorktreeBranch(repoRoot string) (bool, string, error) {
	gitDir, err := gitOutput(repoRoot, "rev-parse", "--path-format=absolute", "--git-dir")
	if err != nil {
		if strings.Contains(err.Error(), "not a git repository") {
			return false, "", nil
		}
		return false, "", err
	}
	commonDir, err := gitOutput(repoRoot, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return false, "", err
	}
	if filepath.Clean(gitDir) == filepath.Clean(commonDir) {
		return false, "", nil
	}

	branch, err := gitOutput(repoRoot, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err == nil {
		return true, branch, nil
	}
	commit, commitErr := gitOutput(repoRoot, "rev-parse", "--verify", "HEAD")
	if commitErr != nil {
		return false, "", commitErr
	}
	return true, "detached@" + commit, nil
}

func gitOutput(repoRoot string, args ...string) (string, error) {
	cmdArgs := append([]string{"-C", repoRoot}, args...)
	out, err := exec.Command("git", cmdArgs...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect git worktree: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func readMetadata(lapsDir string) (metadata, bool, error) {
	path := filepath.Join(lapsDir, metadataName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return metadata{}, false, nil
	}
	if err != nil {
		return metadata{}, false, fmt.Errorf("read dispatch binding %s: %w", path, err)
	}
	var record metadata
	if err := json.Unmarshal(data, &record); err != nil {
		return metadata{}, false, fmt.Errorf("read dispatch binding %s: %w", path, err)
	}
	if record.Version != metadataVersion || strings.TrimSpace(record.Branch) == "" {
		return metadata{}, false, fmt.Errorf("invalid dispatch binding %s", path)
	}
	return record, true, nil
}

func writeMetadata(lapsDir string, record metadata) error {
	if err := os.MkdirAll(lapsDir, 0o755); err != nil {
		return fmt.Errorf("create laps directory: %w", err)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode dispatch binding: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(lapsDir, metadataName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write dispatch binding %s: %w", path, err)
	}
	return nil
}

func hasDispatchState(lapsDir string) (bool, error) {
	entries, err := os.ReadDir(lapsDir)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect laps directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == metadataName || name == "hooks.json" || name == ".gitignore" || strings.HasPrefix(name, "stale-") {
			continue
		}
		if name == "laps.json" {
			empty, emptyErr := emptyRootQueue(filepath.Join(lapsDir, name))
			if emptyErr != nil {
				return false, emptyErr
			}
			if empty {
				continue
			}
		}
		return true, nil
	}
	return false, nil
}

func emptyRootQueue(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("inspect %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return true, nil
	}
	var queue struct {
		Tasks []json.RawMessage `json:"tasks"`
	}
	if err := json.Unmarshal(data, &queue); err != nil {
		return false, fmt.Errorf("inspect %s: %w", path, err)
	}
	return len(queue.Tasks) == 0, nil
}

func archiveState(lapsDir string, now time.Time) (string, error) {
	entries, err := os.ReadDir(lapsDir)
	if err != nil {
		return "", fmt.Errorf("archive stale laps state: %w", err)
	}
	archive := filepath.Join(lapsDir, "stale-"+now.UTC().Format("20060102T150405Z"))
	if _, err := os.Stat(archive); err == nil {
		return "", fmt.Errorf("archive stale laps state: target already exists: %s", archive)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("archive stale laps state: inspect target: %w", err)
	}
	if err := os.Mkdir(archive, 0o755); err != nil {
		return "", fmt.Errorf("archive stale laps state: create %s: %w", archive, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "hooks.json" || name == ".gitignore" || strings.HasPrefix(name, "stale-") {
			continue
		}
		src := filepath.Join(lapsDir, name)
		dst := filepath.Join(archive, name)
		if err := os.Rename(src, dst); err != nil {
			return "", fmt.Errorf("archive stale laps state: move %s: %w", src, err)
		}
	}
	return archive, nil
}
