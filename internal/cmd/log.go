package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchell-wallace/laps/internal/eventlog"
	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
)

var (
	logLimit    int
	lapFlag     string
	sessionFlag string
	sinceFlag   string
	logScope    string
)

type logEventLine struct {
	TS       time.Time              `json:"ts"`
	Event    string                 `json:"event"`
	Cmd      string                 `json:"cmd"`
	File     string                 `json:"file"`
	Lap      string                 `json:"lap,omitempty"`
	Title    string                 `json:"title,omitempty"`
	Assignee string                 `json:"assignee,omitempty"`
	Scope    string                 `json:"scope"`
	Detail   map[string]interface{} `json:"detail"`
	Session  string                 `json:"session"`
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent event log history",
	Long:  `Show recent event log history, newest last (chronological order).`,
	Run: func(cmd *cobra.Command, args []string) {
		path, repoRoot, beadsDir := getStorePath()
		checkDefault(beadsDir)
		exitCode := 0
		if fileFlag != "" {
			loadFile(path, repoRoot, beadsDir)
		} else if activeScopeSelected() {
			file := loadFile(path, repoRoot, beadsDir)
			ctx, err := resolveActiveContext(path, repoRoot, beadsDir, file)
			if err != nil {
				exitCode = 2
				exit(2, "%v", err)
			}
			path = ctx.Path
		}

		var output string
		var task *store.Task
		defer runAfterHooksDeferred(cmd.Name(), beadsDir, path, &task, &output, &exitCode, args)()
		runBeforeHooks(cmd.Name(), beadsDir, path, nil, args)

		logPath := filepath.Join(beadsDir, eventlog.LogFileName)
		f, err := os.Open(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Missing log file behaves as empty
				if jsonOutput {
					printJSON(map[string]interface{}{"events": []logEventLine{}})
				}
				return
			}
			exitCode = 2
			exit(2, "failed to open log file: %v", err)
		}
		defer func() {
			_ = f.Close()
		}()

		var events []logEventLine
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			lineBytes := scanner.Bytes()
			if len(bytes.TrimSpace(lineBytes)) == 0 {
				continue
			}
			var ev logEventLine
			if err := json.Unmarshal(lineBytes, &ev); err != nil {
				fmt.Fprintf(os.Stderr, "laps: log: skipping malformed line: %v\n", err)
				continue
			}
			events = append(events, ev)
		}
		if err := scanner.Err(); err != nil {
			exitCode = 2
			exit(2, "failed to read log: %v", err)
		}

		var sinceTime time.Time
		if sinceFlag != "" {
			var err error
			sinceTime, err = time.Parse(time.RFC3339, sinceFlag)
			if err != nil {
				exitCode = 2
				exit(2, "invalid since timestamp %q: %v", sinceFlag, err)
			}
		}

		resolvedFile := fileNameForClaim(beadsDir, path)
		filtered := make([]logEventLine, 0)
		for _, ev := range events {
			if ev.File != resolvedFile {
				continue
			}
			if logScope != "" && ev.Scope != logScope {
				continue
			}
			if lapFlag != "" && ev.Lap != lapFlag {
				continue
			}
			if sessionFlag != "" && ev.Session != sessionFlag {
				continue
			}
			// since filter is inclusive, so we only exclude if it's strictly before sinceTime
			if !sinceTime.IsZero() && ev.TS.Before(sinceTime) {
				continue
			}
			filtered = append(filtered, ev)
		}

		if logLimit < 0 {
			exitCode = 2
			exit(2, "limit must be non-negative: %d", logLimit)
		}
		if len(filtered) > logLimit {
			filtered = filtered[len(filtered)-logLimit:]
		}

		var lines []string
		for _, ev := range filtered {
			lines = append(lines, formatEventHuman(&ev))
		}
		output = strings.Join(lines, "\n")

		if jsonOutput {
			// Emit a single JSON object whose "events" field is the array of matching events
			printJSON(map[string]interface{}{
				"events": filtered,
			})
		} else if output != "" {
			fmt.Println(output)
		}
	},
}

func formatEventHuman(ev *logEventLine) string {
	tsStr := ev.TS.UTC().Format(time.RFC3339)
	var parts []string
	parts = append(parts, tsStr)
	parts = append(parts, ev.Event)
	if ev.Lap != "" {
		parts = append(parts, ev.Lap)
	}
	if ev.Title != "" {
		parts = append(parts, fmt.Sprintf("%q", ev.Title))
	}
	if ev.Assignee != "" {
		parts = append(parts, "("+ev.Assignee+")")
	}
	if ev.Session != "" {
		parts = append(parts, fmt.Sprintf("[session: %s]", ev.Session))
	}
	if len(ev.Detail) > 0 {
		var detailParts []string
		for k, v := range ev.Detail {
			detailParts = append(detailParts, fmt.Sprintf("%s:%v", k, v))
		}
		parts = append(parts, "{"+strings.Join(detailParts, ", ")+"}")
	}
	return strings.Join(parts, " ")
}

func init() {
	logCmd.Flags().IntVarP(&logLimit, "limit", "n", 20, "limit the number of events shown")
	logCmd.Flags().StringVar(&lapFlag, "lap", "", "filter events by lap ID")
	logCmd.Flags().StringVar(&sessionFlag, "session", "", "filter events by session ID")
	logCmd.Flags().StringVar(&sinceFlag, "since", "", "filter events since timestamp (RFC3339, inclusive)")
	logCmd.Flags().StringVar(&logScope, "scope", "", "filter events by exact scope")
	addScopeFlags(logCmd)
	rootCmd.AddCommand(logCmd)
}
