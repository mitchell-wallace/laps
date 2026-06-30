package cmd

import (
	"path/filepath"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
)

// logEvent appends one best-effort event-log line for an applied state change.
// When the caller leaves File empty it is stamped to the resolved .laps-relative
// task file selected by --file (defaulting to laps.json). A caller that already
// knows the affected file — e.g. clearing a claim that belongs to a DIFFERENT
// file than the one currently selected — sets Entry.File itself and it is
// preserved here. By the event-log contract this never returns an error and
// never influences a command's exit code: callers MUST only invoke it after the
// underlying store.Save (or claim write) has already succeeded.
func logEvent(beadsDir string, e *eventlog.Entry) {
	if e.File == "" {
		e.File = store.ResolveFile(fileFlag)
	}
	eventlog.Append(beadsDir, e)
}

// lapMeta best-effort resolves the title and assignee of lap within the given
// .laps-relative task file under beadsDir. It is used to denormalize event-log
// metadata for a lap that may live in a different file than the one currently
// selected (e.g. a claim being retired by a cross-file replacement or undo).
// A missing/unreadable file or an absent lap yields empty strings so the event
// still records whatever identity is available.
func lapMeta(beadsDir, file, lap string) (title, assignee string) {
	if file == "" || lap == "" {
		return "", ""
	}
	f, err := store.Load(filepath.Join(beadsDir, file))
	if err != nil {
		return "", ""
	}
	for i := range f.Tasks {
		if f.Tasks[i].ID == lap {
			return f.Tasks[i].Title, f.Tasks[i].Assignee
		}
	}
	return "", ""
}
