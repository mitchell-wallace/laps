package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/mitchell-wallace/microbeads/internal/hooks"
	"github.com/mitchell-wallace/microbeads/internal/store"
)

func runBeforeHooks(cmdName string, beadsDir string, path string, task *store.Task) {
	hf, err := hooks.Load(beadsDir)
	if err != nil {
		exit(2, "hooks: %v", err)
	}
	if hf == nil {
		return
	}
	vars := buildHookVars(task, path, cmdName, "", "")
	_, err = hooks.Dispatch(hf, cmdName, "before", vars)
	if err != nil {
		exit(4, "hook: %v", err)
	}
}

func runAfterHooksDeferred(cmdName string, beadsDir string, path string, task **store.Task, output *string, exitCode *int) func() {
	return func() {
		hf, err := hooks.Load(beadsDir)
		if err != nil || hf == nil {
			return
		}
		var t *store.Task
		if task != nil {
			t = *task
		}
		vars := buildHookVars(t, path, cmdName, fmt.Sprintf("%d", *exitCode), *output)
		passback, err := hooks.Dispatch(hf, cmdName, "after", vars)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mb: after hook: %v\n", err)
		}
		if passback != "" {
			fmt.Print(passback)
		}
	}
}

func buildHookVars(task *store.Task, file, command, exitCode, output string) map[string]string {
	vars := map[string]string{
		"command":   command,
		"file":      file,
		"exit_code": exitCode,
		"output":    output,
		"id":        "",
		"title":     "",
		"description": "",
	}
	if task != nil {
		vars["id"] = task.ID
		vars["title"] = task.Title
		vars["description"] = task.Description
	}
	return vars
}

func isKnownCommand(name string) bool {
	switch name {
	case "add", "get", "list", "done", "delete", "prune", "on", "off", "help", "--version":
		return true
	}
	return false
}

func firstNonFlagArg(args []string) string {
	skipNext := false
	for _, a := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(a, "-") {
			if a == "-f" || a == "--file" {
				skipNext = true
			}
			continue
		}
		return a
	}
	return ""
}
