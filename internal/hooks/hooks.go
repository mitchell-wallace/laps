package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Hook represents a single hook entry.
type Hook struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command"`
	When        string `json:"when"`
	Run         string `json:"run"`
	Passback    bool   `json:"passback"`
}

// File is the top-level hooks envelope.
type File struct {
	Version int    `json:"version"`
	Hooks   []Hook `json:"hooks"`
}

// Load reads the hooks file from beadsDir. Returns an empty File if the file does not exist.
func Load(beadsDir string) (*File, error) {
	path := filepath.Join(beadsDir, "hooks.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Version: 1}, nil
		}
		return nil, fmt.Errorf("read hooks file: %w", err)
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse hooks file: %w", err)
	}
	return &f, nil
}

// Dispatch runs all hooks matching (command, when) in array order.
// It substitutes variables from vars, executes each hook via the system shell,
// and returns the combined stdout of all passback hooks.
// If any hook exits non-zero, it returns an error immediately.
// Hook stderr is always written to os.Stderr.
//
// dir is the working directory hooks execute in (the repository root). This
// ensures hooks behave identically regardless of which subdirectory the laps
// command was invoked from. An empty dir leaves the process default in place.
func Dispatch(f *File, command, when string, vars map[string]string, dir string) (string, error) {
	var passbackOut strings.Builder
	for _, h := range f.Hooks {
		if h.Command != command || h.When != when {
			continue
		}
		run := substitute(h.Run, vars)
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", run)
		} else {
			cmd = exec.Command("/bin/sh", "-c", run)
		}
		cmd.Dir = dir
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			return passbackOut.String(), fmt.Errorf("hook %q failed: %w", h.Title, err)
		}
		if h.Passback {
			passbackOut.Write(out)
		}
	}
	return passbackOut.String(), nil
}

var varRe = regexp.MustCompile(`\$\{(\w+)\}|\$(\w+)`)

func substitute(s string, vars map[string]string) string {
	return varRe.ReplaceAllStringFunc(s, func(match string) string {
		// match is either ${name} or $name
		name := varRe.FindStringSubmatch(match)[1]
		if name == "" {
			name = varRe.FindStringSubmatch(match)[2]
		}
		if v, ok := vars[name]; ok {
			return v
		}
		return match
	})
}
