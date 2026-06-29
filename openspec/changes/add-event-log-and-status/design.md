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
  `scope` field match (added later), not separate files. Every line carries `file`, the
  resolved `.laps`-relative task file, because task ids are only unique inside a file.
  `title`/`assignee` are denormalised so the log reads standalone.
- **Attribution via `LAPS_SESSION`.** Each line stamps `session` from the env var (Rally sets
  it to its try/run id); empty when unset. Harmless without an orchestrator.
- **`scope` defaults to `root`.** Present from day one so `add-stints` populates it additively
  without a schema change to the log.
- **Gitignored.** `init` appends `.laps/log.jsonl` to `.gitignore` alongside `.laps/claim`;
  append-only logs conflict constantly across branches and are local runtime history.
- **Structured claim, back-compat.** The claim file becomes `{lap, file, claimedAt}`. A legacy
  bare-id (non-JSON) file is read as `{lap: <id>, file: <selected file>, claimedAt: null}`.
  `claimedAt` drives the
  "active since" / stale-claim signal in `status`. (`add-stints` adds a `scope` field to the
  same object.) Claim JSON parsing SHALL ignore unknown fields so later `{scope}` additions do
  not break older structured-claim readers.
- **Status states.** `active` (a valid todo lap is claimed / work in progress), `ready` (todo
  laps exist and nothing valid is claimed), `empty` (no laps), and `complete` (laps exist, all
  done). `status` exits 0 for valid repo/store snapshots; malformed stores, unreadable stores,
  malformed claim JSON, and serialization errors follow the normal CLI error path rather than
  being hidden as healthy snapshots. Claims pointing at deleted, pruned, done, or wrong-file laps
  produce an exit-0 degraded snapshot with `claim.valid=false`; commands do not auto-clear such
  claims silently. Gate-related states arrive with `add-stints-gating`.
- **One event per applied state change.** Batch and multi-row commands append one line per
  affected lap/transition after the store save succeeds: batch `add --json` emits one `created`
  line per new lap, and `prune` emits one `pruned` line per removed lap. Claim-only mutations
  append only after `WriteClaim` or `RemoveClaim` succeeds, and do not log failed claim writes
  or removals. Reclaiming the same lap preserves `claimedAt` and does not duplicate log entries;
  replacing a different claimed lap emits `unclaimed` with `detail.reason:"replaced"` followed
  by `claimed` for the new lap.
- **Init preserves `.gitignore`.** `init` must scan the complete `.gitignore`, preserve all
  existing lines, and append only missing `.laps/claim` / `.laps/log.jsonl` entries.

## Open Product Calls

- Claim-clear replay: choose whether `done` emits a separate `unclaimed` event when it clears a
  claim, or records claim clearing in `completed.detail`.
- Status JSON shape and stale-claim policy: choose exact field names/nullability and whether a
  `stale` boolean exists now or only `claimedAt`/`ageSeconds` are exposed until a threshold is
  selected.
- Log reader filter semantics: choose default limit, `--since` timestamp format/inclusivity,
  filter-before-limit ordering, malformed JSONL behavior, and JSON output shape.

## Risks

- Wiring a log call into every mutating command is broad but mechanical; the best-effort
  contract bounds the blast radius (a logging bug can never break a queue operation).
