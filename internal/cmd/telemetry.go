package cmd

import (
	"errors"
	"strings"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/mitchell-wallace/laps/internal/telemetry"
)

var telemetrySink telemetry.Sink
var telemetryCommandName string

func queueDepth(file *store.File) int {
	if file == nil {
		return 0
	}
	depth := 0
	for i := range file.Tasks {
		if !file.Tasks[i].IsDone {
			depth++
		}
	}
	return depth
}

func readClaim(beadsDir, selectedFile string) (store.Claim, error) {
	claim, err := store.ReadClaim(beadsDir, selectedFile)
	if errors.Is(err, store.ErrMalformedClaim) {
		recordDiagnostic("claim_file", "corruption", err)
	}
	return claim, err
}

func recordQueueDiagnostic(err error) {
	if err == nil || !strings.Contains(err.Error(), "not a valid laps task file") {
		return
	}
	recordDiagnostic("queue_file", "corruption", err)
}

func recordDiagnostic(source, kind string, err error) {
	if telemetrySink == nil || err == nil {
		return
	}
	telemetrySink.RecordDiagnostic(telemetry.DiagnosticEvent{
		Command: telemetryCommandName,
		Source:  source,
		Kind:    kind,
		Message: err.Error(),
	})
}
