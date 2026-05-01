package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func setupTempRepo(t *testing.T) (beadsDir string, cleanup func()) {
	t.Helper()
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	beadsDir = filepath.Join(root, ".beads")
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
	listAll = false
	listDone = false
	addTitle = ""
	addDescription = ""
	addAssignee = ""
	addJSON = ""
	for _, f := range []*pflag.FlagSet{
		rootCmd.PersistentFlags(),
		addCmd.Flags(),
		listCmd.Flags(),
	} {
		f.VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

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

	outBuf := make([]byte, 4096)
	n, _ := rOut.Read(outBuf)
	errBuf := make([]byte, 4096)
	m, _ := rErr.Read(errBuf)

	return string(outBuf[:n]), string(errBuf[:m]), code
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

	data, err := os.ReadFile(filepath.Join(beadsDir, "mb.json"))
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

	out, _, code = runMB("list")
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
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestListOutputUnchangedWithoutAssignee(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "tail", "--title", "A")
	id := strings.TrimSpace(out)
	out, _, code := runMB("list")
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
	out, _, code := runMB("list")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	want := fmt.Sprintf("1. %s — A (assignee: alice)\n", id)
	if out != want {
		t.Fatalf("expected assignee in list output %q, got %q", want, out)
	}
}

func TestDone(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	out, _, _ := runMB("add", "head", "--title", "Do me")
	id := strings.TrimSpace(out)
	out, _, code := runMB("done")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.TrimSpace(out) != id {
		t.Fatalf("expected done to print %s, got %s", id, out)
	}
}

func TestDoneNoHead(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	_, errStr, code := runMB("done")
	if code != 3 {
		t.Fatalf("expected code 3, got %d", code)
	}
	if !strings.Contains(errStr, "no head task") {
		t.Fatalf("expected no head task, got: %s", errStr)
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
		_, _, _ = runMB("done")
	}

	out, _, code := runMB("prune")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.TrimSpace(out) != "5" {
		t.Fatalf("expected 5 removed, got %s", out)
	}

	out, _, code = runMB("list", "--done")
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
	os.WriteFile(filepath.Join(".beads", "mb-hooks.json"), []byte(hooks), 0644)

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
	os.WriteFile(filepath.Join(".beads", "mb-hooks.json"), []byte(hooks), 0644)

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
	os.WriteFile(filepath.Join(".beads", "mb-hooks.json"), []byte(hooks), 0644)

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

	if _, err := os.Stat(filepath.Join(beadsDir, "mb.json")); err != nil {
		t.Fatalf("mb.json was not created: %v", err)
	}
}

func TestAddRejectsNonMbJSON(t *testing.T) {
	_, cleanup := setupTempRepo(t)
	defer cleanup()

	otherPath := filepath.Join(".beads", "other.json")
	if err := os.WriteFile(otherPath, []byte(`{"foo":"bar"}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, errStr, code := runMB("add", "head", "--title", "X", "-f", "other")
	if code != 2 {
		t.Fatalf("expected code 2, got %d, stderr: %s", code, errStr)
	}
	if !strings.Contains(errStr, "not a valid mb task file") {
		t.Fatalf("expected invalid mb task file error, got: %s", errStr)
	}
}

func TestAddWorksWhenEmptyMbJSONExistsWithOtherJSON(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(beadsDir, "mb.json"), []byte(`{"version":1,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "other.json"), []byte(`{"version":1,"tasks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	out, errStr, code := runMB("add", "head", "--title", "Task with empty existing mb")
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
	os.WriteFile(filepath.Join(".beads", "mb-hooks.json"), []byte(hooks), 0644)

	out, _, code := runMB("get")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out, "alice") {
		t.Fatalf("expected assignee variable in hook output, got: %s", out)
	}
}
