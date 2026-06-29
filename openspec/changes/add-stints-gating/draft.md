# Draft — add-stints-gating (3b)

> Pre-formalisation notes. Status: draft. Change #3b of four. Depends on 3a.

## Why

Stints auto-advance by default (pipeline for free). But we want to **pause** the pipeline at a
chosen stint until an operator releases it — "auto, but pausable" — and give Rally clean signals
to tell **gated** apart from **empty** and **complete**, so the relay loop can decide whether to
stop, wait, or finish.

## In scope

- **`laps stints hold <name>` / `laps stints release <name>`**: mark a stint held / clear the hold.
  The hold flag travels with the stint; it only matters when the stint reaches the resolved head.
- **Gated flow-op behaviour**: when the head resolves to a held stint, `get`/`claim` return
  **no lap** (agent-facing: nothing to do; the relay stops cleanly rather than erroring).
- **Gate exit codes** on flow ops (`get`/`claim`), avoiding existing 2/3/4:
  - `0` — lap returned
  - `10` — held / gated (head stint on hold)
  - `11` — empty (nothing prepared anywhere)
  - `12` — complete (all laps done)
- **`done` under hold**: completing the **claimed** lap still works — hold blocks *starting* the
  next lap (`get`/`claim`), not *finishing* in-flight work.
- **`status` + `stints ls`** surface the gate: `state: held`, which stint, and the gate message.
  `status` always exits `0` and reports state in text + `--json-output`.
- **Log events**: `stint.held`, `stint.released`.

## Out / deferred

- Auto-hold heuristics (hold-on-risk, etc.) — manual hold only.
- Per-lap holds (stint-level gating only).

## Dependencies

- **3a** (stints core). Builds on **change 2** (status, `--json-output`, log).
- Contract change for Rally: `get`/`claim` move from exit `3` (empty today) to `10/11/12`.
  Coordinate with the Rally relay loop; version-bump-worthy.

## Open questions (for formalisation)

- Does `claim` under a held head also return `10` (lean: yes — agents shouldn't claim into a held
  stint), or is pre-claim allowed?
- When a `done` drains the active stint **into** a held next stint, anything special? (Lean: normal
  advance; the next `get` returns `10`.)
- Exact `status` text + JSON shape for the held state and gate message.
