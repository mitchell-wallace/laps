## Why

`.laps/laps.json` is a single flat queue, drained one change at a time. We want to prepare
laps for **multiple** OpenSpec changes ahead of time — one prepared queue per change — and
pull them into execution as a pipeline, **without agents or Rally seeing any change**.
`prepare-laps` (in the sibling `rally` repo) will write one **stint** per change; the
operator enqueues it when ready.

Multi-file storage already exists via the `--file`/`-f` flag (`ResolveFile` maps `-f auth`
→ `.laps/auth.json`), so this change is mostly an **ergonomics + resolution layer** over
storage that already works — low risk.

## What Changes

- **Schema v3** — a `kind` discriminator on queue entries (`lap` default, `stint`).
  Back-compat: an entry with no `kind` is a lap. Migration v2→v3 stamps `kind:"lap"` on
  existing entries. A stint reference reuses the entry envelope plus `ref` (stint name).
- **Stint files** at `.laps/stints/<name>.laps.json` (same schema); archived stints under
  `.laps/stints/archive/`.
- **Read-through resolution** — flow ops (`get`/`claim`/`done`/`list`) descend from the root
  head through active stint refs to the first real lap. Recursive (nesting-ready) and
  **invisible to agents**.
- **Scope flags** on queue-targeting verbs, mutually exclusive, default `--active`: `--active`/`-c`
  (deepest active, recursive), `--root`/`-r` (root, no descent), `--stint <name>`/`-s`
  (named stint, no descent).
- **Scoped structure ops** — `add`/`move`/`edit`/`assign`/`delete` default to the active scope; an
  explicit id resolves within scope, and when it lives elsewhere the error names the stint.
- **Claim records scope** — bare `done` completes the claimed lap within its recorded scope,
  immune to head changes from preemption or another session.
- **Enqueue** — `laps stints enqueue <name> [head|tail|after <id>]`, default tail; `head`
  preempts the active stint non-destructively (its progress resumes from its file).
- **Drain → auto-archive** — a stint with no todo laps left flips its ref to done and its file
  moves to `.laps/stints/archive/`, in the operation that drains it.
- **Commands** — `laps stints ls|new|enqueue|show|rm`, `st` alias, and `list --tree`.
- **Integration** — stint events (`stint.enqueued`/`completed`/`archived`) and a populated
  `scope` in the event log; active-stint and per-stint progress in `laps status`.

## Capabilities

### Added Capabilities
- `stints`: the stint subsystem — schema v3 stint refs, stint files, `laps stints` commands,
  enqueue/preemption, drain/auto-archive, and stint reporting.
- `scope-resolution`: how commands target a layer — read-through flow resolution, the
  `--active`/`--root`/`--stint` scope flags, scoped explicit-id resolution, and claim-scope.

## Impact

- **Code**: schema v3 (`internal/store` — `kind`, stint-ref, migration); a resolution layer
  used by `get`/`claim`/`done`/`list`; shared scope flags on queue-targeting commands;
  `internal/cmd/stints.go`; `claim.go` (scope field); drain + auto-archive; `list --tree`;
  event-log scope/events; `status` extension.
- **Behavior**: `laps.json` may contain stint refs; agents are unaffected (bare verbs
  descend); operators gain scope flags and `laps stints`.
- **Coordination**: depends on `add-event-log-and-status` (event-log infra, `status`, and the
  structured claim with `claimedAt`); this change adds the `scope` field to the claim. It also
  depends on `improve-cli-ergonomics` if `move` and `edit` remain in this change's scoped
  structure-ops contract.
- **Out of scope**: hold/release, gate exit codes, the `held` state (`add-stints-gating`);
  cross-layer moves; nested-stint **creation** tooling (the engine recurses, but creation
  stays flat — `enqueue` targets root); color/TTY theming and the TUI.
