package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestModelRendersSummaryStintAndGate(t *testing.T) {
	m := modelWithSnapshot(fixtureSnapshot(), 80, 12)
	view := ansi.Strip(m.View())

	for _, want := range []string{
		"state held | todo 3 | done 1/4 | claim root-1 age 5m",
		"auth/ (stint 1/3) ⛔ held",
		"  ⛔ laps: stint auth is held; do not implement laps in it yet.",
		"  · auth-1 Add login (AUTH)",
		"✓ root-2 Done lap (QA)",
		"laps tui",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

func TestModelRendersEmptyCompleteAndMissingStates(t *testing.T) {
	tests := []struct {
		name     string
		snapshot Snapshot
		err      error
		want     string
	}{
		{name: "empty", snapshot: Snapshot{State: "empty"}, want: "queue empty"},
		{name: "complete", snapshot: Snapshot{State: "complete"}, want: "queue complete"},
		{name: "missing", snapshot: Snapshot{Missing: true}, want: "no laps workspace"},
		{name: "error", err: errString("store missing"), want: "laps unavailable: store missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(Runner{})
			m.width = 50
			m.height = 6
			m.snapshot = tt.snapshot
			m.err = tt.err
			m.refresh()
			if view := ansi.Strip(m.View()); !strings.Contains(view, tt.want) {
				t.Fatalf("View() missing %q:\n%s", tt.want, view)
			}
		})
	}
}

func TestCursorMovementSkipsNonSelectableRowsAndStaysInBounds(t *testing.T) {
	m := modelWithSnapshot(fixtureSnapshot(), 80, 12)
	if m.cursor != 1 {
		t.Fatalf("initial cursor = %d, want first selectable row 1", m.cursor)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if got := selectedID(m); got != "auth-1" {
		t.Fatalf("down selected %q, want auth-1", got)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if got := selectedStint(m); got != "auth" {
		t.Fatalf("up selected stint %q, want auth", got)
	}

	for i := 0; i < 20; i++ {
		m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyUp})
	}
	if m.cursor != 1 {
		t.Fatalf("cursor after repeated up = %d, want 1", m.cursor)
	}

	for i := 0; i < 20; i++ {
		m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	}
	if got := selectedID(m); got != "root-2" {
		t.Fatalf("cursor after repeated down selected %q, want root-2", got)
	}
}

func TestModelLinesFitFortyColumns(t *testing.T) {
	m := modelWithSnapshot(fixtureSnapshot(), 40, 10)
	for _, line := range strings.Split(m.View(), "\n") {
		if width := lipgloss.Width(ansi.Strip(line)); width > 40 {
			t.Fatalf("line width = %d, want <= 40: %q", width, ansi.Strip(line))
		}
	}
}

func TestRefreshCoalescesWhileFetchInFlight(t *testing.T) {
	m := modelWithSnapshot(fixtureSnapshot(), 80, 10)
	next, cmd := m.Update(runeKey("r"))
	if cmd == nil {
		t.Fatalf("first refresh returned nil cmd")
	}
	m = next.(Model)
	next, cmd = m.Update(runeKey("r"))
	if cmd != nil {
		t.Fatalf("second refresh while loading returned cmd")
	}
	m = next.(Model)
	if !m.loading {
		t.Fatalf("loading = false, want true")
	}
}

func modelWithSnapshot(snapshot Snapshot, width, height int) Model {
	m := NewModel(Runner{})
	m.width = width
	m.height = height
	m.snapshot = snapshot
	m.status = ""
	m.refresh()
	return m
}

func updateModel(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	typed, ok := next.(Model)
	if !ok {
		t.Fatalf("model type = %T, want tui.Model", next)
	}
	return typed
}

func selectedID(m Model) string {
	row := m.selectedRow()
	if row == nil || row.entry == nil {
		return ""
	}
	return row.entry.ID
}

func selectedStint(m Model) string {
	row := m.selectedRow()
	if row == nil {
		return ""
	}
	return row.stintName
}

func fixtureSnapshot() Snapshot {
	age := int64(300)
	return Snapshot{
		State:  "held",
		Counts: Counts{Todo: 3, Done: 1, Total: 4},
		Claim:  Claim{Valid: true, Lap: "root-1", AgeSeconds: &age},
		Gate: &Gate{
			State:   "held",
			Stint:   "auth",
			Message: "laps: stint auth is held; do not implement laps in it yet.",
		},
		Entries: []Entry{
			{
				Kind:  kindStint,
				ID:    "root-stint",
				Ref:   "auth",
				Title: "auth",
				Stint: &Stint{Name: "auth", Done: 1, Total: 3, Queued: true},
				Laps: []Entry{
					{Kind: kindLap, ID: "auth-1", Title: "Add login", Assignee: "AUTH"},
					{Kind: kindLap, ID: "auth-2", Title: "Old auth", Assignee: "QA", IsDone: true},
				},
			},
			{Kind: kindLap, ID: "root-1", Title: "Root todo", Assignee: "DEV"},
			{Kind: kindLap, ID: "root-2", Title: "Done lap", Assignee: "QA", IsDone: true},
		},
	}
}

type errString string

func (e errString) Error() string { return string(e) }
