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

		modifiedGitignore := false
		gitignorePath := filepath.Join(repoRoot, ".gitignore")
		claimLine := ".laps/claim"

		var lines []string
		found := false
		if data, err := os.ReadFile(gitignorePath); err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(data))
			for scanner.Scan() {
				text := scanner.Text()
				if strings.TrimSpace(text) == claimLine {
					found = true
					break
				}
				lines = append(lines, text)
			}
		}

		if !found {
			lines = append(lines, claimLine)
			data := []byte(strings.Join(lines, "\n") + "\n")
			if err := os.WriteFile(gitignorePath, data, 0o644); err != nil {
				exit(2, "update .gitignore: %v", err)
			}
			modifiedGitignore = true
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
			fmt.Println("Added .laps/claim to .gitignore")
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
