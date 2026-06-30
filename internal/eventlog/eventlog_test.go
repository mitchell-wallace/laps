package eventlog

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// readLines returns the non-empty lines written to .laps/log.jsonl under dir.
func readLines(t *testing.T, dir string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, LogFileName))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// lastLine parses the final log line into a generic map.
func lastLine(t *testing.T, dir string) map[string]interface{} {
	t.Helper()
	lines := readLines(t, dir)
	if len(lines) == 0 {
		t.Fatal("expected at least one log line, got none")
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &m); err != nil {
		t.Fatalf("unmarshal last line: %v", err)
	}
	return m
}

// TestAppend_WritesValidLine verifies one valid JSON line is written with every
// required schema field populated when the caller supplies a full entry.
func TestAppend_WritesValidLine(t *testing.T) {
	beadsDir := t.TempDir()
	os.Unsetenv("LAPS_SESSION")

	Append(beadsDir, Entry{
		Event:    "created",
		Cmd:      "add",
		File:     "laps.json",
		Lap:      "laps-0b14",
		Title:    "Build event-log writer",
		Assignee: "JUNIOR",
		Detail:   map[string]interface{}{"position": "tail"},
	})

	m := lastLine(t, beadsDir)

	// ts is present and is a UTC RFC3339 string.
	ts, ok := m["ts"].(string)
	if !ok {
		t.Fatalf("ts is %T, want RFC3339 string", m["ts"])
	}
	if !strings.HasSuffix(ts, "Z") {
		t.Errorf("ts = %q, want a UTC ('Z') timestamp", ts)
	}
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Errorf("ts = %q is not valid RFC3339: %v", ts, err)
	}

	want := map[string]interface{}{
		"event":    "created",
		"cmd":      "add",
		"file":     "laps.json",
		"lap":      "laps-0b14",
		"title":    "Build event-log writer",
		"assignee": "JUNIOR",
		"scope":    "root",
		"session":  "",
	}
	for k, w := range want {
		if m[k] != w {
			t.Errorf("%s = %v, want %v", k, m[k], w)
		}
	}
	detail, ok := m["detail"].(map[string]interface{})
	if !ok {
		t.Fatalf("detail is %T, want a JSON object", m["detail"])
	}
	if detail["position"] != "tail" {
		t.Errorf("detail.position = %v, want tail", detail["position"])
	}
}

// TestAppend_OmitsEmptyOptionalFields verifies lap/title/assignee are omitted
// when not applicable, while detail stays present as an empty object.
func TestAppend_OmitsEmptyOptionalFields(t *testing.T) {
	beadsDir := t.TempDir()

	Append(beadsDir, Entry{Event: "completed", Cmd: "done", File: "laps.json"})

	m := lastLine(t, beadsDir)
	for _, k := range []string{"lap", "title", "assignee"} {
		if _, ok := m[k]; ok {
			t.Errorf("optional field %q should be omitted when empty", k)
		}
	}
	if m["detail"] == nil {
		t.Error("detail should be present as an empty object, got nil")
	}
	if d, ok := m["detail"].(map[string]interface{}); !ok || len(d) != 0 {
		t.Errorf("detail = %v, want empty object {}", m["detail"])
	}
}

// TestAppend_Session verifies session is stamped from LAPS_SESSION when set and
// is an empty string when unset.
func TestAppend_Session(t *testing.T) {
	beadsDir := t.TempDir()

	t.Setenv("LAPS_SESSION", "run-42")
	Append(beadsDir, Entry{Event: "created", Cmd: "add", File: "laps.json"})
	if got := lastLine(t, beadsDir)["session"]; got != "run-42" {
		t.Errorf("session = %v, want run-42", got)
	}

	os.Unsetenv("LAPS_SESSION")
	Append(beadsDir, Entry{Event: "completed", Cmd: "done", File: "laps.json"})
	m := lastLine(t, beadsDir)
	if got, ok := m["session"]; !ok || got != "" {
		t.Errorf("session = %v (ok=%v), want present empty string", got, ok)
	}
}

// TestAppend_ScopeDefaultsRoot verifies scope defaults to "root" and that a
// caller-provided scope is honored (so stint population can be additive later).
func TestAppend_ScopeDefaultsRoot(t *testing.T) {
	beadsDir := t.TempDir()

	Append(beadsDir, Entry{Event: "created", Cmd: "add", File: "laps.json"})
	if got := lastLine(t, beadsDir)["scope"]; got != "root" {
		t.Errorf("scope = %v, want root", got)
	}

	Append(beadsDir, Entry{Event: "created", Cmd: "add", File: "auth.json", Scope: "stint-auth"})
	if got := lastLine(t, beadsDir)["scope"]; got != "stint-auth" {
		t.Errorf("scope = %v, want stint-auth", got)
	}
}

// TestAppend_BestEffortOnFailure verifies a write failure emits a one-line
// stderr warning, returns no error, and never affects the caller.
func TestAppend_BestEffortOnFailure(t *testing.T) {
	beadsDir := t.TempDir()
	// Make log.jsonl unwritable by turning it into a directory so OpenFile fails.
	if err := os.Mkdir(filepath.Join(beadsDir, LogFileName), 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	prev := stderr
	stderr = &buf
	t.Cleanup(func() { stderr = prev })

	// Must not panic and returns nothing (caller unaffected by contract).
	Append(beadsDir, Entry{Event: "created", Cmd: "add", File: "laps.json"})

	if buf.Len() == 0 {
		t.Fatal("expected a one-line stderr warning on write failure, got none")
	}
	// One logical line of warning.
	if n := bytes.Count(buf.Bytes(), []byte("\n")); n != 1 {
		t.Errorf("stderr warning spanned %d lines, want exactly 1", n)
	}
}

// TestAppend_Additive verifies existing lines are preserved across appends.
func TestAppend_Additive(t *testing.T) {
	beadsDir := t.TempDir()

	preExisting := `{"ts":"2026-01-01T00:00:00Z","event":"created","cmd":"add","file":"laps.json","scope":"root","detail":{},"session":"old"}`
	if err := os.WriteFile(filepath.Join(beadsDir, LogFileName), []byte(preExisting+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	Append(beadsDir, Entry{Event: "completed", Cmd: "done", File: "laps.json"})

	lines := readLines(t, beadsDir)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (existing + appended)", len(lines))
	}
	if lines[0] != preExisting {
		t.Errorf("first line changed:\n got: %s\nwant: %s", lines[0], preExisting)
	}

	var appended map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &appended); err != nil {
		t.Fatalf("unmarshal appended line: %v", err)
	}
	if appended["event"] != "completed" {
		t.Errorf("appended event = %v, want completed", appended["event"])
	}

	// Confirm the two events are distinct objects (additive, not overwritten).
	if reflect.DeepEqual(appended["event"], "created") {
		t.Error("appended line should not echo the pre-existing event")
	}
}
