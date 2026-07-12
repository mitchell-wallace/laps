package store

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiscoverRepoRoot_FallsBackToLegacyLaps(t *testing.T) {
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

func TestDiscoverRepoRoot_PrefersCircuitLaps(t *testing.T) {
	root := t.TempDir()
	circuitDir := filepath.Join(root, ".circuit", "laps")
	if err := os.MkdirAll(circuitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".laps"), 0755); err != nil {
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
	if gotBeads != circuitDir {
		t.Errorf("beadsDir = %q, want %q", gotBeads, circuitDir)
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
	if hasAncestorMarker(root, ".git") || hasAncestorMarker(root, ".circuit/laps") || hasAncestorMarker(root, ".laps") {
		t.Skip("temp dir ancestors contain .git, .circuit/laps, or .laps; cannot assert no-repo discovery case")
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

func TestSaveFilesAtomicallyRollsBackMidCommitFailure(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "a.json")
	secondPath := filepath.Join(dir, "b.json")
	first := &File{Version: CurrentVersion, Tasks: []Task{{ID: "a-old", Title: "A old"}}}
	second := &File{Version: CurrentVersion, Tasks: []Task{{ID: "b-old", Title: "B old"}}}
	if err := Save(firstPath, first); err != nil {
		t.Fatal(err)
	}
	if err := Save(secondPath, second); err != nil {
		t.Fatal(err)
	}
	firstBefore, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	secondBefore, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}

	originalRename := atomicRename
	t.Cleanup(func() { atomicRename = originalRename })
	renames := 0
	atomicRename = func(oldPath, newPath string) error {
		renames++
		// Sorted path order performs two commit renames per file. Fail while
		// installing the second replacement, after the first was committed.
		if renames == 4 {
			return errors.New("injected second-file failure")
		}
		return os.Rename(oldPath, newPath)
	}

	err = SaveFilesAtomically(map[string]*File{
		firstPath:  {Version: CurrentVersion, Tasks: []Task{{ID: "a-new", Title: "A new"}}},
		secondPath: {Version: CurrentVersion, Tasks: []Task{{ID: "b-new", Title: "B new"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "injected second-file failure") {
		t.Fatalf("SaveFilesAtomically error = %v, want injected failure", err)
	}
	firstAfter, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	secondAfter, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstAfter, firstBefore) || !bytes.Equal(secondAfter, secondBefore) {
		t.Fatalf("transaction was partially applied\nfirst before: %s\nfirst after: %s\nsecond before: %s\nsecond after: %s", firstBefore, firstAfter, secondBefore, secondAfter)
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
	id, err := GenerateID("mypr", "Add list command", created, "Multi-line\ndescription supported.", nil)
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
		prefix     string
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
		t.Run(tt.prefix, func(t *testing.T) {
			id, err := GenerateID(tt.prefix, "t", created, "d", map[string]struct{}{})
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
	scopePrefix := "proj"
	created := time.Date(2026, 4, 28, 10, 15, 0, 0, time.UTC)
	title := "Task"
	desc := "Desc"

	existing := map[string]struct{}{
		"proj-cfe4":  {},
		"proj-cfe40": {},
	}

	id, err := GenerateID(scopePrefix, title, created, desc, existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "proj-cfe401"
	if id != want {
		t.Errorf("GenerateID = %q, want %q", id, want)
	}
}

func TestStintFilePrefixRoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stint.laps.json")
	file := &File{Version: CurrentVersion, Prefix: "auth", Tasks: []Task{}}

	if err := Save(path, file); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Prefix != "auth" {
		t.Fatalf("Prefix = %q, want auth", loaded.Prefix)
	}
}

func TestAllocateStintPrefixAvoidsRepoAndExistingPrefixes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth")
	beadsDir := filepath.Join(root, ".laps")
	if err := os.MkdirAll(StintsDir(beadsDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := Save(filepath.Join(StintsDir(beadsDir), "existing"+stintFileSuffix), &File{
		Version: CurrentVersion,
		Prefix:  "uath",
		Tasks:   []Task{},
	}); err != nil {
		t.Fatalf("Save existing stint: %v", err)
	}

	got, err := AllocateStintPrefix(beadsDir, root, "auth")
	if err != nil {
		t.Fatalf("AllocateStintPrefix: %v", err)
	}
	if got == RepoPrefix(root) || got == "uath" {
		t.Fatalf("allocated colliding prefix %q", got)
	}
	if got == "" || len(got) != 4 {
		t.Fatalf("allocated invalid prefix %q", got)
	}
}

func TestAllocateStintPrefixDeterministic(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth")
	beadsDir := filepath.Join(root, ".laps")
	if err := os.MkdirAll(StintsDir(beadsDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := Save(filepath.Join(StintsDir(beadsDir), "existing"+stintFileSuffix), &File{
		Version: CurrentVersion,
		Prefix:  "auth",
		Tasks:   []Task{},
	}); err != nil {
		t.Fatalf("Save existing stint: %v", err)
	}

	first, err := AllocateStintPrefix(beadsDir, root, "auth api")
	if err != nil {
		t.Fatalf("first AllocateStintPrefix: %v", err)
	}
	second, err := AllocateStintPrefix(beadsDir, root, "auth api")
	if err != nil {
		t.Fatalf("second AllocateStintPrefix: %v", err)
	}
	if first != second {
		t.Fatalf("allocation not deterministic: first %q second %q", first, second)
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

func TestLoad_DistinguishesMissingFromEmpty(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	if _, err := Load(missing); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("missing file error = %v, want ErrFileNotFound", err)
	}

	empty := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(empty); !errors.Is(err, ErrEmptyFile) {
		t.Fatalf("empty file error = %v, want ErrEmptyFile", err)
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

func TestLoadNewerVersionWithNovelEntryFieldReturnsVersionGateError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "laps.json")
	data := `{"version":4,"tasks":[{"kind":"lap","id":"x","title":"y","isDone":false,"order":1,"createdAt":"2026-04-28T10:15:00Z","updatedAt":"2026-04-28T10:15:00Z","futureEntryField":"new-in-v4"}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for newer schema version")
	}
	if !strings.Contains(err.Error(), "schema version 4") || !strings.Contains(err.Error(), "please update laps") {
		t.Fatalf("expected version-gate error, got: %v", err)
	}
	if strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected version gate before strict entry decode, got unknown-field error: %v", err)
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

func TestMigrateV1RunsOrderingBeforeKindStampAndIsIdempotent(t *testing.T) {
	f := &File{
		Version: 1,
		Tasks: []Task{
			{ID: "first"},
			{ID: "second"},
			{ID: "third"},
		},
	}

	if !Migrate(f) {
		t.Fatal("expected Migrate to report a change")
	}
	if f.Version != CurrentVersion {
		t.Fatalf("version = %d, want %d", f.Version, CurrentVersion)
	}
	for i, task := range f.Tasks {
		wantOrder := (i + 1) * orderStep
		if task.Order != wantOrder {
			t.Errorf("task %s order = %d, want %d", task.ID, task.Order, wantOrder)
		}
		if task.Kind != KindLap {
			t.Errorf("task %s kind = %q, want %q", task.ID, task.Kind, KindLap)
		}
	}

	before := append([]Task(nil), f.Tasks...)
	if Migrate(f) {
		t.Fatal("expected second Migrate to be a no-op")
	}
	if f.Version != CurrentVersion {
		t.Fatalf("version after second migrate = %d, want %d", f.Version, CurrentVersion)
	}
	for i := range before {
		if f.Tasks[i] != before[i] {
			t.Fatalf("task %d changed on idempotent migrate: got %+v want %+v", i, f.Tasks[i], before[i])
		}
	}
}

func TestMigrateV2StampsLapKindAndIsIdempotent(t *testing.T) {
	f := &File{
		Version: 2,
		Tasks: []Task{
			{ID: "alpha", Order: 10},
			{ID: "beta", Kind: KindLap, Order: 20},
		},
	}

	if !Migrate(f) {
		t.Fatal("expected Migrate to report a change")
	}
	if f.Version != CurrentVersion {
		t.Fatalf("version = %d, want %d", f.Version, CurrentVersion)
	}
	for _, task := range f.Tasks {
		if task.Kind != KindLap {
			t.Errorf("task %s kind = %q, want %q", task.ID, task.Kind, KindLap)
		}
	}
	if f.Tasks[0].Order != 10 || f.Tasks[1].Order != 20 {
		t.Fatalf("v2 migration should preserve existing order keys, got %d and %d", f.Tasks[0].Order, f.Tasks[1].Order)
	}

	before := append([]Task(nil), f.Tasks...)
	if Migrate(f) {
		t.Fatal("expected second Migrate to be a no-op")
	}
	for i := range before {
		if f.Tasks[i] != before[i] {
			t.Fatalf("task %d changed on idempotent migrate: got %+v want %+v", i, f.Tasks[i], before[i])
		}
	}
}

func TestLoadMissingKindDefaultsToLap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "laps.json")
	data := `{"version":3,"tasks":[{"id":"x","title":"y","isDone":false,"order":1,"createdAt":"2026-04-28T10:15:00Z","updatedAt":"2026-04-28T10:15:00Z"}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	file, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := file.Tasks[0].Kind; got != KindLap {
		t.Fatalf("missing kind loaded as %q, want %q", got, KindLap)
	}
}

// TestLoadHeldDefaultsFalseAndRoundTrips asserts the held flag folds into
// schema v3: a v3 file written without "held" loads as false, an explicit
// "held":true loads as true, and Save omits the field when false / writes it
// when true (task 1.1).
func TestLoadHeldDefaultsFalseAndRoundTrips(t *testing.T) {
	dir := t.TempDir()

	missingPath := filepath.Join(dir, "missing.json")
	missing := `{"version":3,"tasks":[{"id":"x","title":"y","isDone":false,"order":1,"createdAt":"2026-04-28T10:15:00Z","updatedAt":"2026-04-28T10:15:00Z"}]}`
	if err := os.WriteFile(missingPath, []byte(missing), 0644); err != nil {
		t.Fatal(err)
	}
	missingFile, err := Load(missingPath)
	if err != nil {
		t.Fatalf("Load missing held: %v", err)
	}
	if missingFile.Held {
		t.Fatalf("file without held field loaded Held=true; want false")
	}
	if strings.Contains(missing, "\"held\"") {
		t.Fatalf("test fixture should not contain a held field")
	}

	heldPath := filepath.Join(dir, "held.json")
	held := `{"version":3,"held":true,"tasks":[{"id":"x","title":"y","isDone":false,"order":1,"createdAt":"2026-04-28T10:15:00Z","updatedAt":"2026-04-28T10:15:00Z"}]}`
	if err := os.WriteFile(heldPath, []byte(held), 0644); err != nil {
		t.Fatal(err)
	}
	heldFile, err := Load(heldPath)
	if err != nil {
		t.Fatalf("Load held: %v", err)
	}
	if !heldFile.Held {
		t.Fatalf("file with held:true loaded Held=false; want true")
	}

	// Round-trip: a held file persists held, a released file omits it.
	if err := Save(heldPath, heldFile); err != nil {
		t.Fatalf("Save held: %v", err)
	}
	if body, err := os.ReadFile(heldPath); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(string(body), "\"held\": true") {
		t.Fatalf("held file should persist \"held\": true, got: %s", body)
	}

	heldFile.Held = false
	if err := Save(heldPath, heldFile); err != nil {
		t.Fatalf("Save released: %v", err)
	}
	if body, err := os.ReadFile(heldPath); err != nil {
		t.Fatal(err)
	} else if strings.Contains(string(body), "\"held\"") {
		t.Fatalf("released file should omit held, got: %s", body)
	}
}

func TestMixedQueueRoundTripsLapAndStintRefsOrderedTogether(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "laps.json")
	created := time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC)
	file := &File{
		Version: CurrentVersion,
		Tasks: []Task{
			{Kind: KindLap, ID: "lap-tail", Title: "Tail lap", Order: 30, CreatedAt: created, UpdatedAt: created},
			{Kind: KindStint, ID: "stint-ref", Ref: "auth", Title: "Auth stint", Order: 10, CreatedAt: created, UpdatedAt: created},
			{Kind: KindLap, ID: "lap-middle", Title: "Middle lap", Order: 20, CreatedAt: created, UpdatedAt: created},
		},
	}

	if err := Save(path, file); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Tasks) != 3 {
		t.Fatalf("len(Tasks) = %d, want 3", len(loaded.Tasks))
	}
	want := []struct {
		id   string
		kind string
		ref  string
	}{
		{id: "stint-ref", kind: KindStint, ref: "auth"},
		{id: "lap-middle", kind: KindLap},
		{id: "lap-tail", kind: KindLap},
	}
	for i, exp := range want {
		if loaded.Tasks[i].ID != exp.id || loaded.Tasks[i].Kind != exp.kind || loaded.Tasks[i].Ref != exp.ref {
			t.Fatalf("task %d = id %q kind %q ref %q, want id %q kind %q ref %q",
				i, loaded.Tasks[i].ID, loaded.Tasks[i].Kind, loaded.Tasks[i].Ref, exp.id, exp.kind, exp.ref)
		}
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

func TestStintNameSafetyRejectsUnsafeAndCollidingNames(t *testing.T) {
	beadsDir := filepath.Join(t.TempDir(), ".laps")
	if err := os.MkdirAll(StintsDir(beadsDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(StintArchiveDir(beadsDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(StintsDir(beadsDir), "active"+stintFileSuffix), []byte(`{"version":3,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(StintArchiveDir(beadsDir), "archived"+stintFileSuffix), []byte(`{"version":3,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		want string
	}{
		{name: "", want: "blank"},
		{name: "auth" + string(os.PathSeparator) + "api", want: "not a path"},
		{name: ".", want: "invalid stint name"},
		{name: "..", want: "invalid stint name"},
		{name: "active", want: "active stint file already exists"},
		{name: "archived", want: "archived stint file already exists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckStintNameAvailable(beadsDir, tt.name)
			if err == nil {
				t.Fatal("expected stint name to be rejected")
			}
			if !errors.Is(err, ErrStore) {
				t.Fatalf("expected ErrStore, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error to contain %q, got %v", tt.want, err)
			}
		})
	}
}

func TestArchiveStintRefusesToOverwriteArchivedFile(t *testing.T) {
	beadsDir := filepath.Join(t.TempDir(), ".laps")
	active, err := ResolveStintFile(beadsDir, "auth")
	if err != nil {
		t.Fatal(err)
	}
	archived, err := ResolveArchivedStintFile(beadsDir, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(active), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(archived), 0755); err != nil {
		t.Fatal(err)
	}
	activeBody := []byte(`{"version":3,"tasks":[{"kind":"lap","id":"active","title":"active","isDone":false,"order":1,"createdAt":"2026-04-28T10:15:00Z","updatedAt":"2026-04-28T10:15:00Z"}]}`)
	archivedBody := []byte(`{"version":3,"tasks":[{"kind":"lap","id":"archived","title":"archived","isDone":false,"order":1,"createdAt":"2026-04-28T10:15:00Z","updatedAt":"2026-04-28T10:15:00Z"}]}`)
	if err := os.WriteFile(active, activeBody, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archived, archivedBody, 0644); err != nil {
		t.Fatal(err)
	}

	err = ArchiveStint(beadsDir, "auth")
	if err == nil {
		t.Fatal("expected archive collision error")
	}
	if !strings.Contains(err.Error(), "archived stint file already exists") {
		t.Fatalf("expected archive no-overwrite error, got %v", err)
	}
	gotArchived, err := os.ReadFile(archived)
	if err != nil {
		t.Fatalf("archived file should remain readable: %v", err)
	}
	if string(gotArchived) != string(archivedBody) {
		t.Fatalf("archived file was overwritten: got %s want %s", gotArchived, archivedBody)
	}
	gotActive, err := os.ReadFile(active)
	if err != nil {
		t.Fatalf("active file should remain after failed archive: %v", err)
	}
	if string(gotActive) != string(activeBody) {
		t.Fatalf("active file changed: got %s want %s", gotActive, activeBody)
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
	body := `{"lap":"lap-9","file":"laps.json","scope":"root","claimedAt":null,"futureMode":"delegated"}`
	if err := os.WriteFile(ClaimPath(beadsDir), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	claim, err := ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("ReadClaim with unknown field should not error: %v", err)
	}
	if claim.Lap != "lap-9" || claim.File != "laps.json" || claim.Scope != "root" {
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
