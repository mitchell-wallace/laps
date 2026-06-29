# Draft — add-event-log-and-status (2, observability)

> Pre-formalisation notes. Status: draft. Change #2 of four.

## Why

Rally drives laps and surfaces **its own summaries** — the operator rarely watches the laps queue
directly. Rally's QA reports repeatedly note they **inferred laps' behaviour from transcripts**
because they couldn't see what laps actually did (the `wrong_lap_consumed` / `multi_lap_consumed`
integrity bugs). This change gives laps **machine-readable ground truth** plus a **status snapshot**
Rally can reconcile against and surface ("lap 4/12 · change-x · VERIFY active").

Highest-leverage non-stints item given how the operator actually works.

## In scope

### Event log — `.laps/log.jsonl`

- Append-only JSONL, **native** to each mutating command (not a user hook, so it can't be
  configured away), **best-effort**: an append failure never fails the command — warn on stderr.
  Observability never blocks the queue.
- **Gitignored** via `init` (alongside `.laps/claim`). Runtime history is local; append-only logs
  conflict constantly across branches.
- **Logged events** — state transitions + `claim`, **not** reads (`get`/`list`/`status`):
  `created`, `completed`, `reopened` (done-undo), `claimed`, `unclaimed`, `moved`, `edited`,
  `deleted`, `pruned`. (Stint events `stint.*` added in 3a.)
- **Line schema**: `{ts, event, cmd, file, lap?, title?, assignee?, scope, detail{}, session}`.
  - `file` is the resolved `.laps`-relative task file.
  - `scope` defaults to `root` (stints populate it in 3a).
  - `title`/`assignee` denormalised so the log reads standalone.
  - `detail` is event-specific (`pos` for add, `from`/`to` for move/edit, `count` for prune).
- **Attribution**: `session` stamped from env var **`LAPS_SESSION`** (Rally sets it to its
  try/run id); harmless when unset. **Central, stint-tagged** log (single source of truth; per-stint
  filtering is a `scope` field match) — not per-stint files.
- **Retention**: **grow forever** (~200 B/event ⇒ ~2 MB / 10k events). Document a manual trim path;
  no auto-rotation in v1.
- **Reader**: `laps log [-n N] [--lap <id>] [--session <id>] [--since <t>] [--json-output]`
  (+ scope filter once stints exist). `laps log --lap <id>` shows one lap's full lifecycle
  (`created → claimed → completed`) — exposes `multi_lap_consumed` (two `completed` in one session)
  and `wrong_lap_consumed` (`claimed` X but `completed` Y).
- **Sleeper value**: the log is the **only** place transition history survives `done undo`,
  `delete`, and `prune` — `laps.json` overwrites `completedAt` on undo and drops the row on
  delete/prune.

### Status — `laps status [--json-output]`

- Reports: selected file, counts (todo/done), claimed/active lap, head lap, assignee breakdown,
  and state `active | ready | empty | complete`. (`held` + per-stint progress added in 3a/3b.)
- The `--json-output` form is the integration surface Rally consumes for its summary.

### Claim file → structured

- Becomes `{lap, file, claimedAt}` (back-compat read of legacy bare-id: non-JSON token ⇒
  `{lap: token, file: selected file, claimedAt: nil}`). `claimedAt` powers status "active since
  N" + future stale-claim policy. (`scope` field added in 3a.)

### List active-lap marker

- Owned by change 1; it uses the shared claim reader rather than parsing `.laps/claim` directly.

## Out / deferred

- `scope` population + `stint.*` events → 3a. `held` state → 3b.
- Auto-rotation / size caps. Failed-command logging (only applied state changes logged in v1).

## Dependencies

- None hard. Recommended **after change 1** so the log covers `move`/`edit` from the start.

## Open questions (for formalisation)

- `status` text layout vs JSON shape.
- Stale-claim threshold/policy.
- `done` claim-clear replay shape.
