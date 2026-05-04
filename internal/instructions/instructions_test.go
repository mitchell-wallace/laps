package instructions

import (
	"os"
	"strings"
	"testing"
)

func TestEnableFreshRepo(t *testing.T) {
	root := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origWd)

	if err := Enable(); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	b, err := os.ReadFile("AGENTS.md")
	if err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}
	if !strings.Contains(string(b), "<laps-instructions>") {
		t.Fatal("AGENTS.md missing block")
	}
}

func TestEnableExistingAGENTS(t *testing.T) {
	root := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origWd)

	os.WriteFile("AGENTS.md", []byte("# Existing content\n\nSome text.\n"), 0644)

	if err := Enable(); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	b, _ := os.ReadFile("AGENTS.md")
	content := string(b)
	if !strings.Contains(content, "Existing content") {
		t.Error("original content lost")
	}
	if !strings.Contains(content, "<laps-instructions>") {
		t.Error("block not added")
	}
}

func TestEnableIdempotent(t *testing.T) {
	root := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origWd)

	os.WriteFile("AGENTS.md", []byte("# Existing\n"), 0644)

	if err := Enable(); err != nil {
		t.Fatal(err)
	}
	b1, _ := os.ReadFile("AGENTS.md")

	if err := Enable(); err != nil {
		t.Fatal(err)
	}
	b2, _ := os.ReadFile("AGENTS.md")

	if strings.Count(string(b2), "<laps-instructions>") != 1 {
		t.Fatalf("block duplicated: count=%d", strings.Count(string(b2), "<laps-instructions>"))
	}
	if string(b1) != string(b2) {
		t.Error("content changed on second Enable")
	}
}

func TestEnableAllThree(t *testing.T) {
	root := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origWd)

	os.WriteFile("AGENTS.md", []byte("# A\n"), 0644)
	os.WriteFile("CLAUDE.md", []byte("# C\n"), 0644)
	os.WriteFile("GEMINI.md", []byte("# G\n"), 0644)

	if err := Enable(); err != nil {
		t.Fatal(err)
	}

	for _, f := range []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"} {
		b, _ := os.ReadFile(f)
		if !strings.Contains(string(b), "<laps-instructions>") {
			t.Errorf("%s missing block", f)
		}
	}
}

func TestDisable(t *testing.T) {
	root := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origWd)

	os.WriteFile("AGENTS.md", []byte("# A\n\n<laps-instructions>\nx\n</laps-instructions>\n\nfooter\n"), 0644)
	os.WriteFile("CLAUDE.md", []byte("# C\n\n<laps-instructions>\ny\n</laps-instructions>\n"), 0644)

	if err := Disable(); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile("AGENTS.md")
	content := string(b)
	if strings.Contains(content, "<laps-instructions>") {
		t.Error("AGENTS.md still has block")
	}
	if !strings.Contains(content, "footer") {
		t.Error("AGENTS.md lost footer")
	}

	b, _ = os.ReadFile("CLAUDE.md")
	if strings.Contains(string(b), "<laps-instructions>") {
		t.Error("CLAUDE.md still has block")
	}
}
