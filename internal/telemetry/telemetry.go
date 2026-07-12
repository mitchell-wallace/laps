// Package telemetry provides best-effort Laps usage telemetry.
package telemetry

import (
	"os"
	"strings"
	"time"
	"unicode/utf8"

	nr "github.com/newrelic/go-agent/v3/newrelic"
)

const (
	EventLapsClaim      = "LapsClaim"
	EventLapsComplete   = "LapsComplete"
	EventLapsDiagnostic = "LapsDiagnostic"

	envLicenseKey = "LAPS_NEW_RELIC_LICENSE_KEY"
	envAppName    = "LAPS_NEW_RELIC_APP_NAME"

	defaultAppName = "Laps CLI"
	flushTimeout   = 2 * time.Second
)

// Config contains fallback values used when the Laps-prefixed environment
// variables are unset. Environment values take precedence.
type Config struct {
	LicenseKey      string
	AppName         string
	ShutdownTimeout time.Duration
}

// ClaimEvent describes an attempt to claim work from the selected queue.
type ClaimEvent struct {
	Outcome    string
	LapID      string
	Scope      string
	QueueDepth int
	Explicit   bool
	Reclaim    bool
}

// CompleteEvent describes a successful queue completion operation.
type CompleteEvent struct {
	Operation     string
	LapID         string
	Scope         string
	QueueDepth    int
	Claimed       bool
	Duration      time.Duration
	DurationKnown bool
}

// DiagnosticEvent describes a queue or claim persistence warning.
type DiagnosticEvent struct {
	Command string
	Source  string
	Kind    string
	Message string
}

// Sink records Laps custom events. Implementations must be safe to call even
// when telemetry is disabled.
type Sink interface {
	RecordClaim(ClaimEvent)
	RecordComplete(CompleteEvent)
	RecordDiagnostic(DiagnosticEvent)
}

type noopSink struct{}

func (noopSink) RecordClaim(ClaimEvent)           {}
func (noopSink) RecordComplete(CompleteEvent)     {}
func (noopSink) RecordDiagnostic(DiagnosticEvent) {}

type newRelicSink struct {
	app *nr.Application
}

// Init conditionally initializes New Relic. With no license key it returns a
// no-op sink without constructing an agent. Agent errors are deliberately
// swallowed so observability can never prevent Laps from running.
func Init(cfg Config) (Sink, func()) {
	license, appName := resolveConfig(cfg)
	if license == "" {
		return noopSink{}, func() {}
	}

	app, err := nr.NewApplication(
		nr.ConfigLicense(license),
		nr.ConfigAppName(appName),
		nr.ConfigEnabled(true),
	)
	if err != nil {
		return noopSink{}, func() {}
	}

	timeout := cfg.ShutdownTimeout
	if timeout <= 0 {
		timeout = flushTimeout
	}
	sink := &newRelicSink{app: app}
	return sink, func() { app.Shutdown(timeout) }
}

func resolveConfig(cfg Config) (license string, appName string) {
	license = strings.TrimSpace(os.Getenv(envLicenseKey))
	if license == "" {
		license = strings.TrimSpace(cfg.LicenseKey)
	}
	appName = strings.TrimSpace(os.Getenv(envAppName))
	if appName == "" {
		appName = strings.TrimSpace(cfg.AppName)
	}
	if appName == "" {
		appName = defaultAppName
	}
	return license, appName
}

func (s *newRelicSink) RecordClaim(event ClaimEvent) {
	if s == nil || s.app == nil {
		return
	}
	s.app.RecordCustomEvent(EventLapsClaim, map[string]interface{}{
		"outcome":     cleanAttribute(event.Outcome),
		"lap_id":      cleanAttribute(event.LapID),
		"scope":       cleanAttribute(event.Scope),
		"queue_depth": event.QueueDepth,
		"explicit":    event.Explicit,
		"reclaim":     event.Reclaim,
	})
}

func (s *newRelicSink) RecordComplete(event CompleteEvent) {
	if s == nil || s.app == nil {
		return
	}
	s.app.RecordCustomEvent(EventLapsComplete, map[string]interface{}{
		"operation":      cleanAttribute(event.Operation),
		"lap_id":         cleanAttribute(event.LapID),
		"scope":          cleanAttribute(event.Scope),
		"queue_depth":    event.QueueDepth,
		"claimed":        event.Claimed,
		"duration_ms":    event.Duration.Milliseconds(),
		"duration_known": event.DurationKnown,
	})
}

func (s *newRelicSink) RecordDiagnostic(event DiagnosticEvent) {
	if s == nil || s.app == nil {
		return
	}
	s.app.RecordCustomEvent(EventLapsDiagnostic, map[string]interface{}{
		"command": cleanAttribute(event.Command),
		"source":  cleanAttribute(event.Source),
		"kind":    cleanAttribute(event.Kind),
		"message": cleanAttribute(event.Message),
	})
}

func cleanAttribute(value string) string {
	value = strings.ToValidUTF8(strings.TrimSpace(value), "�")
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		value = strings.ReplaceAll(value, home, "~")
	}
	const maxBytes = 1024
	if len(value) > maxBytes {
		value = value[:maxBytes]
		for !utf8.ValidString(value) {
			value = value[:len(value)-1]
		}
	}
	return value
}
