package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize laps in the current repository",
	Run: func(cmd *cobra.Command, args []string) {
		repoRoot, beadsDir, err := store.DiscoverRepoRoot()
		if err != nil {
			exit(2, "%v", err)
		}

		createdDefault := false
		defaultPath := filepath.Join(beadsDir, store.ResolveFile(""))

		if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
			f := &store.File{Version: store.CurrentVersion, Tasks: []store.Task{}}
			if err := store.Save(defaultPath, f); err != nil {
				exit(2, "create default file: %v", err)
			}
			createdDefault = true
		}

		gitignorePath := filepath.Join(repoRoot, ".gitignore")
		gitignoreEntries := []string{".laps/claim", ".laps/log.jsonl"}

		var lines []string
		found := map[string]bool{}
		if data, err := os.ReadFile(gitignorePath); err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(data))
			for scanner.Scan() {
				text := scanner.Text()
				lines = append(lines, text)
				trimmed := strings.TrimSpace(text)
				for _, e := range gitignoreEntries {
					if trimmed == e {
						found[e] = true
					}
				}
			}
		}

		var added []string
		for _, e := range gitignoreEntries {
			if !found[e] {
				added = append(added, e)
				lines = append(lines, e)
			}
		}

		modifiedGitignore := len(added) > 0
		if modifiedGitignore {
			data := []byte(strings.Join(lines, "\n") + "\n")
			if err := os.WriteFile(gitignorePath, data, 0o644); err != nil {
				exit(2, "update .gitignore: %v", err)
			}
		}

		if jsonOutput {
			printJSON(map[string]interface{}{
				"created":           createdDefault,
				"gitignoreModified": modifiedGitignore,
			})
			return
		}

		if createdDefault {
			fmt.Println("Created .laps/laps.json")
		}
		if modifiedGitignore {
			fmt.Printf("Added %s to .gitignore\n", joinGitignoreEntries(added))
		}
		if !createdDefault && !modifiedGitignore {
			fmt.Println("Already initialized")
		}

		if createdDefault || modifiedGitignore {
			var files []string
			if modifiedGitignore {
				files = append(files, gitignorePath)
			}
			if createdDefault {
				files = append(files, defaultPath)
			}
			addCmd := exec.Command("git", append([]string{"add"}, files...)...)
			addCmd.Dir = repoRoot
			if out, err := addCmd.CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "laps: note: auto-commit (git add) failed: %s\n", strings.TrimSpace(string(out)))
			} else {
				commitCmd := exec.Command("git", "commit", "-m", "chore: laps init")
				commitCmd.Dir = repoRoot
				if out, err := commitCmd.CombinedOutput(); err != nil {
					fmt.Fprintf(os.Stderr, "laps: note: auto-commit (git commit) failed: %s\n", strings.TrimSpace(string(out)))
				} else {
					fmt.Println("Committed initialization")
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// joinGitignoreEntries renders the list of entries added to .gitignore as a
// human-readable phrase for the success message. The expected set is fixed at
// two entries; ordering is preserved from the caller.
func joinGitignoreEntries(entries []string) string {
	switch len(entries) {
	case 0:
		return ""
	case 1:
		return entries[0]
	case 2:
		return entries[0] + " and " + entries[1]
	default:
		return strings.Join(entries[:len(entries)-1], ", ") + ", and " + entries[len(entries)-1]
	}
}
