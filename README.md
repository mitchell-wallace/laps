# Laps

A minimal, single-binary task tracker for AI coding agents.

## Quickstart

```bash
# Install (Linux/macOS)
curl -sSL https://raw.githubusercontent.com/mitchell-wallace/laps/main/install.sh | bash

# Enable agent instructions
laps on

# Add your first task
laps add head --title "Implement login endpoint"

# Read the next task (head)
laps get

# Mark it done when finished
laps done

# See what's in the queue
laps list
```

## Command Reference

### `laps add <head|tail|after> [id]`
Add a task to the queue.

- `head` — insert at the front of the todo queue (the new head, below any completed laps).
- `tail` — append to the end of the queue.
- `after <id>` — insert immediately after the specified task id. If `<id>` is already complete, the lap is added at the head instead (with a notice on stderr).

Flags:
- `--title <string>` — task title (required unless using `--json`).
- `--description <string>` — task description. Supports `\n` for newlines.
- `--assignee <string>` — optional task assignee.
- `--json <object|array|->` — provide one task as an object, multiple tasks as an array, or read either form from stdin with `-`. Mutually exclusive with the field flags.

JSON arrays are validated before the queue is modified, written in one update,
and preserve their input order for `head`, `tail`, and `after` insertion. This is
useful for generated plans:

```bash
laps add tail --json '[
  {"title":"Implement parser","assignee":"SENIOR"},
  {"title":"Verify parser","assignee":"VERIFY"}
]'

jq -c . plan.json | laps add tail --json -
```

Prints each new task id on success. With `--json-output`, object input returns a
`task` object and array input returns a `tasks` array.

### `laps init`
Initialize laps in the current repository. Creates `.laps/laps.json` (if absent)
and appends `.laps/claim` to `.gitignore` (if not already present). When changes
are made, attempts to auto-commit them as `chore: laps init`.

### `laps get [head|<id>]`
Get a task by id, or read the head task if no argument is given.

Output is title, optional assignee, blank line, description.

### `laps claim [head|<id>]`
Claim a task for the current session. Writes the claimed task id to
`.laps/claim` so that a subsequent bare `laps done` knows which task to
complete. Defaults to the head task.

After claiming, a hint is printed suggesting `laps claim undo` if the wrong
task was claimed.

### `laps claim undo`
Clear the claimed lap stored in `.laps/claim`. Prints the id and title of the
un-claimed lap.

### `laps list [--all | --done]`
List tasks as a markdown numbered list. `ls` is an alias for `list`.

The default format renders each task across two lines: the title, then the
task id, assignee, and status:

```
1. First task
   laps-f0f4 · JUNIOR · todo
2. Second task
   laps-6b4a · — · todo
```

- Default: todo tasks only, head first.
- `--all` — include done tasks after todo items (struck through).
- `--done` — show only completed tasks, most recent first.
- `--oneline` — render each task on a single line (the prior format), e.g. `laps-6b4a — Second task (assignee: JUNIOR)`.

### `laps move <id> <head|tail|after> [target]`
Reorder an existing todo task, preserving its id.

- `head` — move to the front of the queue.
- `tail` — move to the end of the queue.
- `after <id>` — move to immediately after the specified task id.

Prints the moved task id on success.

### `laps edit <id> [--title] [--description] [--assignee]`
Edit fields of an existing task in place, preserving its id and order. At
least one of the flags must be provided. Each field is updated only when its
flag is set, so passing an empty string (`""`) clears that field.

- `--title <string>` — set a new (non-blank) title.
- `--description <string>` — set the description; pass `""` to clear it.
- `--assignee <string>` — set the assignee; pass `""` to clear it.

Editing a completed lap succeeds with a warning and does not reopen it.
Prints the affected task id on success.

### `laps assign <id> <role>`
Assign a task to a role. A shortcut for `edit <id> --assignee <role>`.

A blank role clears the assignee. Assigning a completed lap succeeds with a
warning and does not reopen it. Prints the affected task id on success.

### `laps done [<id>]`
Complete a task. When called with a task id, completes that task directly.
When called without arguments, reads the claimed lap from `.laps/claim` (set by
`laps claim`) and completes it. If no task is claimed and no id is given,
prints a hint with the head task's id and title.

Prints the task title on success. If the completed task matches the claimed
lap, `.laps/claim` is cleared automatically.

A hint is printed suggesting `laps done undo` if the wrong task was marked done.

### `laps done undo [-y]`
Re-open the most recently completed lap. If it was completed more than
5 minutes ago, the command fails unless `-y` (or `--yes`) is passed.

Prints the task title on success.

### `laps delete <id>`
Delete a task by id, regardless of whether it is todo or done.

### `laps prune [N]`
Remove old done tasks, keeping the `N` most recent. Default `N` is 20.
`laps prune 0` removes all done tasks. Todo tasks are never touched.

Prints the number of tasks removed.

### `laps on`
Add the `<laps-instructions>` block to `AGENTS.md` (creating it if absent).
Also updates `CLAUDE.md` and `GEMINI.md` if they already exist. Idempotent.

### `laps off`
Remove the `<laps-instructions>` block from `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md`.

### `laps update`
Check the Laps GitHub repository for a newer version.

Prints the current and latest versions. If a newer release exists, prompts for
Y/n confirmation before downloading and installing it via the official install
script. Use `laps update --yes` to install non-interactively.

## Hooks

Laps can run custom shell commands before or after any laps command via `.laps/hooks.json`.

### Hook fields

| Field | Description |
|-------|-------------|
| `title` | Human-readable name for the hook. |
| `description` | What the hook does. |
| `command` | The laps command that triggers this hook (e.g. `done`, `add`). Can also be a custom name for hook-only commands. Subcommands use a hyphenated form, e.g. `done-undo`, `claim-undo`. |
| `when` | `before` or `after`. |
| `run` | Shell command to execute. |
| `passback` | If `true`, the hook's stdout is printed after laps's output. |

### Variables

Use `$var` or `${var}` in `run`:

- `$id`, `$title`, `$description` — the relevant task (empty for commands like `list`, `prune`).
- `$assignee` — the relevant task's assignee, or empty when none is set.
- `$command` — the laps command name.
- `$exit_code` — laps's exit code (`after` hooks only).
- `$output` — laps's stdout (`after` hooks only).
- `$file` — the resolved task file path.

For a batch `laps add --json '[...]'`, hooks run once for the add command with
empty single-task variables (`$id`, `$title`, `$description`, and `$assignee`).

### Hook-only commands

If `command` does not match a built-in laps command (e.g. `worktree`), you can invoke it directly:

```bash
laps worktree
```

laps will fire every matching `before` and `after` hook and nothing else.

### Reserved command names

The following names are reserved for potential future built-in commands and should be avoided for hook-only commands:

`init`, `tui`, `view`, `sync`, `project`, `graph`, `tree`, `ready`, `blocked`

Other names that may conflict with future features in a task-tracking CLI include:
`start`, `stop`, `pause`, `resume`, `reorder`, `search`, `filter`, `tag`, `untag`, `unassign`, `note`, `log`, `status`, `priority`, `label`, `archive`, `unarchive`, `import`, `export`, `backup`, `restore`, `template`, `config`, `setting`

> **Tip:** When in doubt, prefix your hook-only commands with a project-specific namespace (e.g. `myproject-deploy`) to avoid future collisions.

## Versioning

Version is tracked in `VERSION` at the repo root. Pushing a change to `VERSION` on `main` triggers an automated release:

1. `auto-tag` workflow reads the new version from `VERSION`, creates a `vX.Y.Z` git tag, and dispatches `release.yml`.
2. `release` workflow runs [GoReleaser](https://goreleaser.com) to build binaries for Linux, macOS, and Windows, then publishes a GitHub Release with checksums and the install script.

See `.github/workflows/auto-tag.yml` and `.github/workflows/release.yml` for details.

### Example `.laps/hooks.json`

See [`examples/hooks.json`](examples/hooks.json) for a working auto-commit example.

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
