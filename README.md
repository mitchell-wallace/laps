# Microbeads

A minimal, single-binary task tracker for AI coding agents.

## Quickstart

```bash
# Install (Linux/macOS)
curl -sSL https://raw.githubusercontent.com/mitchell-wallace/microbeads/main/install.sh | bash

# Enable agent instructions
mb on

# Add your first task
mb add head --title "Implement login endpoint"

# Read the next task (head)
mb get

# Mark it done when finished
mb done

# See what's in the queue
mb list
```

## Command Reference

### `mb add <head|tail|after> [id]`
Add a task to the queue.

- `head` — insert at the front of the queue.
- `tail` — append to the end of the queue.
- `after <id>` — insert immediately after the specified task id.

Flags:
- `--title <string>` — task title (required unless using `--json`).
- `--description <string>` — task description. Supports `\n` for newlines.
- `--assignee <string>` — optional task assignee.
- `--json <object>` — provide task as `{"title": "...", "description": "...", "assignee": "..."}`. Mutually exclusive with the field flags.

Prints the new task's id on success.

### `mb get [head|<id>]`
Get a task by id, or read the head task if no argument is given.

Output is title, optional assignee, blank line, description.

### `mb list [--all | --done]`
List tasks as a markdown numbered list.

- Default: todo tasks only, head first.
- `--all` — include done tasks after todo items (struck through).
- `--done` — show only completed tasks, most recent first.

### `mb done`
Complete the head task. Sets it to done and prints the task id.

If there is no head task, exits non-zero with `no head task`.

### `mb delete <id>`
Delete a task by id, regardless of whether it is todo or done.

### `mb prune [N]`
Remove old done tasks, keeping the `N` most recent. Default `N` is 20.
`mb prune 0` removes all done tasks. Todo tasks are never touched.

Prints the number of tasks removed.

### `mb on`
Add the `<mb-instructions>` block to `AGENTS.md` (creating it if absent).
Also updates `CLAUDE.md` and `GEMINI.md` if they already exist. Idempotent.

### `mb off`
Remove the `<mb-instructions>` block from `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md`.

### `mb update`
Check the Microbeads GitHub repository for a newer version.

Prints the current and latest versions. If a newer release exists, prompts for
Y/n confirmation before downloading and installing it via the official install
script. Use `mb update --yes` to install non-interactively.

## Hooks

Microbeads can run custom shell commands before or after any mb command via `.beads/mb-hooks.json`.

### Hook fields

| Field | Description |
|-------|-------------|
| `title` | Human-readable name for the hook. |
| `description` | What the hook does. |
| `command` | The mb command that triggers this hook (e.g. `done`, `add`). Can also be a custom name for hook-only commands. |
| `when` | `before` or `after`. |
| `run` | Shell command to execute. |
| `passback` | If `true`, the hook's stdout is printed after mb's output. |

### Variables

Use `$var` or `${var}` in `run`:

- `$id`, `$title`, `$description` — the relevant task (empty for commands like `list`, `prune`).
- `$assignee` — the relevant task's assignee, or empty when none is set.
- `$command` — the mb command name.
- `$exit_code` — mb's exit code (`after` hooks only).
- `$output` — mb's stdout (`after` hooks only).
- `$file` — the resolved task file path.

### Hook-only commands

If `command` does not match a built-in mb command (e.g. `worktree`), you can invoke it directly:

```bash
mb worktree
```

mb will fire every matching `before` and `after` hook and nothing else.

### Reserved command names

The following names are reserved for potential future built-in commands and should be avoided for hook-only commands:

`init`, `tui`, `view`, `edit`, `sync`, `project`, `graph`, `tree`, `ready`, `blocked`

Other names that may conflict with future features in a task-tracking CLI include:
`start`, `stop`, `pause`, `resume`, `move`, `reorder`, `search`, `filter`, `tag`, `untag`, `assign`, `unassign`, `note`, `log`, `status`, `priority`, `label`, `archive`, `unarchive`, `import`, `export`, `backup`, `restore`, `template`, `config`, `setting`

> **Tip:** When in doubt, prefix your hook-only commands with a project-specific namespace (e.g. `myproject-deploy`) to avoid future collisions.

### Example `.beads/mb-hooks.json`

See [`examples/mb-hooks.json`](examples/mb-hooks.json) for a working auto-commit example.

```json
{
  "version": 1,
  "hooks": [
    {
      "title": "Commit on done",
      "description": "Auto-commit when a task is completed.",
      "command": "done",
      "when": "after",
      "run": "git add -A && git commit -m \"done: $title ($id)\"",
      "passback": false
    }
  ]
}
```
