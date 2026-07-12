package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/mitchell-wallace/laps/internal/telemetry"
)

type recordingTelemetrySink struct {
	claims      []telemetry.ClaimEvent
	completions []telemetry.CompleteEvent
	diagnostics []telemetry.DiagnosticEvent
}

func (s *recordingTelemetrySink) RecordClaim(event telemetry.ClaimEvent) {
	s.claims = append(s.claims, event)
}

func (s *recordingTelemetrySink) RecordComplete(event telemetry.CompleteEvent) {
	s.completions = append(s.completions, event)
}

func (s *recordingTelemetrySink) RecordDiagnostic(event telemetry.DiagnosticEvent) {
	s.diagnostics = append(s.diagnostics, event)
}

func TestClaimAndDoneRecordTelemetry(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	task := store.Task{
		ID:        "laps-fixed",
		Title:     "Telemetry task",
		Order:     1,
		CreatedAt: resolverTestTime,
		UpdatedAt: resolverTestTime,
	}
	if err := store.Save(filepath.Join(beadsDir, "laps.json"), &store.File{
		Version: store.CurrentVersion,
		Tasks:   []store.Task{task},
	}); err != nil {
		t.Fatal(err)
	}

	sink := &recordingTelemetrySink{}
	previous := telemetrySink
	telemetrySink = sink
	defer func() { telemetrySink = previous }()

	if _, stderr, code := runMB("claim"); code != 0 {
		t.Fatalf("claim: code=%d stderr=%q", code, stderr)
	}
	if _, stderr, code := runMB("done"); code != 0 {
		t.Fatalf("done: code=%d stderr=%q", code, stderr)
	}

	wantClaim := telemetry.ClaimEvent{
		Outcome:    "success",
		LapID:      task.ID,
		Scope:      "root",
		QueueDepth: 1,
	}
	if !reflect.DeepEqual(sink.claims, []telemetry.ClaimEvent{wantClaim}) {
		t.Fatalf("claim events = %#v, want %#v", sink.claims, wantClaim)
	}
	if len(sink.completions) != 1 {
		t.Fatalf("complete events = %#v", sink.completions)
	}
	complete := sink.completions[0]
	if complete.Operation != "done" || complete.LapID != task.ID || complete.Scope != "root" || complete.QueueDepth != 0 || !complete.Claimed || !complete.DurationKnown || complete.Duration < 0 {
		t.Fatalf("complete event = %#v", complete)
	}
}

func TestMalformedPersistenceRecordsDiagnostics(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()

	sink := &recordingTelemetrySink{}
	previous := telemetrySink
	telemetrySink = sink
	defer func() { telemetrySink = previous }()

	if err := os.WriteFile(filepath.Join(beadsDir, "laps.json"), []byte("{\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, code := runMB("list"); code != 2 {
		t.Fatalf("corrupt queue exit code = %d, want 2", code)
	}
	if len(sink.diagnostics) != 1 || sink.diagnostics[0].Source != "queue_file" || sink.diagnostics[0].Kind != "corruption" {
		t.Fatalf("queue diagnostics = %#v", sink.diagnostics)
	}

	if err := store.Save(filepath.Join(beadsDir, "laps.json"), &store.File{Version: store.CurrentVersion, Tasks: []store.Task{}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.ClaimPath(beadsDir), []byte("{bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, code := runMB("status"); code != 2 {
		t.Fatalf("malformed claim exit code = %d, want 2", code)
	}
	if len(sink.diagnostics) != 2 || sink.diagnostics[1].Source != "claim_file" || sink.diagnostics[1].Kind != "corruption" {
		t.Fatalf("claim diagnostics = %#v", sink.diagnostics)
	}
}

func TestClaimDoneAndHandoffSequenceUnchangedWithConfiguredLicense(t *testing.T) {
	type result struct {
		stdout string
		stderr string
		code   int
	}

	run := func(t *testing.T, scenario, license string) result {
		beadsDir, cleanup := setupTempRepo(t)
		defer cleanup()

		task := store.Task{
			ID:        "laps-fixed",
			Title:     "Parity task",
			Order:     1,
			CreatedAt: resolverTestTime,
			UpdatedAt: resolverTestTime,
		}
		if err := store.Save(filepath.Join(beadsDir, "laps.json"), &store.File{
			Version: store.CurrentVersion,
			Tasks:   []store.Task{task},
		}); err != nil {
			t.Fatal(err)
		}
		if scenario == "done" {
			claimedAt := resolverTestTime
			if err := store.WriteClaim(beadsDir, store.Claim{Lap: task.ID, File: "laps.json", Scope: "root", ClaimedAt: &claimedAt}); err != nil {
				t.Fatal(err)
			}
		}

		t.Setenv("LAPS_NEW_RELIC_LICENSE_KEY", license)
		t.Setenv("LAPS_NEW_RELIC_APP_NAME", "Laps Test")
		sink, sinkCleanup := telemetry.Init(telemetry.Config{ShutdownTimeout: time.Millisecond})
		previous := telemetrySink
		telemetrySink = sink
		defer func() {
			telemetrySink = previous
			sinkCleanup()
		}()

		switch scenario {
		case "claim":
			out, errOut, code := runMB("claim")
			return result{out, errOut, code}
		case "done":
			out, errOut, code := runMB("done")
			return result{out, errOut, code}
		case "handoff-sequence":
			claimOut, claimErr, claimCode := runMB("claim")
			doneOut, doneErr, doneCode := runMB("done")
			if claimCode != 0 {
				return result{claimOut, claimErr, claimCode}
			}
			return result{claimOut + doneOut, claimErr + doneErr, doneCode}
		default:
			t.Fatalf("unknown scenario %q", scenario)
			return result{}
		}
	}

	for _, scenario := range []string{"claim", "done", "handoff-sequence"} {
		t.Run(scenario, func(t *testing.T) {
			without := run(t, scenario, "")
			with := run(t, scenario, strings.Repeat("1", 40))
			if !reflect.DeepEqual(without, with) {
				t.Fatalf("behavior differs without/with license:\nwithout=%#v\nwith=%#v", without, with)
			}
		})
	}
}
