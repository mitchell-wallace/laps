package instructions

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mitchell-wallace/laps/internal/store"
)

const blockStart = "<laps-instructions>"
const blockEnd = "</laps-instructions>"

const blockContent = `<laps-instructions>
This project uses laps (` + "`laps`" + `), a minimal task tracker.
- ` + "`laps get head`" + ` — read the next task. Title and description only.
- ` + "`laps list`" + ` — see the queue.
- ` + "`laps done`" + ` — when you finish the head task. You MUST run this; do not skip.
- ` + "`laps add head|tail|after <id> --title ...`" + ` — add a task. Use ` + "`head`" + ` if it must be done before the current head; otherwise ` + "`tail`" + `.
- If you hit a blocker that prevents finishing the head task this session, add the unblock work to ` + "`head`" + ` and stop.
- Commit after each ` + "`laps done`" + ` unless the user said otherwise.
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
	startIdx := strings.Index(content, blockStart)
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
