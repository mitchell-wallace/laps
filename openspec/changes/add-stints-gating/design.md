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
- Nested stint drain is correct before hold semantics depend on final-lap drain/archive behavior.

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
  hook). Text mode emits no command stdout for `10`/`11`/`12`; held cases warn on stderr that
  the stint is held and should not be implemented yet. Hook passback, if configured, may still
  write its own stdout. JSON mode emits a small state object on stdout.
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
  after-hooks still receive the final exit code. The mechanism is load-bearing and must be
  specified, because the natural implementations break it: today `exit()` (`root.go:39-51`)
  always writes a `laps:` stderr line (or an `{error,exitCode}` JSON object to **stderr**), and
  the only way a nonzero OS code is produced is the `panic(*exitError)` → `recover` → `os.Exit`
  path (`root.go:50,57-64`); the deferred after-hook (`hooks.go:32-51`, registered at
  `get.go:46`/`claim.go:46`) reads `*exitCode` only during that panic unwind. So a naive
  `os.Exit(11)` or `return`-with-error would skip the after-hook and violate "hook sees the
  code". Define a dedicated clean-state exit (e.g. `exitState(code int)`) that, after the Run
  handler has (1) set the captured `exitCode = code` and (2) in JSON mode printed the small
  queue-state object to **stdout** via `printJSON` (note `exit()` puts JSON on stderr — this
  stdout-vs-stderr split is the contract), panics `*exitError{code}` with **no** stderr error
  line. Held cases additionally warn on stderr. Routing `10`/`11`/`12` through the existing
  panic/recover path is what keeps `runAfterHooksDeferred` firing.
- **Empty/complete is distinguished at the existing post-defer `task == nil` site.** Today
  `get`/`claim` reach `exit(3,"no head task")` at `get.go:49-52` / `claim.go:49-55` *after* the
  after-hook defer is registered (`get.go:46`/`claim.go:46`), so the hook already observes the
  code — no reordering is needed. This change replaces that single empty `exit(3)` with
  `exitState(11)` (empty) vs `exitState(12)` (laps exist but all done), per the empty-vs-complete
  taxonomy product call. `checkDefault` (`root.go:136-146`) is **not** the empty path and needs no
  change: `CheckDefaultStore` returns `nil` for a missing/empty default store (`store.go:397-404`,
  via `ErrEmptyFile`), and its `ErrEmptyState→exit(3)` branch is **dead** (nothing in the codebase
  produces `store.ErrEmptyState`). `checkDefault`'s only live effect is `exit(2)` on a *corrupt*
  default store, which stays `2`. Explicit-id-not-found stays `exit(3)` in the `target != "head"`
  branch.
- **Flow-start resolution returns one typed queue-state.** `add-stints`' resolver returns "the
  first lap"; gating needs a single typed outcome (`lap | held | empty | complete`) shared by
  `get`/`claim` and the `status` gate probe, so the exit codes and the `held` status derive from
  **one** resolution pass rather than two divergent ad-hoc detections.
- **Existing failures keep existing codes.** Explicit id not found remains exit `3`, store/io
  failures remain `2`, and hook failures remain `4`; `11`/`12` apply only to head/flow
  operations that find no lap to start because the queue is empty or complete. Exit `10` applies
  to head flow-start operations gated by hold and to explicit `claim <id>` attempts into a held
  stint.
- **Final-lap drain still wins.** `done` for a claimed final lap inside a held stint SHALL
  complete the lap and allow the `add-stints` drain/archive behavior to run; a held drained
  stint must not stay stuck as the next gate.
- **Nested drain uses the actual parent queue.** Review of `add-stints` found that drain currently
  identifies the completed stint by physical file, then looks for that ref only in root. That is
  sufficient for root-level stints but fails for `root -> auth -> search`: completing the final lap
  in `search` cannot mark `auth`'s `search` ref done or archive `search`. Before implementing hold
  gates, extend active resolution/drain preparation to preserve the physical parent chain: for each
  descended stint ref, keep parent path, parent file, ref pointer/index, child path, child name, and
  canonical scope. A final-lap drain updates the immediate parent ref for the completed child,
  archives that child file with the same partial-failure guarantees as root-level drains, then
  re-evaluates the parent file. If the parent now has no todo laps, drain cascades upward one level
  at a time until root is reached or a parent still has todo work. This keeps drain
  content-based/position-independent for nested stints and prevents held, already-drained child
  stints from becoming permanent gates.
- **Nested drain partial-failure ordering.** Preserve the `add-stints` fail-safe ordering at each
  cascade step: preflight archive target/collision and parent/root save target before mutating;
  persist the completed lap or child ref in its containing file; rename the drained child file to
  archive; then save the containing parent ref as done. If a post-rename save fails, restore the
  archived file back to active before returning. Never leave a done parent ref over a present
  active child file, and never leave a todo parent ref pointing at a missing active child file when
  the archived file exists.
- **Nested undo follows the same parent chain.** `done undo` for an archived nested stint lap must
  restore the child file, reopen the immediate parent ref that pointed to that child, and reopen any
  ancestor refs/files needed to make the restored lap reachable. The existing 5-minute age gate and
  global latest-completion selection still apply.

## Resolved Product Calls

- **Empty vs complete across stints — DECIDED (distinct mapping).** `complete` (exit `12`) means
  every entry resolvable from the root head is done — including drained/archived done refs and
  root queues holding only done refs — and nothing enqueueable remains. `empty` (exit `11`) means
  the root resolves to zero todo entries because nothing was ever enqueued — including a repo with
  only unqueued stint files, and an empty active stint file. The two stay distinct so Rally can
  tell "pipeline finished, stop" (`12`) from "idle, wait for work" (`11`). This mapping governs
  both the `get`/`claim` exit codes (tasks 2.2/2.6) and `status` empty/complete (tasks 4.2/4.3).
- **`stints ls` rendering — DECIDED (separate badge).** `held` is rendered as a separate
  badge/marker alongside the queued/active/done lifecycle state, not as a replacement for it.
  Hold is orthogonal to lifecycle: a stint can be held-and-queued, held-and-active, or
  held-and-done, and all three combinations are renderable. The lifecycle column keeps its
  queued/active/done value; an additional held marker flags stints whose held metadata is set,
  regardless of whether the hold has taken effect at the head yet.
- **Non-starting scoped commands under hold — DECIDED (hold gates only flow-start).** A hold
  gates ONLY flow-start (`get`/`claim`). `list`, `count`, `add`, `edit`, `assign`, and `delete`
  operate normally inside/under a held stint, consistent with explicit `get <id>` being allowed
  to inspect a held stint (with the held warning) and `done` being allowed to finish. Hold is a
  pause on serving the next lap, not a freeze on the stint's data.

## Risks

- The exit-code change is the one breaking edge; it is intentional and coordinated with Rally,
  and `status --json-output` provides the same state for any consumer that prefers parsing.
