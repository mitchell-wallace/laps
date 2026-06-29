## Why

Rally drives laps and surfaces **its own summaries** — the operator rarely watches the
laps queue directly. Rally's QA reports repeatedly note that laps' behaviour had to be
**inferred from transcripts** because there was no record of what laps actually did; this
is the root of the `wrong_lap_consumed` / `multi_lap_consumed` integrity bugs.

This change gives laps **machine-readable ground truth** (an append-only event log) and a
**status snapshot** Rally can reconcile against and surface ("lap 4/12 · VERIFY active").
It is the highest-leverage observability item given how the operator actually works.

## What Changes

- **Event log `.laps/log.jsonl`** — append-only JSONL, **native** to each mutating command
  (not a user hook, so it cannot be disabled) and **best-effort** (an append failure never
  fails the command). Gitignored via `init`. Logs state transitions plus `claim`, never
  reads. Line schema `{ts, event, cmd, lap?, title?, assignee?, scope, detail{}, session}`;
  `session` from the `LAPS_SESSION` env var; `scope` defaults to `root`. Grow-forever (no
  rotation).
- **`laps log`** reader with `-n`, `--lap`, `--session`, `--since`, `--json-output`.
  `laps log --lap <id>` shows one lap's full lifecycle.
- **`laps status [--json-output]`** — counts, active (claimed) lap, head, assignee
  breakdown, and a queue state of `active | empty | complete`.
- **Structured claim** — the claim file becomes `{lap, claimedAt}` (legacy bare-id read
  back-compatibly). `claimedAt` lets `status` show how long the active lap has been held and
  flag stale claims from crashed sessions.

## Capabilities

### Added Capabilities
- `event-log`: the append-only `.laps/log.jsonl` history and the `laps log` reader.
- `status`: the `laps status` snapshot and the claim `claimedAt` timestamp it surfaces.

## Impact

- **Code**: new event-log writer wired into every mutating command (`add`, `done`,
  `done-undo`, `claim`, `claim-undo`, `delete`, `prune`, `move`, `edit`, `assign`); new
  `internal/cmd/log.go` and `status.go`; `internal/store/claim.go` (structured claim,
  back-compat read); `init.go` (gitignore `.laps/log.jsonl`); `LAPS_SESSION` read.
- **Behavior**: each mutating command appends a best-effort log line; `init` gitignores the
  log; the claim file becomes JSON (legacy bare-id still read). The log is the only place
  transition history survives `done undo`, `delete`, and `prune`.
- **Out of scope**: `scope` population and `stint.*` events (added by `add-stints`); the
  `held` state (added by `add-stints-gating`); log rotation; failed-command logging.
