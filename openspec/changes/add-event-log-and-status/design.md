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
  not break older structured-claim readers. In this change `done` continues to match the claim
  by **id within the selected file** and ignores the new `claim.file` field (only `status`
  surfaces a `file` mismatch as `claim.valid=false`); `add-stints` is what makes `done` honor the
  recorded file/scope. Note `done.go:92` re-reads the claim with the error ignored before
  clearing it — decide whether that read should now surface a malformed-claim error or stay
  best-effort (recommend staying best-effort: a clear failure must not block a completed `done`).
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
  or removals.   Reclaiming the same lap preserves `claimedAt` and does not duplicate log entries;
  replacing a different claimed lap emits `unclaimed` with `detail.reason:"replaced"` followed
  by `claimed` for the new lap.
- **Claim-clear replay on `done` — DECIDED.** `done` emits a separate `unclaimed` event with
  `detail.reason:"completed"` immediately after the `completed` event, rather than folding claim
  clearing into `completed.detail`. This keeps the log uniform with the claim-replacement case
  (already `unclaimed` / `detail.reason:"replaced"`): one `unclaimed` event per applied claim
  removal, tagged with the reason, regardless of whether the trigger was a replace or a
  completion.
- **Init preserves `.gitignore`.** `init` must scan the complete `.gitignore`, preserve all
  existing lines, and append only missing `.laps/claim` / `.laps/log.jsonl` entries.
- **Status JSON shape and stale-claim policy — DECIDED.** The status snapshot exposes `claimedAt`
  (nullable RFC3339 UTC timestamp; `null` when no lap is claimed) and `ageSeconds` (integer seconds
  since `claimedAt`; `null` when `claimedAt` is `null`). No `stale` boolean is added in this
  change; a stale flag is deferred until a threshold/policy is selected, since "stale" is a
  policy judgement that needs the operator threshold the structured claim now feeds.
- **Log reader filter semantics — DECIDED.** `laps log` applies all filters (`--lap`,
  `--session`, `--since`) first and then truncates to `-n` (filter-then-limit), so the limit is
  the number of matching events shown, not lines scanned. Output is newest-last (chronological
  order). The default limit is `-n 20`. `--since` takes an RFC3339 timestamp and is inclusive of
  the exact timestamp. Malformed JSONL lines are skipped with a one-line stderr note per line and
  never abort the read (the best-effort contract applied to the reader). `--json-output` emits a
  single object `{ "events": [ ... ] }`.

## Risks

- Wiring a log call into every mutating command is broad but mechanical; the best-effort
  contract bounds the blast radius (a logging bug can never break a queue operation).
