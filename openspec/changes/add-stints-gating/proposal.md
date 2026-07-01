## Why

Stints auto-advance by default (a pipeline for free). But we want to **pause** the pipeline
at a chosen stint until an operator releases it — "auto, but pausable" — and give Rally clean
signals to tell **gated** apart from **empty** and **complete**, so the relay loop can decide
whether to stop, wait, or finish.

## What Changes

- **`laps stints hold <name>` / `laps stints release <name>`** — mark any non-archived stint
  held / clear the hold, even before it is enqueued. The held flag lives with the stint file and
  takes effect when a reference to that stint is encountered during flow resolution.
- **Gated flow ops** — when resolution sees a held stint at the current context head,
  head `get`/`claim` return **no lap** (a clean stop, not an error) instead of descending, and
  warn that the stint is held and should not be implemented yet. Explicit `get <id>` may inspect
  a held stint with the same warning; explicit `claim <id>` into a held stint is blocked.
- **Queue-state exit codes** on `get`/`claim`: `0` lap returned, `10` held/gated, `11` empty,
  `12` complete.
- **Hold blocks starting, not finishing** — `done` for the claimed lap still succeeds while a
  stint is held.
- **`status` + `stints ls`** surface the gate: a `held` state, the held stint, and the gate
  message; `status` still exits 0 for valid snapshots.
- **Log events** `stint.held` / `stint.released`.
- **Nested drain hardening** — before layering holds on top of drain, fix the `add-stints`
  nested-drain gap: completing the final lap of a nested stint updates the parent queue's stint
  ref (not only root), archives the child file, and cascades upward when parent stints become
  empty.

## Capabilities

### Added Capabilities
- `stint-gating`: holding/releasing a stint, gated flow-op behaviour, the queue-state exit
  codes, and the `held` state surfaced by `status`.

## Impact

- **Code**: nested-drain parent-chain/cascade support in the resolver/drain path; a held flag on
  stints (`internal/store`); resolution short-circuits at a held head with the new exit codes
  (`get`/`claim`); `done` for the claimed lap is unaffected; `status` and `stints ls` render the
  held state; `stint.held`/`stint.released` log events.
- **Behavior**: `get`/`claim` exit codes change from `3` (empty today) to `10`/`11`/`12` — a
  contract change the Rally relay loop must adopt; version-bump-worthy.
- **Coordination**: depends on `add-stints` (3a) and on `add-event-log-and-status` (`status`,
  `--json-output`, the event log).
- **Out of scope**: auto-hold heuristics (manual hold only); per-lap holds (stint-level only).
