package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/mitchell-wallace/microbeads/internal/hooks"
	"github.com/mitchell-wallace/microbeads/internal/store"
)

func runBeforeHooks(cmdName string, beadsDir string, path string, task *store.Task, args []string) {
	hf, err := hooks.Load(beadsDir)
	if err != nil {
		exit(2, "hooks: %v", err)
	}
	if hf == nil {
		return
	}
	vars := buildHookVars(task, path, cmdName, "", "", args)
	_, err = hooks.Dispatch(hf, cmdName, "before", vars)
	if err != nil {
		exit(4, "hook: %v", err)
	}
}

func runAfterHooksDeferred(cmdName string, beadsDir string, path string, task **store.Task, output *string, exitCode *int, args []string) func() {
	return func() {
		hf, err := hooks.Load(beadsDir)
		if err != nil || hf == nil {
			return
		}
		var t *store.Task
		if task != nil {
			t = *task
		}
		vars := buildHookVars(t, path, cmdName, fmt.Sprintf("%d", *exitCode), *output, args)
		passback, err := hooks.Dispatch(hf, cmdName, "after", vars)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mb: after hook: %v\n", err)
		}
		if passback != "" {
			fmt.Print(passback)
		}
	}
}

func buildHookVars(task *store.Task, file, command, exitCode, output string, args []string) map[string]string {
	vars := map[string]string{
		"command":     command,
		"file":        file,
		"exit_code":   exitCode,
		"output":      output,
		"id":          "",
		"title":       "",
		"description": "",
		"assignee":    "",
		"args":        shellQuoteArgs(args),
	}
	for i, arg := range args {
		vars[fmt.Sprintf("%d", i+1)] = arg
	}
	if task != nil {
		vars["id"] = task.ID
		vars["title"] = task.Title
		vars["description"] = task.Description
		vars["assignee"] = task.Assignee
	}
	return vars
}

func shellQuoteArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	var quoted []string
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\n\"'\\$|&;<>()`") {
			arg = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
		}
		quoted = append(quoted, arg)
	}
	return strings.Join(quoted, " ")
}

func isKnownCommand(name string) bool {
	switch name {
	case "add", "get", "list", "done", "delete", "prune", "on", "off", "help", "--version":
		return true
	}
	return false
}

func splitArgs(args []string) (cmd string, posArgs []string) {
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
		if cmd == "" {
			cmd = a
		} else {
			posArgs = append(posArgs, a)
		}
	}
	return cmd, posArgs
}
