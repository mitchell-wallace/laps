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
and appends `.laps/claim` and `.laps/log.jsonl` to `.gitignore` (if either is
not already present). When changes are made, attempts to auto-commit them as
`chore: laps init`.

### `laps get [head|<id>]`
Get a task by id, or read the head task if no argument is given.

Output is title, optional assignee, blank line, description.

### `laps claim [head|<id>]`
Claim a task for the current session. Writes the claim to `.laps/claim` as a structured JSON object so that a subsequent bare `laps done` knows which task to complete and when it was claimed. Defaults to the head task.

After claiming, a hint is printed suggesting `laps claim undo` if the wrong
task was claimed.

#### Claim File Format
The claim file `.laps/claim` is stored as a single JSON object:
```json
{
  "lap": "laps-f0f4",
  "file": "laps.json",
  "scope": "root",
  "claimedAt": "2026-06-30T10:28:42Z"
}
```
Fields:
- `lap` — the claimed task ID.
- `file` — the relative path of the task file containing the lap.
- `scope` — the canonical logical scope the lap was claimed in (`root`, a
  root-level stint name like `auth`, or a slash path for nesting like
  `auth/search`). A bare `laps done` completes the claimed lap **within its
  recorded scope**, so it survives head changes from preemption or another
  session. See [Stints](#stints).
- `claimedAt` — the UTC timestamp of when the lap was claimed (RFC3339). When the same lap is re-claimed, the original `claimedAt` timestamp is preserved.

**Backward Compatibility**: If `.laps/claim` contains a legacy bare task ID (a plain string token, not JSON), it is successfully parsed as a legacy claim with the lap ID assigned, `file` defaulting to the current selected task file, `scope` defaulting to `root`, and `claimedAt` treated as `null`. Structured claims also ignore unknown JSON fields for forward compatibility.

### `laps claim undo`
Clear the claimed lap stored in `.laps/claim`. Prints the id and title of the
un-claimed lap.

### `laps list [--all | --done] [--tree]`
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
- `--tree` — render the full recursive overview. Descends into every queued
  stint ref and indents its laps, so root laps, queued stints, and the laps
  inside each are visible in one picture. A stint ref renders as a single
  line, e.g. `2. auth/ (stint - 4 laps)`; a missing stint file or a ref cycle
  is flagged inline rather than silently skipped. `--tree` operates on the
  whole root queue and ignores the scope flags (it always starts from root).

`list` is a flow op: a bare `laps list` resolves the **active** context and
shows the active stint's laps (not root) when a stint is active. Use
`list --root` for the root queue or `list --tree` for the full pipeline (see
[Stints](#stints)).

### `laps status`
Show a snapshot of the lap queue status. Reports the selected task file path, the queue state, todo/done/total task counts, the head (next todo) lap, the active (claimed) lap, a breakdown of todo tasks by assignee, and — when stints are present — the active stint and per-stint progress.

Queue state is one of:
- `active` — a valid todo lap is currently claimed (work in progress).
- `ready` — todo laps exist but nothing valid is currently claimed.
- `empty` — no laps exist in the file.
- `complete` — laps exist and all of them are marked done.

If there is an active claim pointing to a deleted task, a completed task, or a task in a different file, it is considered a "dangling" claim. In this case, the claim is surfaced with `valid: false` (in JSON) or `invalid` (in human-readable output) without being silently/automatically cleared, and the command exits successfully with a degraded status snapshot.

Actual errors, such as a corrupt task file, a malformed claim file (bad JSON syntax or wrong field types), or marshal failures, exit with a normal non-zero error code.

With the global `--json-output` flag, the output is a single JSON object with the following shape (note the explicit absence of any stale flag in this version):

```json
{
  "file": "laps.json",
  "state": "active",
  "counts": {
    "todo": 2,
    "done": 12,
    "total": 14
  },
  "head": {
    "id": "laps-2ef1",
    "title": "Document event log, log/status commands, LAPS_SESSION, and structured claim in README",
    "assignee": "JUNIOR"
  },
  "claim": {
    "valid": true,
    "lap": "laps-2ef1",
    "file": "laps.json",
    "claimedAt": "2026-06-30T10:28:42Z",
    "ageSeconds": 3600
  },
  "assignees": [
    {
      "assignee": "JUNIOR",
      "todo": 1
    },
    {
      "assignee": "VERIFY",
      "todo": 1
    }
  ]
}
```

Fields in JSON:
- `file` — the relative path of the active task file.
- `state` — the queue state string (`active` | `ready` | `empty` | `complete`).
- `counts` — object containing `todo`, `done`, and `total` lap counts.
- `head` — object containing `id`, `title`, and `assignee` of the first todo task (null if none).
- `claim` — object representing the active claim:
  - `valid` — boolean indicating if the claim is valid (names a todo lap in the current task file).
  - `lap` — the claimed task ID.
  - `file` — the file the claim was made against.
  - `claimedAt` — nullable RFC3339 UTC timestamp (string) of when the claim was created. This is `null` if no lap is claimed or if the claim file contains a legacy bare-id (no timestamp).
  - `ageSeconds` — nullable integer representing the seconds elapsed since the claim was made. This is `null` whenever `claimedAt` is `null`.
- `assignees` — array of `{assignee, todo}` objects summarizing todo counts by assignee role, sorted alphabetically by assignee name (with unassigned tasks grouped under `"unassigned"`).
- `activeStint` — when the active context is a stint, a `{scope, todo, done, total}` object for that stint (otherwise `null`).
- `stints` — array of `{scope, todo, done, total}` objects giving per-stint progress for every queued stint (empty when no stints are queued).

### `laps log`
Show recent event log history, newest last (chronological order).

Flags:
- `-n, --limit <int>` — limit the number of events shown (default 20, must be non-negative).
- `--lap <string>` — filter events to only those matching the specified lap ID.
- `--session <string>` — filter events to only those matching the specified session ID.
- `--since <string>` — filter events since the specified RFC3339 timestamp (inclusive).

#### Behavior
- **Filter-then-Limit**: The filters (`--lap`, `--session`, `--since`) are applied to the full event log first, and then the limit (`-n`) is applied to the filtered subset.
- **Newest-Last Order**: Events are printed chronologically, with the most recent event at the bottom of the list.
- **Malformed Line Handling**: Any malformed JSON lines in the log file are skipped with a warning on `stderr` and do not cause the command to fail.
- **JSON Output**: When the global `--json-output` flag is requested, the output is a single JSON object containing an `"events"` array of matching events:

```json
{
  "events": [
    {
      "ts": "2026-06-30T10:28:42Z",
      "event": "claimed",
      "cmd": "claim",
      "file": "laps.json",
      "lap": "laps-2ef1",
      "title": "Document event log, log/status commands, LAPS_SESSION, and structured claim in README",
      "assignee": "JUNIOR",
      "scope": "root",
      "detail": {},
      "session": "session-test-123"
    }
  ]
}
```

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

By default `delete` refuses to remove a currently claimed lap (completing a
claimed lap is legitimate progress, but deleting it discards in-flight work),
printing a warning on stderr. `delete --force <id>` removes it and clears the
matching claim. Prints the deleted task id on success.

### `laps prune [N]`
Remove old done tasks, keeping the `N` most recent. Default `N` is 20.
`laps prune 0` removes all done tasks. Todo tasks are never touched.

Prints the number of tasks removed.

### `laps stints <ls|new|enqueue|show|rm>`
Manage stints (prepared per-change queues). `st` is an alias for `stints`.
See [Stints](#stints) for the model behind these commands.

- `stints ls` — list every stint file with its lap count and `queued`/
  `archived` flags. With `--json-output`, returns `{"stints": [...]}`.
- `stints new <name>` — create an empty stint file at
  `.laps/stints/<name>.laps.json` and allocate its id prefix. Prints the
  allocated 4-character prefix.
- `stints enqueue <name> [head|tail|after <id>]` — add a stint reference to
  the root queue (default `tail`). `head` preempts the active stint
  non-destructively; the paused stint resumes from its own file when the
  interloper drains. `after <id>` resolves the anchor id **in root only**; if
  the id lives inside a stint, the command fails naming that stint. Prints the
  new stint-ref id.
- `stints show <name>` — print a stint's queue (active or archived).
- `stints rm <name>` — remove a stint file. By default this allows unqueued
  non-archived stints and archived stints (including archived stints that still
  have a done root ref), and refuses non-archived queued/active/claimed stints
  unless `--force` is supplied; forced removal also drops matching root refs
  and clears a matching claim.

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

## Stints

A **stint** is a prepared queue for a single change, stored at
`.laps/stints/<name>.laps.json`. The canonical `laps.json` queue stays flat but
may now hold laps, **stint references**, or any mix: a stint ref is a queue
entry that points at a stint file. Operators stage one stint per change and
enqueue them as a pipeline, while agents keep seeing the same
read-head → do-it → mark-done contract without noticing stints at all.

The **active context** is derived from queue position, never stored: it is the
deepest stint ref on the path from the resolved root head down to the head lap.
There is no active-pointer file to desynchronize.

### Read-through resolution
The flow ops `get`, `claim`, `done`, and `list` start at the root head and
**descend** through active stint refs to the first real lap (recursive, so
nesting is supported by the engine). Agent output stays identical — a nested
lap's title and description look the same as a root lap. Resolver failures
(missing child file, malformed ref, malformed child file, or a ref cycle) fail
deterministically instead of looping or silently skipping.

### Scope flags
Queue-targeting commands (`add`, `get`, `claim`, `done`, `list`, `count`,
`delete`, `prune`, `move`, `edit`, `assign`) accept three mutually exclusive
scope flags that select *which* layer a command targets:

- `--active` / `-c` — the deepest active queue. This is the **default** when no
  scope flag is given. Flow ops (`get`/`claim`/`done`/`list`) recursively
  descend; `count`/`prune` resolve the active chain only to *locate* the target
  file and then operate on that single file (no aggregation into nested
  children).
- `--root` / `-r` — the root queue, with no descent.
- `--stint <name>` / `-s` — a named stint queue, with no descent.

The raw `--file`/`-f` flag is **mutually exclusive** with all three scope
flags, so one invocation has exactly one target model (`--stint auth` resolves
to `.laps/stints/auth.laps.json`, a different path than `-f auth` →
`.laps/auth.json`). Combining two scope flags, or `--file` with a scope flag,
errors out.

Bare verbs used by agents default to `--active` and descend; operators reach
for the long forms (`--root`, `--stint`) for explicit structural control.

### Scoped explicit-id resolution
Every id-taking operation (`get <id>`, `claim <id>`, `done <id>`,
`add after <id>`, `move`, `edit`, `assign`, `delete`) resolves the id **within
the selected scope first**. If the id lives in another stint, the command fails
naming that stint (e.g. `a7 is in stint search - re-run with -s search`) and
mutates no file. `stints enqueue after <id>` resolves the anchor in root only.

### Per-stint id prefixes (globally unique ids)
A lap id is `<prefix>-<hash>`. Root laps use the **repo prefix** (the first 4
lowercase alphanumerics of the repo directory name). Each stint gets its own
allocated 4-character prefix, recorded in the stint file metadata, and laps
created inside that stint carry the stint's prefix. Allocation happens once at
`stints new` and is made unique against the repo prefix and every existing
stint prefix.

Consequence: **a lap id's prefix identifies its owning stint (or root)**, so
ids are globally unique across all files. This is what lets scoped explicit-id
resolution and the active-lap marker in `list` work unambiguously even though
`list` descends — the marker and scope are read straight from the id, with no
separate file/scope comparison.

### Enqueue, drain, and auto-archive
`stints enqueue` adds a ref to the root queue (default `tail`). A `head`
enqueue preempts the active stint; because each stint's progress lives in its
own file, preemption is non-destructive and the paused stint resumes when the
interloper drains.

A stint with no todo laps left is **drained**: the draining operation flips its
root ref to done and moves the file to `.laps/stints/archive/`. Draining is
content-based and position-independent — a preempted, non-head stint still
drains when its last lap completes, and a done ref is skipped on later
advance. `done undo` scans all queue files (root, active stints, and the
archive) for the globally latest completion and unarchives when that lap lives
in a drained stint (the 5-minute age gate still applies).

### Schema version
The on-disk schema reaches **v3** with this feature: queue entries gain a
`kind` discriminator (`lap` by default, `stint` for refs), and existing
entries are migrated to `kind:"lap"`. The binary `VERSION`, however, stays at
`0.8.1` until a later change bumps `0.9.0`, so a v3-writing build still reports
`0.8.1`. An entry with no `kind` is treated as a `lap`, so older data files
remain readable.

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
- `$scope` — the canonical logical scope the command ran in (`root`, a stint
  name like `auth`, or a slash path for nesting like `auth/search`). Defaults
  to `root` for hook-only commands.

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

## Event Log

Laps features a native, best-effort, append-only event log that records state
changes from mutating commands.

### Characteristics
- **Native**: Built directly into `laps` commands (not configured as a hook).
- **Best-Effort**: Log writes are non-blocking and fail-safe. If writing to the event log fails, a one-line warning is printed to `stderr` but the command still completes with its normal exit code.
- **Append-Only**: Events are appended as JSON lines to `.laps/log.jsonl`.
- **Gitignored**: The `.laps/log.jsonl` file is automatically appended to `.gitignore` during `laps init` to keep local orchestration events out of git history.

### LAPS_SESSION Attribution
If the `LAPS_SESSION` environment variable is set, its value is automatically stamped into the `session` field of every event log entry. This allows multiple commands run as part of the same try, build, or agent stint to be grouped and analyzed together.

### Event Schema
Each line in `.laps/log.jsonl` is a JSON object with the following fields:

- `ts` — UTC timestamp of the event (RFC3339 format).
- `event` — the event type (e.g. `claimed`, `unclaimed`, `completed`, `created`, `moved`, `edited`, `stint.enqueued`, `stint.completed`, `stint.archived`).
- `cmd` — the name of the command that triggered the event (e.g. `claim`, `claim-undo`, `done`, `add`, `move`, `edit`, `stints-enqueue`).
- `file` — the relative path of the task file affected (e.g. `laps.json`).
- `lap` — the ID of the affected task (omitted if not task-specific).
- `title` — the title of the affected task (omitted if not task-specific).
- `assignee` — the assignee role of the task (omitted if none or not task-specific).
- `scope` — the canonical logical scope of the event (`root`, a stint name like `auth`, or a slash path for nesting like `auth/search`).
- `detail` — an object containing additional event-specific details.
- `session` — the session ID from the `LAPS_SESSION` environment variable (empty string if unset).

## Versioning

Version is tracked in `VERSION` at the repo root. Pushing a change to `VERSION` on `main` triggers an automated release:

1. `auto-tag` workflow reads the new version from `VERSION`, creates a `vX.Y.Z` git tag, and dispatches `release.yml`.
2. `release` workflow runs [GoReleaser](https://goreleaser.com) to build binaries for Linux, macOS, and Windows, then publishes a GitHub Release with checksums and the install script.

See `.github/workflows/auto-tag.yml` and `.github/workflows/release.yml` for details.
