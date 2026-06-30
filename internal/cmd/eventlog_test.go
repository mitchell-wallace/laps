package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchell-wallace/laps/internal/store"
)

// readEventLog parses .laps/log.jsonl under beadsDir into a slice of line
// maps. A missing file is reported as nil (no events), matching the reader
// contract.
func readEventLog(t *testing.T, beadsDir string) []map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(beadsDir, "log.jsonl"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read log.jsonl: %v", err)
	}
	var out []map[string]interface{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("unmarshal log line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

// eventsOf filters parsed log lines down to just their "event" value, in order.
func eventsOf(lines []map[string]interface{}) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i], _ = l["event"].(string)
	}
	return out
}

// assertNoLAPS_SESSION keeps the event-log session stamp deterministic in case
// the surrounding environment sets LAPS_SESSION (e.g. a Rally run).
func assertNoLAPS_SESSION(t *testing.T) {
	t.Helper()
	t.Setenv("LAPS_SESSION", "")
}

// TestEventLog_CreatedOnAdd verifies a single 'add' appends exactly one
// 'created' line stamped with the resolved file, lap, title, and detail.position.
func TestEventLog_CreatedOnAdd(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, code := runMB("add", "tail", "--title", "First", "--assignee", "JUNIOR")
	if code != 0 {
		t.Fatalf("add failed: code %d", code)
	}
	id := strings.TrimSpace(out)

	lines := readEventLog(t, beadsDir)
	if got := eventsOf(lines); len(got) != 1 || got[0] != "created" {
		t.Fatalf("expected exactly one created event, got %v", got)
	}
	m := lines[0]
	if m["cmd"] != "add" {
		t.Errorf("cmd = %v, want add", m["cmd"])
	}
	if m["file"] != "laps.json" {
		t.Errorf("file = %v, want laps.json", m["file"])
	}
	if m["lap"] != id {
		t.Errorf("lap = %v, want %s", m["lap"], id)
	}
	if m["title"] != "First" {
		t.Errorf("title = %v, want First", m["title"])
	}
	if m["assignee"] != "JUNIOR" {
		t.Errorf("assignee = %v, want JUNIOR", m["assignee"])
	}
	if m["scope"] != "root" {
		t.Errorf("scope = %v, want root", m["scope"])
	}
	detail, ok := m["detail"].(map[string]interface{})
	if !ok || detail["position"] != "tail" {
		t.Errorf("detail = %v, want {position: tail}", m["detail"])
	}
}

// TestEventLog_CreatedFileIdentity verifies --file is reflected as the resolved
// .laps-relative task file on every emitted line.
func TestEventLog_CreatedFileIdentity(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	_, _, code := runMB("add", "tail", "--file", "auth", "--title", "X")
	if code != 0 {
		t.Fatalf("add --file auth failed: code %d", code)
	}
	lines := readEventLog(t, beadsDir)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0]["file"] != "auth.json" {
		t.Errorf("file = %v, want auth.json", lines[0]["file"])
	}
}

// TestEventLog_BatchAddCreatedPerLap verifies a batch --json add appends one
// 'created' line per new lap (N events for N laps), in insertion order.
func TestEventLog_BatchAddCreatedPerLap(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	batch := `[{"title":"A"},{"title":"B"},{"title":"C"}]`
	out, _, code := runMB("add", "tail", "--json", batch)
	if code != 0 {
		t.Fatalf("batch add failed: code %d", code)
	}
	// Without --json-output, add prints the created ids newline-joined.
	ids := strings.Fields(strings.TrimSpace(out))
	if len(ids) != 3 {
		t.Fatalf("expected 3 created ids on stdout, got %d (%q)", len(ids), out)
	}

	lines := readEventLog(t, beadsDir)
	if got := eventsOf(lines); len(got) != 3 {
		t.Fatalf("expected 3 created events, got %v", got)
	}
	wantTitles := []string{"A", "B", "C"}
	seen := map[string]bool{}
	for i, l := range lines {
		if l["event"] != "created" {
			t.Errorf("line %d event = %v, want created", i, l["event"])
		}
		seen[l["lap"].(string)] = true
		if l["title"] != wantTitles[i] {
			t.Errorf("line %d title = %v, want %s", i, l["title"], wantTitles[i])
		}
	}
	for _, id := range ids {
		if !seen[id] {
			t.Errorf("created id %s has no matching log event", id)
		}
	}
}

// TestEventLog_CompletedOnDone verifies 'done' appends a 'completed' event for
// the finished lap. Claim-clear semantics (a later 'unclaimed' line) are owned
// by a separate change and are intentionally not asserted here.
func TestEventLog_CompletedOnDone(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "Finish me")
	id := strings.TrimSpace(out)

	_, _, code := runMB("done", id)
	if code != 0 {
		t.Fatalf("done failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	got := eventsOf(lines)
	if len(got) != 2 || got[0] != "created" || got[1] != "completed" {
		t.Fatalf("expected [created, completed], got %v", got)
	}
	last := lines[len(lines)-1]
	if last["cmd"] != "done" || last["lap"] != id || last["title"] != "Finish me" {
		t.Errorf("completed line = %v", last)
	}
}

// TestEventLog_ReopenedOnDoneUndo verifies 'done undo' appends a 'reopened'
// event for the re-opened lap.
func TestEventLog_ReopenedOnDoneUndo(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "Reopen me")
	id := strings.TrimSpace(out)
	if _, _, code := runMB("done", id); code != 0 {
		t.Fatalf("done failed: code %d", code)
	}
	if _, _, code := runMB("done", "undo"); code != 0 {
		t.Fatalf("done undo failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	got := eventsOf(lines)
	if len(got) != 3 || got[2] != "reopened" {
		t.Fatalf("expected last event reopened, got %v", got)
	}
	last := lines[len(lines)-1]
	if last["cmd"] != "done-undo" || last["lap"] != id {
		t.Errorf("reopened line = %v", last)
	}
}

// TestEventLog_DeletedOnDelete verifies 'delete' appends a 'deleted' event.
func TestEventLog_DeletedOnDelete(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "Remove me")
	id := strings.TrimSpace(out)
	if _, _, code := runMB("delete", id); code != 0 {
		t.Fatalf("delete failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	got := eventsOf(lines)
	if len(got) != 2 || got[1] != "deleted" {
		t.Fatalf("expected [created, deleted], got %v", got)
	}
	last := lines[len(lines)-1]
	if last["cmd"] != "delete" || last["lap"] != id {
		t.Errorf("deleted line = %v", last)
	}
}

// TestEventLog_PrunedPerRemovedLap verifies 'prune' appends one 'pruned' event
// per removed lap and none for the kept laps.
func TestEventLog_PrunedPerRemovedLap(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	var ids []string
	for i := 0; i < 3; i++ {
		out, _, _ := runMB("add", "tail", "--title", "Done task")
		ids = append(ids, strings.TrimSpace(out))
		// 'done <id>' avoids the claim path so the log only carries completed.
		if _, _, code := runMB("done", ids[i]); code != 0 {
			t.Fatalf("done %s failed: code %d", ids[i], code)
		}
	}

	if _, _, code := runMB("prune", "0"); code != 0 {
		t.Fatalf("prune 0 failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	var pruned []map[string]interface{}
	for _, l := range lines {
		if l["event"] == "pruned" {
			pruned = append(pruned, l)
		}
	}
	if len(pruned) != 3 {
		t.Fatalf("expected 3 pruned events, got %d", len(pruned))
	}
	got := map[string]bool{}
	for _, l := range pruned {
		if l["cmd"] != "prune" {
			t.Errorf("pruned cmd = %v, want prune", l["cmd"])
		}
		got[l["lap"].(string)] = true
	}
	for _, id := range ids {
		if !got[id] {
			t.Errorf("expected a pruned event for lap %s", id)
		}
	}
}

// TestEventLog_MovedDetail verifies 'move' appends a 'moved' event whose detail
// carries the target position and the from/to order keys.
func TestEventLog_MovedDetail(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	outA, _, _ := runMB("add", "tail", "--title", "A")
	outB, _, _ := runMB("add", "tail", "--title", "B")
	idA := strings.TrimSpace(outA)
	idB := strings.TrimSpace(outB)

	if _, _, code := runMB("move", idA, "tail"); code != 0 {
		t.Fatalf("move failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	last := lines[len(lines)-1]
	if last["event"] != "moved" {
		t.Fatalf("expected last event moved, got %v", last["event"])
	}
	if last["cmd"] != "move" || last["lap"] != idA {
		t.Errorf("moved line = %v", last)
	}
	detail, ok := last["detail"].(map[string]interface{})
	if !ok {
		t.Fatalf("detail not an object: %v", last["detail"])
	}
	if detail["position"] != "tail" {
		t.Errorf("detail.position = %v, want tail", detail["position"])
	}
	from, _ := detail["from"].(float64)
	to, _ := detail["to"].(float64)
	if from == to {
		t.Errorf("expected from != to (a real reorder), got from=%v to=%v", from, to)
	}

	// 'after' position stamps the target id in detail.
	if _, _, code := runMB("move", idB, "after", idA); code != 0 {
		t.Fatalf("move after failed: code %d", code)
	}
	lines = readEventLog(t, beadsDir)
	after := lines[len(lines)-1]
	if after["event"] != "moved" {
		t.Fatalf("expected moved, got %v", after["event"])
	}
	detail, _ = after["detail"].(map[string]interface{})
	if detail["position"] != "after" || detail["after"] != idA {
		t.Errorf("after-move detail = %v", detail)
	}
}

// TestEventLog_EditedDetail verifies 'edit' appends an 'edited' event listing
// the changed fields in detail, and that 'assign' (an edit shortcut) also logs
// 'edited' with cmd 'assign'.
func TestEventLog_EditedDetail(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "Orig", "--assignee", "QA")
	id := strings.TrimSpace(out)

	if _, _, code := runMB("edit", id, "--title", "Renamed", "--description", "d"); code != 0 {
		t.Fatalf("edit failed: code %d", code)
	}
	lines := readEventLog(t, beadsDir)
	last := lines[len(lines)-1]
	if last["event"] != "edited" || last["cmd"] != "edit" {
		t.Fatalf("expected edited/edit, got %v", last)
	}
	detail, _ := last["detail"].(map[string]interface{})
	fields, _ := detail["fields"].([]interface{})
	gotFields := map[string]bool{}
	for _, f := range fields {
		gotFields[f.(string)] = true
	}
	if !gotFields["title"] || !gotFields["description"] || len(fields) != 2 {
		t.Errorf("edited detail.fields = %v, want [title description]", fields)
	}

	// assign is an edit shortcut: event 'edited', cmd 'assign', fields [assignee].
	if _, _, code := runMB("assign", id, "DEV"); code != 0 {
		t.Fatalf("assign failed: code %d", code)
	}
	lines = readEventLog(t, beadsDir)
	assign := lines[len(lines)-1]
	if assign["event"] != "edited" || assign["cmd"] != "assign" {
		t.Fatalf("expected edited/assign, got %v", assign)
	}
	detail, _ = assign["detail"].(map[string]interface{})
	fields, _ = detail["fields"].([]interface{})
	if len(fields) != 1 || fields[0] != "assignee" {
		t.Errorf("assign detail.fields = %v, want [assignee]", fields)
	}
	if assign["assignee"] != "DEV" {
		t.Errorf("assign line assignee = %v, want DEV", assign["assignee"])
	}
}

// TestEventLog_ReadsAppendNothing verifies read-only commands append nothing to
// the event log.
func TestEventLog_ReadsAppendNothing(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "Seed")
	_ = strings.TrimSpace(out)
	before := len(readEventLog(t, beadsDir))

	for _, args := range [][]string{
		{"get"},
		{"list"},
		{"count"},
	} {
		if _, _, code := runMB(args...); code != 0 {
			t.Fatalf("%v failed: code %d", args, code)
		}
	}
	after := readEventLog(t, beadsDir)
	if len(after) != before {
		t.Fatalf("read-only commands appended events: before=%d after=%d (%v)", before, len(after), eventsOf(after))
	}
}

// TestEventLog_BestEffortDoesNotChangeExitCode verifies a forced log-write
// failure leaves the command's exit code unchanged (0) and the mutation still
// applies. (The one-line stderr warning contract itself is locked by the
// eventlog package's own TestAppend_BestEffortOnFailure.)
func TestEventLog_BestEffortDoesNotChangeExitCode(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	// Turn the log into a directory so the append OpenFile fails.
	if err := os.Mkdir(filepath.Join(beadsDir, "log.jsonl"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, _, code := runMB("add", "tail", "--title", "Still works")
	if code != 0 {
		t.Fatalf("expected exit code 0 despite log failure, got %d", code)
	}
	// The command must still have applied its state change.
	id := strings.TrimSpace(out)
	data, err := store.Load(filepath.Join(beadsDir, "laps.json"))
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	found := false
	for _, tk := range data.Tasks {
		if tk.ID == id && tk.Title == "Still works" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the task to persist despite the log-write failure")
	}
}
