package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func setupTempRepo(t *testing.T) (beadsDir string, cleanup func()) {
	t.Helper()
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	beadsDir = filepath.Join(root, ".laps")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	os.Chdir(root)
	return beadsDir, func() { os.Chdir(origWd) }
}

func runMB(args ...string) (stdout string, stderr string, code int) {
	oldOut := os.Stdout
	oldErr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	rootCmd.SetArgs(args)
	fileFlag = ""
	jsonOutput = false
	listAll = false
	listDone = false
	listOneline = false
	listTree = false
	addTitle = ""
	addDescription = ""
	addAssignee = ""
	addJSON = ""
	addStdin = false
	updateYes = false
	forceUndo = false
	deleteForce = false
	stintsRmForce = false
	editTitle = ""
	editDescription = ""
	editAssignee = ""
	logLimit = 20
	lapFlag = ""
	sessionFlag = ""
	sinceFlag = ""
	scopeActive = false
	scopeRoot = false
	scopeStint = ""
	resetCommandFlags(rootCmd)

	func() {
		defer func() {
			if r := recover(); r != nil {
				if ee, ok := r.(*exitError); ok {
					code = ee.code
				} else {
					panic(r)
				}
			}
		}()
		rootCmd.Execute()
	}()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	outBuf, _ := io.ReadAll(rOut)
	errBuf, _ := io.ReadAll(rErr)

	return string(outBuf), string(errBuf), code
}

func runMBExecute(args ...string) (stdout string, stderr string, err error) {
	oldArgs := os.Args
	os.Args = append([]string{"laps"}, args...)
	defer func() { os.Args = oldArgs }()

	oldOut := os.Stdout
	oldErr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr
	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
	}()

	rootCmd.SetArgs(args)
	fileFlag = ""
	jsonOutput = false
	listAll = false
	listDone = false
	listOneline = false
	listTree = false
	addTitle = ""
	addDescription = ""
	addAssignee = ""
	addJSON = ""
	addStdin = false
	updateYes = false
	forceUndo = false
	deleteForce = false
	stintsRmForce = false
	editTitle = ""
	editDescription = ""
	editAssignee = ""
	logLimit = 20
	lapFlag = ""
	sessionFlag = ""
	sinceFlag = ""
	scopeActive = false
	scopeRoot = false
	scopeStint = ""
	resetCommandFlags(rootCmd)

	err = Execute("test")

	wOut.Close()
	wErr.Close()

	outBuf, _ := io.ReadAll(rOut)
	errBuf, _ := io.ReadAll(rErr)

	return string(outBuf), string(errBuf), err
}

func resetCommandFlags(cmd *cobra.Command) {
	for _, f := range []*pflag.FlagSet{
		cmd.PersistentFlags(),
		cmd.Flags(),
	} {
		f.VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}
	for _, child := range cmd.Commands() {
		resetCommandFlags(child)
	}
}

var resolverTestTime = time.Date(2026, 4, 28, 10, 15, 0, 0, time.UTC)

func writeResolverQueue(t *testing.T, path, prefix string, tasks ...store.Task) {
	t.Helper()

	if err := store.Save(path, &store.File{
		Version: store.CurrentVersion,
		Prefix:  prefix,
		Tasks:   tasks,
	}); err != nil {
		t.Fatalf("Save %s: %v", path, err)
	}
}

func readResolverFiles(t *testing.T, paths ...string) map[string]string {
	t.Helper()

	files := make(map[string]string, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile %s: %v", path, err)
		}
		files[path] = string(data)
	}
	return files
}

func TestAddHead(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("add", "head", "--title", "First task", "--description", "Desc")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	id := strings.TrimSpace(out)
	if id == "" {
		t.Fatal("expected id output")
	}
}

func TestAddTail(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "A")
	out, _, code := runMB("add", "tail", "--title", "B")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected id output")
	}
}

func TestAddAfter(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "A")
	out, _, _ := runMB("add", "tail", "--title", "B")
	bid := strings.TrimSpace(out)

	out, _, code := runMB("add", "after", bid, "--title", "C")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
}

func TestStintsNewAllocatesPrefixAndStintAddUsesIt(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("stints", "new", "auth")
	if code != 0 {
		t.Fatalf("stints new exit %d, stderr: %s", code, errStr)
	}
	prefix := strings.TrimSpace(out)
	if prefix != "auth" {
		t.Fatalf("stints new prefix = %q, want auth", prefix)
	}

	stintPath := filepath.Join(beadsDir, "stints", "auth.laps.json")
	stintFile, err := store.Load(stintPath)
	if err != nil {
		t.Fatalf("Load stint file: %v", err)
	}
	if stintFile.Prefix != prefix {
		t.Fatalf("stint prefix metadata = %q, want %q", stintFile.Prefix, prefix)
	}

	rootOut, errStr, code := runMB("add", "head", "--title", "Root")
	if code != 0 {
		t.Fatalf("root add exit %d, stderr: %s", code, errStr)
	}
	stintOut, errStr, code := runMB("--file", "stints/auth.laps", "add", "head", "--title", "Inside stint")
	if code != 0 {
		t.Fatalf("stint add exit %d, stderr: %s", code, errStr)
	}
	rootID := strings.TrimSpace(rootOut)
	stintID := strings.TrimSpace(stintOut)
	if !strings.HasPrefix(stintID, prefix+"-") {
		t.Fatalf("stint id %q does not start with prefix %q", stintID, prefix)
	}
	if rootID == stintID {
		t.Fatalf("root and stint ids collided: %q", rootID)
	}
	if strings.SplitN(rootID, "-", 2)[0] == strings.SplitN(stintID, "-", 2)[0] {
		t.Fatalf("root and stint ids used the same prefix: root %q stint %q", rootID, stintID)
	}
}

func TestRawStintFileInitializationAllocatesPrefix(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("--file", "stints/auth.laps", "add", "head", "--title", "Inside stint")
	if code != 0 {
		t.Fatalf("raw stint add exit %d, stderr: %s", code, errStr)
	}

	stintID := strings.TrimSpace(out)
	if !strings.HasPrefix(stintID, "auth-") {
		t.Fatalf("stint id %q does not start with allocated prefix auth", stintID)
	}
	stintPath := filepath.Join(beadsDir, "stints", "auth.laps.json")
	stintFile, err := store.Load(stintPath)
	if err != nil {
		t.Fatalf("Load stint file: %v", err)
	}
	if stintFile.Prefix != "auth" {
		t.Fatalf("stint prefix metadata = %q, want auth", stintFile.Prefix)
	}
}

func TestExistingEmptyRawStintFileGetsPrefixBeforeAdd(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	stintPath := filepath.Join(beadsDir, "stints", "auth.laps.json")
	if err := os.MkdirAll(filepath.Dir(stintPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stintPath, []byte(`{"version":3,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	out, errStr, code := runMB("--file", "stints/auth.laps", "add", "head", "--title", "Inside stint")
	if code != 0 {
		t.Fatalf("raw stint add exit %d, stderr: %s", code, errStr)
	}
	stintID := strings.TrimSpace(out)
	if !strings.HasPrefix(stintID, "auth-") {
		t.Fatalf("stint id %q does not start with allocated prefix auth", stintID)
	}
	stintFile, err := store.Load(stintPath)
	if err != nil {
		t.Fatalf("Load stint file: %v", err)
	}
	if stintFile.Prefix != "auth" {
		t.Fatalf("stint prefix metadata = %q, want auth", stintFile.Prefix)
	}
}

func TestExistingPopulatedRawStintFileRequiresPrefix(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	stintPath := filepath.Join(beadsDir, "stints", "auth.laps.json")
	if err := os.MkdirAll(filepath.Dir(stintPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stintPath, []byte(`{"version":3,"tasks":[{"kind":"lap","id":"repo-1234","title":"Existing","description":"","isDone":false,"order":65536,"createdAt":"2026-04-28T10:15:00Z","updatedAt":"2026-04-28T10:15:00Z","completedAt":null}]}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, errStr, code := runMB("--file", "stints/auth.laps", "add", "tail", "--title", "Inside stint")
	if code != 2 {
		t.Fatalf("raw populated stint add exit %d, want 2; stderr: %s", code, errStr)
	}
	if !strings.Contains(errStr, "missing prefix metadata") {
		t.Fatalf("expected missing prefix metadata error, got: %s", errStr)
	}
}

func TestScopeFlagsResolveQueuePaths(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	if out, errStr, code := runMB("add", "head", "--root", "--title", "Root scoped"); code != 0 {
		t.Fatalf("add --root exit %d, stderr: %s, stdout: %s", code, errStr, out)
	}
	rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	if len(rootFile.Tasks) != 1 || rootFile.Tasks[0].Title != "Root scoped" {
		t.Fatalf("root file tasks = %#v", rootFile.Tasks)
	}

	if out, errStr, code := runMB("add", "head", "--stint", "auth", "--title", "Stint scoped"); code != 0 {
		t.Fatalf("add --stint exit %d, stderr: %s, stdout: %s", code, errStr, out)
	}
	stintFile, err := store.Load(filepath.Join(beadsDir, "stints", "auth.laps.json"))
	if err != nil {
		t.Fatalf("Load stint file: %v", err)
	}
	if len(stintFile.Tasks) != 1 || stintFile.Tasks[0].Title != "Stint scoped" {
		t.Fatalf("stint file tasks = %#v", stintFile.Tasks)
	}

	if out, errStr, code := runMB("list", "--active"); code != 0 {
		t.Fatalf("list --active exit %d, stderr: %s", code, errStr)
	} else if !strings.Contains(out, "Root scoped") {
		t.Fatalf("list --active output = %q, want root task", out)
	}
}

func TestScopeFlagsAreMutuallyExclusive(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	tests := [][]string{
		{"add", "head", "--root", "--active", "--title", "X"},
		{"add", "head", "--root", "--stint", "auth", "--title", "X"},
		{"add", "head", "--active", "--stint", "auth", "--title", "X"},
		{"--file", "other", "add", "head", "--root", "--title", "X"},
		{"--file", "other", "add", "head", "--active", "--title", "X"},
		{"--file", "other", "add", "head", "--stint", "auth", "--title", "X"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			_, _, err := runMBExecute(args...)
			if err == nil {
				t.Fatalf("expected mutual-exclusion error for args %v", args)
			}
		})
	}
}

func TestScopeFlagsAreOnlyOnQueueTargetingCommands(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	tests := [][]string{
		{"init", "--root"},
		{"on", "--root"},
		{"off", "--root"},
		{"update", "--root"},
		{"version", "--root"},
		{"log", "--root"},
		{"status", "--root"},
		{"stints", "new", "auth", "--root"},
		{"customcmd", "--root"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			_, _, err := runMBExecute(args...)
			if err == nil {
				t.Fatalf("expected unknown flag error for args %v", args)
			}
		})
	}
}

func TestScopeFlagChangedStateDoesNotLeakBetweenRuns(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	if out, errStr, code := runMB("add", "head", "--root", "--title", "A"); code != 0 {
		t.Fatalf("add --root exit %d, stderr: %s, stdout: %s", code, errStr, out)
	}
	if out, errStr, code := runMB("--file", "other", "add", "head", "--title", "B"); code != 0 {
		t.Fatalf("add --file after --root exit %d, stderr: %s, stdout: %s", code, errStr, out)
	}
}

// setupActiveStintRepo seeds a root queue whose head is an active "auth" stint
// ref (so the default --active scope descends into auth) plus an auth stint
// file holding one lap. It returns beadsDir and the id of the lap inside the
// active auth stint.
func setupActiveStintRepo(t *testing.T) (beadsDir, authLapID string) {
	t.Helper()
	beadsDir, cleanup := setupTempRepo(t)
	t.Cleanup(cleanup)

	rootTask := store.Task{
		Kind:      store.KindStint,
		ID:        "root-auth",
		Ref:       "auth",
		Title:     "Auth stint",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	}
	writeResolverQueue(t, filepath.Join(beadsDir, "laps.json"), "", rootTask)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "auth.laps.json"), "auth", store.Task{
		Kind:        store.KindLap,
		ID:          "auth-aaaa",
		Title:       "Inside auth",
		Description: "auth details",
		Order:       1,
		CreatedAt:   resolverTestTime,
		UpdatedAt:   resolverTestTime,
	})
	return beadsDir, "auth-aaaa"
}

// setupTwoStints seeds a root queue with an active "auth" stint, an auth stint
// file with one lap, and a separate "search" stint file with one lap. The
// search lap is out of scope when the default --active scope resolves into auth.
// It returns beadsDir, the active auth lap id, the out-of-scope search lap id,
// and the owning search stint name.
func setupTwoStints(t *testing.T) (beadsDir, authLapID, searchLapID, searchStint string) {
	t.Helper()
	beadsDir, authLapID = setupActiveStintRepo(t)

	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "search.laps.json"), "srch", store.Task{
		Kind:        store.KindLap,
		ID:          "srch-bbbb",
		Title:       "Inside search",
		Description: "search details",
		Order:       1,
		CreatedAt:   resolverTestTime,
		UpdatedAt:   resolverTestTime,
	})
	return beadsDir, authLapID, "srch-bbbb", "search"
}

// TestAddHeadDefaultsToActiveStint verifies the default scope (--active) lands
// a new head lap inside the active stint, not in root, while --root bypasses the
// descent and writes to root. (task 4.4)
func TestAddHeadDefaultsToActiveStint(t *testing.T) {
	beadsDir, _ := setupActiveStintRepo(t)

	out, errStr, code := runMB("add", "head", "--title", "Added into active stint")
	if code != 0 {
		t.Fatalf("add head (active) exit %d, stderr: %s, stdout: %s", code, errStr, out)
	}
	stintFile, err := store.Load(filepath.Join(beadsDir, "stints", "auth.laps.json"))
	if err != nil {
		t.Fatalf("Load auth stint file: %v", err)
	}
	found := false
	for _, task := range stintFile.Tasks {
		if task.Title == "Added into active stint" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected new head lap in active auth stint, tasks = %#v", stintFile.Tasks)
	}
	// The new lap's id must carry the auth stint prefix, proving it was written
	// under the descended scope rather than root.
	if id := strings.TrimSpace(out); !strings.HasPrefix(id, "auth-") {
		t.Fatalf("expected active-add id to use auth prefix, got %q", id)
	}
}

// TestAddRootLandsInRootQueue verifies --root bypasses the active-stint descent
// and writes the new lap into the root queue with the repo prefix. (task 4.4)
func TestAddRootLandsInRootQueue(t *testing.T) {
	beadsDir, _ := setupActiveStintRepo(t)

	out, errStr, code := runMB("add", "head", "--root", "--title", "Root not stint")
	if code != 0 {
		t.Fatalf("add head --root exit %d, stderr: %s, stdout: %s", code, errStr, out)
	}
	rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	found := false
	for _, task := range rootFile.Tasks {
		if task.Kind != store.KindLap {
			continue
		}
		if task.Title == "Root not stint" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected --root lap in root queue, tasks = %#v", rootFile.Tasks)
	}
	stintFile, err := store.Load(filepath.Join(beadsDir, "stints", "auth.laps.json"))
	if err != nil {
		t.Fatalf("Load auth stint file: %v", err)
	}
	for _, task := range stintFile.Tasks {
		if task.Title == "Root not stint" {
			t.Fatalf("--root lap leaked into active stint file: %#v", stintFile.Tasks)
		}
	}
	if id := strings.TrimSpace(out); strings.HasPrefix(id, "auth-") {
		t.Fatalf("expected --root id to use repo prefix, not auth, got %q", id)
	}
}

// outOfScopeIDCases exercises every id-taking command group. Each entry is run
// against the two-stint fixture (default --active scope resolves into auth), so
// the search lap id is out of scope and the error must name the owning stint.
func outOfScopeIDCases() []struct {
	kind string
	args []string
} {
	return []struct {
		kind string
		args []string
	}{
		{"get", []string{"get", "srch-bbbb"}},
		{"claim", []string{"claim", "srch-bbbb"}},
		{"done", []string{"done", "srch-bbbb"}},
		{"add-after", []string{"add", "after", "srch-bbbb", "--title", "X"}},
		{"move", []string{"move", "srch-bbbb", "head"}},
		{"edit", []string{"edit", "srch-bbbb", "--title", "X"}},
		{"assign", []string{"assign", "srch-bbbb", "role"}},
		{"delete", []string{"delete", "srch-bbbb"}},
	}
}

// TestOutOfScopeIDErrorNamesOwningStint asserts that every id-taking command
// group fails with a message naming the stint that owns the out-of-scope id,
// and does not mutate any file. (task 4.4)
func TestOutOfScopeIDErrorNamesOwningStint(t *testing.T) {
	for _, tc := range outOfScopeIDCases() {
		t.Run(tc.kind, func(t *testing.T) {
			beadsDir, _, searchID, searchStint := setupTwoStints(t)

			before := readResolverFiles(t,
				filepath.Join(beadsDir, "laps.json"),
				filepath.Join(beadsDir, "stints", "auth.laps.json"),
				filepath.Join(beadsDir, "stints", "search.laps.json"),
			)

			_, errStr, code := runMB(tc.args...)
			if code == 0 {
				t.Fatalf("%s: expected non-zero exit, got 0", tc.kind)
			}
			if !strings.Contains(errStr, searchID) {
				t.Fatalf("%s: error must name the out-of-scope id %q, got: %s", tc.kind, searchID, errStr)
			}
			if !strings.Contains(errStr, searchStint) {
				t.Fatalf("%s: error must name the owning stint %q, got: %s", tc.kind, searchStint, errStr)
			}

			after := readResolverFiles(t,
				filepath.Join(beadsDir, "laps.json"),
				filepath.Join(beadsDir, "stints", "auth.laps.json"),
				filepath.Join(beadsDir, "stints", "search.laps.json"),
			)
			for path, beforeData := range before {
				if after[path] != beforeData {
					t.Fatalf("%s: out-of-scope op mutated %s\nbefore: %s\nafter:  %s", tc.kind, path, beforeData, after[path])
				}
			}
		})
	}
}

// TestOutOfScopeIDErrorInRootScope asserts that when --root is selected and the
// id lives in a stint, the error names the stint, and when a root lap id is
// referenced from an active stint scope, the error names root. (task 4.4)
func TestOutOfScopeIDErrorInRootScope(t *testing.T) {
	_, authLapID, _, _ := setupTwoStints(t)

	// --root scope: an auth-stint lap id is out of scope and must name its stint.
	_, errStr, code := runMB("get", "--root", authLapID)
	if code == 0 {
		t.Fatalf("expected non-zero exit for root-scoped stint lap, got 0")
	}
	if !strings.Contains(errStr, authLapID) {
		t.Fatalf("error must name the out-of-scope id %q, got: %s", authLapID, errStr)
	}
	if !strings.Contains(errStr, "auth") {
		t.Fatalf("error must name the owning stint auth, got: %s", errStr)
	}

	// Active (auth) scope: a root lap id must name root as its owner. Add a root
	// lap at the tail (so the auth stint ref stays the root head and active
	// descent still enters auth) so it carries the real repo prefix, then
	// reference it from the active (auth) scope.
	rootOut, errStr, code := runMB("add", "tail", "--root", "--title", "Root lap")
	if code != 0 {
		t.Fatalf("add --root setup failed, code %d, stderr: %s", code, errStr)
	}
	rootID := strings.TrimSpace(rootOut)

	_, errStr, code = runMB("get", rootID)
	if code == 0 {
		t.Fatalf("expected non-zero exit for root lap under active scope, got 0")
	}
	if !strings.Contains(errStr, rootID) || !strings.Contains(errStr, "root") {
		t.Fatalf("expected error naming root for %q, got: %s", rootID, errStr)
	}
}

// TestDeleteRefusesClaimedLap asserts that delete refuses a claimed lap unless
// --force is supplied, leaving both the lap and the claim intact. (task 4.4)
func TestDeleteRefusesClaimedLap(t *testing.T) {
	beadsDir, authLapID := setupActiveStintRepo(t)

	if _, errStr, code := runMB("claim", authLapID); code != 0 {
		t.Fatalf("claim setup failed, code %d, stderr: %s", code, errStr)
	}

	_, errStr, code := runMB("delete", authLapID)
	if code == 0 {
		t.Fatalf("expected delete of claimed lap to fail, got exit 0")
	}
	if !strings.Contains(errStr, "claimed") || !strings.Contains(errStr, "--force") {
		t.Fatalf("expected claimed-lap refusal naming --force, got: %s", errStr)
	}

	// The lap and the claim must both survive the refused delete.
	stintFile, err := store.Load(filepath.Join(beadsDir, "stints", "auth.laps.json"))
	if err != nil {
		t.Fatalf("Load auth stint file: %v", err)
	}
	if !fileContainsID(stintFile, authLapID) {
		t.Fatalf("claimed lap %s must survive refused delete", authLapID)
	}
	claim, err := store.ReadClaim(beadsDir, "stints/auth.laps.json")
	if err != nil {
		t.Fatalf("read claim: %v", err)
	}
	if claim.Lap != authLapID {
		t.Fatalf("claim must survive refused delete, got %+v", claim)
	}
}

// TestDeleteForceClearsClaim asserts that delete --force removes a claimed lap
// and clears the matching claim. (task 4.4)
func TestDeleteForceClearsClaim(t *testing.T) {
	beadsDir, authLapID := setupActiveStintRepo(t)

	if _, errStr, code := runMB("claim", authLapID); code != 0 {
		t.Fatalf("claim setup failed, code %d, stderr: %s", code, errStr)
	}

	_, errStr, code := runMB("delete", "--force", authLapID)
	if code != 0 {
		t.Fatalf("delete --force exit %d, stderr: %s", code, errStr)
	}

	stintFile, err := store.Load(filepath.Join(beadsDir, "stints", "auth.laps.json"))
	if err != nil {
		t.Fatalf("Load auth stint file: %v", err)
	}
	if fileContainsID(stintFile, authLapID) {
		t.Fatalf("forced delete must remove lap %s", authLapID)
	}
	if _, err := os.Stat(filepath.Join(beadsDir, "claim")); !os.IsNotExist(err) {
		t.Fatalf("forced delete must clear the claim file, got stat err: %v", err)
	}
}

// writeScopeCountFixture seeds a repo whose root, active auth stint, and an
// inactive search stint each carry a distinct lap set, so a scope-aware command
// can be told apart by its counts. Root's head is the auth stint ref, so the
// default --active scope descends into auth; root also holds one root lap.
func writeScopeCountFixture(t *testing.T) (beadsDir string) {
	t.Helper()
	beadsDir, cleanup := setupTempRepo(t)
	t.Cleanup(cleanup)

	writeResolverQueue(t, filepath.Join(beadsDir, "laps.json"), "",
		store.Task{Kind: store.KindStint, ID: "root-auth", Ref: "auth", Title: "Auth stint", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
		store.Task{Kind: store.KindLap, ID: "root-0001", Title: "Root lap", Assignee: "rooty", Order: 2, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "auth.laps.json"), "auth",
		store.Task{Kind: store.KindLap, ID: "auth-d1", Title: "Auth done", Assignee: "coder", IsDone: true, CompletedAt: &resolverTestTime, Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
		store.Task{Kind: store.KindLap, ID: "auth-t1", Title: "Auth todo", Assignee: "reviewer", Order: 2, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "search.laps.json"), "srch",
		store.Task{Kind: store.KindLap, ID: "srch-1", Title: "Search done", Assignee: "qa", IsDone: true, CompletedAt: &resolverTestTime, Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)
	return beadsDir
}

// TestCountScopeActiveDescends asserts the default scope descends into the
// deepest active stint (auth) while --root and -s <stint> remain direct, by
// reading each scope's distinct counts/breakdown. Regression for count bypassing
// active-context resolution. (task: count/prune --active handling)
func TestCountScopeActiveDescends(t *testing.T) {
	writeScopeCountFixture(t)

	// Default (--active) resolves into the auth stint: 1 done / 2 total.
	out, errStr, code := runMB("count")
	if code != 0 {
		t.Fatalf("count (active) exit %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "Laps done: 1 out of 2") {
		t.Fatalf("active count must reflect auth stint, got: %s", out)
	}
	if !strings.Contains(out, "coder: 1 complete, 0 incomplete") {
		t.Fatalf("active count must show auth coder breakdown, got: %s", out)
	}

	// --root targets the root queue directly: 0 done / 2 total (stint ref + lap).
	out, errStr, code = runMB("count", "--root")
	if code != 0 {
		t.Fatalf("count --root exit %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "Laps done: 0 out of 2") {
		t.Fatalf("root count must reflect root queue, got: %s", out)
	}
	if !strings.Contains(out, "rooty: 0 complete, 1 incomplete") {
		t.Fatalf("root count must show rooty breakdown, got: %s", out)
	}

	// -s search targets the inactive search stint directly: 1 done / 1 total.
	out, errStr, code = runMB("count", "-s", "search")
	if code != 0 {
		t.Fatalf("count -s search exit %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "Laps done: 1 out of 1") {
		t.Fatalf("search count must reflect search stint, got: %s", out)
	}
	if !strings.Contains(out, "qa: 1 complete, 0 incomplete") {
		t.Fatalf("search count must show qa breakdown, got: %s", out)
	}
}

// TestCountExplicitActiveFlagMatchesDefault asserts --active behaves identically
// to the implicit default descent, so the flag is genuinely wired (not just the
// default path). (task: count/prune --active handling)
func TestCountExplicitActiveFlagMatchesDefault(t *testing.T) {
	writeScopeCountFixture(t)

	defOut, _, defCode := runMB("count")
	if defCode != 0 {
		t.Fatalf("count default exit %d", defCode)
	}
	flagOut, errStr, flagCode := runMB("count", "--active")
	if flagCode != 0 {
		t.Fatalf("count --active exit %d, stderr: %s", flagCode, errStr)
	}
	if defOut != flagOut {
		t.Fatalf("--active must match default descent\n default: %q\n active:  %q", defOut, flagOut)
	}
}

// TestPruneScopeActiveDescends asserts prune with the default scope prunes only
// the deepest active stint's done laps while leaving the root queue untouched,
// and that --root prunes root directly. The active stint carries 2 done laps
// while root carries 1, so the removed count itself proves which file was
// targeted. Regression for prune bypassing active-context resolution.
// (task: count/prune --active handling)
func TestPruneScopeActiveDescends(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	writeResolverQueue(t, filepath.Join(beadsDir, "laps.json"), "",
		store.Task{Kind: store.KindStint, ID: "root-auth", Ref: "auth", Title: "Auth stint", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
		store.Task{Kind: store.KindLap, ID: "root-0001", Title: "Root done", IsDone: true, CompletedAt: &resolverTestTime, Order: 2, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "auth.laps.json"), "auth",
		store.Task{Kind: store.KindLap, ID: "auth-d1", Title: "Auth done 1", IsDone: true, CompletedAt: &resolverTestTime, Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
		store.Task{Kind: store.KindLap, ID: "auth-d2", Title: "Auth done 2", IsDone: true, CompletedAt: &resolverTestTime, Order: 2, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)

	// Default (--active) descends into auth: prune 0 removes its 2 done laps.
	out, errStr, code := runMB("prune", "0")
	if code != 0 {
		t.Fatalf("prune 0 (active) exit %d, stderr: %s", code, errStr)
	}
	if strings.TrimSpace(out) != "2" {
		t.Fatalf("active prune must remove the 2 auth done laps, got: %s", out)
	}

	authFile, err := store.Load(filepath.Join(beadsDir, "stints", "auth.laps.json"))
	if err != nil {
		t.Fatalf("Load auth stint file: %v", err)
	}
	for _, tk := range authFile.Tasks {
		if tk.IsDone {
			t.Fatalf("active prune must clear auth done tasks, got %#v", authFile.Tasks)
		}
	}

	// Root's done lap must survive an active-scope prune (root has 1 done, so a
	// misrouted prune would have reported 1 above and removed root-0001 here).
	rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	rootDoneSurvives := false
	for _, tk := range rootFile.Tasks {
		if tk.ID == "root-0001" && tk.IsDone {
			rootDoneSurvives = true
		}
	}
	if !rootDoneSurvives {
		t.Fatalf("active prune must not touch root done lap, root tasks = %#v", rootFile.Tasks)
	}

	// --root prunes the root queue directly: removes root-0001, leaves auth empty.
	out, errStr, code = runMB("prune", "0", "--root")
	if code != 0 {
		t.Fatalf("prune 0 --root exit %d, stderr: %s", code, errStr)
	}
	if strings.TrimSpace(out) != "1" {
		t.Fatalf("root prune must remove the 1 root done lap, got: %s", out)
	}
	rootFile, err = store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	for _, tk := range rootFile.Tasks {
		if tk.ID == "root-0001" {
			t.Fatalf("root prune must remove root-0001, root tasks = %#v", rootFile.Tasks)
		}
	}
}

// TestClaimRecordsScope asserts a claim taken from an active stint persists the
// canonical logical scope alongside the lap and physical file. (task 5.5)
func TestClaimRecordsScope(t *testing.T) {
	beadsDir, authLapID := setupActiveStintRepo(t)

	if _, errStr, code := runMB("claim"); code != 0 {
		t.Fatalf("claim exit %d, stderr: %s", code, errStr)
	}

	claim, err := store.ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("ReadClaim: %v", err)
	}
	if claim.Lap != authLapID {
		t.Fatalf("claim lap = %q, want %q", claim.Lap, authLapID)
	}
	if claim.File != "stints/auth.laps.json" {
		t.Fatalf("claim file = %q, want stints/auth.laps.json", claim.File)
	}
	if claim.Scope != "auth" {
		t.Fatalf("claim scope = %q, want auth", claim.Scope)
	}
}

// TestDoneCompletesClaimedLapAfterEnqueueHeadPreemption is the headline
// preemption-safety scenario: a bare done follows the recorded claim scope, not
// the new root head introduced by enqueue head. (task 5.5)
func TestDoneCompletesClaimedLapAfterEnqueueHeadPreemption(t *testing.T) {
	beadsDir, authLapID := setupActiveStintRepo(t)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "search.laps.json"), "srch",
		store.Task{Kind: store.KindLap, ID: "srch-bbbb", Title: "Search head", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)

	if _, errStr, code := runMB("claim"); code != 0 {
		t.Fatalf("claim exit %d, stderr: %s", code, errStr)
	}
	if _, errStr, code := runMB("stints", "enqueue", "search", "head"); code != 0 {
		t.Fatalf("stints enqueue head exit %d, stderr: %s", code, errStr)
	}

	if _, errStr, code := runMB("done"); code != 0 {
		t.Fatalf("bare done exit %d, stderr: %s", code, errStr)
	}

	authFile, err := store.Load(filepath.Join(beadsDir, "stints", "archive", "auth.laps.json"))
	if err != nil {
		t.Fatalf("Load archived auth stint file: %v", err)
	}
	authLap := taskByIDInFile(t, authFile, authLapID)
	if !authLap.IsDone {
		t.Fatalf("bare done must complete originally claimed auth lap %s", authLapID)
	}

	searchFile, err := store.Load(filepath.Join(beadsDir, "stints", "search.laps.json"))
	if err != nil {
		t.Fatalf("Load search stint file: %v", err)
	}
	searchLap := taskByIDInFile(t, searchFile, "srch-bbbb")
	if searchLap.IsDone {
		t.Fatalf("bare done must not complete preempting search head")
	}
}

// TestStintsEnqueueDefaultsToTail asserts omitted enqueue position appends the
// stint reference after existing root queue entries. (task 6.5)
func TestStintsEnqueueDefaultsToTail(t *testing.T) {
	beadsDir, _ := setupActiveStintRepo(t)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "search.laps.json"), "srch")

	out, errStr, code := runMB("stints", "enqueue", "search")
	if code != 0 {
		t.Fatalf("stints enqueue default exit %d, stderr: %s", code, errStr)
	}
	searchRefID := strings.TrimSpace(out)

	rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	if got := rootTodoIDs(rootFile); len(got) != 2 || got[0] != "root-auth" || got[1] != searchRefID {
		t.Fatalf("root todo order = %v, want [root-auth %s]", got, searchRefID)
	}
	searchRef := taskByIDInFile(t, rootFile, searchRefID)
	if searchRef.Kind != store.KindStint || searchRef.Ref != "search" {
		t.Fatalf("enqueued task = %+v, want search stint ref", searchRef)
	}
}

// TestStintsEnqueueHeadPreemptsAndLaterResumesWithProgress asserts head enqueue
// makes a new stint active without changing the paused stint file, and once the
// preempting root ref is drained, the original stint resumes with its remaining
// todo progress intact. (task 6.5)
func TestStintsEnqueueHeadPreemptsAndLaterResumesWithProgress(t *testing.T) {
	beadsDir, _ := setupActiveStintRepo(t)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "auth.laps.json"), "auth",
		store.Task{Kind: store.KindLap, ID: "auth-done", Title: "Auth done", IsDone: true, CompletedAt: &resolverTestTime, Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
		store.Task{Kind: store.KindLap, ID: "auth-next", Title: "Auth next", Order: 2, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "search.laps.json"), "srch",
		store.Task{Kind: store.KindLap, ID: "srch-next", Title: "Search next", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)

	out, errStr, code := runMB("stints", "enqueue", "search", "head")
	if code != 0 {
		t.Fatalf("stints enqueue head exit %d, stderr: %s", code, errStr)
	}
	searchRefID := strings.TrimSpace(out)

	out, errStr, code = runMB("get")
	if code != 0 {
		t.Fatalf("get after preemption exit %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "Search next") {
		t.Fatalf("head enqueue must make search active, got: %s", out)
	}

	authFile, err := store.Load(filepath.Join(beadsDir, "stints", "auth.laps.json"))
	if err != nil {
		t.Fatalf("Load auth stint file: %v", err)
	}
	if !taskByIDInFile(t, authFile, "auth-done").IsDone || taskByIDInFile(t, authFile, "auth-next").IsDone {
		t.Fatalf("preemption must preserve auth progress, tasks = %#v", authFile.Tasks)
	}

	// Drain/archive is a later OpenSpec task. Mark the preempting root ref done
	// here to model the post-drain root queue state and prove active resolution
	// resumes the paused stint with its existing progress.
	rootPath := filepath.Join(beadsDir, "laps.json")
	rootFile, err := store.Load(rootPath)
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	now := resolverTestTime.Add(time.Minute)
	searchRef := taskByIDInFile(t, rootFile, searchRefID)
	searchRef.IsDone = true
	searchRef.CompletedAt = &now
	searchRef.UpdatedAt = now
	if err := store.Save(rootPath, rootFile); err != nil {
		t.Fatalf("Save drained root file: %v", err)
	}

	out, errStr, code = runMB("get")
	if code != 0 {
		t.Fatalf("get after simulated drain exit %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "Auth next") {
		t.Fatalf("after drain, active scope must resume auth's remaining lap, got: %s", out)
	}
}

// TestStintsEnqueueAfterIsRootOnly asserts enqueue after never resolves through
// the active stint; a stint-owned id is out of root scope and the enqueue is
// rejected without mutation. (task 6.5)
func TestStintsEnqueueAfterIsRootOnly(t *testing.T) {
	beadsDir, authLapID := setupActiveStintRepo(t)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "search.laps.json"), "srch")
	before := readResolverFiles(t, filepath.Join(beadsDir, "laps.json"))

	_, errStr, code := runMB("stints", "enqueue", "search", "after", authLapID)
	if code == 0 {
		t.Fatalf("expected root-only enqueue after to fail, got exit 0")
	}
	if !strings.Contains(errStr, authLapID) || !strings.Contains(errStr, "auth") {
		t.Fatalf("error must name out-of-root auth lap and owning stint, got: %s", errStr)
	}

	after := readResolverFiles(t, filepath.Join(beadsDir, "laps.json"))
	if after[filepath.Join(beadsDir, "laps.json")] != before[filepath.Join(beadsDir, "laps.json")] {
		t.Fatalf("failed root-only enqueue after must not mutate root\nbefore: %s\nafter:  %s",
			before[filepath.Join(beadsDir, "laps.json")],
			after[filepath.Join(beadsDir, "laps.json")])
	}
}

// TestEmptyStintEnqueuesAndResolvesAsNoHead asserts an empty stint is accepted
// as an ordinary queued stint file; active resolution then reaches that empty
// file and reports the normal no-head condition rather than a special enqueue
// error. (task 6.5)
func TestEmptyStintEnqueuesAndResolvesAsNoHead(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "empty.laps.json"), "empt")
	out, errStr, code := runMB("stints", "enqueue", "empty", "head")
	if code != 0 {
		t.Fatalf("stints enqueue empty exit %d, stderr: %s", code, errStr)
	}
	emptyRefID := strings.TrimSpace(out)

	rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	emptyRef := taskByIDInFile(t, rootFile, emptyRefID)
	if emptyRef.Kind != store.KindStint || emptyRef.Ref != "empty" || emptyRef.IsDone {
		t.Fatalf("empty enqueue ref = %+v, want todo empty stint ref", emptyRef)
	}

	_, errStr, code = runMB("get")
	if code != 3 {
		t.Fatalf("get through empty active stint exit %d, want 3; stderr: %s", code, errStr)
	}
	if !strings.Contains(errStr, "no head task") {
		t.Fatalf("empty active stint should use normal no-head error, got: %s", errStr)
	}
}

// TestDoneLastLapDrainsAndArchives asserts completing the final todo lap in an
// active stint marks its root ref done and moves the stint file to the archive.
// (task 7.5)
func TestDoneLastLapDrainsAndArchives(t *testing.T) {
	beadsDir, authLapID := setupActiveStintRepo(t)

	if _, errStr, code := runMB("done", authLapID); code != 0 {
		t.Fatalf("done final stint lap exit %d, stderr: %s", code, errStr)
	}

	activePath := filepath.Join(beadsDir, "stints", "auth.laps.json")
	archivePath := filepath.Join(beadsDir, "stints", "archive", "auth.laps.json")
	if _, err := os.Stat(activePath); !os.IsNotExist(err) {
		t.Fatalf("active stint file should be archived, stat err: %v", err)
	}
	archivedFile, err := store.Load(archivePath)
	if err != nil {
		t.Fatalf("Load archived auth stint file: %v", err)
	}
	archivedLap := taskByIDInFile(t, archivedFile, authLapID)
	if !archivedLap.IsDone || archivedLap.CompletedAt == nil {
		t.Fatalf("archived lap should be completed, got %+v", archivedLap)
	}

	rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	rootRef := taskByIDInFile(t, rootFile, "root-auth")
	if !rootRef.IsDone || rootRef.CompletedAt == nil {
		t.Fatalf("root stint ref should be done after drain, got %+v", rootRef)
	}
}

// TestDoneNonHeadStintDrainsContentBased asserts a preempted/non-head stint
// still drains when its final lap is completed explicitly in that stint scope.
// (task 7.5)
func TestDoneNonHeadStintDrainsContentBased(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	writeResolverQueue(t, filepath.Join(beadsDir, "laps.json"), "",
		store.Task{Kind: store.KindStint, ID: "root-search", Ref: "search", Title: "Search stint", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
		store.Task{Kind: store.KindStint, ID: "root-auth", Ref: "auth", Title: "Auth stint", Order: 2, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "search.laps.json"), "srch",
		store.Task{Kind: store.KindLap, ID: "srch-head", Title: "Search head", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "auth.laps.json"), "auth",
		store.Task{Kind: store.KindLap, ID: "auth-tail", Title: "Auth tail", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)

	if _, errStr, code := runMB("done", "--stint", "auth", "auth-tail"); code != 0 {
		t.Fatalf("done non-head auth lap exit %d, stderr: %s", code, errStr)
	}

	if _, err := os.Stat(filepath.Join(beadsDir, "stints", "auth.laps.json")); !os.IsNotExist(err) {
		t.Fatalf("non-head drained auth file should be archived, stat err: %v", err)
	}
	if _, err := store.Load(filepath.Join(beadsDir, "stints", "archive", "auth.laps.json")); err != nil {
		t.Fatalf("Load archived non-head auth file: %v", err)
	}
	if _, err := store.Load(filepath.Join(beadsDir, "stints", "search.laps.json")); err != nil {
		t.Fatalf("preempting search stint should remain active: %v", err)
	}

	rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	searchRef := taskByIDInFile(t, rootFile, "root-search")
	authRef := taskByIDInFile(t, rootFile, "root-auth")
	if searchRef.IsDone {
		t.Fatalf("head search ref should remain todo, got %+v", searchRef)
	}
	if !authRef.IsDone || authRef.CompletedAt == nil {
		t.Fatalf("non-head auth ref should be done after drain, got %+v", authRef)
	}
}

// TestDoneUndoUnarchivesAndReopensArchivedStint asserts undo honors the age
// gate before unarchiving, then with --yes restores the stint file, reopens the
// root ref, and reopens the lap. (task 7.5)
func TestDoneUndoUnarchivesAndReopensArchivedStint(t *testing.T) {
	beadsDir, authLapID := setupActiveStintRepo(t)

	if _, errStr, code := runMB("done", authLapID); code != 0 {
		t.Fatalf("done final stint lap exit %d, stderr: %s", code, errStr)
	}

	archivePath := filepath.Join(beadsDir, "stints", "archive", "auth.laps.json")
	archivedFile, err := store.Load(archivePath)
	if err != nil {
		t.Fatalf("Load archived auth stint file: %v", err)
	}
	oldCompletion := time.Now().UTC().Add(-UndoAgeLimit - time.Minute)
	archivedLap := taskByIDInFile(t, archivedFile, authLapID)
	archivedLap.CompletedAt = &oldCompletion
	archivedLap.UpdatedAt = oldCompletion
	if err := store.Save(archivePath, archivedFile); err != nil {
		t.Fatalf("Save old archived auth stint file: %v", err)
	}

	if _, errStr, code := runMB("done", "undo"); code != 3 {
		t.Fatalf("old archived undo exit %d, want 3; stderr: %s", code, errStr)
	} else if !strings.Contains(errStr, "use 'laps done undo -y' to force") {
		t.Fatalf("old archived undo should enforce age gate, got: %s", errStr)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("age-gated undo should leave archived file in place: %v", err)
	}
	if _, err := os.Stat(filepath.Join(beadsDir, "stints", "auth.laps.json")); !os.IsNotExist(err) {
		t.Fatalf("age-gated undo should not restore active file, stat err: %v", err)
	}

	if _, errStr, code := runMB("done", "undo", "-y"); code != 0 {
		t.Fatalf("forced undo archived stint exit %d, stderr: %s", code, errStr)
	}

	activePath := filepath.Join(beadsDir, "stints", "auth.laps.json")
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Fatalf("forced undo should remove archived file, stat err: %v", err)
	}
	activeFile, err := store.Load(activePath)
	if err != nil {
		t.Fatalf("Load restored auth stint file: %v", err)
	}
	reopenedLap := taskByIDInFile(t, activeFile, authLapID)
	if reopenedLap.IsDone || reopenedLap.CompletedAt != nil {
		t.Fatalf("forced undo should reopen archived lap, got %+v", reopenedLap)
	}
	rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	rootRef := taskByIDInFile(t, rootFile, "root-auth")
	if rootRef.IsDone || rootRef.CompletedAt != nil {
		t.Fatalf("forced undo should reopen root stint ref, got %+v", rootRef)
	}
}

// TestStintsLifecycleLsShowAliasAndUnqueuedDisplay covers the task-8 command
// surface for creating, listing, showing, enqueueing, and the `st` alias.
func TestStintsLifecycleLsShowAliasAndUnqueuedDisplay(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	if out, errStr, code := runMB("stints", "new", "auth"); code != 0 {
		t.Fatalf("stints new exit %d, stderr: %s, stdout: %s", code, errStr, out)
	}
	if out, errStr, code := runMB("add", "head", "--stint", "auth", "--title", "Inside auth"); code != 0 {
		t.Fatalf("add --stint exit %d, stderr: %s, stdout: %s", code, errStr, out)
	}

	out, errStr, code := runMB("stints", "ls")
	if code != 0 {
		t.Fatalf("stints ls exit %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "auth") || !strings.Contains(out, "laps=1") || !strings.Contains(out, "queued=false") {
		t.Fatalf("unqueued stint should be listed as ordinary queued=false file, got: %s", out)
	}

	aliasOut, aliasErr, err := runMBExecute("st", "ls")
	if err != nil {
		t.Fatalf("st alias Execute returned error: %v, stderr: %s", err, aliasErr)
	}
	if aliasOut != out {
		t.Fatalf("st alias output mismatch\nstints: %q\nst:     %q", out, aliasOut)
	}

	out, errStr, code = runMB("stints", "show", "auth")
	if code != 0 {
		t.Fatalf("stints show exit %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "auth/") || !strings.Contains(out, "Inside auth") {
		t.Fatalf("stints show output should include name and lap, got: %s", out)
	}

	if out, errStr, code := runMB("stints", "enqueue", "auth"); code != 0 {
		t.Fatalf("stints enqueue exit %d, stderr: %s, stdout: %s", code, errStr, out)
	}
	out, errStr, code = runMB("stints", "ls")
	if code != 0 {
		t.Fatalf("stints ls queued exit %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "queued=true") {
		t.Fatalf("enqueued stint should list queued=true, got: %s", out)
	}

	if _, err := store.Load(filepath.Join(beadsDir, "stints", "auth.laps.json")); err != nil {
		t.Fatalf("lifecycle should leave auth stint readable: %v", err)
	}
}

// TestListTreeAndStintRefLineRendering verifies the mixed root queue remains
// legible without --tree and --tree descends through nested stint refs.
func TestListTreeAndStintRefLineRendering(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	writeResolverQueue(t, filepath.Join(beadsDir, "laps.json"), "",
		store.Task{Kind: store.KindStint, ID: "root-auth", Ref: "auth", Title: "Auth stint", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
		store.Task{Kind: store.KindLap, ID: "root-lap", Title: "Root work", Order: 2, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "auth.laps.json"), "auth",
		store.Task{Kind: store.KindLap, ID: "auth-lap", Title: "Auth work", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
		store.Task{Kind: store.KindStint, ID: "auth-search", Ref: "search", Title: "Search stint", Order: 2, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "search.laps.json"), "srch",
		store.Task{Kind: store.KindLap, ID: "srch-lap", Title: "Search work", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
	)

	out, errStr, code := runMB("list", "--root")
	if code != 0 {
		t.Fatalf("list --root exit %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "auth/ (stint - 2 laps)") || !strings.Contains(out, "Root work") {
		t.Fatalf("mixed root list should render stint ref as one summary line, got: %s", out)
	}
	if strings.Contains(out, "Auth work") {
		t.Fatalf("non-tree root list should not descend into auth, got: %s", out)
	}

	out, errStr, code = runMB("list", "--tree", "--root")
	if code != 0 {
		t.Fatalf("list --tree --root exit %d, stderr: %s", code, errStr)
	}
	for _, want := range []string{"auth/ (stint - 2 laps)", "Auth work", "search/ (stint - 1 laps)", "Search work", "Root work"} {
		if !strings.Contains(out, want) {
			t.Fatalf("tree output missing %q:\n%s", want, out)
		}
	}
	if !idxBefore(out, "auth/ (stint - 2 laps)", "Auth work") || !idxBefore(out, "search/ (stint - 1 laps)", "Search work") {
		t.Fatalf("tree output should nest children after their refs, got:\n%s", out)
	}
}

// TestStintsRmSafetyMatrix covers default refusal for non-archived queued or
// claimed stints, --force root-ref and claim cleanup, default unqueued removal,
// and default archived cleanup including a done root ref.
func TestStintsRmSafetyMatrix(t *testing.T) {
	t.Run("refuses queued active by default", func(t *testing.T) {
		beadsDir, _ := setupActiveStintRepo(t)

		_, errStr, code := runMB("stints", "rm", "auth")
		if code == 0 {
			t.Fatalf("expected queued active rm to fail")
		}
		if !strings.Contains(errStr, "queued") || !strings.Contains(errStr, "--force") {
			t.Fatalf("queued refusal should name --force, got: %s", errStr)
		}
		if _, err := os.Stat(filepath.Join(beadsDir, "stints", "auth.laps.json")); err != nil {
			t.Fatalf("refused rm should leave active file: %v", err)
		}
		rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
		if err != nil {
			t.Fatalf("Load root file: %v", err)
		}
		if findStintRef(rootFile, "auth") == nil {
			t.Fatalf("refused rm should leave root ref, tasks = %#v", rootFile.Tasks)
		}
	})

	t.Run("force removes queued and clears claim", func(t *testing.T) {
		beadsDir, authLapID := setupActiveStintRepo(t)
		if _, errStr, code := runMB("claim", authLapID); code != 0 {
			t.Fatalf("claim setup failed, code %d, stderr: %s", code, errStr)
		}

		if _, errStr, code := runMB("stints", "rm", "auth", "--force"); code != 0 {
			t.Fatalf("forced rm exit %d, stderr: %s", code, errStr)
		}
		if _, err := os.Stat(filepath.Join(beadsDir, "stints", "auth.laps.json")); !os.IsNotExist(err) {
			t.Fatalf("forced rm should remove active file, stat err: %v", err)
		}
		rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
		if err != nil {
			t.Fatalf("Load root file: %v", err)
		}
		if findStintRef(rootFile, "auth") != nil {
			t.Fatalf("forced rm should remove matching root refs, tasks = %#v", rootFile.Tasks)
		}
		if _, err := os.Stat(filepath.Join(beadsDir, "claim")); !os.IsNotExist(err) {
			t.Fatalf("forced rm should clear matching claim, stat err: %v", err)
		}
	})

	t.Run("removes unqueued non archived by default", func(t *testing.T) {
		beadsDir, cleanup := setupTempRepo(t)
		defer cleanup()
		writeResolverQueue(t, filepath.Join(beadsDir, "stints", "draft.laps.json"), "drft",
			store.Task{Kind: store.KindLap, ID: "drft-lap", Title: "Draft", Order: 1, CreatedAt: resolverTestTime, UpdatedAt: resolverTestTime},
		)

		if _, errStr, code := runMB("stints", "rm", "draft"); code != 0 {
			t.Fatalf("unqueued rm exit %d, stderr: %s", code, errStr)
		}
		if _, err := os.Stat(filepath.Join(beadsDir, "stints", "draft.laps.json")); !os.IsNotExist(err) {
			t.Fatalf("unqueued rm should remove active file, stat err: %v", err)
		}
	})

	t.Run("removes archived with done ref by default", func(t *testing.T) {
		beadsDir, authLapID := setupActiveStintRepo(t)
		if _, errStr, code := runMB("done", authLapID); code != 0 {
			t.Fatalf("done setup exit %d, stderr: %s", code, errStr)
		}

		if _, errStr, code := runMB("stints", "rm", "auth"); code != 0 {
			t.Fatalf("archived rm exit %d, stderr: %s", code, errStr)
		}
		if _, err := os.Stat(filepath.Join(beadsDir, "stints", "archive", "auth.laps.json")); !os.IsNotExist(err) {
			t.Fatalf("archived rm should remove archived file, stat err: %v", err)
		}
		rootFile, err := store.Load(filepath.Join(beadsDir, "laps.json"))
		if err != nil {
			t.Fatalf("Load root file: %v", err)
		}
		if findStintRef(rootFile, "auth") != nil {
			t.Fatalf("archived rm should remove done root ref, tasks = %#v", rootFile.Tasks)
		}
	})
}

// TestDoneArchiveCollisionIsRefusedWithoutOverwriting asserts a drain blocked
// by an existing archive target does not overwrite the archived file. (task 7.5)
func TestDoneArchiveCollisionIsRefusedWithoutOverwriting(t *testing.T) {
	beadsDir, authLapID := setupActiveStintRepo(t)
	archivePath := filepath.Join(beadsDir, "stints", "archive", "auth.laps.json")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := []byte(`{"version":3,"prefix":"old","tasks":[]}`)
	if err := os.WriteFile(archivePath, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}

	_, errStr, code := runMB("done", authLapID)
	if code == 0 {
		t.Fatalf("expected archive collision to fail, got exit 0")
	}
	if !strings.Contains(errStr, "archived stint file already exists") {
		t.Fatalf("expected archive collision error, got: %s", errStr)
	}
	got, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Read archive collision target: %v", err)
	}
	if string(got) != string(sentinel) {
		t.Fatalf("archive collision target was overwritten: got %s want %s", got, sentinel)
	}
}

// TestFinishStintDrainPartialFailureKeepsCompletedLapWithoutDoneRef asserts the
// fail-safe ordering after the stint lap has been persisted: archive failure
// leaves the completed stint file in place and does not flip the root ref done.
// (task 7.5)
func TestFinishStintDrainPartialFailureKeepsCompletedLapWithoutDoneRef(t *testing.T) {
	beadsDir, authLapID := setupActiveStintRepo(t)
	activePath := filepath.Join(beadsDir, "stints", "auth.laps.json")
	archivePath := filepath.Join(beadsDir, "stints", "archive", "auth.laps.json")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := []byte(`{"version":3,"prefix":"old","tasks":[]}`)
	if err := os.WriteFile(archivePath, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	activeFile, err := store.Load(activePath)
	if err != nil {
		t.Fatalf("Load active auth stint file: %v", err)
	}
	completedLap := taskByIDInFile(t, activeFile, authLapID)
	completedLap.IsDone = true
	completedLap.CompletedAt = &now
	completedLap.UpdatedAt = now
	if err := store.Save(activePath, activeFile); err != nil {
		t.Fatalf("Save completed active auth stint file: %v", err)
	}

	rootPath := filepath.Join(beadsDir, "laps.json")
	rootFile, err := store.Load(rootPath)
	if err != nil {
		t.Fatalf("Load root file: %v", err)
	}
	rootRef := taskByIDInFile(t, rootFile, "root-auth")
	err = finishStintDrain(&pendingStintDrain{
		RootPath: rootPath,
		RootFile: rootFile,
		RootRef:  rootRef,
		Stint:    "auth",
		Src:      activePath,
		Dst:      archivePath,
	}, now)
	if err == nil {
		t.Fatalf("expected archive collision to fail")
	}

	activeAfter, err := store.Load(activePath)
	if err != nil {
		t.Fatalf("completed active file should remain loadable: %v", err)
	}
	lapAfter := taskByIDInFile(t, activeAfter, authLapID)
	if !lapAfter.IsDone || lapAfter.CompletedAt == nil {
		t.Fatalf("partial failure should preserve completed lap, got %+v", lapAfter)
	}
	rootAfter, err := store.Load(rootPath)
	if err != nil {
		t.Fatalf("Load root after partial failure: %v", err)
	}
	refAfter := taskByIDInFile(t, rootAfter, "root-auth")
	if refAfter.IsDone || refAfter.CompletedAt != nil {
		t.Fatalf("partial failure must not leave done ref over present file, got %+v", refAfter)
	}
	gotArchive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Read archive collision target after partial failure: %v", err)
	}
	if string(gotArchive) != string(sentinel) {
		t.Fatalf("partial failure overwrote archive target: got %s want %s", gotArchive, sentinel)
	}
}

func TestAddMissingPosition(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("add", "--title", "X")
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errStr, "position required") {
		t.Fatalf("expected position required error, got: %s", errStr)
	}
}

func TestAddMutualExclusion(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("add", "head", "--title", "X", "--json", "{}")
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errStr, "mutually exclusive") {
		t.Fatalf("expected mutual exclusion error, got: %s", errStr)
	}
}

func TestAddWithAssignee(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("add", "head", "--title", "Assigned task", "--assignee", "  alice  ")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	id := strings.TrimSpace(out)

	data, err := os.ReadFile(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"assignee": "alice"`) {
		t.Fatalf("expected trimmed assignee in store for %s, got: %s", id, data)
	}
}

func TestAddJSONWithAssignee(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("add", "head", "--json", `{"title":"JSON task","assignee":"bob"}`)
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	id := strings.TrimSpace(out)

	out, _, code = runMB("list", "--oneline")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, id+" — JSON task (assignee: bob)") {
		t.Fatalf("expected assignee in list output, got: %s", out)
	}
}

func TestAddJSONRejectsOtherInputFlags(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("add", "head", "--json", `{"title":"JSON task"}`, "--assignee", "alice")
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errStr, "mutually exclusive") {
		t.Fatalf("expected mutual exclusion error, got: %s", errStr)
	}
}

func TestAddRequiresJSONTitle(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("add", "head", "--json", `{"assignee":"alice"}`)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errStr, "title is required") {
		t.Fatalf("expected title required error, got: %s", errStr)
	}
}

func TestAddJSONArrayPreservesOrderByPosition(t *testing.T) {
	tests := []struct {
		name     string
		position []string
		setup    func() string
	}{
		{name: "head", position: []string{"head"}},
		{name: "tail", position: []string{"tail"}},
		{
			name:     "after",
			position: []string{"after"},
			setup: func() string {
				out, _, _ := runMB("add", "tail", "--title", "Anchor")
				return strings.TrimSpace(out)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupTempRepo(t)
			defer cleanup()

			args := append([]string{"add"}, tt.position...)
			if tt.setup != nil {
				args = append(args, tt.setup())
			}
			args = append(args, "--json", `[{"title":"First"},{"title":"Second"},{"title":"Third"}]`)
			out, errStr, code := runMB(args...)
			if code != 0 {
				t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
			}
			if got := len(strings.Fields(out)); got != 3 {
				t.Fatalf("expected 3 ids, got %d: %q", got, out)
			}

			list, _, _ := runMB("list")
			if !idxBefore(list, "First", "Second") || !idxBefore(list, "Second", "Third") {
				t.Fatalf("expected input order to be preserved, got:\n%s", list)
			}
			if tt.name == "after" && !idxBefore(list, "Anchor", "First") {
				t.Fatalf("expected batch after anchor, got:\n%s", list)
			}
		})
	}
}

func TestAddJSONArrayValidationIsAtomic(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("add", "tail", "--json", `[{"title":"Valid"},{"assignee":"alice"}]`)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errStr, "task 2 title is required") {
		t.Fatalf("expected indexed title error, got: %s", errStr)
	}
	list, _, _ := runMB("list")
	if strings.Contains(list, "Valid") {
		t.Fatalf("expected no tasks from invalid batch, got:\n%s", list)
	}
}

func TestAddJSONArrayRejectsEmptyArray(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("add", "tail", "--json", `[]`)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errStr, "task array must not be empty") {
		t.Fatalf("expected empty array error, got: %s", errStr)
	}
}

func TestAddJSONDashReadsArrayFromStdin(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(`[{"title":"From stdin 1"},{"title":"From stdin 2"}]`); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
	}()

	out, errStr, code := runMB("add", "tail", "--json", "-")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if got := len(strings.Fields(out)); got != 2 {
		t.Fatalf("expected 2 ids, got output %q", out)
	}
	list, _, _ := runMB("list")
	if !idxBefore(list, "From stdin 1", "From stdin 2") {
		t.Fatalf("expected stdin batch order, got:\n%s", list)
	}
}

func TestAddJSONArrayHooksRunOnceWithoutSingleTaskContext(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	hooks := `{"version":1,"hooks":[{"title":"batch","command":"add","when":"after","run":"printf '%s|%s|%s' \"$id\" \"$title\" \"$output\" > .laps/add-hook.txt"}]}`
	if err := os.WriteFile(filepath.Join(beadsDir, "hooks.json"), []byte(hooks), 0o644); err != nil {
		t.Fatal(err)
	}

	out, errStr, code := runMB("add", "tail", "--json", `[{"title":"A"},{"title":"B"}]`)
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	data, err := os.ReadFile(filepath.Join(beadsDir, "add-hook.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := "||" + strings.TrimSpace(out)
	if got := string(data); got != want {
		t.Fatalf("expected one command-level hook context %q, got %q", want, got)
	}
}

func TestGetHead(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Task One", "--description", "Details")
	out, errStr, code := runMB("get")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "Task One") {
		t.Fatalf("expected Task One in output, got: %s", out)
	}
	if !strings.Contains(out, "Details") {
		t.Fatalf("expected Details in output, got: %s", out)
	}
}

func TestResolverGetDescendsIntoActiveStint(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	writeResolverQueue(t, filepath.Join(beadsDir, "laps.json"), "", store.Task{
		Kind:      store.KindStint,
		ID:        "root-auth",
		Ref:       "auth",
		Title:     "Auth stint",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	})
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "auth.laps.json"), "auth", store.Task{
		Kind:        store.KindLap,
		ID:          "auth-1111",
		Title:       "Inside auth",
		Description: "Nested details",
		Order:       1,
		CreatedAt:   resolverTestTime,
		UpdatedAt:   resolverTestTime,
	})

	out, errStr, code := runMB("get")
	if code != 0 {
		t.Fatalf("get exit %d, stderr: %s", code, errStr)
	}
	if out != "Inside auth\n\nNested details\n" {
		t.Fatalf("get output = %q", out)
	}
}

func TestResolverGetRecursesAcrossNestedStint(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	writeResolverQueue(t, filepath.Join(beadsDir, "laps.json"), "", store.Task{
		Kind:      store.KindStint,
		ID:        "root-auth",
		Ref:       "auth",
		Title:     "Auth stint",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	})
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "auth.laps.json"), "auth", store.Task{
		Kind:      store.KindStint,
		ID:        "auth-search",
		Ref:       "search",
		Title:     "Search stint",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	})
	writeResolverQueue(t, filepath.Join(beadsDir, "stints", "search.laps.json"), "srch", store.Task{
		Kind:        store.KindLap,
		ID:          "srch-1111",
		Title:       "Inside search",
		Description: "Deep details",
		Order:       1,
		CreatedAt:   resolverTestTime,
		UpdatedAt:   resolverTestTime,
	})

	out, errStr, code := runMB("get")
	if code != 0 {
		t.Fatalf("get exit %d, stderr: %s", code, errStr)
	}
	if out != "Inside search\n\nDeep details\n" {
		t.Fatalf("get output = %q", out)
	}
}

func TestResolverMissingChildFileError(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	writeResolverQueue(t, filepath.Join(beadsDir, "laps.json"), "", store.Task{
		Kind:      store.KindStint,
		ID:        "root-missing",
		Ref:       "missing",
		Title:     "Missing stint",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	})

	_, errStr, code := runMB("get")
	if code != 2 {
		t.Fatalf("get exit %d, want 2; stderr: %s", code, errStr)
	}
	if !strings.Contains(errStr, "missing child file") || !strings.Contains(errStr, "missing") {
		t.Fatalf("missing-child error was not classified: %s", errStr)
	}
}

func TestResolverMalformedRefError(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	writeResolverQueue(t, filepath.Join(beadsDir, "laps.json"), "", store.Task{
		Kind:      store.KindStint,
		ID:        "root-bad-ref",
		Ref:       "../auth",
		Title:     "Bad ref",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	})

	_, errStr, code := runMB("get")
	if code != 2 {
		t.Fatalf("get exit %d, want 2; stderr: %s", code, errStr)
	}
	if !strings.Contains(errStr, "malformed stint ref") || !strings.Contains(errStr, "../auth") {
		t.Fatalf("malformed-ref error was not classified: %s", errStr)
	}
}

func TestResolverMalformedChildFileError(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	writeResolverQueue(t, filepath.Join(beadsDir, "laps.json"), "", store.Task{
		Kind:      store.KindStint,
		ID:        "root-auth",
		Ref:       "auth",
		Title:     "Auth stint",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	})
	stintPath := filepath.Join(beadsDir, "stints", "auth.laps.json")
	if err := os.MkdirAll(filepath.Dir(stintPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stintPath, []byte(`{"version":3,"prefix":"auth","tasks":[`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, errStr, code := runMB("get")
	if code != 2 {
		t.Fatalf("get exit %d, want 2; stderr: %s", code, errStr)
	}
	if !strings.Contains(errStr, "malformed child file") || !strings.Contains(errStr, "auth") {
		t.Fatalf("malformed-child error was not classified: %s", errStr)
	}
}

func TestResolverCycleErrorForActualStintGraphDoesNotMutateFiles(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	rootPath := filepath.Join(beadsDir, "laps.json")
	authPath := filepath.Join(beadsDir, "stints", "auth.laps.json")
	searchPath := filepath.Join(beadsDir, "stints", "search.laps.json")
	writeResolverQueue(t, rootPath, "", store.Task{
		Kind:      store.KindStint,
		ID:        "root-auth",
		Ref:       "auth",
		Title:     "Auth stint",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	})
	writeResolverQueue(t, authPath, "auth", store.Task{
		Kind:      store.KindStint,
		ID:        "auth-search",
		Ref:       "search",
		Title:     "Search stint",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	})
	writeResolverQueue(t, searchPath, "srch", store.Task{
		Kind:      store.KindStint,
		ID:        "search-auth",
		Ref:       "auth",
		Title:     "Auth again",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	})
	before := readResolverFiles(t, rootPath, authPath, searchPath)

	_, errStr, code := runMB("get")
	if code != 2 {
		t.Fatalf("get exit %d, want 2; stderr: %s", code, errStr)
	}
	if !strings.Contains(errStr, "cycle detected") || !strings.Contains(errStr, "auth") {
		t.Fatalf("cycle error was not classified: %s", errStr)
	}
	after := readResolverFiles(t, rootPath, authPath, searchPath)
	for path, beforeData := range before {
		if after[path] != beforeData {
			t.Fatalf("%s mutated during cycle resolution\nbefore: %s\nafter:  %s", path, beforeData, after[path])
		}
	}
}

func TestGetOutputUnchangedWithoutAssignee(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Task One", "--description", "Details")
	out, errStr, code := runMB("get")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if out != "Task One\n\nDetails\n" {
		t.Fatalf("expected legacy get output, got: %q", out)
	}
}

func TestGetOutputIncludesAssignee(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Task One", "--description", "Details", "--assignee", "alice")
	out, errStr, code := runMB("get")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if out != "Task One\nAssignee: alice\n\nDetails\n" {
		t.Fatalf("expected assignee in get output, got: %q", out)
	}
}

func TestGetNoHead(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("get")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "no head task") {
		t.Fatalf("expected no head task, got: %s", errStr)
	}
}

func TestGetByID(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Task A")
	id := strings.TrimSpace(out)
	out, _, code := runMB("get", id)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "Task A") {
		t.Fatalf("expected Task A, got: %s", out)
	}
}

func TestListDefault(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	runMB("add", "tail", "--title", "B")
	out, _, code := runMB("list")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	// Two-line default: two laps render on four lines.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %q", len(lines), out)
	}
}

func TestListOutputUnchangedWithoutAssignee(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "tail", "--title", "A")
	id := strings.TrimSpace(out)
	// --oneline preserves the prior single-line shape byte-for-byte.
	out, _, code := runMB("list", "--oneline")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	want := fmt.Sprintf("1. %s — A\n", id)
	if out != want {
		t.Fatalf("expected legacy list output %q, got %q", want, out)
	}
}

func TestListOutputIncludesAssignee(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "tail", "--title", "A", "--assignee", "alice")
	id := strings.TrimSpace(out)
	// --oneline appends the assignee clause when the assignee is set.
	out, _, code := runMB("list", "--oneline")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	want := fmt.Sprintf("1. %s — A (assignee: alice)\n", id)
	if out != want {
		t.Fatalf("expected assignee in list output %q, got %q", want, out)
	}
}

func TestListTwoLineStructure(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "tail", "--title", "Title One", "--assignee", "alice")
	id := strings.TrimSpace(out)

	out, _, code := runMB("list")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines for 1 lap (two-line default), got %d: %q", len(lines), out)
	}
	if want := "1. Title One"; lines[0] != want {
		t.Fatalf("line 1 = %q, want %q", lines[0], want)
	}
	if want := "   " + id + " · alice · todo"; lines[1] != want {
		t.Fatalf("line 2 = %q, want %q", lines[1], want)
	}
}

func TestListTwoLineEmDashWhenUnassigned(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "Unassigned")

	out, _, code := runMB("list")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	// Unset assignee renders as an em dash, never an empty field.
	if !strings.Contains(out, " · — · todo") {
		t.Fatalf("expected em-dash assignee placeholder, got: %s", out)
	}
}

func TestListMarksClaimedLap(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	outA, _, _ := runMB("add", "head", "--title", "A")
	idA := strings.TrimSpace(outA)
	runMB("add", "tail", "--title", "B")
	if _, _, code := runMB("claim", idA); code != 0 {
		t.Fatalf("claim failed, code %d", code)
	}

	out, _, code := runMB("list")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	// Exactly one active marker, and it sits on the claimed lap's title line.
	if n := strings.Count(out, "> "); n != 1 {
		t.Fatalf("expected exactly one active marker, got %d: %q", n, out)
	}
	if !strings.Contains(out, "1. > A") {
		t.Fatalf("expected claimed lap A marked, got: %s", out)
	}
	if strings.Contains(out, "2. > B") {
		t.Fatalf("unclaimed lap B must not be marked, got: %s", out)
	}
}

func TestListNoMarkerWithoutClaim(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "A")
	runMB("add", "tail", "--title", "B")

	out, _, code := runMB("list")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.Contains(out, "> ") {
		t.Fatalf("expected no active marker without a claim, got: %s", out)
	}
}

func TestListNoMarkerWhenClaimOutsideResult(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "A")
	outB, _, _ := runMB("add", "tail", "--title", "B")
	idB := strings.TrimSpace(outB)
	// Delete B, then write a stale claim pointing at it. The claim is outside the
	// rendered result, so no marker may appear.
	if _, _, code := runMB("delete", idB); code != 0 {
		t.Fatalf("delete failed, code %d", code)
	}
	if err := store.WriteClaim(beadsDir, store.Claim{Lap: idB, File: "laps.json"}); err != nil {
		t.Fatalf("write stale claim: %v", err)
	}

	out, _, code := runMB("list")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.Contains(out, "> ") {
		t.Fatalf("expected no marker when claim is outside result, got: %s", out)
	}
	if !strings.Contains(out, "1. A") {
		t.Fatalf("expected remaining lap A in list, got: %s", out)
	}
}

func TestListOnelineOmitsMarker(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "A")
	id := strings.TrimSpace(out)
	runMB("claim") // claim the head (A)

	// The active marker only applies to the two-line default; --oneline keeps the
	// legacy shape, so even a claimed lap renders without a marker.
	out, _, code := runMB("list", "--oneline")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	want := fmt.Sprintf("1. %s — A\n", id)
	if out != want {
		t.Fatalf("expected oneline shape without marker %q, got %q", want, out)
	}
}

func TestListStrikesDoneTitleOnlyInTwoLineMode(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Done one")
	id := strings.TrimSpace(out)
	runMB("claim", id)
	if _, _, code := runMB("done"); code != 0 {
		t.Fatalf("setup done failed, code %d", code)
	}

	out, _, code := runMB("list", "--done")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	want := fmt.Sprintf("1. ~~Done one~~\n   %s · — · done\n", id)
	if out != want {
		t.Fatalf("expected title-only strike %q, got %q", want, out)
	}
}

func TestListOnelineStrikesWholeDoneLine(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Done one", "--assignee", "al")
	id := strings.TrimSpace(out)
	runMB("claim", id)
	if _, _, code := runMB("done"); code != 0 {
		t.Fatalf("setup done failed, code %d", code)
	}

	out, _, code := runMB("list", "--done", "--oneline")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	want := fmt.Sprintf("1. ~~%s — Done one (assignee: al)~~\n", id)
	if out != want {
		t.Fatalf("expected whole-line strike %q, got %q", want, out)
	}
}

func TestLSAliasMatchesList(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	runMB("add", "tail", "--title", "B", "--assignee", "alice")

	cases := [][]string{
		{},
		{"--oneline"},
		{"--json-output"},
	}
	for _, flags := range cases {
		listArgs := append([]string{"list"}, flags...)
		lsArgs := append([]string{"ls"}, flags...)
		listOut, _, listCode := runMB(listArgs...)
		lsOut, _, lsCode := runMB(lsArgs...)
		if lsCode != listCode {
			t.Fatalf("ls %v exit %d != list exit %d", flags, lsCode, listCode)
		}
		if lsOut != listOut {
			t.Fatalf("ls %v output != list output\nlist: %q\nls:   %q", flags, listOut, lsOut)
		}
	}
}

func TestLSAliasDispatchThroughExecute(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Via alias")

	// A hook registered for the command name "ls" can only fire through the
	// hook-only intercept. Because ls is a recognized built-in, the intercept is
	// skipped and listCmd runs instead — so this hook must NOT run.
	hooks := `{"version":1,"hooks":[{"title":"intercept","command":"ls","when":"before","run":"echo HOOK-RAN","passback":true}]}`
	if err := os.WriteFile(filepath.Join(beadsDir, "hooks.json"), []byte(hooks), 0o644); err != nil {
		t.Fatal(err)
	}

	out, errStr, err := runMBExecute("ls")
	if err != nil {
		t.Fatalf("expected nil error dispatching ls, got %v, stderr: %s", err, errStr)
	}
	if strings.Contains(out, "HOOK-RAN") {
		t.Fatalf("ls must not be intercepted as a hook-only command, got: %s", out)
	}
	if !strings.Contains(out, "Via alias") {
		t.Fatalf("expected list output through ls alias, got: %s", out)
	}
}

func TestLSAliasJSONOutput(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	runMB("add", "tail", "--title", "B")

	listOut, _, code := runMB("list", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	lsOut, _, code := runMB("ls", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if lsOut != listOut {
		t.Fatalf("ls --json-output != list --json-output\nlist: %q\nls:   %q", listOut, lsOut)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(lsOut)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", lsOut)
	}
	tasks, ok := result["tasks"].([]interface{})
	if !ok {
		t.Fatalf("expected tasks array in JSON, got: %s", lsOut)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestDoneWithClaim(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Do me")
	id := strings.TrimSpace(out)
	_, _, code := runMB("claim")
	if code != 0 {
		t.Fatalf("claim failed, code %d", code)
	}
	out, _, code = runMB("done")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "Do me") {
		t.Fatalf("expected title 'Do me' in output, got: %s", out)
	}
	if !strings.Contains(out, "laps done undo") {
		t.Fatalf("expected undo hint in output, got: %s", out)
	}
	claimPath := filepath.Join(beadsDir, "claim")
	if _, err := os.Stat(claimPath); !os.IsNotExist(err) {
		t.Fatal("expected claim file to be removed after done")
	}
	data, _ := store.Load(filepath.Join(beadsDir, "laps.json"))
	for _, task := range data.Tasks {
		if task.ID == id && !task.IsDone {
			t.Fatal("expected task to be done")
		}
	}
}

func TestDoneNoHeadNoClaim(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("done")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "no claimed lap and no head task") {
		t.Fatalf("expected no claimed lap and no head task, got: %s", errStr)
	}
}

func TestDelete(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Delete me")
	id := strings.TrimSpace(out)
	_, _, code := runMB("delete", id)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	_, errStr, code := runMB("get", id)
	if code != 3 {
		t.Fatalf("expected code 3 after delete, got %d", code)
	}
	if !strings.Contains(errStr, "task not found") {
		t.Fatalf("expected task not found, got: %s", errStr)
	}
}

func TestDeleteNotFound(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("delete", "nonexistent")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "task not found") {
		t.Fatalf("expected task not found, got: %s", errStr)
	}
}

func TestPruneDefault(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	for i := 0; i < 25; i++ {
		_, _, _ = runMB("add", "tail", "--title", fmt.Sprintf("Task %d", i))
	}
	for i := 0; i < 25; i++ {
		runMB("claim")
		_, _, _ = runMB("done")
	}

	out, _, code := runMB("prune")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.TrimSpace(out) != "5" {
		t.Fatalf("expected 5 removed, got %s", out)
	}

	// --oneline keeps one line per lap so the count maps to lap count.
	out, _, code = runMB("list", "--done", "--oneline")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 20 {
		t.Fatalf("expected 20 done tasks, got %d", len(lines))
	}
}

func TestPruneZero(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, _, _ = runMB("add", "tail", "--title", "A")
	runMB("claim")
	_, _, _ = runMB("done")

	out, _, code := runMB("prune", "0")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.TrimSpace(out) != "1" {
		t.Fatalf("expected 1 removed, got %s", out)
	}

	out, _, code = runMB("list", "--done")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty done list, got %s", out)
	}
}

func TestPruneEmpty(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, code := runMB("prune")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.TrimSpace(out) != "0" {
		t.Fatalf("expected 0 removed, got %s", out)
	}
}

func TestHookBeforeAbort(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	hooks := `{"version":1,"hooks":[{"title":"abort","command":"get","when":"before","run":"exit 1","passback":false}]}`
	os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644)

	_, errStr, code := runMB("get")
	if code != 4 {
		t.Fatalf("expected code 4, got %d", code)
	}
	if !strings.Contains(errStr, "hook") {
		t.Fatalf("expected hook error, got: %s", errStr)
	}
}

func TestHookAfterRunsOnFailure(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	hooks := `{"version":1,"hooks":[{"title":"after","command":"get","when":"after","run":"echo after","passback":true}]}`
	os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644)

	out, _, code := runMB("get")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(out, "after") {
		t.Fatalf("expected after hook output, got: %s", out)
	}
}

func TestHookPassback(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "A")
	hooks := `{"version":1,"hooks":[{"title":"pass","command":"done","when":"after","run":"echo passback","passback":true}]}`
	os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644)

	runMB("claim")
	out, _, code := runMB("done")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "passback") {
		t.Fatalf("expected passback in output, got: %s", out)
	}
}

func TestAddCreatesMbJSONDespiteOtherJSON(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(beadsDir, "other.json"), []byte(`{"version":1,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	out, errStr, code := runMB("add", "head", "--title", "First task")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	id := strings.TrimSpace(out)
	if id == "" {
		t.Fatal("expected id output")
	}

	if _, err := os.Stat(filepath.Join(beadsDir, "laps.json")); err != nil {
		t.Fatalf("laps.json was not created: %v", err)
	}
}

func TestAddRejectsNonMbJSON(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	otherPath := filepath.Join(".laps", "other.json")
	if err := os.WriteFile(otherPath, []byte(`{"foo":"bar"}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, errStr, code := runMB("add", "head", "--title", "X", "-f", "other")
	if code != 2 {
		t.Fatalf("expected code 2, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(errStr, "not a valid laps task file") {
		t.Fatalf("expected invalid laps task file error, got: %s", errStr)
	}
}

func TestAddWorksWhenEmptyMbJSONExistsWithOtherJSON(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(beadsDir, "laps.json"), []byte(`{"version":1,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "other.json"), []byte(`{"version":1,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	out, errStr, code := runMB("add", "head", "--title", "Task with empty existing laps")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	id := strings.TrimSpace(out)
	if id == "" {
		t.Fatal("expected id output")
	}
}

func TestHookAssigneeVariable(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "A", "--assignee", "alice")
	hooks := `{"version":1,"hooks":[{"title":"assignee","command":"get","when":"after","run":"echo $assignee","passback":true}]}`
	os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644)

	out, _, code := runMB("get")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "alice") {
		t.Fatalf("expected assignee variable in hook output, got: %s", out)
	}
}

func TestBeforeHookTaskVariables(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "TaskA", "--description", "DescA", "--assignee", "alice")
	id := strings.TrimSpace(out)

	// Test GET before hook
	hooksGet := `{"version":1,"hooks":[{"title":"beforeGet","command":"get","when":"before","run":"echo GET:$id:$title:$description:$assignee","passback":true}]}`
	if err := os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooksGet), 0644); err != nil {
		t.Fatal(err)
	}
	out, _, _ = runMB("get", id)
	if !strings.Contains(out, "GET:"+id+":TaskA:DescA:alice") {
		t.Fatalf("expected GET before-hook variables, got: %s", out)
	}

	// Test DONE before hook
	hooksDone := `{"version":1,"hooks":[{"title":"beforeDone","command":"done","when":"before","run":"echo DONE:$id:$title:$description:$assignee","passback":true}]}`
	if err := os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooksDone), 0644); err != nil {
		t.Fatal(err)
	}
	runMB("claim")
	out, _, _ = runMB("done")
	if !strings.Contains(out, "DONE:"+id+":TaskA:DescA:alice") {
		t.Fatalf("expected DONE before-hook variables, got: %s", out)
	}

	// Add another task to delete
	out, _, _ = runMB("add", "head", "--title", "TaskB", "--description", "DescB", "--assignee", "bob")
	id2 := strings.TrimSpace(out)

	// Test DELETE before hook
	hooksDelete := `{"version":1,"hooks":[{"title":"beforeDelete","command":"delete","when":"before","run":"echo DELETE:$id:$title:$description:$assignee","passback":true}]}`
	if err := os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooksDelete), 0644); err != nil {
		t.Fatal(err)
	}
	out, _, _ = runMB("delete", id2)
	if !strings.Contains(out, "DELETE:"+id2+":TaskB:DescB:bob") {
		t.Fatalf("expected DELETE before-hook variables, got: %s", out)
	}
}

func TestHookArgsPassthrough(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "A")
	id := strings.TrimSpace(out)
	hooks := `{"version":1,"hooks":[{"title":"args","command":"get","when":"after","run":"echo $args $1 $2 $3","passback":true}]}`
	os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644)

	out, _, code := runMB("get", id, "extra1", "extra2")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	want := id + " extra1 extra2 " + id + " extra1 extra2"
	if !strings.Contains(out, want) {
		t.Fatalf("expected args in hook output, got: %s", out)
	}
}

func TestHookOnlyCommandArgsPassthrough(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	hooks := `{"version":1,"hooks":[{"title":"args","command":"worktree","when":"before","run":"echo $command $args $1 $2","passback":true}]}`
	if err := os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644); err != nil {
		t.Fatal(err)
	}

	out, errStr, err := runMBExecute("worktree", "feature", "*")
	if err != nil {
		t.Fatalf("expected nil error, got %v, stderr: %s", err, errStr)
	}
	if !strings.Contains(out, "worktree feature * feature *") {
		t.Fatalf("expected literal hook-only args in output, got: %s", out)
	}
}

// TestHookOnlyCommandFromSubdir is a regression test for running a hook-backed
// command (e.g. laps wrapup / laps done) from a nested subdirectory rather than
// the directory containing .laps/. Resolution must walk up to the ancestor
// .laps/, and hooks must execute in the repo root so repo-relative paths work.
func TestHookOnlyCommandFromSubdir(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	hooks := `{"version":1,"hooks":[{"title":"wrapup","command":"wrapup","when":"after","run":"printf wrapped > .laps/wrapup-hook.txt","passback":false}]}`
	if err := os.WriteFile(filepath.Join(beadsDir, "hooks.json"), []byte(hooks), 0644); err != nil {
		t.Fatal(err)
	}

	// setupTempRepo chdir'd us into the repo root; descend into a nested subdir.
	sub := filepath.Join(beadsDir, "..", "sub", "deeper")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}

	_, errStr, err := runMBExecute("wrapup")
	if err != nil {
		t.Fatalf("expected nil error running wrapup from subdir, got %v, stderr: %s", err, errStr)
	}

	data, readErr := os.ReadFile(filepath.Join(beadsDir, "wrapup-hook.txt"))
	if readErr != nil {
		t.Fatalf("expected hook to resolve ancestor .laps/ and run in repo root: %v", readErr)
	}
	if string(data) != "wrapped" {
		t.Fatalf("expected hook output 'wrapped', got %q", string(data))
	}
}

func TestHookOnlyCommandFileParsing(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	hooks := `{"version":1,"hooks":[{"title":"printfile","command":"customcmd","when":"before","run":"echo $file","passback":true}]}`
	if err := os.WriteFile(filepath.Join(beadsDir, "hooks.json"), []byte(hooks), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		args     []string
		expected string
	}{
		{[]string{"customcmd"}, "laps.json"},
		{[]string{"-f", "other.json", "customcmd"}, "other.json"},
		{[]string{"--file", "other2.json", "customcmd"}, "other2.json"},
		{[]string{"-f=other3.json", "customcmd"}, "other3.json"},
		{[]string{"--file=other4.json", "customcmd"}, "other4.json"},
	}

	for _, tc := range tests {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			out, errStr, err := runMBExecute(tc.args...)
			if err != nil {
				t.Fatalf("expected nil error, got %v, stderr: %s", err, errStr)
			}
			expectedPath := filepath.Join(beadsDir, tc.expected)
			if !strings.Contains(strings.TrimSpace(out), expectedPath) {
				t.Fatalf("expected output to contain %q, got: %q", expectedPath, out)
			}
		})
	}
}

func TestGetAcceptsExtraArgs(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "A")
	id := strings.TrimSpace(out)
	out, errStr, code := runMB("get", id, "extra")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "A") {
		t.Fatalf("expected task in output, got: %s", out)
	}
}

func TestDeleteAcceptsExtraArgs(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "A")
	id := strings.TrimSpace(out)
	_, errStr, code := runMB("delete", id, "extra")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
}

func TestDeleteRequiresId(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("delete")
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errStr, "task id required") {
		t.Fatalf("expected task id required error, got: %s", errStr)
	}
}

func TestPruneAcceptsExtraArgs(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, _, _ = runMB("add", "tail", "--title", "A")
	runMB("claim")
	_, _, _ = runMB("done")

	out, errStr, code := runMB("prune", "0", "extra")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if strings.TrimSpace(out) != "1" {
		t.Fatalf("expected 1 removed, got %s", out)
	}
}

func TestUpdateDevVersion(t *testing.T) {
	version = "dev"
	out, errStr, code := runMB("update")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "dev") {
		t.Fatalf("expected dev message, got: %s", out)
	}
}

func TestUpdateRunsHooksInRepo(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	hooks := `{"version":1,"hooks":[{"title":"before-update","command":"update","when":"before","run":"printf before > .laps/update-hook.txt"},{"title":"after-update","command":"update","when":"after","run":"printf after >> .laps/update-hook.txt"}]}`
	if err := os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644); err != nil {
		t.Fatal(err)
	}

	version = "dev"
	out, errStr, code := runMB("update")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "Current version: dev") {
		t.Fatalf("expected dev output, got: %s", out)
	}

	data, err := os.ReadFile(filepath.Join(".laps", "update-hook.txt"))
	if err != nil {
		t.Fatalf("expected update hook output file: %v", err)
	}
	if string(data) != "beforeafter" {
		t.Fatalf("expected update hooks to run, got %q", string(data))
	}
}

func TestUpdateYesInstallsWithoutPrompt(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	oldFetch := fetchLatestVersionFunc
	oldInstall := installLatestVersionFn
	defer func() {
		fetchLatestVersionFunc = oldFetch
		installLatestVersionFn = oldInstall
	}()

	fetchLatestVersionFunc = func() (string, error) {
		return "0.4.2", nil
	}

	installed := false
	installLatestVersionFn = func() error {
		installed = true
		return nil
	}

	version = "0.4.1"
	out, errStr, code := runMB("update", "--yes")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if !installed {
		t.Fatal("expected install to run")
	}
	if strings.Contains(out, "Update to latest version?") {
		t.Fatalf("expected no prompt with --yes, got: %s", out)
	}
	if !strings.Contains(out, "Latest version:  0.4.2") {
		t.Fatalf("expected latest version in output, got: %s", out)
	}
}

func TestShellQuoteArgs(t *testing.T) {
	got := shellQuoteArgs([]string{"plain", "*", "", "two words", "quote's"})
	want := "plain '*' '' 'two words' 'quote'\\''s'"
	if got != want {
		t.Fatalf("shellQuoteArgs() = %q, want %q", got, want)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b    string
		want    int
		wantErr bool
	}{
		{"0.2.0", "0.3.0", -1, false},
		{"0.3.0", "0.2.0", 1, false},
		{"0.3.0", "0.3.0", 0, false},
		{"0.2.1", "0.2.10", -1, false},
		{"1.0.0", "0.9.9", 1, false},
		{"dev", "0.3.0", 0, true},
	}
	for _, tt := range tests {
		got, err := compareVersions(tt.a, tt.b)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("compareVersions(%q, %q) expected error", tt.a, tt.b)
			}
			continue
		}
		if err != nil {
			t.Fatalf("compareVersions(%q, %q) unexpected error: %v", tt.a, tt.b, err)
		}
		if got != tt.want {
			t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCountCommand(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	// Initially, when no tasks are present
	out, errStr, code := runMB("count")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	expectedEmpty := "Laps done: 0 out of 0\n\nNo tasks found.\n"
	if out != expectedEmpty {
		t.Fatalf("expected output %q, got %q", expectedEmpty, out)
	}

	// Add some tasks
	_, _, code1 := runMB("add", "tail", "--title", "Task 1", "--assignee", "coder")
	_, _, code2 := runMB("add", "tail", "--title", "Task 2", "--assignee", "coder")
	_, _, code3 := runMB("add", "tail", "--title", "Task 3", "--assignee", "reviewer")
	_, _, code4 := runMB("add", "tail", "--title", "Task 4") // unassigned
	if code1 != 0 || code2 != 0 || code3 != 0 || code4 != 0 {
		t.Fatal("failed to add tasks")
	}

	// Complete the first task (Task 1, assigned to coder)
	runMB("claim")
	_, _, code5 := runMB("done")
	if code5 != 0 {
		t.Fatal("failed to complete task")
	}

	// Output should be:
	// Laps done: 1 out of 4
	//
	// Breakdown by role:
	// - coder: 1 complete, 1 incomplete
	// - reviewer: 0 complete, 1 incomplete
	// - unassigned: 0 complete, 1 incomplete
	out, errStr, code = runMB("count")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}

	expected := "Laps done: 1 out of 4\n\nBreakdown by role:\n- coder: 1 complete, 1 incomplete\n- reviewer: 0 complete, 1 incomplete\n- unassigned: 0 complete, 1 incomplete\n"
	if out != expected {
		t.Fatalf("expected output %q, got %q", expected, out)
	}
}

func TestOrderingHeadLandsBelowDone(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	runMB("add", "tail", "--title", "B")
	out, _, _ := runMB("add", "head", "--title", "C")
	cid := strings.TrimSpace(out)

	// Complete C (the head todo).
	runMB("claim")
	runMB("done")

	// Default list shows only todos, head first: A then B.
	// --oneline keeps the `— <title>` substring the legacy assertion relies on.
	list, _, _ := runMB("list", "--oneline")
	if !strings.Contains(list, "1. ") || !idxBefore(list, "— A", "— B") {
		t.Fatalf("expected A before B in todo list, got:\n%s", list)
	}
	if strings.Contains(list, cid) {
		t.Fatalf("completed lap %s should not appear in default list:\n%s", cid, list)
	}

	// New head lands at the top of the todo section, still below the done lap.
	runMB("add", "head", "--title", "D")
	list2, _, _ := runMB("list", "--oneline")
	if !idxBefore(list2, "— D", "— A") {
		t.Fatalf("expected new head D before A, got:\n%s", list2)
	}

	// The done lap is the most recent completion.
	done, _, _ := runMB("list", "--done")
	if !strings.Contains(done, cid) {
		t.Fatalf("expected completed lap %s in --done, got:\n%s", cid, done)
	}
}

func TestAddAfterDoneWarnsAndBecomesHead(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	out, _, _ := runMB("add", "head", "--title", "C")
	cid := strings.TrimSpace(out)
	runMB("claim")
	runMB("done") // completes C

	_, errStr, code := runMB("add", "after", cid, "--title", "D")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(errStr, "already complete") || !strings.Contains(errStr, "head") {
		t.Fatalf("expected fallback-to-head warning, got stderr: %q", errStr)
	}
	list, _, _ := runMB("list", "--oneline")
	if !idxBefore(list, "— D", "— A") {
		t.Fatalf("expected D to be the new head, got:\n%s", list)
	}
}

func TestAutoMigrateV1File(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	v1 := `{"version":1,"tasks":[` +
		`{"id":"old-1","title":"First","description":"","isDone":false,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","completedAt":null},` +
		`{"id":"old-2","title":"Second","description":"","isDone":false,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","completedAt":null}` +
		`]}`
	path := filepath.Join(beadsDir, "laps.json")
	if err := os.WriteFile(path, []byte(v1), 0644); err != nil {
		t.Fatal(err)
	}

	// Any command triggers auto-migration on load.
	list, _, _ := runMB("list")
	if !idxBefore(list, "First", "Second") {
		t.Fatalf("expected migration to preserve todo order, got:\n%s", list)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, `"version": 3`) {
		t.Fatalf("expected version 3 after migration, got:\n%s", got)
	}
	if !strings.Contains(got, `"order"`) {
		t.Fatalf("expected order keys after migration, got:\n%s", got)
	}
	if !strings.Contains(got, `"kind": "lap"`) {
		t.Fatalf("expected kind after migration, got:\n%s", got)
	}
}

// idxBefore reports whether substring a appears before substring b in s.
func idxBefore(s, a, b string) bool {
	ia := strings.Index(s, a)
	ib := strings.Index(s, b)
	return ia >= 0 && ib >= 0 && ia < ib
}

// fileContainsID reports whether the given file holds a task with id.
func fileContainsID(f *store.File, id string) bool {
	for i := range f.Tasks {
		if f.Tasks[i].ID == id {
			return true
		}
	}
	return false
}

// taskByIDInFile returns the task with id from an already loaded file, failing
// the test if it is absent.
func taskByIDInFile(t *testing.T, f *store.File, id string) *store.Task {
	t.Helper()
	for i := range f.Tasks {
		if f.Tasks[i].ID == id {
			return &f.Tasks[i]
		}
	}
	t.Fatalf("task %s not found in file: %#v", id, f.Tasks)
	return nil
}

// rootTodoIDs returns unfinished task ids in normalized root queue order.
func rootTodoIDs(f *store.File) []string {
	store.Normalize(f)
	var ids []string
	for i := range f.Tasks {
		if !f.Tasks[i].IsDone {
			ids = append(ids, f.Tasks[i].ID)
		}
	}
	return ids
}

// taskByID loads the laps store under beadsDir and returns the task with the
// given id, failing the test if it is absent.
func taskByID(t *testing.T, beadsDir, id string) *store.Task {
	t.Helper()
	data, err := store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	for i := range data.Tasks {
		if data.Tasks[i].ID == id {
			return &data.Tasks[i]
		}
	}
	t.Fatalf("task %s not found in store", id)
	return nil
}

func TestJSONOutputAdd(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("add", "head", "--title", "JSON task", "--description", "desc", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	task, ok := result["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object in JSON, got: %s", out)
	}
	if task["title"] != "JSON task" {
		t.Fatalf("expected title 'JSON task', got: %v", task["title"])
	}
	if task["description"] != "desc" {
		t.Fatalf("expected description 'desc', got: %v", task["description"])
	}
}

func TestJSONOutputAddArray(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("add", "tail", "--json", `[{"title":"A"},{"title":"B"}]`, "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if len(result.Tasks) != 2 || result.Tasks[0]["title"] != "A" || result.Tasks[1]["title"] != "B" {
		t.Fatalf("expected ordered tasks array, got: %s", out)
	}
}

func TestJSONOutputGet(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Get me", "--description", "details")
	out, errStr, code := runMB("get", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	task, ok := result["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object in JSON, got: %s", out)
	}
	if task["title"] != "Get me" {
		t.Fatalf("expected title 'Get me', got: %v", task["title"])
	}
}

func TestJSONOutputList(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	runMB("add", "tail", "--title", "B")
	out, errStr, code := runMB("list", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	tasks, ok := result["tasks"].([]interface{})
	if !ok {
		t.Fatalf("expected tasks array in JSON, got: %s", out)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got: %d", len(tasks))
	}
}

func TestJSONOutputListEmpty(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("list", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	tasks, ok := result["tasks"].([]interface{})
	if !ok {
		t.Fatalf("expected tasks array in JSON, got: %s", out)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got: %d", len(tasks))
	}
}

func TestJSONOutputDone(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Do me")
	runMB("claim")
	out, errStr, code := runMB("done", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	task, ok := result["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object in JSON, got: %s", out)
	}
	if task["isDone"] != true {
		t.Fatalf("expected isDone true, got: %v", task["isDone"])
	}
}

func TestJSONOutputDelete(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Delete me", "--json-output")
	var addResult map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &addResult); err != nil {
		t.Fatalf("expected valid JSON from add, got: %s", out)
	}
	id := addResult["task"].(map[string]interface{})["id"].(string)

	out, errStr, code := runMB("delete", id, "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["deleted"] != id {
		t.Fatalf("expected deleted id %s, got: %v", id, result["deleted"])
	}
}

func TestJSONOutputCount(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A", "--assignee", "alice")
	runMB("add", "tail", "--title", "B", "--assignee", "alice")
	runMB("claim")
	runMB("done")

	out, errStr, code := runMB("count", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["done"].(float64) != 1 {
		t.Fatalf("expected done=1, got: %v", result["done"])
	}
	if result["total"].(float64) != 2 {
		t.Fatalf("expected total=2, got: %v", result["total"])
	}
	breakdown, ok := result["breakdown"].([]interface{})
	if !ok || len(breakdown) != 1 {
		t.Fatalf("expected 1 breakdown entry, got: %v", result["breakdown"])
	}
}

func TestJSONOutputPrune(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	runMB("claim")
	runMB("done")

	out, errStr, code := runMB("prune", "0", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["removed"].(float64) != 1 {
		t.Fatalf("expected removed=1, got: %v", result["removed"])
	}
}

func TestJSONOutputVersion(t *testing.T) {
	version = "0.6.0"
	out, _, code := runMB("version", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["version"] != "0.6.0" {
		t.Fatalf("expected version '0.6.0', got: %v", result["version"])
	}
}

func TestJSONOutputError(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("get", "--json-output")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(errStr)), &result); err != nil {
		t.Fatalf("expected valid JSON on stderr, got: %s", errStr)
	}
	if result["error"] != "no head task" {
		t.Fatalf("expected error 'no head task', got: %v", result["error"])
	}
	if result["exitCode"].(float64) != 3 {
		t.Fatalf("expected exitCode 3, got: %v", result["exitCode"])
	}
}

func TestJSONOutputUpdateDev(t *testing.T) {
	version = "dev"
	out, errStr, code := runMB("update", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["currentVersion"] != "dev" {
		t.Fatalf("expected currentVersion 'dev', got: %v", result["currentVersion"])
	}
}

func TestJSONOutputUpdateUpToDate(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	oldFetch := fetchLatestVersionFunc
	defer func() { fetchLatestVersionFunc = oldFetch }()
	fetchLatestVersionFunc = func() (string, error) { return "0.5.0", nil }

	version = "0.5.0"
	out, errStr, code := runMB("update", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["upToDate"] != true {
		t.Fatalf("expected upToDate true, got: %v", result["upToDate"])
	}
}

func TestJSONOutputUpdateAvailable(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	oldFetch := fetchLatestVersionFunc
	defer func() { fetchLatestVersionFunc = oldFetch }()
	fetchLatestVersionFunc = func() (string, error) { return "0.6.0", nil }

	version = "0.5.0"
	out, errStr, code := runMB("update", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["upToDate"] != false {
		t.Fatalf("expected upToDate false, got: %v", result["upToDate"])
	}
	if result["updated"] != false {
		t.Fatalf("expected updated false without --yes, got: %v", result["updated"])
	}
}

func TestJSONOutputOnOff(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("on", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0 for on, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON from on, got: %s", out)
	}
	if result["status"] != "enabled" {
		t.Fatalf("expected status 'enabled', got: %v", result["status"])
	}

	out, errStr, code = runMB("off", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0 for off, got %d, stderr: %s", code, errStr)
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON from off, got: %s", out)
	}
	if result["status"] != "disabled" {
		t.Fatalf("expected status 'disabled', got: %v", result["status"])
	}
}

func TestIsKnownCommand(t *testing.T) {
	known := []string{
		"add", "count", "get", "list", "ls", "move", "edit", "assign", "done",
		"delete", "prune", "on", "off", "update", "version", "help", "--version",
		"claim", "init", "log", "status",
	}
	for _, name := range known {
		if !isKnownCommand(name) {
			t.Fatalf("expected isKnownCommand(%q) = true", name)
		}
	}
	// log and status are the newly-registered built-ins; assert them explicitly
	// since they back the new reader/status commands and must not fall through to
	// the hook-only intercept path.
	if !isKnownCommand("log") {
		t.Fatal("expected isKnownCommand(\"log\") = true")
	}
	if !isKnownCommand("status") {
		t.Fatal("expected isKnownCommand(\"status\") = true")
	}
	if isKnownCommand("not-a-real-command") {
		t.Fatal("expected unknown command to return false")
	}
}

func TestIsJSONOutput(t *testing.T) {
	if !isJSONOutput([]string{"list", "--json-output"}) {
		t.Fatal("expected true for --json-output")
	}
	if isJSONOutput([]string{"list"}) {
		t.Fatal("expected false without --json-output")
	}
	if !isJSONOutput([]string{"--json-output", "list"}) {
		t.Fatal("expected true for flag before command")
	}
	if !isJSONOutput([]string{"--json-output=true"}) {
		t.Fatal("expected true for --json-output=true")
	}
	if !isJSONOutput([]string{"worktree", "--json-output=true", "feature"}) {
		t.Fatal("expected true for --json-output=true after command")
	}
}

func TestJSONOutputGetByID(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "By ID")
	id := strings.TrimSpace(out)

	out, errStr, code := runMB("get", id, "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	task, ok := result["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object in JSON, got: %s", out)
	}
	if task["id"] != id {
		t.Fatalf("expected id %s, got: %v", id, task["id"])
	}
}

func TestJSONOutputListDone(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	runMB("add", "tail", "--title", "B")
	runMB("claim")
	runMB("done")

	out, errStr, code := runMB("list", "--done", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	tasks, ok := result["tasks"].([]interface{})
	if !ok {
		t.Fatalf("expected tasks array in JSON, got: %s", out)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 done task, got: %d", len(tasks))
	}
	task := tasks[0].(map[string]interface{})
	if task["isDone"] != true {
		t.Fatalf("expected isDone true, got: %v", task["isDone"])
	}
}

func TestJSONOutputListAll(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "Todo")
	runMB("add", "tail", "--title", "Done")
	runMB("claim")
	runMB("done")

	out, errStr, code := runMB("list", "--all", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	tasks, ok := result["tasks"].([]interface{})
	if !ok || len(tasks) != 2 {
		t.Fatalf("expected 2 tasks (todo+done), got: %v", tasks)
	}
}

func TestJSONOutputCountEmpty(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("count", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["done"].(float64) != 0 {
		t.Fatalf("expected done=0, got: %v", result["done"])
	}
	if result["total"].(float64) != 0 {
		t.Fatalf("expected total=0, got: %v", result["total"])
	}
	breakdown, ok := result["breakdown"].([]interface{})
	if !ok || len(breakdown) != 0 {
		t.Fatalf("expected empty breakdown, got: %v", result["breakdown"])
	}
}

func TestJSONOutputPruneDefault(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	for i := 0; i < 25; i++ {
		runMB("add", "tail", "--title", fmt.Sprintf("Task %d", i))
	}
	for i := 0; i < 25; i++ {
		runMB("claim")
		runMB("done")
	}

	out, errStr, code := runMB("prune", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["removed"].(float64) != 5 {
		t.Fatalf("expected removed=5, got: %v", result["removed"])
	}
}

func TestJSONOutputDoneNoHeadNoClaim(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("done", "--json-output")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(errStr)), &result); err != nil {
		t.Fatalf("expected valid JSON on stderr, got: %s", errStr)
	}
	if !strings.Contains(result["error"].(string), "no claimed lap") {
		t.Fatalf("expected error containing 'no claimed lap', got: %v", result["error"])
	}
}

func TestJSONOutputDeleteNotFound(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("delete", "nonexistent", "--json-output")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(errStr)), &result); err != nil {
		t.Fatalf("expected valid JSON on stderr, got: %s", errStr)
	}
	if result["error"] != "task not found" {
		t.Fatalf("expected error 'task not found', got: %v", result["error"])
	}
}

func TestJSONOutputUpdateYes(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	oldFetch := fetchLatestVersionFunc
	oldInstall := installLatestVersionFn
	defer func() {
		fetchLatestVersionFunc = oldFetch
		installLatestVersionFn = oldInstall
	}()

	fetchLatestVersionFunc = func() (string, error) { return "0.7.0", nil }
	installed := false
	installLatestVersionFn = func() error {
		installed = true
		return nil
	}

	version = "0.6.0"
	out, errStr, code := runMB("update", "--yes", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if !installed {
		t.Fatal("expected install to run")
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["upToDate"] != false {
		t.Fatalf("expected upToDate false, got: %v", result["upToDate"])
	}
	if result["updated"] != true {
		t.Fatalf("expected updated true, got: %v", result["updated"])
	}
	if result["currentVersion"] != "0.6.0" {
		t.Fatalf("expected currentVersion 0.6.0, got: %v", result["currentVersion"])
	}
	if result["latestVersion"] != "0.7.0" {
		t.Fatalf("expected latestVersion 0.7.0, got: %v", result["latestVersion"])
	}
}

func TestJSONOutputHookPassbackSuppression(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "A")
	runMB("claim")
	hooks := `{"version":1,"hooks":[{"title":"pass","command":"done","when":"after","run":"echo passback","passback":true}]}`
	os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644)

	out, errStr, code := runMB("done", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	// Output must be valid JSON and must not contain raw hook passback text.
	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "{") {
		t.Fatalf("expected JSON output, got: %s", out)
	}
	if strings.Contains(trimmed, "passback") {
		t.Fatalf("hook passback should be suppressed in JSON mode, got: %s", out)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if _, ok := result["task"]; !ok {
		t.Fatalf("expected task key in JSON, got: %s", out)
	}
}

func TestJSONOutputVersionFlag(t *testing.T) {
	out, errStr, err := runMBExecute("--json-output", "--version")
	if err != nil {
		t.Fatalf("expected nil error, got %v, stderr: %s", err, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON for --version --json-output, got: %s", out)
	}
	if result["version"] != "test" {
		t.Fatalf("expected version 'test', got: %v", result["version"])
	}

	// Verify --version --json-output (inverted order) also works
	out, errStr, err = runMBExecute("--version", "--json-output")
	if err != nil {
		t.Fatalf("expected nil error, got %v, stderr: %s", err, errStr)
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON for --version --json-output (inverted), got: %s", out)
	}
	if result["version"] != "test" {
		t.Fatalf("expected version 'test', got: %v", result["version"])
	}
}

// --- Claim tests ---

func TestClaimHead(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "First task", "--description", "Desc")
	runMB("add", "tail", "--title", "Second task")

	out, _, code := runMB("claim")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "First task") {
		t.Fatalf("expected title 'First task' in output, got: %s", out)
	}
	if !strings.Contains(out, "Desc") {
		t.Fatalf("expected description 'Desc' in output, got: %s", out)
	}
	if !strings.Contains(out, "laps claim undo") {
		t.Fatalf("expected undo hint in output, got: %s", out)
	}

	data, err := os.ReadFile(filepath.Join(beadsDir, "claim"))
	if err != nil {
		t.Fatalf("expected claim file: %v", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		t.Fatal("expected non-empty claim file")
	}
}

func TestClaimByID(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Task A")
	runMB("add", "tail", "--title", "Task B")
	idA := strings.TrimSpace(out)

	out, _, code := runMB("claim", idA)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "Task A") {
		t.Fatalf("expected title 'Task A' in output, got: %s", out)
	}
}

func TestClaimNoHead(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("claim")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "no head task") {
		t.Fatalf("expected 'no head task', got: %s", errStr)
	}
}

func TestClaimNotFound(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("claim", "nonexistent-id")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "task not found") {
		t.Fatalf("expected 'task not found', got: %s", errStr)
	}
}

func TestClaimMalformedStructuredClaimErrors(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Task")
	claimPath := filepath.Join(beadsDir, "claim")
	if err := os.WriteFile(claimPath, []byte("{"), 0644); err != nil {
		t.Fatalf("write malformed claim: %v", err)
	}

	_, errStr, code := runMB("claim")
	if code != 2 {
		t.Fatalf("expected code 2, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(errStr, "malformed claim") {
		t.Fatalf("expected malformed claim error, got: %s", errStr)
	}
	data, err := os.ReadFile(claimPath)
	if err != nil {
		t.Fatalf("read claim: %v", err)
	}
	if string(data) != "{" {
		t.Fatalf("expected malformed claim to remain untouched, got: %q", data)
	}
}

func TestClaimUndo(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Task to claim")
	runMB("claim")

	out, _, code := runMB("claim", "undo")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "Claim cleared for") {
		t.Fatalf("expected 'Claim cleared for' in output, got: %s", out)
	}
	if !strings.Contains(out, "Task to claim") {
		t.Fatalf("expected title in output, got: %s", out)
	}
}

func TestClaimUndoNothingClaimed(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("claim", "undo")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "no claimed lap to clear") {
		t.Fatalf("expected 'no claimed lap to clear', got: %s", errStr)
	}
}

// --- Done tests ---

func TestDoneBareNoClaimHasHead(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Head task")

	_, errStr, code := runMB("done")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "no claimed lap") {
		t.Fatalf("expected 'no claimed lap' in error, got: %s", errStr)
	}
	if !strings.Contains(errStr, "Head task") {
		t.Fatalf("expected head task title in error, got: %s", errStr)
	}
}

func TestDoneExplicitID(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Do me", "--description", "desc")
	id := strings.TrimSpace(out)

	out, _, code := runMB("done", id)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "Do me") {
		t.Fatalf("expected title 'Do me' in output, got: %s", out)
	}
	if !strings.Contains(out, "laps done undo") {
		t.Fatalf("expected undo hint, got: %s", out)
	}
}

func TestDoneExplicitIDAlreadyDone(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Do me")
	id := strings.TrimSpace(out)
	runMB("claim")
	runMB("done")

	_, errStr, code := runMB("done", id)
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "already done") {
		t.Fatalf("expected 'already done' in error, got: %s", errStr)
	}
}

func TestDoneExplicitIDNotFound(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("done", "nonexistent-id")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "task not found") {
		t.Fatalf("expected 'task not found', got: %s", errStr)
	}
}

func TestDoneClaimedNotFound(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Task")
	runMB("claim")
	// Remove the claimed task from the store
	data, _ := store.Load(filepath.Join(beadsDir, "laps.json"))
	data.Tasks = nil
	store.Save(filepath.Join(beadsDir, "laps.json"), data)

	_, errStr, code := runMB("done")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "not found") {
		t.Fatalf("expected 'not found' in error, got: %s", errStr)
	}
}

func TestDoneClaimedAlreadyDone(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Task")
	id := strings.TrimSpace(out)
	runMB("claim")
	// Mark done externally
	data, _ := store.Load(filepath.Join(beadsDir, "laps.json"))
	for i := range data.Tasks {
		if data.Tasks[i].ID == id {
			now := time.Now().UTC()
			data.Tasks[i].IsDone = true
			data.Tasks[i].CompletedAt = &now
		}
	}
	store.Save(filepath.Join(beadsDir, "laps.json"), data)

	_, errStr, code := runMB("done")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "already done") {
		t.Fatalf("expected 'already done' in error, got: %s", errStr)
	}
}

func TestDoneExplicitIDWithClaim(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "tail", "--title", "Task A")
	idA := strings.TrimSpace(out)
	runMB("add", "head", "--title", "Task B")
	runMB("claim") // claims Task B

	// Complete Task A explicitly (different from claimed task)
	out, _, code := runMB("done", idA)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "Task A") {
		t.Fatalf("expected 'Task A' title, got: %s", out)
	}

	// Claim should still exist (we completed a different task)
	claim, _ := store.ReadClaim(beadsDir, store.ResolveFile(""))
	if claim.Lap == "" {
		t.Fatal("expected claim to persist after completing different task")
	}
}

func TestDoneExplicitIDMatchesClaim(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Task X")
	out, _, _ := runMB("add", "tail", "--title", "Task Y")
	idY := strings.TrimSpace(out)
	runMB("claim", idY) // claim Task Y

	// Complete Task Y (matches claim)
	out, _, code := runMB("done", idY)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "Task Y") {
		t.Fatalf("expected 'Task Y' title, got: %s", out)
	}

	// Claim should be cleared
	claim, _ := store.ReadClaim(beadsDir, store.ResolveFile(""))
	if claim.Lap != "" {
		t.Fatal("expected claim cleared when completing matching task")
	}
}

func TestDoneExplicitIDDoesNotClearWrongFileClaim(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Default file task")
	id := strings.TrimSpace(out)
	if err := store.WriteClaim(beadsDir, store.Claim{Lap: id, File: "auth.json"}); err != nil {
		t.Fatalf("write wrong-file claim: %v", err)
	}

	_, _, code := runMB("done", id)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	claim, err := store.ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("read claim: %v", err)
	}
	if claim.Lap != id || claim.File != "auth.json" {
		t.Fatalf("expected wrong-file claim to remain, got %+v", claim)
	}
}

func TestDoneUndoRecent(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Recent task")
	runMB("claim")
	runMB("done")

	out, _, code := runMB("done", "undo")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "Done state cleared for Recent task") {
		t.Fatalf("expected 'Done state cleared for Recent task' in output, got: %s", out)
	}

	// Verify task is no longer done
	data, _ := store.Load(filepath.Join(beadsDir, "laps.json"))
	for _, task := range data.Tasks {
		if task.Title == "Recent task" && task.IsDone {
			t.Fatal("expected task to be undone")
		}
	}
}

func TestDoneUndoOld(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Old task")
	id := strings.TrimSpace(out)
	runMB("claim")
	runMB("done")

	// Manually set completedAt to 6 minutes ago
	path := filepath.Join(beadsDir, "laps.json")
	data, _ := store.Load(path)
	for i := range data.Tasks {
		if data.Tasks[i].ID == id {
			old := time.Now().UTC().Add(-6 * time.Minute)
			data.Tasks[i].CompletedAt = &old
		}
	}
	store.Save(path, data)

	_, errStr, code := runMB("done", "undo")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "was completed") {
		t.Fatalf("expected age warning, got: %s", errStr)
	}
	if !strings.Contains(errStr, "Old task") {
		t.Fatalf("expected task title in error, got: %s", errStr)
	}
	if !strings.Contains(errStr, "undo -y") {
		t.Fatalf("expected -y hint in error, got: %s", errStr)
	}
}

func TestDoneUndoForce(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Old task")
	id := strings.TrimSpace(out)
	runMB("claim")
	runMB("done")

	path := filepath.Join(beadsDir, "laps.json")
	data, _ := store.Load(path)
	for i := range data.Tasks {
		if data.Tasks[i].ID == id {
			old := time.Now().UTC().Add(-6 * time.Minute)
			data.Tasks[i].CompletedAt = &old
		}
	}
	store.Save(path, data)

	out, _, code := runMB("done", "undo", "-y")
	if code != 0 {
		t.Fatalf("expected code 0 with -y, got %d", code)
	}
	if !strings.Contains(out, "Done state cleared for Old task") {
		t.Fatalf("expected 'Done state cleared for Old task' in output, got: %s", out)
	}
}

func TestDoneUndoNoCompleted(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("done", "undo")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "no completed task to undo") {
		t.Fatalf("expected 'no completed task to undo', got: %s", errStr)
	}
}

// --- JSON output tests ---

func TestJSONOutputClaim(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "JSON claim task")
	out, errStr, code := runMB("claim", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if _, ok := result["task"]; !ok {
		t.Fatalf("expected task key in JSON, got: %s", out)
	}
	if result["claimedId"] == "" {
		t.Fatalf("expected claimedId in JSON, got: %s", out)
	}
}

func TestJSONOutputClaimUndo(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Undo JSON")
	runMB("claim")
	out, errStr, code := runMB("claim", "undo", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["unclaimedId"] == "" {
		t.Fatalf("expected unclaimedId in JSON, got: %s", out)
	}
	if result["title"] != "Undo JSON" {
		t.Fatalf("expected title 'Undo JSON', got: %v", result["title"])
	}
}

func TestJSONOutputDoneUndo(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Undo JSON done")
	runMB("claim")
	runMB("done")
	out, errStr, code := runMB("done", "undo", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	task, ok := result["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object in JSON, got: %s", out)
	}
	if task["isDone"] != false {
		t.Fatalf("expected isDone false, got: %v", task["isDone"])
	}
}

func TestJSONOutputClaimBareError(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("claim", "--json-output")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(errStr)), &result); err != nil {
		t.Fatalf("expected valid JSON on stderr, got: %s", errStr)
	}
	if result["error"] != "no head task" {
		t.Fatalf("expected error 'no head task', got: %v", result["error"])
	}
}

// --- Init tests ---

func TestInit(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	os.WriteFile(".gitignore", []byte("/bin/\n"), 0644)

	// Remove .laps/laps.json that setupTempRepo created (it only creates dir)
	// Actually setupTempRepo just creates the dir, so laps.json doesn't exist yet.

	out, _, code := runMB("init")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, output: %s", code, out)
	}
	if !strings.Contains(out, "Created .laps/laps.json") {
		t.Fatalf("expected created message, got: %s", out)
	}
	// Both target entries are missing, so the message must name both.
	if !strings.Contains(out, "Added .laps/claim and .laps/log.jsonl to .gitignore") {
		t.Fatalf("expected both-entries gitignore message, got: %s", out)
	}

	if _, err := os.Stat(filepath.Join(beadsDir, "laps.json")); err != nil {
		t.Fatalf("expected laps.json to exist: %v", err)
	}

	gitignoreData, _ := os.ReadFile(".gitignore")
	if !strings.Contains(string(gitignoreData), ".laps/claim") {
		t.Fatal("expected .laps/claim in .gitignore")
	}
	if !strings.Contains(string(gitignoreData), ".laps/log.jsonl") {
		t.Fatal("expected .laps/log.jsonl in .gitignore")
	}
	// Pre-existing arbitrary content must survive verbatim.
	if !strings.Contains(string(gitignoreData), "/bin/") {
		t.Fatal("expected pre-existing /bin/ line preserved in .gitignore")
	}

	// Run again - should say Already initialized
	out, _, code = runMB("init")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "Already initialized") {
		t.Fatalf("expected 'Already initialized', got: %s", out)
	}
}

func TestInitGitignoreAlreadyExists(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	// Regression: the old scan broke at the first .laps/claim match, so it never
	// noticed .laps/log.jsonl was missing. With the fix, init scans the complete
	// file and appends only the missing entry while preserving the rest verbatim.
	orig := []byte("/bin/\n.laps/claim\n# keep me\n")
	os.WriteFile(".gitignore", orig, 0644)

	out, _, code := runMB("init")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "Created .laps/laps.json") {
		t.Fatalf("expected created message, got: %s", out)
	}
	// Only the log entry was missing, so the message names just it.
	if !strings.Contains(out, "Added .laps/log.jsonl to .gitignore") {
		t.Fatalf("expected log-only gitignore message, got: %s", out)
	}
	if strings.Contains(out, ".laps/claim and") {
		t.Fatalf("should not claim to add .laps/claim (already present), got: %s", out)
	}

	data, _ := os.ReadFile(".gitignore")
	got := string(data)
	// .laps/log.jsonl must now be appended...
	if !strings.Contains(got, ".laps/log.jsonl") {
		t.Fatalf("expected .laps/log.jsonl appended to .gitignore, got: %s", got)
	}
	// ...and every pre-existing line preserved verbatim and in order.
	for _, line := range []string{"/bin/", ".laps/claim", "# keep me"} {
		if !strings.Contains(got, line) {
			t.Fatalf("expected pre-existing line %q preserved, got: %s", line, got)
		}
	}
	if c := strings.Count(got, ".laps/claim"); c != 1 {
		t.Fatalf("expected .laps/claim exactly once, got %d: %s", c, got)
	}
}

func TestInitGitignoreBothPresentNoChange(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	orig := []byte("/bin/\n.laps/claim\n.laps/log.jsonl\n")
	os.WriteFile(".gitignore", orig, 0644)

	out, _, code := runMB("init")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.Contains(out, "Added") {
		t.Fatalf("should not modify .gitignore when both entries present, got: %s", out)
	}

	// File must be byte-for-byte unchanged.
	data, _ := os.ReadFile(".gitignore")
	if string(data) != string(orig) {
		t.Fatalf("expected .gitignore unchanged, got: %s", string(data))
	}
}

func TestInitPreservesArbitraryGitignoreContent(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	orig := "/build/\n*.tmp\n\n# trailing comment\n"
	os.WriteFile(".gitignore", []byte(orig), 0644)

	if _, _, code := runMB("init"); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}

	data, _ := os.ReadFile(".gitignore")
	got := string(data)
	// Original content (including the blank line and comment) is preserved, and
	// the two laps entries are appended at the end in their canonical order.
	if !strings.HasPrefix(got, "/build/\n*.tmp\n\n# trailing comment\n") {
		t.Fatalf("expected original .gitignore content preserved verbatim at top, got: %s", got)
	}
	if !strings.HasSuffix(got, ".laps/claim\n.laps/log.jsonl\n") {
		t.Fatalf("expected both laps entries appended in order at end, got: %s", got)
	}
}

// --- Move tests ---

func TestMovePositions(t *testing.T) {
	tests := []struct {
		name   string
		move   []string // args after "move"
		before func(t *testing.T, list string)
	}{
		{
			name: "head",
			move: []string{"head"},
			before: func(t *testing.T, list string) {
				if !idxBefore(list, "— C", "— A") || !idxBefore(list, "— C", "— B") {
					t.Fatalf("expected C at head, got:\n%s", list)
				}
			},
		},
		{
			name: "tail",
			move: []string{"tail"},
			before: func(t *testing.T, list string) {
				if !idxBefore(list, "— B", "— A") || !idxBefore(list, "— C", "— A") {
					t.Fatalf("expected A at tail, got:\n%s", list)
				}
			},
		},
		{
			name: "after",
			move: []string{"after"},
			before: func(t *testing.T, list string) {
				if !idxBefore(list, "— A", "— C") || !idxBefore(list, "— C", "— B") {
					t.Fatalf("expected C right after A, got:\n%s", list)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupTempRepo(t)
			defer cleanup()

			outA, _, _ := runMB("add", "tail", "--title", "A")
			outB, _, _ := runMB("add", "tail", "--title", "B")
			outC, _, _ := runMB("add", "tail", "--title", "C")
			idA := strings.TrimSpace(outA)
			idB := strings.TrimSpace(outB)
			idC := strings.TrimSpace(outC)

			args := []string{"move"}
			moveID := idC
			if tt.name == "tail" {
				moveID = idA
			}
			args = append(args, moveID)
			args = append(args, tt.move...)
			if tt.name == "after" {
				args = append(args, idA) // move C after A
			}

			out, errStr, code := runMB(args...)
			if code != 0 {
				t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
			}
			if got := strings.TrimSpace(out); got != moveID {
				t.Fatalf("expected move to echo moved id %s, got %q", moveID, got)
			}

			list, _, _ := runMB("list", "--oneline")
			tt.before(t, list)
			// No id is lost in the move: A, B, C all remain.
			for _, id := range []string{idA, idB, idC} {
				if !strings.Contains(list, id) {
					t.Fatalf("expected %s to survive move, got:\n%s", id, list)
				}
			}
		})
	}
}

func TestMovePreservesID(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Original")
	id := strings.TrimSpace(out)

	// move prints the moved task id; it must be byte-identical to the original.
	moveOut, errStr, code := runMB("move", id, "tail")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if moveOut != id+"\n" {
		t.Fatalf("expected move to preserve id %q, got %q", id, moveOut)
	}

	// Re-getting by the original id still resolves the same lap.
	getOut, _, code := runMB("get", id, "--json-output")
	if code != 0 {
		t.Fatalf("expected get %s to succeed after move, code %d", id, code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(getOut)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", getOut)
	}
	task, ok := result["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object, got: %s", getOut)
	}
	if task["id"] != id {
		t.Fatalf("expected preserved id %s, got %v", id, task["id"])
	}
}

func TestMoveUnknownIDFails(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("move", "does-not-exist", "head")
	if code != 1 {
		t.Fatalf("expected code 1 for unknown moved id, got %d", code)
	}
	if !strings.Contains(errStr, "not found") {
		t.Fatalf("expected not-found error, got: %s", errStr)
	}
}

func TestMoveDoneIDFails(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Done one")
	id := strings.TrimSpace(out)
	runMB("claim")
	if _, _, code := runMB("done"); code != 0 {
		t.Fatalf("setup done failed, code %d", code)
	}

	_, errStr, code := runMB("move", id, "head")
	if code != 1 {
		t.Fatalf("expected code 1 for done moved id, got %d", code)
	}
	if !strings.Contains(errStr, "already done") {
		t.Fatalf("expected already-done error, got: %s", errStr)
	}
}

func TestMoveUsageFails(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "no position", args: []string{"someid"}, want: "usage"},
		{name: "invalid position", args: []string{"someid", "sideways"}, want: "position must be head"},
		{name: "after without target", args: []string{"someid", "after"}, want: "after requires a target"},
		{name: "head with extra arg", args: []string{"someid", "head", "extra"}, want: "usage"},
		{name: "after with extra arg", args: []string{"someid", "after", "target", "extra"}, want: "exactly one target"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupTempRepo(t)
			defer cleanup()

			_, errStr, code := runMB(append([]string{"move"}, tt.args...)...)
			if code != 1 {
				t.Fatalf("expected code 1, got %d", code)
			}
			if !strings.Contains(errStr, tt.want) {
				t.Fatalf("expected error containing %q, got: %s", tt.want, errStr)
			}
		})
	}
}

func TestMoveAfterSelfReferenceFails(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Self")
	id := strings.TrimSpace(out)

	_, errStr, code := runMB("move", id, "after", id)
	if code != 1 {
		t.Fatalf("expected code 1 for self-reference, got %d", code)
	}
	if !strings.Contains(errStr, "cannot move a lap after itself") {
		t.Fatalf("expected self-reference error, got: %s", errStr)
	}
}

func TestMoveAfterMissingTargetFails(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Mover")
	id := strings.TrimSpace(out)

	_, errStr, code := runMB("move", id, "after", "ghost-id")
	if code != 3 {
		t.Fatalf("expected code 3 for missing after target, got %d", code)
	}
	if !strings.Contains(errStr, "not found") {
		t.Fatalf("expected not-found error, got: %s", errStr)
	}
}

func TestMoveAfterDoneFallsBackToHead(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	out, _, _ := runMB("add", "head", "--title", "Done target")
	doneID := strings.TrimSpace(out)
	runMB("claim")
	if _, _, code := runMB("done"); code != 0 {
		t.Fatalf("setup done failed, code %d", code)
	}

	out, _, _ = runMB("add", "tail", "--title", "Mover")
	moveID := strings.TrimSpace(out)

	_, errStr, code := runMB("move", moveID, "after", doneID)
	if code != 0 {
		t.Fatalf("expected code 0 for after-done fallback, got %d, stderr: %s", code, errStr)
	}
	// The after-done fallback notice is emitted on stderr by move.go itself.
	if !strings.Contains(errStr, "already complete") || !strings.Contains(errStr, "head") {
		t.Fatalf("expected fallback-to-head stderr notice, got: %q", errStr)
	}

	list, _, _ := runMB("list", "--oneline")
	// Mover lands at the head of the todo queue, ahead of A.
	if !idxBefore(list, "— Mover", "— A") {
		t.Fatalf("expected mover at head after fallback, got:\n%s", list)
	}
}

func TestMoveJSONOutput(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	out, _, _ := runMB("add", "tail", "--title", "Mover")
	id := strings.TrimSpace(out)

	out, errStr, code := runMB("move", id, "head", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	task, ok := result["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object in JSON, got: %s", out)
	}
	// The JSON payload carries the preserved id.
	if task["id"] != id {
		t.Fatalf("expected preserved id %s, got %v", id, task["id"])
	}
	if task["title"] != "Mover" {
		t.Fatalf("expected title 'Mover', got %v", task["title"])
	}
}

func TestMoveAdvancesUpdatedAt(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Mover", "--description", "desc")
	id := strings.TrimSpace(out)
	runMB("add", "tail", "--title", "Other")

	// Seed a known-old UpdatedAt so the advance is deterministic regardless of
	// clock resolution between the two time.Now() calls.
	path := filepath.Join(beadsDir, "laps.json")
	data, _ := store.Load(path)
	old := time.Now().UTC().Add(-6 * time.Minute)
	for i := range data.Tasks {
		if data.Tasks[i].ID == id {
			data.Tasks[i].UpdatedAt = old
		}
	}
	if err := store.Save(path, data); err != nil {
		t.Fatal(err)
	}

	if _, _, code := runMB("move", id, "tail"); code != 0 {
		t.Fatalf("expected move code 0")
	}

	after, _ := store.Load(path)
	for _, task := range after.Tasks {
		if task.ID == id {
			if !task.UpdatedAt.After(old) {
				t.Fatalf("expected updatedAt to advance past %v, got %v", old, task.UpdatedAt)
			}
			return
		}
	}
	t.Fatalf("moved task %s not found after move", id)
}

func TestMoveDispatchThroughExecute(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "A")
	out, _, _ := runMB("add", "tail", "--title", "Mover")
	id := strings.TrimSpace(out)

	// move is a registered built-in (isKnownCommand), so Execute must dispatch
	// to moveCmd rather than the hook-only intercept path.
	execOut, errStr, err := runMBExecute("move", id, "head")
	if err != nil {
		t.Fatalf("expected nil error dispatching move, got %v, stderr: %s", err, errStr)
	}
	if got := strings.TrimSpace(execOut); got != id {
		t.Fatalf("expected dispatched move to echo id %s, got %q", id, got)
	}

	list, _, _ := runMB("list", "--oneline")
	if !idxBefore(list, "— Mover", "— A") {
		t.Fatalf("expected mover at head after Execute dispatch, got:\n%s", list)
	}
}

func TestMoveHookContext(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "Anchor")
	out, _, _ := runMB("add", "tail", "--title", "Mover")
	id := strings.TrimSpace(out)

	// before and after hooks for "move" must both receive the affected lap's
	// id and title in the standard hook variables.
	hooks := `{"version":1,"hooks":[` +
		`{"title":"beforeMove","command":"move","when":"before","run":"echo BEFORE:$id:$title","passback":true},` +
		`{"title":"afterMove","command":"move","when":"after","run":"echo AFTER:$id:$title","passback":true}` +
		`]}`
	if err := os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644); err != nil {
		t.Fatal(err)
	}

	out, errStr, code := runMB("move", id, "head")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "BEFORE:"+id+":Mover") {
		t.Fatalf("expected before hook with task context, got: %s", out)
	}
	if !strings.Contains(out, "AFTER:"+id+":Mover") {
		t.Fatalf("expected after hook with task context, got: %s", out)
	}
}

// --- Edit / Assign tests ---

// TestEditTitleOnlyDoesNotLeakFieldClears is the regression guard for the
// flag-reset harness. edit's semantics hinge on cmd.Flags().Changed: a cleared
// description/assignee is encoded as "flag set, empty value". If the reset
// harness fails to clear editCmd's Changed state, a title-only edit following
// a clear-edit would silently wipe the other fields. Clearing fields on one
// task must NOT affect a later title-only edit of a different task.
func TestEditTitleOnlyDoesNotLeakFieldClears(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	outA, _, _ := runMB("add", "tail", "--title", "A", "--description", "descA", "--assignee", "alA")
	outB, _, _ := runMB("add", "tail", "--title", "B", "--description", "descB", "--assignee", "alB")
	idA := strings.TrimSpace(outA)
	idB := strings.TrimSpace(outB)

	// Clear both flagged fields on A. This sets description/assignee Changed=true
	// and leaves editDescription/editAssignee at "". Without a reset, those leak.
	if _, _, code := runMB("edit", idA, "--description", "", "--assignee", ""); code != 0 {
		t.Fatalf("clear-edit on A failed, code %d", code)
	}

	// Now edit only B's title. description/assignee must be untouched.
	out, errStr, code := runMB("edit", idB, "--title", "B2")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if got := strings.TrimSpace(out); got != idB {
		t.Fatalf("expected edit to echo id %s, got %q", idB, got)
	}

	task := taskByID(t, beadsDir, idB)
	if task.Title != "B2" {
		t.Fatalf("expected title B2, got %q", task.Title)
	}
	if task.Description != "descB" {
		t.Fatalf("title-only edit leaked a description clear; got %q", task.Description)
	}
	if task.Assignee != "alB" {
		t.Fatalf("title-only edit leaked an assignee clear; got %q", task.Assignee)
	}
}

func TestEditTitleField(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Old", "--description", "keep", "--assignee", "al")
	id := strings.TrimSpace(out)

	if _, errStr, code := runMB("edit", id, "--title", "New Title"); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	task := taskByID(t, beadsDir, id)
	if task.Title != "New Title" {
		t.Fatalf("expected title 'New Title', got %q", task.Title)
	}
	if task.Description != "keep" || task.Assignee != "al" {
		t.Fatalf("title edit must not touch other fields; desc=%q assignee=%q", task.Description, task.Assignee)
	}
}

func TestEditRejectsExtraArgs(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	_, errStr, code := runMB("edit", id, "extra", "--title", "T2")
	if code != 1 {
		t.Fatalf("expected code 1 with extra arg, got %d", code)
	}
	if !strings.Contains(errStr, "usage") {
		t.Fatalf("expected usage error, got: %s", errStr)
	}
}

func TestEditAdvancesUpdatedAt(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)
	path := filepath.Join(beadsDir, "laps.json")
	data, _ := store.Load(path)
	old := time.Now().UTC().Add(-6 * time.Minute)
	for i := range data.Tasks {
		if data.Tasks[i].ID == id {
			data.Tasks[i].UpdatedAt = old
		}
	}
	if err := store.Save(path, data); err != nil {
		t.Fatal(err)
	}

	if _, errStr, code := runMB("edit", id, "--title", "T2"); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if task := taskByID(t, beadsDir, id); !task.UpdatedAt.After(old) {
		t.Fatalf("expected updatedAt to advance past %v, got %v", old, task.UpdatedAt)
	}
}

func TestEditDescriptionField(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	if _, errStr, code := runMB("edit", id, "--description", "fresh desc"); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if task := taskByID(t, beadsDir, id); task.Description != "fresh desc" {
		t.Fatalf("expected description 'fresh desc', got %q", task.Description)
	}
}

func TestEditDescriptionUnescapesNewlines(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	// The literal two-character sequence backslash-n is converted to a newline.
	if _, errStr, code := runMB("edit", id, "--description", "line1\\nline2"); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	task := taskByID(t, beadsDir, id)
	if task.Description != "line1\nline2" {
		t.Fatalf("expected escaped newline expansion to %q, got %q", "line1\nline2", task.Description)
	}
}

func TestEditClearsDescription(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T", "--description", "to clear")
	id := strings.TrimSpace(out)

	if _, errStr, code := runMB("edit", id, "--description", ""); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if task := taskByID(t, beadsDir, id); task.Description != "" {
		t.Fatalf("expected description cleared, got %q", task.Description)
	}
}

func TestEditAssigneeTrims(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	if _, errStr, code := runMB("edit", id, "--assignee", "  carol  "); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if task := taskByID(t, beadsDir, id); task.Assignee != "carol" {
		t.Fatalf("expected trimmed assignee 'carol', got %q", task.Assignee)
	}
}

func TestEditClearsAssignee(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T", "--assignee", "al")
	id := strings.TrimSpace(out)

	if _, errStr, code := runMB("edit", id, "--assignee", ""); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if task := taskByID(t, beadsDir, id); task.Assignee != "" {
		t.Fatalf("expected assignee cleared, got %q", task.Assignee)
	}
}

func TestEditBlankTitleErrors(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	_, errStr, code := runMB("edit", id, "--title", "   ")
	if code != 1 {
		t.Fatalf("expected code 1 for blank title, got %d", code)
	}
	if !strings.Contains(errStr, "title must not be blank") {
		t.Fatalf("expected blank-title error, got: %s", errStr)
	}
}

func TestEditNoFieldFlagsExitsOne(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	_, errStr, code := runMB("edit", id)
	if code != 1 {
		t.Fatalf("expected code 1 with no field flags, got %d", code)
	}
	if !strings.Contains(errStr, "at least one of") {
		t.Fatalf("expected field-required error, got: %s", errStr)
	}
}

func TestEditRequiresID(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	// A field flag is set so the field-required check passes, but no id is given.
	_, errStr, code := runMB("edit", "--title", "X")
	if code != 1 {
		t.Fatalf("expected code 1 with no id, got %d", code)
	}
	if !strings.Contains(errStr, "a task id is required") {
		t.Fatalf("expected id-required error, got: %s", errStr)
	}
}

func TestEditNotFound(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("edit", "ghost-id", "--title", "X")
	if code != 1 {
		t.Fatalf("expected code 1 for unknown id, got %d", code)
	}
	if !strings.Contains(errStr, "not found") {
		t.Fatalf("expected not-found error, got: %s", errStr)
	}
}

func TestEditSuccessPrintsOnlyID(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	out, errStr, code := runMB("edit", id, "--title", "T2")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if out != id+"\n" {
		t.Fatalf("expected stdout to be only the id %q, got %q", id, out)
	}
}

func TestEditJSONOutput(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	out, errStr, code := runMB("edit", id, "--title", "T2", "--description", "d", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	task, ok := result["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object in JSON, got: %s", out)
	}
	if task["id"] != id {
		t.Fatalf("expected preserved id %s, got %v", id, task["id"])
	}
	if task["title"] != "T2" {
		t.Fatalf("expected title 'T2', got %v", task["title"])
	}
	if task["description"] != "d" {
		t.Fatalf("expected description 'd', got %v", task["description"])
	}
}

func TestEditDoneLapWarnsAndPreservesCompletion(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Done one", "--description", "d")
	id := strings.TrimSpace(out)
	runMB("claim")
	if _, _, code := runMB("done"); code != 0 {
		t.Fatalf("setup done failed, code %d", code)
	}

	_, errStr, code := runMB("edit", id, "--title", "Renamed")
	if code != 0 {
		t.Fatalf("expected code 0 editing a done lap, got %d, stderr: %s", code, errStr)
	}
	// The warning is emitted on stderr and must not reopen the lap.
	if !strings.Contains(errStr, "already complete") || !strings.Contains(errStr, "without reopening") {
		t.Fatalf("expected done-lap edit warning on stderr, got: %q", errStr)
	}

	task := taskByID(t, beadsDir, id)
	if task.Title != "Renamed" {
		t.Fatalf("expected title applied to done lap, got %q", task.Title)
	}
	if !task.IsDone {
		t.Fatal("editing a done lap must not reopen it (IsDone must stay true)")
	}
	if task.CompletedAt == nil {
		t.Fatal("editing a done lap must preserve completedAt")
	}
}

func TestEditHookContext(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "EditMe", "--assignee", "al")
	id := strings.TrimSpace(out)

	// before and after hooks for "edit" must both receive the affected lap's
	// id and title in the standard hook variables.
	hooks := `{"version":1,"hooks":[` +
		`{"title":"beforeEdit","command":"edit","when":"before","run":"echo BEFORE:$id:$title","passback":true},` +
		`{"title":"afterEdit","command":"edit","when":"after","run":"echo AFTER:$id:$title","passback":true}` +
		`]}`
	if err := os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644); err != nil {
		t.Fatal(err)
	}

	out, errStr, code := runMB("edit", id, "--title", "EditMe2")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	// The before hook sees the pre-edit title; the after hook sees the new one.
	if !strings.Contains(out, "BEFORE:"+id+":EditMe") {
		t.Fatalf("expected before hook with task context, got: %s", out)
	}
	if !strings.Contains(out, "AFTER:"+id+":EditMe2") {
		t.Fatalf("expected after hook with updated task context, got: %s", out)
	}
}

func TestAssignSetsAssignee(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	if _, errStr, code := runMB("assign", id, "reviewer"); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if task := taskByID(t, beadsDir, id); task.Assignee != "reviewer" {
		t.Fatalf("expected assignee 'reviewer', got %q", task.Assignee)
	}
}

func TestAssignRejectsExtraArgs(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	_, errStr, code := runMB("assign", id, "role", "extra")
	if code != 1 {
		t.Fatalf("expected code 1 with extra arg, got %d", code)
	}
	if !strings.Contains(errStr, "usage") {
		t.Fatalf("expected usage error, got: %s", errStr)
	}
}

func TestAssignTrimsAssignee(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	if _, errStr, code := runMB("assign", id, "  reviewer  "); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if task := taskByID(t, beadsDir, id); task.Assignee != "reviewer" {
		t.Fatalf("expected trimmed assignee 'reviewer', got %q", task.Assignee)
	}
}

func TestAssignBlankClearsAssignee(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T", "--assignee", "al")
	id := strings.TrimSpace(out)

	if _, errStr, code := runMB("assign", id, ""); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if task := taskByID(t, beadsDir, id); task.Assignee != "" {
		t.Fatalf("expected blank role to clear assignee, got %q", task.Assignee)
	}
}

func TestAssignUsageError(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	_, errStr, code := runMB("assign", id)
	if code != 1 {
		t.Fatalf("expected code 1 with missing role, got %d", code)
	}
	if !strings.Contains(errStr, "usage") {
		t.Fatalf("expected usage error, got: %s", errStr)
	}
}

func TestAssignSuccessPrintsOnlyID(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	out, errStr, code := runMB("assign", id, "role")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if out != id+"\n" {
		t.Fatalf("expected stdout to be only the id %q, got %q", id, out)
	}
}

func TestAssignJSONOutput(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	out, errStr, code := runMB("assign", id, "reviewer", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	task, ok := result["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object in JSON, got: %s", out)
	}
	if task["id"] != id {
		t.Fatalf("expected preserved id %s, got %v", id, task["id"])
	}
	if task["assignee"] != "reviewer" {
		t.Fatalf("expected assignee reviewer, got %v", task["assignee"])
	}
}

func TestAssignDoneLapWarnsAndPreservesCompletion(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Done one")
	id := strings.TrimSpace(out)
	runMB("claim", id)
	if _, _, code := runMB("done"); code != 0 {
		t.Fatalf("setup done failed, code %d", code)
	}

	_, errStr, code := runMB("assign", id, "reviewer")
	if code != 0 {
		t.Fatalf("expected code 0 assigning done lap, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(errStr, "already complete") || !strings.Contains(errStr, "without reopening") {
		t.Fatalf("expected done-lap assign warning on stderr, got: %q", errStr)
	}
	task := taskByID(t, beadsDir, id)
	if task.Assignee != "reviewer" {
		t.Fatalf("expected assignee applied to done lap, got %q", task.Assignee)
	}
	if !task.IsDone {
		t.Fatal("assigning a done lap must not reopen it")
	}
	if task.CompletedAt == nil {
		t.Fatal("assigning a done lap must preserve completedAt")
	}
}

func TestAssignAdvancesUpdatedAt(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)
	path := filepath.Join(beadsDir, "laps.json")
	data, _ := store.Load(path)
	old := time.Now().UTC().Add(-6 * time.Minute)
	for i := range data.Tasks {
		if data.Tasks[i].ID == id {
			data.Tasks[i].UpdatedAt = old
		}
	}
	if err := store.Save(path, data); err != nil {
		t.Fatal(err)
	}

	if _, errStr, code := runMB("assign", id, "reviewer"); code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if task := taskByID(t, beadsDir, id); !task.UpdatedAt.After(old) {
		t.Fatalf("expected updatedAt to advance past %v, got %v", old, task.UpdatedAt)
	}
}

func TestAssignHookContext(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "AssignMe")
	id := strings.TrimSpace(out)
	hooks := `{"version":1,"hooks":[` +
		`{"title":"beforeAssign","command":"assign","when":"before","run":"echo BEFORE:$id:$title:$assignee","passback":true},` +
		`{"title":"afterAssign","command":"assign","when":"after","run":"echo AFTER:$id:$title:$assignee","passback":true}` +
		`]}`
	if err := os.WriteFile(filepath.Join(".laps", "hooks.json"), []byte(hooks), 0644); err != nil {
		t.Fatal(err)
	}

	out, errStr, code := runMB("assign", id, "reviewer")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "BEFORE:"+id+":AssignMe:") {
		t.Fatalf("expected before hook with task context, got: %s", out)
	}
	if !strings.Contains(out, "AFTER:"+id+":AssignMe:reviewer") {
		t.Fatalf("expected after hook with updated task context, got: %s", out)
	}
}

func TestEditAndAssignDispatchThroughExecute(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "T")
	id := strings.TrimSpace(out)

	// edit and assign are registered built-ins (isKnownCommand), so Execute
	// must dispatch to their cobra commands rather than the hook-only path.
	editOut, errStr, err := runMBExecute("edit", id, "--title", "ViaExec")
	if err != nil {
		t.Fatalf("expected nil error dispatching edit, got %v, stderr: %s", err, errStr)
	}
	if got := strings.TrimSpace(editOut); got != id {
		t.Fatalf("expected dispatched edit to echo id %s, got %q", id, got)
	}

	assignOut, errStr, err := runMBExecute("assign", id, "executor")
	if err != nil {
		t.Fatalf("expected nil error dispatching assign, got %v, stderr: %s", err, errStr)
	}
	if got := strings.TrimSpace(assignOut); got != id {
		t.Fatalf("expected dispatched assign to echo id %s, got %q", id, got)
	}
}

func writeTestLog(t *testing.T, beadsDir string, lines []string) {
	t.Helper()
	path := filepath.Join(beadsDir, eventlog.LogFileName)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test log: %v", err)
	}
}

func TestLogBasic(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	lines := []string{
		`{"ts":"2026-01-01T12:00:00Z","event":"created","cmd":"add","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
		`{"ts":"2026-01-01T12:05:00Z","event":"claimed","cmd":"claim","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
	}
	writeTestLog(t, beadsDir, lines)

	stdout, stderr, code := runMB("log")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr)
	}

	if !strings.Contains(stdout, "2026-01-01T12:00:00Z") || !strings.Contains(stdout, "created") {
		t.Errorf("stdout missing first event: %q", stdout)
	}
	if !strings.Contains(stdout, "2026-01-01T12:05:00Z") || !strings.Contains(stdout, "claimed") {
		t.Errorf("stdout missing second event: %q", stdout)
	}

	// Verify newest-last ordering
	idx1 := strings.Index(stdout, "2026-01-01T12:00:00Z")
	idx2 := strings.Index(stdout, "2026-01-01T12:05:00Z")
	if idx1 == -1 || idx2 == -1 || idx1 > idx2 {
		t.Errorf("expected newest-last order (2026-01-01T12:00:00Z before 2026-01-01T12:05:00Z), got index order: %d, %d", idx1, idx2)
	}
}

func TestLogLimit(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	var lines []string
	for i := 1; i <= 25; i++ {
		lines = append(lines, fmt.Sprintf(
			`{"ts":"2026-01-01T12:%02d:00Z","event":"created","cmd":"add","file":"laps.json","lap":"laps-%d","title":"T","assignee":"JUNIOR","scope":"root","session":"s","detail":{}}`,
			i, i,
		))
	}
	writeTestLog(t, beadsDir, lines)

	// Default limit is 20
	stdout, _, code := runMB("log")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	outLines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(outLines) != 20 {
		t.Errorf("expected 20 events by default, got %d", len(outLines))
	}
	// Verify that the 20 events shown are the most recent ones (laps-6 to laps-25)
	if !strings.Contains(outLines[0], "laps-6") {
		t.Errorf("expected oldest of the 20 to be laps-6, got: %q", outLines[0])
	}
	if !strings.Contains(outLines[19], "laps-25") {
		t.Errorf("expected newest of the 20 to be laps-25, got: %q", outLines[19])
	}

	// Custom limit via -n
	stdout, _, code = runMB("log", "-n", "5")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	outLines = strings.Split(strings.TrimSpace(stdout), "\n")
	if len(outLines) != 5 {
		t.Errorf("expected 5 events, got %d", len(outLines))
	}
	if !strings.Contains(outLines[0], "laps-21") {
		t.Errorf("expected oldest of the 5 to be laps-21, got: %q", outLines[0])
	}
	if !strings.Contains(outLines[4], "laps-25") {
		t.Errorf("expected newest of the 5 to be laps-25, got: %q", outLines[4])
	}
}

func TestLogFilters(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	lines := []string{
		`{"ts":"2026-01-01T12:00:00Z","event":"created","cmd":"add","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
		`{"ts":"2026-01-01T12:01:00Z","event":"created","cmd":"add","file":"laps.json","lap":"laps-2","title":"T2","assignee":"JUNIOR","scope":"root","session":"s2","detail":{}}`,
		`{"ts":"2026-01-01T12:02:00Z","event":"completed","cmd":"done","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
		`{"ts":"2026-01-01T12:03:00Z","event":"claimed","cmd":"claim","file":"laps.json","lap":"laps-2","title":"T2","assignee":"JUNIOR","scope":"root","session":"s2","detail":{}}`,
		`{"ts":"2026-01-01T12:04:00Z","event":"completed","cmd":"done","file":"laps.json","lap":"laps-2","title":"T2","assignee":"JUNIOR","scope":"root","session":"s2","detail":{}}`,
	}
	writeTestLog(t, beadsDir, lines)

	// Filter by --lap
	stdout, _, code := runMB("log", "--lap", "laps-1")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	outLines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(outLines) != 2 {
		t.Fatalf("expected 2 matching lap events, got %d", len(outLines))
	}
	if !strings.Contains(outLines[0], "created") || !strings.Contains(outLines[1], "completed") {
		t.Errorf("unexpected output for lap filter: %v", outLines)
	}

	// Filter by --session
	stdout, _, code = runMB("log", "--session", "s2")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	outLines = strings.Split(strings.TrimSpace(stdout), "\n")
	if len(outLines) != 3 {
		t.Fatalf("expected 3 matching session events, got %d", len(outLines))
	}

	// Filter-then-limit: filter by lap AND limit
	stdout, _, code = runMB("log", "--lap", "laps-2", "-n", "2")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	outLines = strings.Split(strings.TrimSpace(stdout), "\n")
	if len(outLines) != 2 {
		t.Fatalf("expected 2 matching limited events, got %d", len(outLines))
	}
	if !strings.Contains(outLines[0], "claimed") || !strings.Contains(outLines[1], "completed") {
		t.Errorf("expected most recent matching: claimed then completed, got: %v", outLines)
	}
}

func TestLogSinceInclusive(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	lines := []string{
		`{"ts":"2026-01-01T12:00:00Z","event":"created","cmd":"add","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
		`{"ts":"2026-01-01T12:01:00Z","event":"claimed","cmd":"claim","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
		`{"ts":"2026-01-01T12:02:00Z","event":"completed","cmd":"done","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
	}
	writeTestLog(t, beadsDir, lines)

	// Since 12:01:00Z (inclusive)
	stdout, _, code := runMB("log", "--since", "2026-01-01T12:01:00Z")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	outLines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(outLines) != 2 {
		t.Fatalf("expected 2 events since 12:01:00Z, got %d", len(outLines))
	}
	if !strings.Contains(outLines[0], "claimed") || !strings.Contains(outLines[1], "completed") {
		t.Errorf("expected claimed and completed, got: %v", outLines)
	}
}

func TestLogMalformed(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	lines := []string{
		`{"ts":"2026-01-01T12:00:00Z","event":"created","cmd":"add","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
		`not valid JSON at all`,
		`{"ts":"2026-01-01T12:02:00Z","event":"completed","cmd":"done","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
	}
	writeTestLog(t, beadsDir, lines)

	stdout, stderr, code := runMB("log")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr)
	}

	outLines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(outLines) != 2 {
		t.Errorf("expected 2 output lines after skipping malformed line, got %d: %q", len(outLines), stdout)
	}
	if !strings.Contains(stderr, "laps: log: skipping malformed line") {
		t.Errorf("expected stderr note, got: %q", stderr)
	}
}

func TestLogReadsLongJSONLLine(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	line, err := json.Marshal(map[string]interface{}{
		"ts":       "2026-01-01T12:00:00Z",
		"event":    "created",
		"cmd":      "add",
		"file":     "laps.json",
		"lap":      "laps-long",
		"title":    strings.Repeat("x", 70*1024),
		"assignee": "JUNIOR",
		"scope":    "root",
		"session":  "s1",
		"detail":   map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("marshal long log line: %v", err)
	}
	writeTestLog(t, beadsDir, []string{string(line)})

	stdout, stderr, code := runMB("log", "--session", "no-match")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected no matching output, got: %q", stdout)
	}
}

func TestLogJSONOutput(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	lines := []string{
		`{"ts":"2026-01-01T12:00:00Z","event":"created","cmd":"add","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
	}
	writeTestLog(t, beadsDir, lines)

	stdout, _, code := runMB("log", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v, output: %q", err, stdout)
	}

	evs, ok := parsed["events"]
	if !ok {
		t.Fatalf("JSON output missing 'events' field: %q", stdout)
	}
	evList, ok := evs.([]interface{})
	if !ok || len(evList) != 1 {
		t.Fatalf("expected events to be array of size 1, got: %v", evs)
	}
	firstEv, ok := evList[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected event to be an object, got: %T", evList[0])
	}
	if firstEv["lap"] != "laps-1" || firstEv["event"] != "created" {
		t.Errorf("event fields mismatch, got: %v", firstEv)
	}
}

func TestLogJSONOutputNoMatchesUsesEmptyArray(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	lines := []string{
		`{"ts":"2026-01-01T12:00:00Z","event":"created","cmd":"add","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
	}
	writeTestLog(t, beadsDir, lines)

	stdout, _, code := runMB("log", "--json-output", "--lap", "laps-missing")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v, output: %q", err, stdout)
	}

	evs, ok := parsed["events"]
	if !ok {
		t.Fatalf("JSON output missing 'events' field: %q", stdout)
	}
	evList, ok := evs.([]interface{})
	if !ok {
		t.Fatalf("expected events to be an array, got %T (%v)", evs, evs)
	}
	if len(evList) != 0 {
		t.Fatalf("expected empty events array, got: %v", evList)
	}
}

func TestLogMissingLog(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	// .laps/log.jsonl does not exist. Should exit 0 and be empty.
	stdout, stderr, code := runMB("log")
	if code != 0 {
		t.Fatalf("expected code 0 for missing log, got %d, stderr: %s", code, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout for missing log, got: %q", stdout)
	}

	// JSON output mode
	stdout, _, code = runMB("log", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	evs, ok := parsed["events"]
	if !ok {
		t.Fatalf("missing events key: %q", stdout)
	}
	if len(evs.([]interface{})) != 0 {
		t.Errorf("expected empty events list, got: %v", evs)
	}
}

func TestLogFileFilter(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	lines := []string{
		`{"ts":"2026-01-01T12:00:00Z","event":"created","cmd":"add","file":"laps.json","lap":"laps-1","title":"T1","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
		`{"ts":"2026-01-01T12:01:00Z","event":"created","cmd":"add","file":"auth.json","lap":"laps-2","title":"T2","assignee":"JUNIOR","scope":"root","session":"s1","detail":{}}`,
	}
	writeTestLog(t, beadsDir, lines)

	// Default file filter: laps.json
	stdout, _, code := runMB("log")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "laps-1") || strings.Contains(stdout, "laps-2") {
		t.Errorf("default log output file filter failed, got: %q", stdout)
	}

	// Filter by --file auth
	stdout, _, code = runMB("log", "--file", "auth")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.Contains(stdout, "laps-1") || !strings.Contains(stdout, "laps-2") {
		t.Errorf("explicit log output --file filter failed, got: %q", stdout)
	}
}

func parseStatusJSON(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("status JSON did not parse: %v\nraw: %q", err, raw)
	}
	return m
}

func TestStatusEmpty(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, errStr, code := runMB("status")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "State: empty") {
		t.Fatalf("expected empty state, got: %s", out)
	}
	if !strings.Contains(out, "Head: none") {
		t.Fatalf("expected no head, got: %s", out)
	}
}

func TestStatusReadyWithCountsAndHead(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	outA, _, _ := runMB("add", "tail", "--title", "Alpha", "--assignee", "SENIOR")
	idA := strings.TrimSpace(outA)
	runMB("add", "tail", "--title", "Beta", "--assignee", "JUNIOR")
	runMB("add", "tail", "--title", "Gamma")

	out, errStr, code := runMB("status")
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(out, "State: ready") {
		t.Fatalf("expected ready state with todos and no claim, got: %s", out)
	}
	if !strings.Contains(out, "Laps: 3 todo, 0 done (3 total)") {
		t.Fatalf("expected counts line, got: %s", out)
	}
	if !strings.Contains(out, "Head: "+idA+" — Alpha") {
		t.Fatalf("expected head to be Alpha, got: %s", out)
	}
	if !strings.Contains(out, "- SENIOR: 1") || !strings.Contains(out, "- JUNIOR: 1") {
		t.Fatalf("expected assignee breakdown, got: %s", out)
	}
	if !strings.Contains(out, "Claim: none") {
		t.Fatalf("expected no claim, got: %s", out)
	}
}

func TestStatusComplete(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "tail", "--title", "Only")
	runMB("claim")
	if _, _, code := runMB("done"); code != 0 {
		t.Fatalf("done failed, code %d", code)
	}

	out, _, code := runMB("status")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "State: complete") {
		t.Fatalf("expected complete state, got: %s", out)
	}
	if !strings.Contains(out, "Laps: 0 todo, 1 done (1 total)") {
		t.Fatalf("expected counts, got: %s", out)
	}
}

func TestStatusActiveWithValidClaim(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	outA, _, _ := runMB("add", "head", "--title", "Alpha")
	idA := strings.TrimSpace(outA)
	if _, _, code := runMB("claim", idA); code != 0 {
		t.Fatalf("claim failed, code %d", code)
	}

	out, _, code := runMB("status")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "State: active") {
		t.Fatalf("expected active state, got: %s", out)
	}
	if !strings.Contains(out, "Claim: "+idA+" (valid)") {
		t.Fatalf("expected valid claim, got: %s", out)
	}
}

func TestStatusJSONShape(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	// No claim: file identity present, claimedAt/ageSeconds explicitly null.
	runMB("add", "head", "--title", "Alpha")
	out, _, code := runMB("status", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	m := parseStatusJSON(t, out)
	if m["file"] != "laps.json" {
		t.Fatalf("expected file identity laps.json, got: %v", m["file"])
	}
	if m["state"] != "ready" {
		t.Fatalf("expected ready, got: %v", m["state"])
	}
	claim, ok := m["claim"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected claim object, got: %v", m["claim"])
	}
	if claim["valid"] != false {
		t.Fatalf("expected claim.valid false, got: %v", claim["valid"])
	}
	if v, present := claim["claimedAt"]; !present || v != nil {
		t.Fatalf("expected null claimedAt when no claim, got present=%v val=%v", present, v)
	}
	if v, present := claim["ageSeconds"]; !present || v != nil {
		t.Fatalf("expected null ageSeconds when no claim, got present=%v val=%v", present, v)
	}

	// With a claim: claimedAt is a non-null RFC3339 string and ageSeconds a number.
	runMB("claim")
	out, _, code = runMB("status", "--json-output")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	m = parseStatusJSON(t, out)
	claim = m["claim"].(map[string]interface{})
	if claim["valid"] != true {
		t.Fatalf("expected claim.valid true, got: %v", claim["valid"])
	}
	ts, ok := claim["claimedAt"].(string)
	if !ok {
		t.Fatalf("expected claimedAt string, got: %v", claim["claimedAt"])
	}
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Fatalf("claimedAt is not RFC3339: %v", err)
	}
	if _, ok := claim["ageSeconds"].(float64); !ok {
		t.Fatalf("expected ageSeconds number, got: %v", claim["ageSeconds"])
	}
}

func TestStatusCorruptStoreErrors(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(beadsDir, "laps.json"), []byte(`{"version":1,"tasks":`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, code := runMB("status")
	if code == 0 {
		t.Fatalf("expected non-zero exit for corrupt store, got 0")
	}
}

func TestStatusMalformedClaimErrors(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	runMB("add", "head", "--title", "Alpha")
	// Structured-looking but invalid JSON => malformed claim, not legacy.
	if err := os.WriteFile(filepath.Join(beadsDir, "claim"), []byte(`{"lap":`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, code := runMB("status")
	if code == 0 {
		t.Fatalf("expected non-zero exit for malformed claim, got 0")
	}
}

func TestStatusDanglingClaimIsDegradedNoAutoClear(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	outA, _, _ := runMB("add", "head", "--title", "Alpha")
	idA := strings.TrimSpace(outA)
	if _, _, code := runMB("delete", idA); code != 0 {
		t.Fatalf("delete failed, code %d", code)
	}
	if err := store.WriteClaim(beadsDir, store.Claim{Lap: idA, File: "laps.json"}); err != nil {
		t.Fatalf("write stale claim: %v", err)
	}

	claimPath := filepath.Join(beadsDir, "claim")
	before, err := os.ReadFile(claimPath)
	if err != nil {
		t.Fatalf("claim file should still exist: %v", err)
	}

	out, _, code := runMB("status", "--json-output")
	if code != 0 {
		t.Fatalf("expected exit 0 for degraded snapshot, got %d", code)
	}
	m := parseStatusJSON(t, out)
	claim := m["claim"].(map[string]interface{})
	if claim["valid"] != false {
		t.Fatalf("expected claim.valid false for dangling claim, got: %v", claim["valid"])
	}
	if claim["lap"] != idA {
		t.Fatalf("expected dangling claim to retain its lap id, got: %v", claim["lap"])
	}
	if m["state"] != "empty" {
		t.Fatalf("expected empty state (lap deleted), got: %v", m["state"])
	}

	after, err := os.ReadFile(claimPath)
	if err != nil {
		t.Fatalf("claim file must not be cleared: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("status must not rewrite the claim file: before=%q after=%q", before, after)
	}
}

func TestStatusDoneClaimIsInvalid(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	outA, _, _ := runMB("add", "head", "--title", "Alpha")
	idA := strings.TrimSpace(outA)
	// Complete the lap without going through claim, then point a claim at the
	// now-done lap directly.
	if _, _, code := runMB("done", idA); code != 0 {
		t.Fatalf("done failed, code %d", code)
	}
	if err := store.WriteClaim(beadsDir, store.Claim{Lap: idA, File: "laps.json"}); err != nil {
		t.Fatalf("write claim: %v", err)
	}

	out, _, code := runMB("status", "--json-output")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	m := parseStatusJSON(t, out)
	claim := m["claim"].(map[string]interface{})
	if claim["valid"] != false {
		t.Fatalf("expected claim.valid false for done lap, got: %v", claim["valid"])
	}
	if m["state"] != "complete" {
		t.Fatalf("expected complete state, got: %v", m["state"])
	}
}

func TestStatusWrongFileClaimIsInvalid(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	outA, _, _ := runMB("add", "head", "--title", "Alpha")
	idA := strings.TrimSpace(outA)
	// A claim recorded against a different task file must not count as valid for
	// the selected (default) file even though the lap id exists there.
	if err := os.WriteFile(filepath.Join(beadsDir, "claim"),
		[]byte(`{"lap":"`+idA+`","file":"other.json","claimedAt":"2026-06-30T00:00:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _, code := runMB("status", "--json-output")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	m := parseStatusJSON(t, out)
	claim := m["claim"].(map[string]interface{})
	if claim["valid"] != false {
		t.Fatalf("expected claim.valid false for wrong-file claim, got: %v", claim["valid"])
	}
	if m["state"] != "ready" {
		t.Fatalf("expected ready state (todo present, no valid claim), got: %v", m["state"])
	}
	// Even an invalid claim surfaces its recorded claimedAt for visibility.
	if _, ok := claim["claimedAt"].(string); !ok {
		t.Fatalf("expected claimedAt surfaced for dangling claim, got: %v", claim["claimedAt"])
	}
}
