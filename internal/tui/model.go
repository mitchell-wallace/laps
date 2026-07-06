package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	runner Runner

	snapshot Snapshot
	err      error
	status   string
	confirm  bool
	loading  bool

	cursor int
	rows   []row
	vp     viewport.Model
	width  int
	height int
}

type row struct {
	text       string
	selectable bool
	entry      *Entry
	stintName  string
	fileArg    string
}

type snapshotMsg struct {
	snapshot Snapshot
	err      error
}

type actionMsg struct {
	line string
	err  error
}

func NewModel(runner Runner) Model {
	return Model{
		runner: runner,
		vp:     viewport.New(80, 22),
		width:  80,
		height: 24,
		status: "loading",
	}
}

func (m Model) Init() tea.Cmd {
	return m.fetchCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = maxInt(1, msg.Width)
		m.height = maxInt(1, msg.Height)
		m.refresh()
		return m, nil
	case snapshotMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.status = msg.err.Error()
		} else {
			m.snapshot = msg.snapshot
			m.err = nil
			if m.status == "loading" {
				m.status = ""
			}
		}
		m.refresh()
		return m, nil
	case actionMsg:
		if msg.line != "" {
			m.status = msg.line
		} else if msg.err != nil {
			m.status = msg.err.Error()
		}
		m.confirm = false
		m.refresh()
		cmd := m.fetchCmd()
		return m, cmd
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) View() string {
	if m.height <= 1 {
		return m.statusBar()
	}
	bodyHeight := maxInt(1, m.height-1)
	vp := m.vp
	vp.Width = maxInt(1, m.width)
	vp.Height = bodyHeight
	return fitBlock(vp.View(), m.width, bodyHeight) + "\n" + m.statusBar()
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.confirm {
		// Swallow every key here: letting one fall through would act on the
		// cursor, and y after a cursor move would delete the wrong lap.
		switch key {
		case "y":
			return *m, m.actionCmd("delete")
		default:
			m.confirm = false
			m.status = "delete cancelled"
			m.refresh()
			return *m, nil
		}
	}
	switch key {
	case "ctrl+c", "q":
		return *m, tea.Quit
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "pgup":
		m.pageCursor(-1)
	case "pgdown":
		m.pageCursor(1)
	case "r":
		cmd := m.fetchCmd()
		return *m, cmd
	case "d":
		return *m, m.actionCmd("done")
	case "x":
		if selected := m.selectedRow(); selected != nil && selected.entry != nil && selected.entry.Kind != kindStint {
			m.confirm = true
			m.status = fmt.Sprintf("delete %s? y/n", selected.entry.ID)
			m.refresh()
		}
	case "K":
		return *m, m.moveAction(-1)
	case "J":
		return *m, m.moveAction(1)
	case "h":
		return *m, m.holdAction()
	}
	m.refresh()
	return *m, nil
}

func (m *Model) fetchCmd() tea.Cmd {
	if m.loading {
		return nil
	}
	m.loading = true
	runner := m.runner
	return func() tea.Msg {
		snapshot, err := runner.Load(context.Background())
		return snapshotMsg{snapshot: snapshot, err: err}
	}
}

func (m *Model) actionCmd(action string) tea.Cmd {
	selected := m.selectedRow()
	if selected == nil || selected.entry == nil || selected.entry.Kind == kindStint {
		return nil
	}
	args := selected.actionPrefix()
	args = append(args, action, selected.entry.ID)
	return m.runAction(args)
}

func (m *Model) moveAction(delta int) tea.Cmd {
	selected := m.selectedRow()
	if selected == nil || selected.entry == nil || selected.entry.Kind == kindStint || selected.entry.IsDone {
		return nil
	}
	todos := m.todoRowsForFile(selected.fileArg)
	index := -1
	for i := range todos {
		if todos[i].entry != nil && todos[i].entry.ID == selected.entry.ID {
			index = i
			break
		}
	}
	if index < 0 {
		return nil
	}
	args := selected.actionPrefix()
	switch {
	case delta < 0 && index == 0:
		return nil
	case delta < 0 && index == 1:
		args = append(args, "move", selected.entry.ID, "head")
	case delta < 0:
		args = append(args, "move", selected.entry.ID, "after", todos[index-2].entry.ID)
	case delta > 0 && index >= len(todos)-1:
		return nil
	case delta > 0:
		args = append(args, "move", selected.entry.ID, "after", todos[index+1].entry.ID)
	default:
		return nil
	}
	return m.runAction(args)
}

func (m *Model) holdAction() tea.Cmd {
	selected := m.selectedRow()
	if selected == nil || selected.entry == nil || selected.entry.Kind != kindStint {
		return nil
	}
	name := selected.stintName
	if name == "" {
		return nil
	}
	action := "hold"
	if m.isHeld(name) {
		action = "release"
	}
	return m.runAction([]string{"stints", action, name})
}

func (m *Model) runAction(args []string) tea.Cmd {
	runner := m.runner
	return func() tea.Msg {
		line, err := runner.Action(context.Background(), args...)
		return actionMsg{line: line, err: err}
	}
}

func (r row) actionPrefix() []string {
	if r.fileArg == "" {
		return nil
	}
	return []string{"-f", r.fileArg}
}

func (m *Model) refresh() {
	m.rows = m.buildRows()
	m.ensureCursor()
	lines := make([]string, len(m.rows))
	for i := range m.rows {
		text := fitLine(m.rows[i].text, m.width)
		if i == m.cursor && m.rows[i].selectable {
			text = selectedStyle.Width(maxInt(1, m.width)).Render(text)
		}
		lines[i] = text
	}
	m.vp.Width = maxInt(1, m.width)
	m.vp.Height = maxInt(1, m.height-1)
	m.vp.SetContent(strings.Join(lines, "\n"))
	m.vp.SetYOffset(m.cursor)
}

func (m Model) buildRows() []row {
	width := maxInt(1, m.width)
	if m.err != nil {
		return []row{{text: fitLine("laps unavailable: "+m.err.Error(), width)}}
	}
	if m.snapshot.Missing {
		return []row{{text: fitLine("no laps workspace", width)}}
	}
	switch m.snapshot.State {
	case "empty":
		return []row{{text: fitLine("queue empty", width)}}
	case "complete":
		return []row{{text: fitLine("queue complete", width)}}
	}
	if len(m.snapshot.Entries) == 0 && m.snapshot.Counts.Total == 0 {
		return []row{{text: fitLine("queue empty", width)}}
	}
	rows := []row{{text: summaryLine(m.snapshot)}}
	for i := range m.snapshot.Entries {
		rows = appendEntryRows(rows, &m.snapshot.Entries[i], m.snapshot.Gate, "")
	}
	return rows
}

func appendEntryRows(rows []row, entry *Entry, gate *Gate, fileArg string) []row {
	switch entry.Kind {
	case kindStint:
		name := entry.Ref
		if entry.Stint != nil && entry.Stint.Name != "" {
			name = entry.Stint.Name
		}
		if name == "" {
			name = entry.Title
		}
		rows = append(rows, row{text: stintHeader(entry, gate, 0), selectable: true, entry: entry, stintName: name})
		if gate != nil && gate.Stint == name && gate.Message != "" {
			rows = append(rows, row{text: "  ⛔ " + gate.Message})
		}
		for i := range entry.Laps {
			rows = appendLapRows(rows, &entry.Laps[i], "  ", stintFileArg(name))
		}
	default:
		rows = appendLapRows(rows, entry, "", fileArg)
	}
	return rows
}

func appendLapRows(rows []row, entry *Entry, indent, fileArg string) []row {
	rows = append(rows, row{text: indent + lapLine(entry), selectable: true, entry: entry, fileArg: fileArg})
	return rows
}

func (m *Model) ensureCursor() {
	if len(m.rows) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.rows[m.cursor].selectable {
		return
	}
	for i := m.cursor; i < len(m.rows); i++ {
		if m.rows[i].selectable {
			m.cursor = i
			return
		}
	}
	for i := m.cursor; i >= 0; i-- {
		if m.rows[i].selectable {
			m.cursor = i
			return
		}
	}
}

func (m *Model) moveCursor(delta int) {
	if len(m.rows) == 0 {
		return
	}
	for i := m.cursor + delta; i >= 0 && i < len(m.rows); i += delta {
		if m.rows[i].selectable {
			m.cursor = i
			return
		}
	}
}

func (m *Model) pageCursor(delta int) {
	step := maxInt(1, m.vp.Height)
	if delta < 0 {
		step = -step
	}
	target := m.cursor + step
	if target < 0 {
		target = 0
	}
	if target >= len(m.rows) {
		target = len(m.rows) - 1
	}
	direction := 1
	if target < m.cursor {
		direction = -1
	}
	for i := target; i >= 0 && i < len(m.rows); i += direction {
		if m.rows[i].selectable {
			m.cursor = i
			return
		}
	}
}

func (m Model) selectedRow() *row {
	if m.cursor < 0 || m.cursor >= len(m.rows) || !m.rows[m.cursor].selectable {
		return nil
	}
	return &m.rows[m.cursor]
}

func (m Model) todoRowsForFile(fileArg string) []row {
	var rows []row
	for _, r := range m.rows {
		if !r.selectable || r.entry == nil || r.entry.Kind == kindStint || r.entry.IsDone || r.fileArg != fileArg {
			continue
		}
		rows = append(rows, r)
	}
	return rows
}

func (m Model) isHeld(name string) bool {
	return m.snapshot.Gate != nil && m.snapshot.Gate.Stint == name
}

func summaryLine(snapshot Snapshot) string {
	parts := []string{
		"state " + valueOr(snapshot.State, "unknown"),
		fmt.Sprintf("todo %d", snapshot.Counts.Todo),
		fmt.Sprintf("done %d/%d", snapshot.Counts.Done, snapshot.Counts.Total),
	}
	if snapshot.Claim.Valid {
		age := int64(0)
		if snapshot.Claim.AgeSeconds != nil {
			age = *snapshot.Claim.AgeSeconds
		}
		parts = append(parts, fmt.Sprintf("claim %s age %s", snapshot.Claim.Lap, formatAge(age)))
	}
	return strings.Join(parts, " | ")
}

func stintHeader(entry *Entry, gate *Gate, indent int) string {
	name := valueOr(entry.Ref, entry.Title)
	done, total := 0, 0
	if entry.Stint != nil {
		name = valueOr(entry.Stint.Name, name)
		done = entry.Stint.Done
		total = entry.Stint.Total
	}
	marker := ""
	if gate != nil && gate.Stint == name {
		marker = " ⛔ held"
	}
	return fmt.Sprintf("%s%s/ (stint %d/%d)%s", strings.Repeat(" ", indent), name, done, total, marker)
}

func lapLine(entry *Entry) string {
	glyph := "·"
	if entry.IsDone {
		glyph = "✓"
	}
	assignee := ""
	if entry.Assignee != "" {
		assignee = " (" + entry.Assignee + ")"
	}
	title := entry.Title
	if title == "" {
		title = entry.ID
	}
	return fmt.Sprintf("%s %s %s%s", glyph, entry.ID, title, assignee)
}

func formatAge(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dh%02dm", minutes/60, minutes%60)
}

func (m Model) statusBar() string {
	width := maxInt(1, m.width)
	left := "laps tui"
	middle := m.status
	if m.loading {
		middle = valueOr(middle, "loading")
	}
	right := "j/k move · d done · x delete · K/J reorder · h hold · r refresh · q quit"
	return statusStyle.Width(width).Render(composeStatus(width, left, middle, right))
}

func Run(runner Runner) error {
	if runner.Binary == "" {
		binary, err := os.Executable()
		if err != nil {
			return err
		}
		runner.Binary = binary
	}
	_, err := tea.NewProgram(NewModel(runner), tea.WithAltScreen()).Run()
	return err
}

var (
	statusStyle   = lipgloss.NewStyle().Reverse(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
)

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
