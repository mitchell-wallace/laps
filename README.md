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

Output is title, optional assignee, blank line, description. When a scoped
operation such as `get --root` targets a stint reference directly, text output
renders it as `<name>/ (stint)` instead of just the raw ref title.

A head `get` (no explicit id) signals queue state through its **exit code**:
`0` a lap was returned, `10` the head is held/gated, `11` the queue is empty,
`12` every lap is complete (see [Queue-state exit codes](#queue-state-exit-codes)).
Text mode prints no command output to stdout for `10`/`11`/`12` and warns on
stderr for held; JSON mode prints a small `{"state","exitCode"}` object to
stdout. Hook passback, if configured, may still write its own stdout.
Explicit `get <id>` (including one inside a held stint) stays its normal
result, warning on stderr when the target stint is held.

### `laps claim [head|<id>]`
Claim a task for the current session. Writes the claim to `.laps/claim` as a structured JSON object so that a subsequent bare `laps done` knows which task to complete and when it was claimed. Defaults to the head task.

A head `claim` (no explicit id) signals queue state through its **exit code**:
`0` a lap was claimed, `10` the head is held/gated, `11` the queue is empty,
`12` every lap is complete (see [Queue-state exit codes](#queue-state-exit-codes)).
Held cases leave the existing claim unchanged and warn on stderr. An explicit
`claim <id>` targeting a lap inside a held stint also exits `10`, leaves the
claim unchanged, and warns on stderr.

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
- `--oneline` — render each task on a single line, e.g. `2. laps-6b4a — Second task (assignee: JUNIOR)`. This is the stable machine form pinned by the [consumer contract](#consumer-contract): exactly one non-blank line per lap, `<n>. <id> — <title>[ (assignee: <a>)]`.
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

`status` accepts the same scope flags as queue-targeting commands:
`--active` (default), `--root`, and `--stint <name>`. Use `status --stint auth`
to inspect `.laps/stints/auth.laps.json` without spelling the path via
`--file stints/auth.laps.json`.

Queue state is one of:
- `active` — a valid todo lap is currently claimed (work in progress).
- `ready` — todo laps exist but nothing valid is currently claimed.
- `held` — the next flow-start operation is gated by a held stint (no valid
  active claim takes precedence). The held stint, scope, and gate message are
  surfaced separately (the `gate` field in JSON; a `Gate:` line in text).
- `empty` — no laps exist in the file.
- `complete` — laps exist and all of them are marked done.

Consumers (orchestrators, dashboards) SHALL NOT treat the four states
`active`/`ready`/`empty`/`complete` as a closed list — `held` extends the
taxonomy. A valid active claim keeps `status.state=active` even when the next
head is held; in that case the held gate is reported separately rather than as
the primary state.

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
- `state` — the queue state string (`active` | `ready` | `held` | `empty` | `complete`).
- `counts` — object containing `todo`, `done`, and `total` lap counts.
- `head` — object containing `id`, `title`, and `assignee` of the first todo task (null if none).
- `gate` — present only when the resolved head is a held stint. A
  `{state, stint, scope, file, message}` object describing the held gate,
  where `state` is `"held"`, `stint` is the held stint name, `scope` is its
  canonical scope, and `message` is a human-readable gate message. Surfaced
  separately even when an active claim keeps `state` as `active`.
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
- `--scope <string>` — filter events to an exact logged scope (`root`, `auth`,
  `auth/search`, etc.).
- `--since <string>` — filter events since the specified RFC3339 timestamp (inclusive).

`log` also accepts the scope-selection flags `--active`, `--root`, and
`--stint <name>` to choose which task file's events are shown. Combine
`--stint auth --scope auth/search` to filter to nested-scope events recorded in
that stint file.

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

### `laps stints <ls|new|enqueue|show|hold|release|rm>`
Manage stints (prepared per-change queues). `st` is an alias for `stints`.
See [Stints](#stints) for the model behind these commands.

- `stints ls` — list every stint file with its id prefix, lap count, and
  `queued`/`archived` flags plus a `held` marker for any held stint
  (`held=true`). With `--json-output`, returns `{"stints": [...]}` (each entry
  carries `prefix` and a `held` boolean).
- `stints new <name>` — create an empty stint file at
  `.laps/stints/<name>.laps.json` and allocate its id prefix. Prints the
  allocated 4-character prefix.
- `stints enqueue <name> [head|tail|after <id>]` — add a stint reference to
  the root queue (default `tail`). `head` preempts the active stint
  non-destructively; the paused stint resumes from its own file when the
  interloper drains. `after <id>` resolves the anchor id **in root only**; if
  the id lives inside a stint, the command fails naming that stint. Prints the
  new stint-ref id.
- `stints show <name>` — print a stint's queue (active or archived), including
  metadata (`prefix`, `queued`, `archived`, `held`) before the lap list.
- `stints hold <name>` — mark a non-archived stint **held** so the queue stops
  once a reference to that stint reaches the head during flow resolution (see
  [Holding a stint](#holding-a-stint-gating)). Works on any non-archived
  stint, including one that is not yet enqueued; archived stints are refused.
  Idempotent — holding an already-held stint is a no-op. Appends a
  `stint.held` event only when the state actually changes. Prints the stint
  name.
- `stints release <name>` — clear the held flag so the stint resumes flowing.
  Same refusals and idempotency as `hold`; appends a `stint.released` event
  only when the state changes. Prints the stint name.
- `stints rm <name>` — remove a stint file. By default this allows unqueued
  non-archived stints and archived stints (including archived stints that still
  have a done root ref), and refuses non-archived queued/active/claimed stints
  unless `--force` is supplied; forced removal also drops matching root refs
  and clears a matching claim.

### `laps on`
Add the versioned `<laps-instructions v="2">` queue contract to `AGENTS.md`
(creating it if absent). Also updates `CLAUDE.md` and `GEMINI.md` if they
already exist. Idempotent; legacy and older versioned blocks are refreshed in
place. The injected contract and `get`/`claim`/`status` help document the
queue-state exit-code actions.

### `laps off`
Remove either legacy or versioned `<laps-instructions>` blocks from `AGENTS.md`,
`CLAUDE.md`, and `GEMINI.md`.

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
`delete`, `prune`, `move`, `edit`, `assign`, plus `status` and `log`) accept
three mutually exclusive scope flags that select *which* layer a command
targets:

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

Raw file targets are fail-closed: read and flow commands exit `3` when the
selected file does not exist and do not create it. `add` may create a missing
file because creation is explicit in that verb; `init` continues to create the
default store. When a raw target resembles an existing stint, the error
suggests the safer `--stint <name>` form. Prefer scope flags over `--file` for
normal queue and stint work.

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

A stint with no todo laps left is **drained**: the draining operation flips the
reference in its immediate parent queue to done, moves the file to
`.laps/stints/archive/`, and cascades toward root while parent stints also have
no todo laps left. Draining is content-based and position-independent — a
preempted, non-head stint still drains when its last lap completes, and a done
ref is skipped on later advance. `done undo` scans all queue files (root,
active stints, and the archive) for the globally latest completion and
unarchives when that lap lives in a drained stint (the 5-minute age gate still
applies).

### Holding a stint (gating)

A **held** stint pauses the pipeline at stint granularity. Mark a non-archived
stint held with `laps stints hold <name>` and resume it with
`laps stints release <name>`. The held flag lives on the stint file metadata,
defaults to `false` when absent, and is **orthogonal to lifecycle** — a stint
can be held-and-queued, held-and-active, or held-and-done, and `stints ls`
shows the held marker alongside the `queued`/`archived` lifecycle value.

The flag only takes effect once a reference to the held stint is encountered
at the **current context head** during flow resolution. A held stint deeper in
the pipeline (not yet at the head) has no effect until descent reaches it.

- **Hold blocks starting, not finishing.** A hold gates `get`/`claim`
  (starting the next lap) but never `done` for the already-claimed lap. An
  agent mid-lap can always finish and record, even while the stint is held.
- **Final-lap drain still wins.** Completing the claimed final lap of a held
  stint runs normal drain/archive behavior, so a drained held stint never
  becomes a permanent gate.
- **Explicit ids.** `get <id>` may inspect a lap inside a held stint (with a
  warning); `claim <id>` into a held stint is blocked — it exits `10`, leaves
  the claim unchanged, and warns.
- **Only `get`/`claim` flow-start are gated.** `list`, `count`, `add`,
  `edit`, `assign`, and `delete` operate normally inside or under a held
  stint: no exit-code change, no held warning, no mutation block.

Held interactions warn on stderr:

```
laps: stint <scope> is held; do not implement laps in it yet.
```

`stint.held` and `stint.released` events are appended to the event log, but
**only when the state actually changes** — holding an already-held stint (or
releasing an already-released one) is idempotent and does not double-log.

### Queue-state exit codes

Head/flow `laps get` and `laps claim` signal queue state through their exit
code, replacing the prior single `3` ("no head task") for the empty/complete
head cases. The codes are chosen to avoid the existing `2`/`3`/`4` failure
codes:

| Exit | Meaning for head `get`/`claim` |
|------|--------------------------------|
| `0`  | A lap was returned/claimed. |
| `10` | The head is **held** (gated by a held stint). |
| `11` | The queue is **empty** — nothing was ever enqueued (resolves to zero todo). |
| `12` | The queue is **complete** — every lap resolvable from the root head is done and nothing enqueueable remains. |

- `11` (empty) and `12` (complete) apply only to head/flow operations that are
  not targeting an explicit id. `10` also applies to an explicit
  `claim <id>` attempt into a held stint.
- Explicit-id **not found** remains exit `3`; store/io failures remain `2`;
  hook failures remain `4`.
- Text mode emits **no command stdout** for `10`/`11`/`12`; held cases warn on
  stderr. Hook passback, if configured, may still write its own stdout. JSON
  mode emits a small queue-state object on stdout instead:
  ```json
  {"state":"held","exitCode":10}
  ```
  where `state` is `held`, `empty`, or `complete` matching the code.
- After-hooks still observe the final exit code (clean-state exits are routed
  through the normal exit path), so any consumer keying off `$exit_code` sees
  `10`/`11`/`12`.

`laps status` reports the same taxonomy (a primary `held` state when the head
is gated and no active claim takes precedence) but always exits `0` for valid
snapshots — see [`laps status`](#laps-status).

### Schema version
The on-disk schema reaches **v3**: queue entries gain a `kind` discriminator
(`lap` by default, `stint` for refs), existing entries are migrated to
`kind:"lap"`, and non-archived stint file metadata carries a `held` boolean
(defaulting to `false` when absent). An entry with no `kind` is treated as a
`lap`, and a stint file with no `held` field is treated as not held, so older
data files remain readable. The binary `VERSION` is **`0.9.0`**, so a v3 build
reports `0.9.0` and writes v3 data.

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

If a hook declares a `command` that does not match a built-in laps command
(e.g. `worktree`), you can invoke it directly:

```bash
laps worktree
```

laps will fire every matching `before` and `after` hook and nothing else. The
custom name must be declared by at least one entry in `.laps/hooks.json`;
undeclared names fail with a non-zero `unknown command` error.

### Command-name compatibility guidance

The following names may become built-in commands in the future and should be
avoided for hook-only commands:

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

## Consumer contract

Consumers (orchestrators such as **Rally**, scripts, other tooling) that read
laps output or `.laps/` files programmatically may rely on exactly four pinned
surfaces. Everything else — the default `laps list` rendering, `laps status`,
and `laps log` output — is **operator-facing and not stable**; it may change
in any release without a consumer migration.

The pinned surfaces:

1. **`laps list --oneline`** — exactly one non-blank line per lap, in the form
   `<n>. <id> — <title>`, with ` (assignee: <a>)` appended only when an
   assignee is set. See [`laps list`](#laps-list---all----done---tree).
2. **`.laps/claim`** — a JSON object `{"lap","file","scope","claimedAt"}`
   where `lap` is the claimed lap id; `scope`/`claimedAt` may be omitted. A
   legacy bare-id claim file is still read back-compatibly. See
   [Claim File Format](#claim-file-format).
3. **`get`/`claim` task-detail stdout** — line 0 is the title, an optional
   `Assignee: <name>` line follows, then a blank line, then the description.
   Stint resolution is transparent: a lap served from a stint renders
   identically to a root-queue lap.
4. **`get`/`claim` queue-state exit codes** — `0` lap returned, `10` head
   held, `11` queue empty, `12` queue complete; `2` store/io, `3` explicit id
   not found, and `4` hook failure are unchanged. See
   [Queue-state exit codes](#queue-state-exit-codes).

**Version-gating rule:** a change that alters any pinned surface must bump
`VERSION`, update this contract section (and the consumer-contract spec), and
record the consumer impact in its proposal. Changes to operator-facing output
carry no such obligation — consumers must use the pinned machine forms, never
parse the default `list`/`status`/`log` renderings.

## Orchestrator & Rally coordination

`laps` is driven by an orchestrator relay loop (e.g. **Rally**): a loop of
`get`/`claim` → run agent → `done`. **0.9.0 makes a deliberate, breaking
contract change to that loop's exit-code signals.**

### Contract change: `get`/`claim` exit codes

Before 0.9.0, head `get`/`claim` exited `3` on an empty/complete queue
("no head task"). 0.9.0 replaces that single signal with distinct
queue-state exit codes so an orchestrator can tell **gated** from **empty**
from **complete** instead of guessing:

| Old (≤ 0.8.1) | New (0.9.0) | Meaning |
|---------------|-------------|---------|
| `3` | `10` | Head is **held** — stop on the gate, do not start work. |
| `3` | `11` | Queue is **empty** — idle, wait for work to be enqueued. |
| `3` | `12` | Queue is **complete** — the pipeline is finished, stop. |
| `0` | `0`  | A lap was returned — run it. |
| `3` (explicit id not found) | `3` | Unchanged. |
| `2` (store/io), `4` (hook) | `2` / `4` | Unchanged. |

Relay loops that previously branched on "exit `3` ⇒ nothing to do" **must** be
updated to handle `10`/`11`/`12` distinctly. In JSON mode the same signal is
`{"state":"held|empty|complete","exitCode":N}` on stdout, and `laps status`
exits `0` with the same state in `state` (plus a `gate` object when held) for
any consumer that prefers parsing over exit codes.

### Operator action required (external to this repo)

The Rally-side relay-loop update is **not** part of this repository. Operators
coordinating the 0.9.0 release **must**:

1. Update the Rally relay loop to branch on `10`/`11`/`12` (or to read
   `laps status`/the JSON queue-state object) before upgrading laps to 0.9.0.
2. Treat `held` (`10`) as a hard stop on starting new laps while a stint is
   gated — finishing the already-claimed lap remains valid.
3. Ship the updated relay loop in lockstep with the 0.9.0 laps binary, since
   the empty/complete cases no longer surface as `3`.

This section flags the coordination need; the relay-loop implementation itself
lives outside this repo.

## Versioning

Version is tracked in `VERSION` at the repo root. Pushing a change to `VERSION` on `main` triggers an automated release:

1. `auto-tag` workflow reads the new version from `VERSION`, creates a `vX.Y.Z` git tag, and dispatches `release.yml`.
2. `release` workflow runs [GoReleaser](https://goreleaser.com) to build binaries for Linux, macOS, and Windows, then publishes a GitHub Release with checksums and the install script.

See `.github/workflows/auto-tag.yml` and `.github/workflows/release.yml` for details.
