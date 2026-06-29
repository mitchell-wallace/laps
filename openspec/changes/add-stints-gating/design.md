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

- **Held stops descent.** A held flag travels with the stint; when the resolved head is a held
  stint, flow resolution stops there rather than descending. The flag only matters once the
  stint is at the head, so a held stint deeper in the pipeline has no effect until it arrives.
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

## Risks

- The exit-code change is the one breaking edge; it is intentional and coordinated with Rally,
  and `status --json-output` provides the same state for any consumer that prefers parsing.
