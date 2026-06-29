## Why

Stints auto-advance by default (a pipeline for free). But we want to **pause** the pipeline
at a chosen stint until an operator releases it — "auto, but pausable" — and give Rally clean
signals to tell **gated** apart from **empty** and **complete**, so the relay loop can decide
whether to stop, wait, or finish.

## What Changes

- **`laps stints hold <name>` / `laps stints release <name>`** — mark a stint held / clear the
  hold. The held flag travels with the stint and takes effect when the stint reaches the
  resolved head.
- **Gated flow ops** — when the resolved head is a held stint, `get`/`claim` return **no lap**
  (a clean stop, not an error) instead of descending.
- **Queue-state exit codes** on `get`/`claim`: `0` lap returned, `10` held/gated, `11` empty,
  `12` complete.
- **Hold blocks starting, not finishing** — `done` for the claimed lap still succeeds while a
  stint is held.
- **`status` + `stints ls`** surface the gate: a `held` state, the held stint, and the gate
  message; `status` still always exits 0.
- **Log events** `stint.held` / `stint.released`.

## Capabilities

### Added Capabilities
- `stint-gating`: holding/releasing a stint, gated flow-op behaviour, the queue-state exit
  codes, and the `held` state surfaced by `status`.

## Impact

- **Code**: a held flag on stints (`internal/store`); resolution short-circuits at a held head
  with the new exit codes (`get`/`claim`); `done` for the claimed lap is unaffected; `status`
  and `stints ls` render the held state; `stint.held`/`stint.released` log events.
- **Behavior**: `get`/`claim` exit codes change from `3` (empty today) to `10`/`11`/`12` — a
  contract change the Rally relay loop must adopt; version-bump-worthy.
- **Coordination**: depends on `add-stints` (3a) and on `add-event-log-and-status` (`status`,
  `--json-output`, the event log).
- **Out of scope**: auto-hold heuristics (manual hold only); per-lap holds (stint-level only).
