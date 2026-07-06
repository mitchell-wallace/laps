package tui

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRunnerLoadUsesExactReadArgSequences(t *testing.T) {
	binary, logPath := writeFakeLaps(t, "")
	runner := Runner{Binary: binary, Timeout: time.Second}

	snapshot, err := runner.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(snapshot.Entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(snapshot.Entries))
	}
	if got := snapshot.Entries[0].Laps[0].ID; got != "auth-1" {
		t.Fatalf("stint lap id = %q, want auth-1", got)
	}

	want := []string{
		"status --json-output",
		"list --root --all --json-output",
		"-f stints/auth.laps list --all --json-output",
	}
	if got := readLog(t, logPath); !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestActionDispatchesAndRefreshes(t *testing.T) {
	tests := []struct {
		name    string
		keys    []tea.KeyMsg
		wantLog []string
	}{
		{
			name: "done root",
			keys: []tea.KeyMsg{
				runeKey("j"),
				runeKey("j"),
				runeKey("j"),
				runeKey("d"),
			},
			wantLog: []string{
				"done root-1",
				"status --json-output",
				"list --root --all --json-output",
				"-f stints/auth.laps list --all --json-output",
			},
		},
		{
			name: "done stint",
			keys: []tea.KeyMsg{
				runeKey("j"),
				runeKey("d"),
			},
			wantLog: []string{
				"-f stints/auth.laps done auth-1",
				"status --json-output",
				"list --root --all --json-output",
				"-f stints/auth.laps list --all --json-output",
			},
		},
		{
			name: "delete confirm",
			keys: []tea.KeyMsg{
				runeKey("j"),
				runeKey("j"),
				runeKey("j"),
				runeKey("x"),
				runeKey("y"),
			},
			wantLog: []string{
				"delete root-1",
				"status --json-output",
				"list --root --all --json-output",
				"-f stints/auth.laps list --all --json-output",
			},
		},
		{
			name: "move up root",
			keys: []tea.KeyMsg{
				runeKey("j"),
				runeKey("j"),
				runeKey("j"),
				runeKey("j"),
				runeKey("K"),
			},
			wantLog: []string{
				"move root-3 head",
				"status --json-output",
				"list --root --all --json-output",
				"-f stints/auth.laps list --all --json-output",
			},
		},
		{
			name: "move down root",
			keys: []tea.KeyMsg{
				runeKey("j"),
				runeKey("j"),
				runeKey("j"),
				runeKey("J"),
			},
			wantLog: []string{
				"move root-1 after root-3",
				"status --json-output",
				"list --root --all --json-output",
				"-f stints/auth.laps list --all --json-output",
			},
		},
		{
			name: "hold stint",
			keys: []tea.KeyMsg{
				runeKey("h"),
			},
			wantLog: []string{
				"stints release auth",
				"status --json-output",
				"list --root --all --json-output",
				"-f stints/auth.laps list --all --json-output",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary, logPath := writeFakeLaps(t, "")
			m := modelWithSnapshot(actionSnapshot(), 80, 12)
			m.runner = Runner{Binary: binary, Timeout: time.Second}
			if got := os.Truncate(logPath, 0); got != nil {
				t.Fatalf("truncate log: %v", got)
			}

			for i, key := range tt.keys {
				next, cmd := m.Update(key)
				typed, ok := next.(*Model)
				if !ok {
					t.Fatalf("model type = %T, want tui.Model", next)
				}
				m = typed
				if i == len(tt.keys)-1 {
					if cmd == nil {
						t.Fatalf("final key returned nil cmd")
					}
					msg := cmd()
					next, cmd = m.Update(msg)
					m = next.(*Model)
					if cmd == nil {
						t.Fatalf("action did not schedule refresh")
					}
					msg = cmd()
					next, _ = m.Update(msg)
					m = next.(*Model)
				} else if cmd != nil {
					t.Fatalf("key %d returned unexpected cmd", i)
				}
			}

			if got := readLog(t, logPath); !reflect.DeepEqual(got, tt.wantLog) {
				t.Fatalf("args = %#v, want %#v", got, tt.wantLog)
			}
		})
	}
}

func TestDeleteRequiresConfirmation(t *testing.T) {
	binary, logPath := writeFakeLaps(t, "")
	m := modelWithSnapshot(actionSnapshot(), 80, 12)
	m.runner = Runner{Binary: binary, Timeout: time.Second}
	m = updateModel(t, m, runeKey("j"))

	next, cmd := m.Update(runeKey("x"))
	if cmd != nil {
		t.Fatalf("x returned cmd before confirmation")
	}
	m = next.(*Model)
	if !m.confirm {
		t.Fatalf("confirm = false, want true")
	}
	if got := readLog(t, logPath); len(got) != 0 {
		t.Fatalf("log before confirmation = %#v, want empty", got)
	}
}

func TestConfirmSwallowsOtherKeysAndCancels(t *testing.T) {
	binary, logPath := writeFakeLaps(t, "")
	m := modelWithSnapshot(actionSnapshot(), 80, 12)
	m.runner = Runner{Binary: binary, Timeout: time.Second}
	m = updateModel(t, m, runeKey("j"))
	m = updateModel(t, m, runeKey("x"))
	if !m.confirm {
		t.Fatal("confirm = false after x, want true")
	}
	before := selectedID(m)

	// A cursor key during confirm must cancel, not move the target under y.
	next, cmd := m.Update(runeKey("j"))
	if cmd != nil {
		t.Fatalf("key during confirm returned cmd")
	}
	m = next.(*Model)
	if m.confirm {
		t.Fatal("confirm still armed after other key, want cancelled")
	}
	if got := selectedID(m); got != before {
		t.Fatalf("cursor moved during confirm: %q -> %q", before, got)
	}

	next, cmd = m.Update(runeKey("y"))
	if cmd != nil {
		t.Fatalf("y after cancelled confirm returned cmd")
	}
	m = next.(*Model)
	if got := readLog(t, logPath); len(got) != 0 {
		t.Fatalf("log after cancelled confirm = %#v, want empty", got)
	}
}

func actionSnapshot() Snapshot {
	s := fixtureSnapshot()
	s.State = "ready"
	s.Gate = &Gate{State: "held", Stint: "auth", Message: "held"}
	s.Entries = []Entry{
		{
			Kind:  kindStint,
			ID:    "root-stint",
			Ref:   "auth",
			Title: "auth",
			Stint: &Stint{Name: "auth", Done: 1, Total: 2, Queued: true},
			Laps: []Entry{
				{Kind: kindLap, ID: "auth-1", Title: "Auth todo"},
				{Kind: kindLap, ID: "auth-2", Title: "Auth next"},
			},
		},
		{Kind: kindLap, ID: "root-1", Title: "Root todo"},
		{Kind: kindLap, ID: "root-3", Title: "Root next"},
		{Kind: kindLap, ID: "root-2", Title: "Done lap", IsDone: true},
	}
	return s
}

func writeFakeLaps(t *testing.T, extra string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "args.log")
	script := filepath.Join(dir, "laps")
	body := `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "` + logPath + `"
` + extra + `
case "$*" in
"status --json-output")
cat <<'JSON'
{"state":"held","counts":{"todo":3,"done":1,"total":4},"claim":{"valid":false,"lap":"","file":""},"gate":{"state":"held","stint":"auth","message":"held"},"assignees":[],"activeStint":null,"stints":[{"name":"auth","scope":"auth","file":"stints/auth.laps.json","todo":2,"done":1,"total":3,"queued":true,"archived":false,"active":false},{"name":"draft","scope":"draft","file":"stints/draft.laps.json","todo":1,"done":0,"total":1,"queued":false,"archived":false,"active":false},{"name":"old","scope":"old","file":"stints/archive/old.laps.json","todo":0,"done":1,"total":1,"queued":true,"archived":true,"active":false}]}
JSON
;;
"list --root --all --json-output")
cat <<'JSON'
{"tasks":[{"kind":"stint","id":"root-stint","ref":"auth","title":"auth","isDone":false,"order":1,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","completedAt":null},{"id":"root-1","title":"Root todo","isDone":false,"order":2,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","completedAt":null},{"id":"root-3","title":"Root next","isDone":false,"order":3,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","completedAt":null}]}
JSON
;;
"-f stints/auth.laps list --all --json-output")
cat <<'JSON'
{"tasks":[{"id":"auth-1","title":"Auth todo","isDone":false,"order":1,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","completedAt":null},{"id":"auth-2","title":"Auth done","isDone":true,"order":2,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","completedAt":"2026-01-01T00:00:00Z"}]}
JSON
;;
*)
printf 'ok %s\n' "$1"
;;
esac
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.WriteFile(logPath, nil, 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	return script, logPath
}

func readLog(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	text := strings.TrimSpace(string(b))
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
