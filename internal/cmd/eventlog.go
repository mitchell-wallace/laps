package cmd

import (
	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
)

// logEvent appends one best-effort event-log line for an applied state change.
// The file field is stamped to the resolved .laps-relative task file selected by
// --file (defaulting to laps.json). By the event-log contract this never returns
// an error and never influences a command's exit code: callers MUST only invoke
// it after the underlying store.Save (or claim write) has already succeeded.
func logEvent(beadsDir string, e *eventlog.Entry) {
	e.File = store.ResolveFile(fileFlag)
	eventlog.Append(beadsDir, e)
}
