package instructions

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mitchell-wallace/laps/internal/store"
)

const blockStartPrefix = "<laps-instructions"
const blockEnd = "</laps-instructions>"

const blockContent = `<laps-instructions v="2">
This project uses ` + "`laps`" + ` for its agent work queue.
1. ` + "`laps claim`" + ` — claim the next lap before starting work.
2. Work only the claimed lap.
3. ` + "`laps done`" + ` — complete the claimed lap. A bare ` + "`done`" + ` uses the claim, not the current head. You MUST run it when finished.

Head ` + "`get`" + `/` + "`claim`" + ` exit codes:
- ` + "`0`" + ` — run: a lap was returned/claimed.
- ` + "`10`" + ` — stop-held: finish an existing claimed lap, but start nothing new; then stop.
- ` + "`11`" + ` — idle: no laps are ready; stop.
- ` + "`12`" + ` — finished: all laps are complete; stop.

Stint resolution is transparent. Prefer ` + "`--active`" + `, ` + "`--root`" + `, or ` + "`--stint <name>`" + ` over raw ` + "`-f/--file`" + ` targeting. Use ` + "`laps status`" + ` or ` + "`laps list`" + ` to inspect the queue.
</laps-instructions>`

var targetFiles = []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"}

// Enable writes the laps-instructions block to AGENTS.md (creating if absent),
// and to CLAUDE.md / GEMINI.md only if they already exist.
func Enable() error {
	for _, name := range targetFiles {
		exists := fileExists(name)
		if name != "AGENTS.md" && !exists {
			continue
		}
		if err := writeBlock(name); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

// Disable removes the laps-instructions block from any target file that contains it.
func Disable() error {
	for _, name := range targetFiles {
		if !fileExists(name) {
			continue
		}
		if err := removeBlock(name); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func writeBlock(path string) error {
	content := ""
	if fileExists(path) {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content = string(b)
	}

	newContent := replaceBlock(content, blockContent)
	return store.SafeWriteFile(path, []byte(newContent), 0o644)
}

func removeBlock(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	newContent := replaceBlock(string(b), "")
	return store.SafeWriteFile(path, []byte(newContent), 0o644)
}

func replaceBlock(content, block string) string {
	startIdx := findBlockStart(content)
	if startIdx == -1 {
		if block == "" {
			return content
		}
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + block + "\n"
	}

	endIdx := strings.Index(content[startIdx:], blockEnd)
	if endIdx == -1 {
		// malformed: just append after start
		if block == "" {
			return content[:startIdx]
		}
		return content[:startIdx] + block + "\n"
	}
	endIdx += startIdx + len(blockEnd)

	before := content[:startIdx]
	after := content[endIdx:]

	if block == "" {
		return strings.TrimRight(before, "\n") + "\n" + strings.TrimLeft(after, "\n")
	}

	return before + block + "\n" + strings.TrimLeft(after, "\n")
}

func findBlockStart(content string) int {
	searchFrom := 0
	for {
		rel := strings.Index(content[searchFrom:], blockStartPrefix)
		if rel == -1 {
			return -1
		}
		idx := searchFrom + rel
		next := idx + len(blockStartPrefix)
		if next < len(content) && (content[next] == '>' || content[next] == ' ' || content[next] == '\t') {
			return idx
		}
		searchFrom = next
	}
}
