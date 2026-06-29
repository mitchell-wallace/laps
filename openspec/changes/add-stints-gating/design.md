## Context

`add-stints` makes the pipeline auto-advance: when a stint reaches the resolved head it
activates and its laps are served. This change adds a pause point — a held stint stops the
queue — and makes the "nothing to serve" cases distinguishable to an orchestrator, so Rally's
relay loop can stop on a gate, finish on completion, or idle on an empty queue without guessing.

## Goals / Non-Goals

**Goals:**
- Operator-controlled pause/resume of the pipeline at stint granularity.
- A clean agent-facing stop (no lap, not an error) when the head is held.
- Distinct, parseable signals for held vs empty vs complete.
- In-flight work can always be finished even while a stint is held.

**Non-Goals:**
- Auto-hold heuristics (e.g. hold-on-risk) — manual hold only.
- Per-lap holds — gating is at the stint level.

## Decisions

- **Held stops flow-start descent.** A held flag travels with the stint reference. For
  `get`/`claim` flow-start resolution and status gate probing, if the current context head is a
  held `kind:"stint"` ref, resolution returns `held` and does not open the child file. The flag
  only matters once the stint is at the current context head, so a held stint deeper in the
  pipeline has no effect until descent reaches its parent context. Non-starting scoped commands
  are covered by an open product call.
- **Exit codes carry queue state.** `get`/`claim` return `0` (lap), `10` (held), `11` (empty),
  `12` (complete), chosen to avoid the existing `2`/`3`/`4` (io/store, not-found/empty,
  hook). The relay loop branches on the code without parsing output. `status` stays exit-0 and
  reports the same state in text and `--json-output`.
- **Hold blocks starting, not finishing.** A hold gates `get`/`claim` (starting the next lap)
  but never `done` for the already-claimed lap — an agent mid-lap can always finish and record.
- **Claim under a held head also returns `10`.** Agents should not claim into a held stint; the
  gate is uniform across both flow entry points.
- **Rally contract change.** Today `get`/`claim` exit `3` on an empty queue; moving to
  `10`/`11`/`12` is a deliberate contract change Rally adopts in lockstep, gated behind a
  version bump.

## Implementation Contracts

- **Clean state exits are not generic errors.** The `10`/`11`/`12` queue-state exits SHALL avoid
  the generic error helper's stderr/error-JSON shape. They are control-flow signals for Rally;
  after-hooks still receive the final exit code.
- **Existing failures keep existing codes.** Explicit id not found remains exit `3`, store/io
  failures remain `2`, and hook failures remain `4`; `10`/`11`/`12` apply only to head/flow
  operations that find no lap to start because the queue is held, empty, or complete.
- **Final-lap drain still wins.** `done` for a claimed final lap inside a held stint SHALL
  complete the lap and allow the `add-stints` drain/archive behavior to run; a held drained
  stint must not stay stuck as the next gate.

## Open Product Calls

- Held schema/version ownership: decide whether `held` folds into the unreleased v3 stint schema
  or bumps the schema to v4, including missing-field default `false` and older-version rejection.
- Hold target semantics: decide whether a stint can be held before enqueue, whether archived
  stints can be targeted, how duplicate refs are handled, and idempotency/logging for already
  held or already released stints.
- Queue-state output: decide exact text/stdout/stderr and `--json-output` shape for clean
  `10`/`11`/`12` exits.
- Empty vs complete across stints: decide how unqueued stint files, archived drained stints,
  root queues with only done refs, and empty active stint files map to `empty` vs `complete`.
- Status precedence: decide whether an existing valid claim keeps status `active` even when the
  next head is held, or whether `held` takes precedence.
- `stints ls` rendering: decide whether held replaces lifecycle state or appears as a separate
  boolean/marker alongside queued/active/done.
- Non-starting scoped commands and explicit ids: decide whether `list`, `count`, `add`, `edit`,
  `delete`, `get <id>`, and `claim <id>` can inspect or mutate inside/under a held stint; the
  current gated contract only covers starting the next lap via head `get`/`claim`.

## Risks

- The exit-code change is the one breaking edge; it is intentional and coordinated with Rally,
  and `status --json-output` provides the same state for any consumer that prefers parsing.
