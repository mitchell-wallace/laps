## Context

Laps drains one flat queue at a time. The operator prepares work per OpenSpec change and
wants several changes staged and pulled in as a pipeline, while agents keep seeing the same
"read head → do it → mark done" contract. Storage already supports multiple files (`-f`), so
the work is a resolution + ergonomics layer, not new storage.

A **stint** is a prepared queue for one change, stored at `.laps/stints/<name>.laps.json`.
`laps.json` stays canonical and may hold laps, **stint references**, or any mix. The active
context is whatever stint ref sits on the path from the resolved root head down to the head
lap — **derived from queue position, never stored** (no active-pointer file), which removes a
class of desync bugs.

## Goals / Non-Goals

**Goals:**
- `laps.json` canonical; a queue entry is a lap or a stint ref (schema v3, back-compat).
- Flow ops descend to the deepest active stint, invisibly to agents.
- One consistent scope vocabulary (`--active`/`--root`/`--stint`) across queue-targeting verbs.
- Preemption-safe completion: claims survive head changes.
- Auto-advancing pipeline with auto-archive of drained stints.

**Non-Goals:**
- Hold / release / gating / exit codes (`add-stints-gating`).
- Cross-layer moves (lap between stints, stint↔root) — within-layer reorder only.
- Nested-stint **creation** UX — the engine recurses, but `enqueue` targets root only.
- Color / TTY theming; the TUI.

## Decisions

- **Schema v3 / `kind`.** Entries gain `kind` (`lap` default, `stint`); missing `kind` ⇒ lap.
  Migration stamps `kind:"lap"` and bumps the version. A stint ref reuses the entry envelope
  plus `ref`, ordered by the same order key so `enqueue head|tail|after` is just
  `ComputeInsertOrder` at the root layer.
- **Read-through resolution.** Flow ops start at the root head and descend through active
  stint refs to the first lap; recursive for nesting. Agents see a lap's title/description and
  never know it was nested.
- **Scope flags.** Shared local flags on queue-targeting commands, mutually exclusive, default
  `--active`. `-r`/`-s`/`-c` are the
  only foreclosure-free shorthands (`-t`→title, `-a`→assignee/all, `-d`→description stay free
  for their natural owners). Mostly sugar over `-f`; `--active`/`--root` add the descend /
  no-descend semantics `-f` can't express. Raw `--file` is mutually exclusive with scope flags
  so one invocation has only one target model. Agents use bare verbs (implicit `--active`);
  `prepare-laps` writes the long forms for an explicit structural model.
- **Scoped id resolution.** Explicit-id structure ops resolve within the selected scope; if the
  id lives in another stint, the error names it ("`a7` is in stint `search` — re-run with
  `-s search`"). No silent cross-file action.
- **Canonical scope strings.** Logical scope values use slash paths: `root` for the root queue,
  `auth` for a root-level stint, and `auth/search` for nested stints. Claims, events, status,
  hook variables, and resolver visited keys SHALL use the same encoding.
- **Claim records scope (preemption-safety).** With literal `enqueue head` preemption, a bare
  lap-id claim could complete against the wrong context. So the claim becomes
  `{lap, file, scope, claimedAt}` and bare `done` resolves the claimed lap within its **recorded**
  scope. Invariant: a claimed, undone lap keeps its stint from draining, so the recorded scope
  file always still exists when `done` looks — auto-archive can never pull the rug out.
- **Deleting claimed laps.** `delete` refuses to remove a claimed lap by default, warning on
  stderr. `delete --force <id>` removes it and clears the matching claim; the forced mutation is
  explicit because it can discard in-flight work.
- **Drain → auto-archive.** A stint with no todo laps is drained; the draining operation flips
  its ref to done and moves the file to `.laps/stints/archive/`. Draining is content-based and
  position-independent — a preempted, non-head stint still drains when its last lap completes,
  and the done ref is skipped on later advance.
- **`done undo` unarchives when needed.** If the most recent completion being undone was inside
  a drained-and-archived stint, `done undo` restores the archived stint file to `.laps/stints/`,
  reopens the root stint ref, and then reopens the lap within the normal undo rules.
- **Enqueue / preemption.** Default tail (planned-work order). `head` preempts the active stint;
  because each stint's progress lives in its own file, preemption is non-destructive and the
  paused stint resumes when the interloper drains.
- **Stint listing includes unqueued files.** `stints ls` lists stint files, shows lap counts for
  each, and shows whether each stint is currently queued. Empty and unqueued stints are ordinary
  stint files with no special state beyond their counts and queued flag.
- **`stints rm` safety.** `stints rm <name>` removes unqueued non-archived stint files and
  archived stints, including archived stints that still have a done root ref. It refuses
  non-archived queued, active, or claimed stints unless `--force` is supplied; forced removal also
  removes matching root refs and clears matching claims.
- **Nesting.** The resolution engine and drain cascade are depth-agnostic (honoring "even if
  stints nest"), but creation tooling stays flat in this change — `enqueue` targets root only.
- **Integration.** Events carry the resolved `scope` and add `stint.enqueued`/`completed`/
  `archived` (the event log already defaults `scope` to `root`); `status` reports the active
  stint and per-stint progress; `list --tree` renders the full recursive overview. Hooks under
  scoped operations receive `$file` as the resolved physical task file and a new `$scope` value
  using the canonical scope string.

## Implementation Contracts

- **Schema loading order.** Before adding strict `kind`/`ref` fields, load enough top-level
  envelope data to reject newer schema versions with the existing clear "update laps" message;
  then run v1->v2 ordering migration and v2->v3 `kind:"lap"` stamping in order.
- **Stint names are file names, not paths.** `stints new`/`enqueue`/archive SHALL reject blank
  names, path separators, `.`/`..`, and names that would collide with an existing active or
  archived stint file. Archive moves SHALL be no-overwrite.
- **Resolver failures are classified.** Resolution SHALL keep a visited set for stint refs and
  fail deterministically for cycles, missing child files, malformed stint refs, and malformed
  child files instead of looping or silently skipping a ref.
- **Command scope matrix.** Queue data commands accept scope flags only when they operate on a
  queue snapshot or queue entry: `add`, `get`, `claim`, `done`, `list`, `count`, `delete`,
  `prune`, `move`, `edit`, and `assign`. Admin/readers without a queue target (`init`, `on`,
  `off`, `update`, `version`, `help`, hook-only commands, `log`, and `status`) do not accept
  these flags unless their own change explicitly adds scope-specific filters.
- **Explicit id resolution.** Every id-taking queue operation (`get <id>`, `claim <id>`,
  `done <id>`, `add after <id>`, `move`, `edit`, `assign`, and `delete`) resolves ids inside
  the selected scope first; if the id exists in another stint, the error names that stint and
  does not mutate any file. `stints enqueue after <id>` is root-queue structural work: it
  resolves the `after` id only in root, and if the id exists only in a stint it fails naming
  that stint.

## Risks

- The resolution layer touches every flow op; the mitigations are that active context is
  derived (not stored) and that claims pin their scope, so head changes can't misroute `done`.
- Schema migration is one-way (v2→v3); older binaries reading a v3 file are rejected with a
  clear "update laps" message (existing version-gate behaviour).
