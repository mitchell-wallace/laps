package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiscoverRepoRoot_BeadsExists(t *testing.T) {
	root := t.TempDir()
	beadsDir := filepath.Join(root, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	gotRoot, gotBeads, err := DiscoverRepoRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRoot != root {
		t.Errorf("repoRoot = %q, want %q", gotRoot, root)
	}
	if gotBeads != beadsDir {
		t.Errorf("beadsDir = %q, want %q", gotBeads, beadsDir)
	}
}

func TestDiscoverRepoRoot_CreateBeadsNextToGit(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	beadsDir := filepath.Join(root, ".laps")
	if _, err := os.Stat(beadsDir); !os.IsNotExist(err) {
		t.Fatal(".laps should not exist before discovery")
	}

	gotRoot, gotBeads, err := DiscoverRepoRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRoot != root {
		t.Errorf("repoRoot = %q, want %q", gotRoot, root)
	}
	if gotBeads != beadsDir {
		t.Errorf("beadsDir = %q, want %q", gotBeads, beadsDir)
	}
	info, err := os.Stat(beadsDir)
	if err != nil {
		t.Fatalf(".laps was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal(".laps is not a directory")
	}
}

func TestDiscoverRepoRoot_NoGitNoBeads(t *testing.T) {
	root := t.TempDir()
	if hasAncestorMarker(root, ".git") || hasAncestorMarker(root, ".laps") {
		t.Skip("temp dir ancestors contain .git or .laps; cannot assert no-repo discovery case")
	}
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	_, _, err := DiscoverRepoRoot()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrStore) {
		t.Errorf("expected ErrStore, got %v", err)
	}
}

func hasAncestorMarker(dir, name string) bool {
	dir = filepath.Clean(dir)
	for {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

func TestDiscoverRepoRoot_StopAtGit(t *testing.T) {
	parent := t.TempDir()
	parentBeads := filepath.Join(parent, ".laps")
	if err := os.MkdirAll(parentBeads, 0755); err != nil {
		t.Fatal(err)
	}

	child := filepath.Join(parent, "child")
	childGit := filepath.Join(child, ".git")
	if err := os.MkdirAll(childGit, 0755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(child, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	gotRoot, gotBeads, err := DiscoverRepoRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRoot != child {
		t.Errorf("repoRoot = %q, want %q", gotRoot, child)
	}
	expectedBeads := filepath.Join(child, ".laps")
	if gotBeads != expectedBeads {
		t.Errorf("beadsDir = %q, want %q", gotBeads, expectedBeads)
	}
}

func TestResolveFile(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "laps.json"},
		{"tasks", "tasks.json"},
		{"tasks.json", "tasks.json"},
		{"my-tasks", "my-tasks.json"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ResolveFile(tt.input)
			if got != tt.want {
				t.Errorf("ResolveFile(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	created := time.Date(2026, 4, 28, 10, 15, 0, 0, time.UTC)
	file := &File{
		Version: 1,
		Tasks: []Task{
			{
				ID:          "test-1234",
				Title:       "Test",
				Description: "A description",
				Assignee:    "alice",
				IsDone:      false,
				CreatedAt:   created,
				UpdatedAt:   created,
			},
		},
	}

	if err := Save(path, file); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Version != 1 {
		t.Errorf("Version = %d, want 1", loaded.Version)
	}
	if len(loaded.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want 1", len(loaded.Tasks))
	}
	if loaded.Tasks[0].ID != "test-1234" {
		t.Errorf("ID = %q, want test-1234", loaded.Tasks[0].ID)
	}
	if loaded.Tasks[0].Assignee != "alice" {
		t.Errorf("Assignee = %q, want alice", loaded.Tasks[0].Assignee)
	}
	if !loaded.Tasks[0].CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", loaded.Tasks[0].CreatedAt, created)
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSave_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "file.json")
	if err := Save(path, &File{Version: 1}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestGenerateID(t *testing.T) {
	created := time.Date(2026, 4, 28, 10, 15, 0, 0, time.UTC)
	id, err := GenerateID("MyProject", "Add list command", created, "Multi-line\ndescription supported.", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "mypr-474a"
	if id != want {
		t.Errorf("GenerateID = %q, want %q", id, want)
	}
}

func TestGenerateID_PrefixNormalization(t *testing.T) {
	tests := []struct {
		repo       string
		wantPrefix string
	}{
		{"ab", "abxx"},
		{"a-b", "abxx"},
		{"12345", "1234"},
		{"A1B2C3", "a1b2"},
		{"a.b.c", "abcx"},
		{"", "xxxx"},
	}
	created := time.Now()
	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			id, err := GenerateID(tt.repo, "t", created, "d", map[string]struct{}{})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(id, tt.wantPrefix+"-") {
				t.Errorf("id %q does not start with prefix %q", id, tt.wantPrefix)
			}
		})
	}
}

func TestGenerateID_CollisionExtension(t *testing.T) {
	repoRoot := "proj"
	created := time.Date(2026, 4, 28, 10, 15, 0, 0, time.UTC)
	title := "Task"
	desc := "Desc"

	existing := map[string]struct{}{
		"proj-cfe4":  {},
		"proj-cfe40": {},
	}

	id, err := GenerateID(repoRoot, title, created, desc, existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "proj-cfe401"
	if id != want {
		t.Errorf("GenerateID = %q, want %q", id, want)
	}
}

func TestCheckDefaultStore_MissingWithCandidates(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "other.json"), []byte(`{"version":1,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CheckDefaultStore(beadsDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckDefaultStore_EmptyWithCandidates(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "laps.json"), []byte(`{"version":1,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "other.json"), []byte(`{"version":1,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CheckDefaultStore(beadsDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_RejectMissingVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "other.json")
	if err := os.WriteFile(path, []byte(`{"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing version")
	}
	if !strings.Contains(err.Error(), "missing version") {
		t.Fatalf("expected missing version error, got: %v", err)
	}
}

func TestLoad_RejectMissingTasks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "other.json")
	if err := os.WriteFile(path, []byte(`{"version":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing tasks")
	}
	if !strings.Contains(err.Error(), "missing tasks") {
		t.Fatalf("expected missing tasks error, got: %v", err)
	}
}

func TestLoad_AcceptsValidEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "laps.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	file, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.Version != 1 {
		t.Errorf("Version = %d, want 1", file.Version)
	}
	if len(file.Tasks) != 0 {
		t.Errorf("len(Tasks) = %d, want 0", len(file.Tasks))
	}
}

func TestLoad_EmptyBraces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "laps.json")
	if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if !errors.Is(err, ErrEmptyFile) {
		t.Fatalf("expected ErrEmptyFile, got: %v", err)
	}
}

func TestLoad_RejectExtraTopLevelField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "laps.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"tasks":[],"extra":"foo"}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for extra top-level field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got: %v", err)
	}
}

func TestLoad_RejectExtraTaskField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "laps.json")
	data := `{"version":1,"tasks":[{"id":"x","title":"y","isDone":false,"createdAt":"2026-04-28T10:15:00Z","updatedAt":"2026-04-28T10:15:00Z","extra":"foo"}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for extra task field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got: %v", err)
	}
}

func TestLoad_RejectInvalidTaskStructure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "laps.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"tasks":"not-an-array"}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid task structure")
	}
}

func TestCheckDefaultStore_OK(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	data := `{"version":1,"tasks":[{"id":"x","title":"y","isDone":false,"createdAt":"2026-04-28T10:15:00Z","updatedAt":"2026-04-28T10:15:00Z"}]}`
	if err := os.WriteFile(filepath.Join(beadsDir, "laps.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CheckDefaultStore(beadsDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckDefaultStore_MissingNoCandidates(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := CheckDefaultStore(beadsDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClaimCRUD(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := WriteClaim(beadsDir, Claim{Lap: "test-id-123", File: "laps.json"}); err != nil {
		t.Fatalf("WriteClaim: %v", err)
	}

	claim, err := ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("ReadClaim: %v", err)
	}
	if claim.Lap != "test-id-123" {
		t.Fatalf("expected lap 'test-id-123', got %q", claim.Lap)
	}
	if claim.File != "laps.json" {
		t.Fatalf("expected file 'laps.json', got %q", claim.File)
	}
	if claim.ClaimedAt == nil {
		t.Fatalf("expected claimedAt to be recorded, got nil")
	}

	cp := ClaimPath(beadsDir)
	if _, err := os.Stat(cp); err != nil {
		t.Fatalf("claim file should exist: %v", err)
	}

	if err := RemoveClaim(beadsDir); err != nil {
		t.Fatalf("RemoveClaim: %v", err)
	}

	claim, err = ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("ReadClaim after remove: %v", err)
	}
	if !claim.IsZero() {
		t.Fatalf("expected no claim after remove, got %+v", claim)
	}

	// Remove when already gone should not error
	if err := RemoveClaim(beadsDir); err != nil {
		t.Fatalf("RemoveClaim on empty: %v", err)
	}

	// ReadClaim on missing file
	claim, err = ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("ReadClaim on empty: %v", err)
	}
	if !claim.IsZero() {
		t.Fatalf("expected no claim, got %+v", claim)
	}
}

func TestClaimRecordsClaimedAt(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	before := time.Now().UTC().Add(-time.Second)
	if err := WriteClaim(beadsDir, Claim{Lap: "lap-1", File: "laps.json"}); err != nil {
		t.Fatalf("WriteClaim: %v", err)
	}

	claim, err := ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("ReadClaim: %v", err)
	}
	if claim.ClaimedAt == nil {
		t.Fatal("expected claimedAt to be set")
	}
	if claim.ClaimedAt.Before(before) {
		t.Fatalf("claimedAt %v is before write time %v", claim.ClaimedAt, before)
	}
	original := *claim.ClaimedAt

	// Re-claiming the same lap preserves the original claimedAt.
	if err := WriteClaim(beadsDir, Claim{Lap: "lap-1", File: "laps.json"}); err != nil {
		t.Fatalf("WriteClaim (re-claim): %v", err)
	}
	claim, err = ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("ReadClaim (re-claim): %v", err)
	}
	if claim.ClaimedAt == nil || !claim.ClaimedAt.Equal(original) {
		t.Fatalf("expected preserved claimedAt %v, got %v", original, claim.ClaimedAt)
	}

	// Claiming a different lap records a fresh claimedAt.
	if err := WriteClaim(beadsDir, Claim{Lap: "lap-2", File: "laps.json"}); err != nil {
		t.Fatalf("WriteClaim (new lap): %v", err)
	}
	claim, err = ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("ReadClaim (new lap): %v", err)
	}
	if claim.Lap != "lap-2" {
		t.Fatalf("expected lap 'lap-2', got %q", claim.Lap)
	}
}

func TestClaimDifferentFileGetsFreshClaimedAt(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	original := time.Date(2026, time.June, 30, 7, 0, 0, 0, time.UTC)
	if err := WriteClaim(beadsDir, Claim{
		Lap:       "lap-1",
		File:      "alpha.json",
		ClaimedAt: &original,
	}); err != nil {
		t.Fatalf("WriteClaim original: %v", err)
	}

	if err := WriteClaim(beadsDir, Claim{Lap: "lap-1", File: "beta.json"}); err != nil {
		t.Fatalf("WriteClaim different file: %v", err)
	}

	claim, err := ReadClaim(beadsDir, "beta.json")
	if err != nil {
		t.Fatalf("ReadClaim: %v", err)
	}
	if claim.File != "beta.json" {
		t.Fatalf("expected file beta.json, got %q", claim.File)
	}
	if claim.ClaimedAt == nil {
		t.Fatal("expected claimedAt to be set")
	}
	if claim.ClaimedAt.Equal(original) {
		t.Fatalf("expected a fresh claimedAt for different-file claim, got preserved %v", claim.ClaimedAt)
	}
}

func TestReadClaimLegacyBareID(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Legacy format: a bare id with trailing newline.
	if err := os.WriteFile(ClaimPath(beadsDir), []byte("legacy-id-7\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	claim, err := ReadClaim(beadsDir, "selected.json")
	if err != nil {
		t.Fatalf("ReadClaim: %v", err)
	}
	if claim.Lap != "legacy-id-7" {
		t.Fatalf("expected lap 'legacy-id-7', got %q", claim.Lap)
	}
	if claim.File != "selected.json" {
		t.Fatalf("expected file to be the selected file 'selected.json', got %q", claim.File)
	}
	if claim.ClaimedAt != nil {
		t.Fatalf("expected nil claimedAt for legacy claim, got %v", claim.ClaimedAt)
	}
}

func TestReadClaimIgnoresUnknownFields(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// A future writer adds a "scope" field that this reader does not know.
	body := `{"lap":"lap-9","file":"laps.json","scope":"root","claimedAt":null}`
	if err := os.WriteFile(ClaimPath(beadsDir), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	claim, err := ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("ReadClaim with unknown field should not error: %v", err)
	}
	if claim.Lap != "lap-9" || claim.File != "laps.json" {
		t.Fatalf("unexpected claim: %+v", claim)
	}
}

func TestReadClaimMalformedStructured(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"truncated":      `{"lap":"x",`,
		"wrong-type-lap": `{"lap":123}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(ClaimPath(beadsDir), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := ReadClaim(beadsDir, "laps.json")
			if !errors.Is(err, ErrMalformedClaim) {
				t.Fatalf("expected ErrMalformedClaim, got %v", err)
			}
		})
	}
}

func TestReadClaimJSONScalarIsMalformed(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"string": `"legacy-id-7"`,
		"array":  `["legacy-id-7"]`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(ClaimPath(beadsDir), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := ReadClaim(beadsDir, "laps.json")
			if !errors.Is(err, ErrMalformedClaim) {
				t.Fatalf("expected ErrMalformedClaim, got %v", err)
			}
		})
	}
}

func TestReadClaimWhitespaceIsNoClaim(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ClaimPath(beadsDir), []byte("   \n\t  \n"), 0o644); err != nil {
		t.Fatal(err)
	}

	claim, err := ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("whitespace claim should not error: %v", err)
	}
	if !claim.IsZero() {
		t.Fatalf("expected no claim for whitespace file, got %+v", claim)
	}
}
