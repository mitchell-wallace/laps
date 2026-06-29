# Draft — add-stints-gating (3b)

> Pre-formalisation notes. Status: draft. Change #3b of four. Depends on 3a.

## Why

Stints auto-advance by default (pipeline for free). But we want to **pause** the pipeline at a
chosen stint until an operator releases it — "auto, but pausable" — and give Rally clean signals
to tell **gated** apart from **empty** and **complete**, so the relay loop can decide whether to
stop, wait, or finish.

## In scope

- **`laps stints hold <name>` / `laps stints release <name>`**: mark any non-archived stint held
  / clear the hold, including before enqueue. The hold flag lives on the stint file; it only
  matters when a ref to that stint reaches the resolved head. Idempotent hold/release does not
  double-log.
- **Gated flow-op behaviour**: when the head resolves to a held stint, head `get`/`claim` return
  **no lap** and warn that the stint is held and should not be implemented yet. Explicit
  `get <id>` can inspect held work with the same warning; explicit `claim <id>` into held work is
  blocked.
- **Gate exit codes** on flow ops (`get`/`claim`), avoiding existing 2/3/4:
  - `0` — lap returned
  - `10` — held / gated (head stint on hold)
  - `11` — empty (nothing prepared anywhere)
  - `12` — complete (all laps done)
- **`done` under hold**: completing the **claimed** lap still works — hold blocks *starting* the
  next lap (`get`/`claim`), not *finishing* in-flight work.
- **`status` + `stints ls`** surface the gate: `state: held`, which stint, and the gate message.
  A valid active claim keeps `status.state=active` with gate metadata separately. `status` exits
  `0` for valid snapshots and reports state in text + `--json-output`.
- **Log events**: `stint.held`, `stint.released`.

## Out / deferred

- Auto-hold heuristics (hold-on-risk, etc.) — manual hold only.
- Per-lap holds (stint-level gating only).

## Dependencies

- **3a** (stints core). Builds on **change 2** (status, `--json-output`, log).
- Contract change for Rally: `get`/`claim` move from exit `3` (empty today) to `10/11/12`.
  Coordinate with the Rally relay loop; version-bump-worthy.

## Open questions (for formalisation)

- Exact `status` text + JSON shape for the held state and gate message.
