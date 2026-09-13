package dispatchguard

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLinkedWorktreeRefusesUnboundStateThenArchivesAndBinds(t *testing.T) {
	repo, linked := linkedFixture(t)
	lapsDir := filepath.Join(linked, ".laps")
	if err := os.MkdirAll(lapsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldQueue := `{"version":3,"tasks":[{"id":"oldx-1234","title":"old dispatch"}]}`
	if err := os.WriteFile(filepath.Join(lapsDir, "laps.json"), []byte(oldQueue), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lapsDir, "hooks.json"), []byte(`{"version":1,"hooks":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Ensure(linked, lapsDir, false, time.Now())
	var stale *StaleError
	if !errors.As(err, &stale) || !strings.Contains(err.Error(), "no branch binding") {
		t.Fatalf("Ensure unbound error = %v, want stale no-binding error", err)
	}

	now := time.Date(2026, 9, 13, 19, 0, 0, 0, time.UTC)
	result, err := Ensure(linked, lapsDir, true, now)
	if err != nil {
		t.Fatalf("Ensure reset: %v", err)
	}
	wantArchive := filepath.Join(lapsDir, "stale-20260913T190000Z")
	if result.Archived != wantArchive {
		t.Fatalf("archive = %q, want %q", result.Archived, wantArchive)
	}
	archived, err := os.ReadFile(filepath.Join(wantArchive, "laps.json"))
	if err != nil || string(archived) != oldQueue {
		t.Fatalf("archived queue = %q, %v", archived, err)
	}
	if _, err := os.Stat(filepath.Join(lapsDir, "hooks.json")); err != nil {
		t.Fatalf("hooks config must remain active: %v", err)
	}

	if _, err := Ensure(linked, lapsDir, false, time.Now()); err != nil {
		t.Fatalf("bound dispatch refused: %v", err)
	}

	git(t, repo, "branch", "dispatch-b")
	git(t, linked, "switch", "dispatch-b")
	_, err = Ensure(linked, lapsDir, false, time.Now())
	if !errors.As(err, &stale) || stale.RecordedBranch != "dispatch-a" || stale.CurrentBranch != "dispatch-b" {
		t.Fatalf("Ensure branch mismatch error = %#v, want dispatch-a -> dispatch-b", err)
	}
}

func TestPrimaryWorktreeRetainsRepoGlobalState(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	git(t, repo, "config", "user.name", "Laps Fixture")
	git(t, repo, "config", "user.email", "laps-fixture@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "tracked.txt")
	git(t, repo, "commit", "-q", "-m", "base")
	lapsDir := filepath.Join(repo, ".laps")
	if err := os.Mkdir(lapsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lapsDir, "laps.json"), []byte(`{"version":3,"tasks":[{}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Ensure(repo, lapsDir, false, time.Now())
	if err != nil || result.Protected {
		t.Fatalf("primary Ensure = %#v, %v; want unprotected success", result, err)
	}
}

func linkedFixture(t *testing.T) (repo, linked string) {
	t.Helper()
	repo = t.TempDir()
	git(t, repo, "init", "-q")
	git(t, repo, "config", "user.name", "Laps Fixture")
	git(t, repo, "config", "user.email", "laps-fixture@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "tracked.txt")
	git(t, repo, "commit", "-q", "-m", "base")
	git(t, repo, "branch", "dispatch-a")
	linked = filepath.Join(t.TempDir(), "linked")
	git(t, repo, "worktree", "add", "-q", linked, "dispatch-a")
	return repo, linked
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"-C", dir}, args...)
	if out, err := exec.Command("git", cmdArgs...).CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
