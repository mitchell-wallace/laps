spec - the more formal document for v0.1.0
---
# Microbeads (`mb`) — Spec

## Intent

A minimal, single-binary task tracker for AI coding agents. Tasks are a flat ordered queue with two states (todo / done). The agent's contract is simple: read the head, do the work, mark it done. No box drawing, no JSON output, no persistent process, no statuses beyond done/not-done.

Microbeads is **not** a place to track messy human-side feature ideas. It is a place to encode the next concrete unit of work an agent should pick up. Humans and agents add tasks; the agent always works the head.

## Non-goals

- Blocked / in-progress / pending statuses. If something blocks the head, write the unblock work as a new head task.
- Parallel work. Microbeads is single-threaded — one head, one current task.
- Search, tags, priorities, labels, estimates, dependencies graph.
- A daemon, server, web UI, sync, or remote storage.
- Built-in awareness of any external tool (including [rally](https://github.com/mitchell-wallace/rally)). Integration is purely via hooks.

## Storage

- Default store: `.beads/mb.json` at the repo root.
- Discovery: walk up from cwd until a `.beads/` directory is found. Stop at a `.git` directory; if none of the ancestors have `.beads/`, create `.beads/` next to the `.git`. If neither exists by the filesystem root, error.
- Multi-file: any `*.json` under `.beads/` other than `mb-hooks.json` (and a future `config.json`) is a valid task file. `-f <name>` selects one. `name` may be given with or without the `.json` extension; do not double-append.
- If `mb.json` is absent or has zero tasks, and other task files exist in `.beads/`, commands that need a store print a hint listing the candidate files and ask the user to pass `-f`. This forces named chains in monorepos: a missing default is a feature, not a bug.

### Task record

```json
{
  "id": "micr-a3f2",
  "title": "Add list command",
  "description": "Multi-line\ndescription supported.",
  "assignee": "alice",
  "isDone": false,
  "createdAt": "2026-04-28T10:15:00Z",
  "updatedAt": "2026-04-28T10:15:00Z",
  "completedAt": null
}
```

### File format

```json
{
  "version": 1,
  "tasks": [ /* ordered list of task records, head first */ ]
}
```

Order in the array IS the queue order. `mb add head` prepends, `mb add tail` appends, `mb add after <id>` inserts after that id. Done tasks remain in place until pruned.

### ID generation

`<prefix>-<hash>` where:
- `prefix` = first 4 lowercase alphanumeric chars of the repo-root folder name (pad with `x` if shorter, strip non-alnum).
- `hash` = first 4 chars of a hex digest of `title | createdAt | description[:200]`.
- On collision within the same file, extend the hash slice by one char until unique.

Hash function: SHA-256, lowercase hex. Cheap and deterministic.

## Commands

All commands accept `-f <name>` to target a non-default file.

`assignee` is optional. When omitted or blank, it is not written to the file and is not shown in command output.

### `mb add <head|tail|after> [id] (--title T [--description D] [--assignee A] | --json '{...}')`

- Position is **required**. With no position, print:

      mb add: position required (head, tail, or after <id>)

- `head` and `tail` take no id; `after` requires `<id>` immediately following.
- Input is either:
  - `--title T [--description D] [--assignee A]` — description and assignee optional; description supports `\n` and real newlines.
  - `--json '{"title": "...", "description": "...", "assignee": "..."}'` — a flat object. `description` and `assignee` are optional.
- The two input modes are mutually exclusive. Mixing them is an error.
- Prints the new task's id on success.

### `mb get [head|<id>]`

- Default target is `head`.
- Output: title, optional `Assignee: ...`, blank line, description. No id or timestamps. The agent has the id from `list` or from the call that created it.
- If the store is empty (no todo tasks for `head`), print `no head task` and exit non-zero so hooks/scripts can branch on it.

### `mb list [--all | --done]`

- Default: todo tasks only, head first, as a markdown numbered list:

      1. micr-a3f2 — Add list command
      2. micr-9b1c — Wire up hooks dispatcher (assignee: alice)

- `--all` includes done tasks (done shown after todo, struck through with `~~...~~`).
- `--done` shows only done tasks, most recently completed first.

### `mb done`

- Completes the head todo task. No id form. If there is no head todo task, print `no head task` and exit non-zero.
- Sets `isDone=true`, `completedAt=now`, `updatedAt=now`. The task stays in place in the array; the next todo task becomes the new head.
- Prints the completed task's id.

### `mb delete <id>`

- Removes the task entirely from the file (todo or done).

### `mb prune [N]`

- Retains the `N` most recently completed done tasks (by `completedAt` descending). Default `N` is 20. `mb prune 0` deletes all done tasks. Todo tasks are never touched.
- Prints the number of tasks removed.

### `mb on`

- Writes a `<mb-instructions>...</mb-instructions>` block into `AGENTS.md` (creating it if absent).
- Additionally writes the same block into `CLAUDE.md` and `GEMINI.md` **if those files already exist**. Never creates them.
- Idempotent: if the tagged block is already present, replace it; never duplicate.
- Block content (subject to refinement):

      <mb-instructions>
      This project uses microbeads (`mb`), a minimal task tracker.
      - `mb get head` — read the next task. Title and description only.
      - `mb list` — see the queue.
      - `mb done` — when you finish the head task. You MUST run this; do not skip.
      - `mb add head|tail|after <id> --title ...` — add a task. Use `head` if it must be done before the current head; otherwise `tail`.
      - If you hit a blocker that prevents finishing the head task this session, add the unblock work to `head` and stop.
      - Commit after each `mb done` unless the user said otherwise.
      </mb-instructions>

### `mb off`

- Removes the `<mb-instructions>` block from `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` (whichever contain it). Leaves the rest of the file untouched. If a file becomes empty after removal and `mb on` had created it, leave it (don't delete user files we may have created).

## Hooks

File: `.beads/mb-hooks.json`. Optional. Schema:

```json
{
  "version": 1,
  "hooks": [
    {
      "title": "Commit on done",
      "description": "Auto-commit completed task.",
      "command": "done",
      "when": "after",
      "run": "git add -A && git commit -m \"done: $title ($id)\"",
      "passback": false
    }
  ]
}
```

### Dispatch rules

- `command`: any string. If it matches a real mb command (`add`, `done`, `get`, ...), the hook fires when that command runs. If it doesn't match a real command (e.g. `worktree`), then `mb worktree` is a "hook-only" invocation: mb itself does nothing but fires every hook bound to that name.
- `when`: `"before"` or `"after"`.
- `before` hooks run before the mb command body. A non-zero exit aborts the mb command and surfaces the hook's stderr.
- `after` hooks always run, even on mb command failure. Their `$exit_code` reflects the mb command's exit.
- Hooks bound to the same `(command, when)` run in array order.
- `passback: true` — the hook's stdout is appended to mb's stdout so the agent reads it. `false` — hook output is silently discarded (stderr is always shown).

### Variables in `run`

Standard shell-style `$var` substitution (also `${var}`):

- `$id`, `$title`, `$description`, `$assignee` — the task in question. For `done`, the just-completed head. For `add`, the newly created task. For `get`, the fetched task. For commands without a single relevant task (`list`, `prune`, `on`, `off`, hook-only commands), these are empty strings.
- `$command` — the mb command name (`done`, `add`, ...).
- `$exit_code` — the mb command's exit code (`after` hooks only; empty for `before`).
- `$output` — the mb command's stdout (`after` hooks only).
- `$file` — the resolved task file path (after `-f` resolution).

`run` is executed by the user's shell (`/bin/sh -c` on unix, `cmd /C` on windows).

## Rally relationship

`mb` is rally-agnostic and does not import, link to, or shell out to rally. Rally's installer is responsible for writing `mb-hooks.json` entries that call `rally robots ...`. This keeps the dependency one-way: rally knows about mb; mb does not know about rally.

## Distribution

- Single static Go binary `mb`.
- Released via goreleaser. Targets: linux/{amd64,arm64}, darwin/{amd64,arm64}, windows/{amd64,arm64}.
- An install script (`install.sh`, served from the repo) detects OS/arch, downloads the latest release tarball, extracts `mb` to `~/.local/bin/mb` (creating the dir), and ensures `~/.local/bin` is on `PATH` by appending an export to the user's shell rc (`~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish`) only if missing. Idempotent — safe to re-run for upgrades.
- A windows install script (`install.ps1`) places `mb.exe` in `%LOCALAPPDATA%\Programs\mb\` and updates the user `PATH` env var.
- `mb --version` prints the build version (set via goreleaser ldflags).

## Errors and exit codes

- `0` — success.
- `1` — usage error (bad flags, missing required arg, mutually exclusive inputs).
- `2` — store error (no `.git` ancestor, file unreadable, hooks file malformed).
- `3` — empty-state error (`mb done`/`mb get head` with no head task; `mb get <id>` with unknown id).
- `4` — hook abort (a `before` hook returned non-zero).

Errors are written to stderr as `mb: <message>`; nothing else.

## What we are explicitly NOT building

- `mb init` / `mb onboarding` — replaced by `on`/`off`.
- `mb worktree` as a built-in — it is the canonical hook-only example, not a real command.
- Any output mode flag (`--json`, `--plain`). The output IS markdown/plain.
- Config file. If we ever need one, it goes at `.beads/config.json`; for v1 there is no config.
