# Draft — add-stints (3a, stints core)

> Pre-formalisation notes. Captures the agreed shape before writing proposal/design/tasks/specs.
> Status: draft. Change #3a of four. Formalise in order 1 → 2 → 3a → 3b.

## Why

Today `.laps/laps.json` is a single flat queue, drained one change at a time. We want to
prepare laps for **multiple** OpenSpec changes ahead of time (one prepared queue per change)
and pull them into execution as a pipeline — without agents or Rally seeing any change.

`prepare-laps` (in the sibling `rally` repo) will write one **stint** per change; the operator
enqueues it when ready.

Storage already supports multiple files via the existing `--file`/`-f` flag (`ResolveFile`
maps `-f auth` → `.laps/auth.json`). So this is mostly an **ergonomics + resolution layer**
over storage that already works — low risk.

## Core model

- `laps.json` stays **canonical**. A queue entry is either a **lap** or a **stint-ref** that
  points to `.laps/stints/<name>.laps.json`. The queue may hold laps, stint-refs, or any mix.
- A stint file is itself a queue (same `File` schema). Drained/done stints are moved to
  `.laps/stints/archive/`.
- The **active context** = whatever stint-ref(s) sit on the path from the resolved root head
  down to the current head lap. **Derived from head position, never stored** (no `.laps/active`
  file) — kills a class of desync bugs.

## In scope

- **Schema v3**: `kind` discriminator on queue entries (`lap` default, `stint`). Back-compat:
  no `kind` ⇒ lap. Migration v2→v3 stamps `kind:"lap"` on existing entries, bumps version.
  Stint-ref reuses the `Task` envelope + `kind:"stint"` + `ref` (stint name); `title` is display.
- **Read-through resolution** for flow ops (`get`/`claim`/`done`/`list`): descend from root head
  through active stint-refs to the first real lap. Recursive (nesting-ready). **Invisible to
  agents** — `get` still returns title + description of a lap; the agent never knows it's nested.
- **Scope flags** (shared local, mutually exclusive, default `--active`), uniform on queue-targeting verbs:
  - `--active` / `-c` — deepest active context, recursive (the default).
  - `--root` / `-r` — root file, no descent.
  - `--stint <name>` / `-s` — named stint file, no descent.
  - Mostly sugar over `-f` (`--stint x` ≈ `-f stints/x`); `--active`/`--root` add descend / no-descend
    semantics `-f` can't express. `-f` is incompatible with scope flags. `-r/-s/-c` chosen as the only
    foreclosure-free shorthands (`-t`→title, `-a`→assignee/all, `-d`→description are protected).
  - Division of use: agents use bare verbs (implicit `--active`); `prepare-laps` uses long forms
    so the planner holds an explicit structural model.
- **Structure ops** (`add`/`move`/`edit`/`assign`/`delete`) default to active scope; `-r`/`-s` redirect.
  Explicit-id resolution: within scope; if not found in scope but found elsewhere, the error
  **names the stint** ("`a7` is in stint `search` — re-run with `-s search`"). No silent cross-file action.
- **Claim records scope** (preemption-correctness): claim file becomes `{lap, scope, claimedAt}`.
  Bare `done` resolves the claimed lap within its **recorded scope**, not the current head — so it's
  immune to head changes from preemption or another session. Invariant: a claimed *undone* lap keeps
  its stint from draining, so the recorded scope file always still exists when `done` looks.
  (`claimedAt` itself is introduced in change 2; this change adds `scope`.)
- **Drain → auto-archive**: a stint with no remaining todo laps is drained. Detected inline by the
  `done` that empties it (and at resolution time for robustness). On drain: stint-ref flips to done
  (sets `completedAt`) **and** the stint file auto-moves to `.laps/stints/archive/`. Draining is
  content-based and position-independent — a preempted, non-head stint still drains when its last
  lap completes; the done stint-ref is later skipped on advance.
- **Auto-activation**: stint-refs activate purely by reaching the resolved head — no explicit step.
- **`enqueue`**: `laps stints enqueue <name> [head|tail|after <id>]`, default **tail** (planned-work
  order), reuses `ComputeInsertOrder` at the root layer. `head` **preempts** the active stint
  non-destructively (its partial progress is safe in its file; resumes when the interloper drains).
  `enqueue` always targets root (it manages the root pipeline).
- **Commands**: `laps stints ls | new <name> | enqueue <name> [pos] | show <name> | rm <name>`;
  `st` alias for `stints`. `stints ls` shows each stint's lap counts and whether it is queued.
  `--tree` render flag on `list` for the full recursive overview.
- **Log integration** (infra from change 2): events carry `scope`; add `stint.enqueued`,
  `stint.completed`/`stint.archived`.
- **Status integration** (status from change 2): report active stint + per-stint progress.
  (`held` state is 3b.)

## Out / deferred

- Hold / release, gate exit codes, `held` state → **3b** (`add-stints-gating`).
- Cross-layer moves (lap between stints, stint↔root) — within-layer reorder only here.
  Future `move <id> --to-stint <name>` / `--to-root`.
- Nested-stint **creation** tooling (enqueue into a stint). Engine recurses (nesting-ready), but
  creation stays flat: `enqueue` → root only.
- Color / TTY theming.

## Dependencies

- Builds on **change 2** (event-log infra + `status` + structured claim file with `claimedAt`).
  Formalise/ship after 2. Hard requirement met here: **claim-scope** is needed for preemption
  correctness, so it lands in this change.
- Schema v2→v3 migration; version-gate messaging for older binaries reading v3.

## Open questions (for formalisation)

- Exact claim-file JSON shape + back-compat parsing (shared with change 2's `claimedAt`).
- `laps stints ls` columns beyond name, queued flag, and lap counts.
- Whether `status` extension and `list --tree` fully land here or are minimal.
- ID generation across stint files: prefix is repo-based (identical across stints), so cross-stint
  collisions are rare-but-possible; confirm generation scans only the target file's ids and that
  scope resolution disambiguates.
- Behaviour of `delete`/`rm` on a stint-ref that still has todo laps (cascade? refuse?).
