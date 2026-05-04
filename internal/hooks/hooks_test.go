package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissing(t *testing.T) {
	f, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil File")
	}
	if len(f.Hooks) != 0 {
		t.Fatalf("expected 0 hooks, got %d", len(f.Hooks))
	}
}

func TestLoadValid(t *testing.T) {
	dir := t.TempDir()
	data := `{"version":1,"hooks":[{"title":"Test","command":"done","when":"after","run":"echo hi","passback":true}]}`
	os.WriteFile(filepath.Join(dir, "laps-hooks.json"), []byte(data), 0644)
	f, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.Hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(f.Hooks))
	}
	if f.Hooks[0].Command != "done" {
		t.Errorf("expected command done, got %s", f.Hooks[0].Command)
	}
}

func TestSubstitute(t *testing.T) {
	vars := map[string]string{"id": "abc", "title": "Hello"}
	s := substitute("$id ${title} $id", vars)
	if s != "abc Hello abc" {
		t.Errorf("got %q", s)
	}
}

func TestSubstituteNoPartial(t *testing.T) {
	vars := map[string]string{"id": "abc"}
	s := substitute("$idea", vars)
	if s != "$idea" {
		t.Errorf("partial substitution: got %q", s)
	}
}

func TestDispatchPassback(t *testing.T) {
	f := &File{
		Version: 1,
		Hooks: []Hook{
			{Title: "Echo", Command: "test", When: "before", Run: "echo hello", Passback: true},
		},
	}
	out, err := Dispatch(f, "test", "before", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected hello in output, got %q", out)
	}
}

func TestDispatchNoPassback(t *testing.T) {
	f := &File{
		Version: 1,
		Hooks: []Hook{
			{Title: "Echo", Command: "test", When: "before", Run: "echo hello", Passback: false},
		},
	}
	out, err := Dispatch(f, "test", "before", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestDispatchAbort(t *testing.T) {
	f := &File{
		Version: 1,
		Hooks: []Hook{
			{Title: "Fail", Command: "test", When: "before", Run: "exit 1", Passback: false},
		},
	}
	_, err := Dispatch(f, "test", "before", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDispatchOrdering(t *testing.T) {
	f := &File{
		Version: 1,
		Hooks: []Hook{
			{Title: "A", Command: "test", When: "before", Run: "echo -n A", Passback: true},
			{Title: "B", Command: "test", When: "before", Run: "echo -n B", Passback: true},
		},
	}
	out, err := Dispatch(f, "test", "before", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// sh -c "echo -n A" produces "A" without newline
	if !strings.Contains(out, "A") || !strings.Contains(out, "B") {
		t.Errorf("expected A and B, got %q", out)
	}
}
