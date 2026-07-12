package telemetry

import (
	"strings"
	"testing"
	"time"
)

func TestInitWithoutLicenseReturnsNoop(t *testing.T) {
	t.Setenv(envLicenseKey, "")
	t.Setenv(envAppName, "")
	sink, cleanup := Init(Config{})
	defer cleanup()
	if _, ok := sink.(noopSink); !ok {
		t.Fatalf("sink = %T, want noopSink", sink)
	}
	sink.RecordClaim(ClaimEvent{})
	sink.RecordComplete(CompleteEvent{})
	sink.RecordDiagnostic(DiagnosticEvent{})
}

func TestInitWithLicenseConstructsAgent(t *testing.T) {
	t.Setenv(envLicenseKey, strings.Repeat("1", 40))
	t.Setenv(envAppName, "Laps Test")
	sink, cleanup := Init(Config{ShutdownTimeout: time.Millisecond})
	defer cleanup()
	if _, ok := sink.(*newRelicSink); !ok {
		t.Fatalf("sink = %T, want *newRelicSink", sink)
	}
}

func TestEnvironmentOverridesConfig(t *testing.T) {
	t.Setenv(envLicenseKey, "env-license")
	t.Setenv(envAppName, "Env App")
	license, appName := resolveConfig(Config{LicenseKey: "config-license", AppName: "Config App"})
	if license != "env-license" || appName != "Env App" {
		t.Fatalf("resolveConfig = (%q, %q)", license, appName)
	}
}

func TestConfigFallbackAndDefaultAppName(t *testing.T) {
	t.Setenv(envLicenseKey, "")
	t.Setenv(envAppName, "")
	license, appName := resolveConfig(Config{LicenseKey: "config-license"})
	if license != "config-license" || appName != defaultAppName {
		t.Fatalf("resolveConfig = (%q, %q)", license, appName)
	}
}
