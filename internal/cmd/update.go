package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const githubAPI = "https://api.github.com/repos/mitchell-wallace/microbeads/releases/latest"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for a newer version and optionally update",
	Long: `Check the GitHub releases page for a newer version of mb.

Prints the current and latest versions. If a newer version is available,
prompts for confirmation before running the install script.`,
	Run: func(cmd *cobra.Command, args []string) {
		if version == "" || version == "dev" {
			fmt.Println("Current version: dev (cannot check for updates)")
			return
		}

		latest, err := fetchLatestVersion()
		if err != nil {
			exit(2, "update: %v", err)
		}

		fmt.Printf("Current version: %s\n", version)
		fmt.Printf("Latest version:  %s\n", latest)

		cmp, err := compareVersions(version, latest)
		if err != nil {
			exit(2, "update: %v", err)
		}

		if cmp >= 0 {
			fmt.Println("You are up to date.")
			return
		}

		fmt.Print("Update to latest version? [Y/n] ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			exit(2, "update: read confirmation: %v", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "" && response != "y" && response != "yes" {
			fmt.Println("Update cancelled.")
			return
		}

		installCmd := exec.Command("sh", "-c", "curl -fsSL https://raw.githubusercontent.com/mitchell-wallace/microbeads/main/install.sh | bash")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			exit(2, "update: install failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", githubAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned %s", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("parse release: %w", err)
	}

	return strings.TrimPrefix(payload.TagName, "v"), nil
}

func compareVersions(a, b string) (int, error) {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}
	for i := 0; i < maxLen; i++ {
		var av, bv int
		if i < len(aParts) {
			v, err := strconv.Atoi(aParts[i])
			if err != nil {
				return 0, fmt.Errorf("invalid version %q", a)
			}
			av = v
		}
		if i < len(bParts) {
			v, err := strconv.Atoi(bParts[i])
			if err != nil {
				return 0, fmt.Errorf("invalid version %q", b)
			}
			bv = v
		}
		if av < bv {
			return -1, nil
		}
		if av > bv {
			return 1, nil
		}
	}
	return 0, nil
}
