package instructions

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mitchell-wallace/laps/internal/queuecontract"
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
	if !strings.Contains(string(b), `<laps-instructions v="2">`) {
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
	if !strings.Contains(content, `<laps-instructions v="2">`) {
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

	if strings.Count(string(b2), blockStartPrefix) != 1 {
		t.Fatalf("block duplicated: count=%d", strings.Count(string(b2), blockStartPrefix))
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
		if !strings.Contains(string(b), `<laps-instructions v="2">`) {
			t.Errorf("%s missing block", f)
		}
	}
}

func TestEnableRefreshesLegacyBlock(t *testing.T) {
	root := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origWd)

	legacy := "# Before\n\n<laps-instructions>\nold contract\n</laps-instructions>\n\nAfter\n"
	if err := os.WriteFile("AGENTS.md", []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Enable(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if strings.Count(content, blockStartPrefix) != 1 || !strings.Contains(content, `<laps-instructions v="2">`) {
		t.Fatalf("legacy block was not replaced by one v2 block:\n%s", content)
	}
	if strings.Contains(content, "old contract") || !strings.Contains(content, "# Before") || !strings.Contains(content, "After") {
		t.Fatalf("refresh did not preserve surrounding content:\n%s", content)
	}
}

func TestDisableVersionedBlock(t *testing.T) {
	root := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origWd)

	content := "# A\n\n" + blockContent + "\n\nfooter\n"
	if err := os.WriteFile("AGENTS.md", []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Disable(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), blockStartPrefix) || !strings.Contains(string(b), "footer") {
		t.Fatalf("versioned block removal failed:\n%s", b)
	}
}

func TestBlockContentMatchesQueueContract(t *testing.T) {
	required := []string{
		"laps claim",
		"laps done",
		"claimed lap",
		"not the current head",
		"finish an existing claimed lap",
		"start nothing new",
		"Stint resolution is transparent",
		"--active",
		"--root",
		"--stint <name>",
		"-f/--file",
	}
	for _, phrase := range required {
		if !strings.Contains(blockContent, phrase) {
			t.Errorf("v2 block missing %q", phrase)
		}
	}
	for code, action := range map[int]string{
		queuecontract.ExitRun:      "run",
		queuecontract.ExitHeld:     "stop-held",
		queuecontract.ExitEmpty:    "idle",
		queuecontract.ExitComplete: "finished",
	} {
		if !strings.Contains(blockContent, fmt.Sprintf("`%d`", code)) || !strings.Contains(blockContent, action) {
			t.Errorf("v2 block missing exit %d action %q", code, action)
		}
	}
}

func TestBlockContentStaysWithinContextBudget(t *testing.T) {
	const legacyBlock = `<laps-instructions>
This project uses laps (` + "`laps`" + `), a minimal task tracker.
- ` + "`laps get head`" + ` — read the next task. Title and description only.
- ` + "`laps list`" + ` — see the queue.
- ` + "`laps done`" + ` — when you finish the head task. You MUST run this; do not skip.
- ` + "`laps add head|tail|after <id> --title ...`" + ` — add a task. Use ` + "`head`" + ` if it must be done before the current head; otherwise ` + "`tail`" + `.
- If you hit a blocker that prevents finishing the head task this session, add the unblock work to ` + "`head`" + ` and stop.
- Commit after each ` + "`laps done`" + ` unless the user said otherwise.
</laps-instructions>`
	legacyWords := len(strings.Fields(legacyBlock))
	v2Words := len(strings.Fields(blockContent))
	if v2Words*2 < legacyWords || v2Words*2 > legacyWords*3 {
		t.Fatalf("v2 block word count %d is outside ±50%% of legacy count %d", v2Words, legacyWords)
	}
	t.Logf("instruction block word counts: legacy=%d v2=%d", legacyWords, v2Words)
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
