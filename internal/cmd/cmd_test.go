package cmd

import (
	"encoding/json"
	"fmt"
	"io"
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
	addTitle = ""
	addDescription = ""
	addAssignee = ""
	addJSON = ""
	addStdin = false
	updateYes = false
	for _, f := range []*pflag.FlagSet{
		rootCmd.PersistentFlags(),
		addCmd.Flags(),
		listCmd.Flags(),
		updateCmd.Flags(),
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
	addTitle = ""
	addDescription = ""
	addAssignee = ""
	addJSON = ""
	addStdin = false
	updateYes = false
	for _, f := range []*pflag.FlagSet{
		rootCmd.PersistentFlags(),
		addCmd.Flags(),
		listCmd.Flags(),
		updateCmd.Flags(),
	} {
		f.VisitAll(func(flag *pflag.Flag) {
			flag.Changed = false
		})
	}

	err = Execute("test")

	wOut.Close()
	wErr.Close()

	outBuf, _ := io.ReadAll(rOut)
	errBuf, _ := io.ReadAll(rErr)

	return string(outBuf), string(errBuf), err
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
	runMB("done")

	// Default list shows only todos, head first: A then B.
	list, _, _ := runMB("list")
	if !strings.Contains(list, "1. ") || !idxBefore(list, "— A", "— B") {
		t.Fatalf("expected A before B in todo list, got:\n%s", list)
	}
	if strings.Contains(list, cid) {
		t.Fatalf("completed lap %s should not appear in default list:\n%s", cid, list)
	}

	// New head lands at the top of the todo section, still below the done lap.
	runMB("add", "head", "--title", "D")
	list2, _, _ := runMB("list")
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
	runMB("done") // completes C

	_, errStr, code := runMB("add", "after", cid, "--title", "D")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(errStr, "already complete") || !strings.Contains(errStr, "head") {
		t.Fatalf("expected fallback-to-head warning, got stderr: %q", errStr)
	}
	list, _, _ := runMB("list")
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
	if !strings.Contains(got, `"version": 2`) {
		t.Fatalf("expected version 2 after migration, got:\n%s", got)
	}
	if !strings.Contains(got, `"order"`) {
		t.Fatalf("expected order keys after migration, got:\n%s", got)
	}
}

// idxBefore reports whether substring a appears before substring b in s.
func idxBefore(s, a, b string) bool {
	ia := strings.Index(s, a)
	ib := strings.Index(s, b)
	return ia >= 0 && ib >= 0 && ia < ib
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

func TestJSONOutputDoneNoHead(t *testing.T) {
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
	if result["error"] != "no head task" {
		t.Fatalf("expected error 'no head task', got: %v", result["error"])
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
