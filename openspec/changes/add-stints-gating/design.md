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

- **Held stops flow-start descent.** A held flag lives on the non-archived stint file metadata,
  defaulting to `false` when absent. It is folded into schema v3 before the first v3/0.9.0 binary
  ships. For `get`/`claim` flow-start resolution and status gate probing, if the current context
  head references a held stint, resolution returns `held` and does not select a lap from that
  child file. The flag only matters once the stint is at the current context head, so a held
  stint deeper in the pipeline has no effect until descent reaches its parent context.
- **Exit codes carry queue state.** `get`/`claim` return `0` (lap), `10` (held), `11` (empty),
  `12` (complete), chosen to avoid the existing `2`/`3`/`4` (io/store, not-found/empty,
  hook). Text mode emits no stdout for `10`/`11`/`12`; held cases warn on stderr that the stint
  is held and should not be implemented yet. JSON mode emits a small state object on stdout.
  `status` stays exit-0 for valid snapshots and reports the same state in text and
  `--json-output`.
- **Hold blocks starting, not finishing.** A hold gates `get`/`claim` (starting the next lap)
  but never `done` for the already-claimed lap — an agent mid-lap can always finish and record.
- **Explicit id behavior under hold.** `get <id>` may inspect a lap inside a held stint, but it
  warns on stderr that the stint is held and should not be implemented yet. `claim <id>` into a
  held stint exits `10`, leaves the claim unchanged, and emits the same warning.
- **Status precedence.** A valid active claim keeps `status.state=active` even when the next head
  is held; status includes gate metadata separately. `held` is reported as the primary state only
  when there is no valid active claim and the next flow-start operation would gate.
- **Rally contract change.** Today `get`/`claim` exit `3` on an empty queue; moving to
  `10`/`11`/`12` is a deliberate contract change Rally adopts in lockstep, gated behind a
  version bump.

## Implementation Contracts

- **Clean state exits are not generic errors.** The `10`/`11`/`12` queue-state exits SHALL avoid
  the generic error helper's stderr/error-JSON shape. They are control-flow signals for Rally;
  after-hooks still receive the final exit code.
- **Existing failures keep existing codes.** Explicit id not found remains exit `3`, store/io
  failures remain `2`, and hook failures remain `4`; `11`/`12` apply only to head/flow
  operations that find no lap to start because the queue is empty or complete. Exit `10` applies
  to head flow-start operations gated by hold and to explicit `claim <id>` attempts into a held
  stint.
- **Final-lap drain still wins.** `done` for a claimed final lap inside a held stint SHALL
  complete the lap and allow the `add-stints` drain/archive behavior to run; a held drained
  stint must not stay stuck as the next gate.

## Open Product Calls

- Empty vs complete across stints: decide how unqueued stint files, archived drained stints,
  root queues with only done refs, and empty active stint files map to `empty` vs `complete`.
- `stints ls` rendering: decide whether held replaces lifecycle state or appears as a separate
  boolean/marker alongside queued/active/done.
- Non-starting scoped commands besides explicit `get`/`claim`: decide whether `list`, `count`,
  `add`, `edit`, and `delete` can inspect or mutate inside/under a held stint.

## Risks

- The exit-code change is the one breaking edge; it is intentional and coordinated with Rally,
  and `status --json-output` provides the same state for any consumer that prefers parsing.
