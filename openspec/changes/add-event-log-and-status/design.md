## Context

Laps has no record of what it did beyond the current `laps.json` state. Rally reconstructs
laps behaviour from agent transcripts, which is how integrity bugs (`wrong_lap_consumed`,
`multi_lap_consumed`) went undiagnosed. The operator monitors via Rally summaries rather
than the laps queue, so the highest-value additions are (a) a ground-truth event history and
(b) a status snapshot Rally can consume.

## Goals / Non-Goals

**Goals:**
- A durable, append-only event history that survives `done undo` / `delete` / `prune`.
- Observability that never affects correctness (best-effort, never blocks a command).
- A `status` snapshot, including `--json-output`, that Rally can surface.
- Run attribution so a cluster of events maps to a Rally run/try.

**Non-Goals:**
- `scope` population and `stint.*` events (owned by `add-stints`); the `held` state
  (owned by `add-stints-gating`).
- Log rotation / size caps (grow-forever for now, with a documented manual trim path).
- Logging failed/refused commands (only applied state changes in this change).
- Logging read commands (`get`, `list`, `status`).

## Decisions

- **Native, not a hook.** The log writer is built into the commands so it cannot be
  configured away — it is ground truth, not user-customisable behaviour. It is **best-effort**:
  an append failure is reported on stderr but never changes the command's exit code.
- **JSONL, central, stint-tagged.** One `.laps/log.jsonl`; per-stint views come from a
  `scope` field match (added later), not separate files. `title`/`assignee` are denormalised
  so the log reads standalone.
- **Attribution via `LAPS_SESSION`.** Each line stamps `session` from the env var (Rally sets
  it to its try/run id); empty when unset. Harmless without an orchestrator.
- **`scope` defaults to `root`.** Present from day one so `add-stints` populates it additively
  without a schema change to the log.
- **Gitignored.** `init` appends `.laps/log.jsonl` to `.gitignore` alongside `.laps/claim`;
  append-only logs conflict constantly across branches and are local runtime history.
- **Structured claim, back-compat.** The claim file becomes `{lap, claimedAt}`. A legacy
  bare-id (non-JSON) file is read as `{lap: <id>, claimedAt: null}`. `claimedAt` drives the
  "active since" / stale-claim signal in `status`. (`add-stints` adds a `scope` field to the
  same object.)
- **Status states.** `active` (a lap is claimed / work in progress), `empty` (no laps),
  `complete` (laps exist, all done). `status` always exits 0; gate-related states arrive with
  `add-stints-gating`.

## Risks

- Wiring a log call into every mutating command is broad but mechanical; the best-effort
  contract bounds the blast radius (a logging bug can never break a queue operation).
