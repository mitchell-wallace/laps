// Package eventlog implements the append-only, best-effort event log written
// to .laps/log.jsonl by every mutating laps command.
//
// The log is NATIVE (built into the commands, not a configurable hook) and
// BEST-EFFORT: a write failure is reported as a one-line warning on stderr and
// MUST NEVER be returned to the caller or change a command's exit code.
package eventlog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LogFileName is the on-disk event-log file, relative to the .laps directory.
const LogFileName = "log.jsonl"

// sessionEnvVar names the environment variable laps stamps into every line so a
// cluster of events maps to a single orchestrator run/try. Empty when unset.
const sessionEnvVar = "LAPS_SESSION"

// defaultScope is the scope value when no stint context applies. It is present
// from day one so stint population (a later change) is purely additive.
const defaultScope = "root"

// stderr is the sink for best-effort failure warnings. It is a package variable
// so tests can capture the warning without touching the real os.Stderr.
var stderr io.Writer = os.Stderr

// Timestamp wraps time.Time so it always serialises as an RFC3339 UTC string,
// the format required by the event-log schema. (Unmarshalling is the reader's
// concern and is intentionally not implemented here.)
type Timestamp time.Time

// MarshalJSON renders the timestamp as an RFC3339 UTC string.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).UTC().Format(time.RFC3339))
}

// Line is a single event-log entry. It is the full, on-disk schema: the writer
// stamps TS, Scope, and Session itself. Optional fields (Lap, Title, Assignee)
// are omitted from the JSON when empty. Detail is always present, defaulting to
// an empty object.
type Line struct {
	TS       Timestamp              `json:"ts"`
	Event    string                 `json:"event"`
	Cmd      string                 `json:"cmd"`
	File     string                 `json:"file"`
	Lap      string                 `json:"lap,omitempty"`
	Title    string                 `json:"title,omitempty"`
	Assignee string                 `json:"assignee,omitempty"`
	Scope    string                 `json:"scope"`
	Detail   map[string]interface{} `json:"detail"`
	Session  string                 `json:"session"`
}

// Entry is the input a mutating command provides for one log line. The writer
// stamps TS (now, UTC), Session (from LAPS_SESSION), and defaults an empty Scope
// to "root"; callers therefore never set those fields.
type Entry struct {
	Event    string
	Cmd      string
	File     string
	Lap      string
	Title    string
	Assignee string
	Scope    string
	Detail   map[string]interface{}
}

// Append writes one event line to .laps/log.jsonl under beadsDir. It is
// best-effort: a failure is reported as a one-line warning on stderr and is
// never returned to the caller or reflected in an exit code.
func Append(beadsDir string, e *Entry) {
	if err := write(beadsDir, e); err != nil {
		_, _ = fmt.Fprintf(stderr, "laps: event log: %v\n", err)
	}
}

// write builds, marshals, and appends a single line. It returns any error so
// Append can report it on stderr without surfacing it to the caller.
func write(beadsDir string, e *Entry) error {
	path := filepath.Join(beadsDir, LogFileName)
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", beadsDir, err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		_ = f.Close()
	}()

	b, err := json.Marshal(buildLine(e))
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	b = append(b, '\n')
	if _, err := f.Write(b); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// buildLine turns caller input into a full schema Line, stamping TS and Session
// and defaulting Scope/Detail.
func buildLine(e *Entry) Line {
	scope := e.Scope
	if scope == "" {
		scope = defaultScope
	}
	detail := e.Detail
	if detail == nil {
		detail = map[string]interface{}{}
	}
	return Line{
		TS:       Timestamp(time.Now().UTC()),
		Event:    e.Event,
		Cmd:      e.Cmd,
		File:     e.File,
		Lap:      e.Lap,
		Title:    e.Title,
		Assignee: e.Assignee,
		Scope:    scope,
		Detail:   detail,
		Session:  os.Getenv(sessionEnvVar),
	}
}
