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

// lastDetailReason returns the detail.reason string of a parsed log line.
func lastDetailReason(t *testing.T, line map[string]interface{}) string {
	t.Helper()
	detail, _ := line["detail"].(map[string]interface{})
	r, _ := detail["reason"].(string)
	return r
}

// TestEventLog_ClaimedOnClaim verifies a fresh claim appends exactly one
// 'claimed' event for the claimed lap.
func TestEventLog_ClaimedOnClaim(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "Claim me", "--assignee", "SENIOR")
	id := strings.TrimSpace(out)

	if _, _, code := runMB("claim", id); code != 0 {
		t.Fatalf("claim failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	got := eventsOf(lines)
	if len(got) != 2 || got[0] != "created" || got[1] != "claimed" {
		t.Fatalf("expected [created, claimed], got %v", got)
	}
	last := lines[len(lines)-1]
	if last["cmd"] != "claim" || last["lap"] != id || last["title"] != "Claim me" || last["assignee"] != "SENIOR" {
		t.Errorf("claimed line = %v", last)
	}
}

// TestEventLog_SameLapReclaimPreservesAndNoDuplicate verifies re-claiming the
// already-claimed lap preserves claimedAt exactly and emits no second 'claimed'.
func TestEventLog_SameLapReclaimPreservesAndNoDuplicate(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "Sticky")
	id := strings.TrimSpace(out)

	if _, _, code := runMB("claim", id); code != 0 {
		t.Fatalf("first claim failed: code %d", code)
	}
	first, err := store.ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("read claim: %v", err)
	}
	if first.ClaimedAt == nil {
		t.Fatal("expected claimedAt to be recorded on first claim")
	}

	if _, _, code := runMB("claim", id); code != 0 {
		t.Fatalf("reclaim failed: code %d", code)
	}
	second, err := store.ReadClaim(beadsDir, "laps.json")
	if err != nil {
		t.Fatalf("read claim after reclaim: %v", err)
	}
	if second.ClaimedAt == nil || !second.ClaimedAt.Equal(*first.ClaimedAt) {
		t.Errorf("claimedAt not preserved on same-lap reclaim: first=%v second=%v", first.ClaimedAt, second.ClaimedAt)
	}

	// Exactly one 'claimed' event across both claims.
	claimed := 0
	for _, ev := range eventsOf(readEventLog(t, beadsDir)) {
		if ev == "claimed" {
			claimed++
		}
	}
	if claimed != 1 {
		t.Fatalf("expected exactly one claimed event after same-lap reclaim, got %d", claimed)
	}
}

// TestEventLog_DifferentLapReplacement verifies replacing a claimed lap with a
// different lap emits unclaimed(replaced) for the prior lap immediately before
// claimed for the new lap.
func TestEventLog_DifferentLapReplacement(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	outA, _, _ := runMB("add", "tail", "--title", "Alpha")
	idA := strings.TrimSpace(outA)
	outB, _, _ := runMB("add", "tail", "--title", "Beta")
	idB := strings.TrimSpace(outB)

	if _, _, code := runMB("claim", idA); code != 0 {
		t.Fatalf("claim A failed: code %d", code)
	}
	if _, _, code := runMB("claim", idB); code != 0 {
		t.Fatalf("claim B failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	got := eventsOf(lines)
	// created A, created B, claimed A, unclaimed(replaced) A, claimed B
	if len(got) != 5 {
		t.Fatalf("expected 5 events, got %v", got)
	}
	if got[2] != "claimed" || got[3] != "unclaimed" || got[4] != "claimed" {
		t.Fatalf("expected [...claimed, unclaimed, claimed], got %v", got)
	}
	replaced := lines[3]
	if replaced["lap"] != idA || lastDetailReason(t, replaced) != "replaced" {
		t.Errorf("expected unclaimed(replaced) for %s, got %v", idA, replaced)
	}
	newClaim := lines[4]
	if newClaim["lap"] != idB {
		t.Errorf("expected claimed for %s, got %v", idB, newClaim)
	}
}

// TestEventLog_DoneClaimedEmitsCompletedThenUnclaimed verifies completing a
// claimed lap emits completed immediately followed by unclaimed(completed).
func TestEventLog_DoneClaimedEmitsCompletedThenUnclaimed(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "Work")
	id := strings.TrimSpace(out)
	if _, _, code := runMB("claim", id); code != 0 {
		t.Fatalf("claim failed: code %d", code)
	}
	if _, _, code := runMB("done"); code != 0 {
		t.Fatalf("done failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	got := eventsOf(lines)
	// created, claimed, completed, unclaimed(completed)
	if len(got) != 4 || got[2] != "completed" || got[3] != "unclaimed" {
		t.Fatalf("expected [..., completed, unclaimed], got %v", got)
	}
	unclaimed := lines[3]
	if unclaimed["cmd"] != "done" || unclaimed["lap"] != id || lastDetailReason(t, unclaimed) != "completed" {
		t.Errorf("expected unclaimed(completed) for %s on done, got %v", id, unclaimed)
	}
}

// TestEventLog_ClaimUndoEmitsUnclaimed verifies 'claim undo' appends an
// 'unclaimed' event after the claim is removed.
func TestEventLog_ClaimUndoEmitsUnclaimed(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "Undo me")
	id := strings.TrimSpace(out)
	if _, _, code := runMB("claim", id); code != 0 {
		t.Fatalf("claim failed: code %d", code)
	}
	if _, _, code := runMB("claim", "undo"); code != 0 {
		t.Fatalf("claim undo failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	got := eventsOf(lines)
	if len(got) != 3 || got[2] != "unclaimed" {
		t.Fatalf("expected last event unclaimed, got %v", got)
	}
	last := lines[len(lines)-1]
	if last["cmd"] != "claim-undo" || last["lap"] != id {
		t.Errorf("unclaimed line = %v", last)
	}
}

// TestEventLog_FailedClaimWriteEmitsNoEvent verifies a failed WriteClaim leaves
// the log untouched (no 'claimed' event).
func TestEventLog_FailedClaimWriteEmitsNoEvent(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "No write")
	id := strings.TrimSpace(out)

	// Turn the claim path into a directory so WriteClaim's rename fails.
	if err := os.Mkdir(store.ClaimPath(beadsDir), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, code := runMB("claim", id); code != 2 {
		t.Fatalf("expected claim to fail with code 2, got %d", code)
	}

	for _, ev := range eventsOf(readEventLog(t, beadsDir)) {
		if ev == "claimed" || ev == "unclaimed" {
			t.Fatalf("a failed claim write must emit no claim event, found %q", ev)
		}
	}
}

// TestEventLog_FailedClaimRemoveEmitsNoEvent verifies a failed RemoveClaim
// leaves the log untouched (no 'unclaimed' event).
func TestEventLog_FailedClaimRemoveEmitsNoEvent(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	out, _, _ := runMB("add", "head", "--title", "No remove")
	id := strings.TrimSpace(out)
	if _, _, code := runMB("claim", id); code != 0 {
		t.Fatalf("claim failed: code %d", code)
	}

	before := len(readEventLog(t, beadsDir))

	// Make the .laps dir non-writable so os.Remove of the claim file fails while
	// the file itself remains readable. Restored before TempDir cleanup.
	if err := os.Chmod(beadsDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(beadsDir, 0o755)

	if _, _, code := runMB("claim", "undo"); code != 2 {
		t.Fatalf("expected claim undo to fail with code 2, got %d", code)
	}

	os.Chmod(beadsDir, 0o755)
	after := readEventLog(t, beadsDir)
	if len(after) != before {
		t.Fatalf("a failed claim remove must emit no event: before=%d after=%d (%v)", before, len(after), eventsOf(after))
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

// TestEventLog_CrossFileReplacementStampsClaimFile verifies that when a claim in
// one --file is replaced by claiming a lap in a DIFFERENT file, the
// unclaimed(replaced) event stamps the retired claim's own file, lap, title, and
// assignee — not the currently selected file (which need not even contain the
// prior lap). Regression for cross-file claim event identity/metadata.
func TestEventLog_CrossFileReplacementStampsClaimFile(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	// Alpha lives in auth.json; Beta lives in the default laps.json.
	outA, _, _ := runMB("add", "tail", "--file", "auth", "--title", "Alpha", "--assignee", "SENIOR")
	idA := strings.TrimSpace(outA)
	outB, _, _ := runMB("add", "tail", "--title", "Beta", "--assignee", "JUNIOR")
	idB := strings.TrimSpace(outB)

	// Claim Alpha while auth is selected, then replace it by claiming Beta on the
	// default file. The retired claim (Alpha) belongs to auth.json.
	if _, _, code := runMB("claim", idA, "--file", "auth"); code != 0 {
		t.Fatalf("claim A failed: code %d", code)
	}
	if _, _, code := runMB("claim", idB); code != 0 {
		t.Fatalf("claim B failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	got := eventsOf(lines)
	// created A (auth), created B (laps), claimed A (auth),
	// unclaimed(replaced) A (auth), claimed B (laps)
	if len(got) != 5 || got[3] != "unclaimed" || got[4] != "claimed" {
		t.Fatalf("expected [..., unclaimed, claimed], got %v", got)
	}
	replaced := lines[3]
	if replaced["cmd"] != "claim" || lastDetailReason(t, replaced) != "replaced" {
		t.Fatalf("expected unclaimed(replaced) from claim, got %v", replaced)
	}
	if replaced["file"] != "auth.json" {
		t.Errorf("retired-claim file = %v, want auth.json", replaced["file"])
	}
	if replaced["lap"] != idA {
		t.Errorf("retired-claim lap = %v, want %s", replaced["lap"], idA)
	}
	if replaced["title"] != "Alpha" {
		t.Errorf("retired-claim title = %v, want Alpha (lost across files)", replaced["title"])
	}
	if replaced["assignee"] != "SENIOR" {
		t.Errorf("retired-claim assignee = %v, want SENIOR (lost across files)", replaced["assignee"])
	}
	// The replacing claim still records the newly selected file.
	if newClaim := lines[4]; newClaim["lap"] != idB || newClaim["file"] != "laps.json" {
		t.Errorf("replacing claim line = %v, want lap %s file laps.json", newClaim, idB)
	}
}

// TestEventLog_CrossFileClaimUndoStampsClaimFile verifies that clearing (undo) a
// claim that lives in a different --file than the currently selected one stamps
// the unclaimed event with the claim's own file, title, and assignee rather than
// the selected file. Regression for cross-file claim event identity/metadata.
func TestEventLog_CrossFileClaimUndoStampsClaimFile(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	assertNoLAPS_SESSION(t)

	// The lap being claimed lives only in auth.json.
	outA, _, _ := runMB("add", "head", "--file", "auth", "--title", "Cross", "--assignee", "SENIOR")
	idA := strings.TrimSpace(outA)
	if _, _, code := runMB("claim", idA, "--file", "auth"); code != 0 {
		t.Fatalf("claim failed: code %d", code)
	}

	// Undo on the default file: the claim still points into auth.json.
	if _, _, code := runMB("claim", "undo"); code != 0 {
		t.Fatalf("claim undo failed: code %d", code)
	}

	lines := readEventLog(t, beadsDir)
	got := eventsOf(lines)
	if len(got) != 3 || got[2] != "unclaimed" {
		t.Fatalf("expected last event unclaimed, got %v", got)
	}
	last := lines[len(lines)-1]
	if last["cmd"] != "claim-undo" || last["lap"] != idA {
		t.Fatalf("unclaimed line = %v", last)
	}
	if last["file"] != "auth.json" {
		t.Errorf("claim-undo file = %v, want auth.json", last["file"])
	}
	if last["title"] != "Cross" {
		t.Errorf("claim-undo title = %v, want Cross (lost across files)", last["title"])
	}
	if last["assignee"] != "SENIOR" {
		t.Errorf("claim-undo assignee = %v, want SENIOR (lost across files)", last["assignee"])
	}
}
