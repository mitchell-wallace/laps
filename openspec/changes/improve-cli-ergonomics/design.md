## Context

Laps today is a flat todo/done queue. `list` prints one terse line per lap
(`<n>. <id> — <title> (assignee: X)`) and cannot indicate which lap is being worked.
Insertion exists (`add head|tail|after`) but there is no primitive to reorder a lap that
is already in the queue — operators must delete and re-add, which changes the lap id and
breaks its claim/hook/log identity. This change closes both gaps without a schema change.

## Goals / Non-Goals

**Goals:**
- A richer default `list` (two lines, no descriptions) that surfaces the active lap.
- Preserve a terse machine-friendly form via `--oneline`.
- A `move` primitive that reorders an existing lap while preserving its id.
- In-place field editing (`edit`) and an `assign` shortcut.
- No schema change; reuse existing ordering and store machinery.

**Non-Goals:**
- Color / TTY theming.
- Scope flags (`--active`/`--root`/`--stint`) — introduced by `add-stints`.
- A TUI (the operator monitors via Rally summaries, not the laps queue directly).
- `$EDITOR`-based editing (flags only for now).

## Decisions

- **Two-line layout.** Line 1: `<n>. [marker]<title>` (title struck through when done).
  Line 2: `   <id> · <assignee|—> · <todo|done>`. Assignee placeholder is an em dash when
  unset.
- **`--oneline` preserves** the existing `<n>. <id> — <title> (assignee: X)` form for
  consumers that parse the terse output. The two-line form is the new default per the
  product decision to make `list` slightly more detailed without showing descriptions.
- **Active marker uses the claim reader.** `list` SHALL call the central claim-reader contract
  (`store.ReadClaim` today) rather than parsing `.laps/claim` in the formatter. Missing claims
  and claims for laps outside the listed result produce no marker; claim-read errors remain
  command errors. This keeps the marker forward-compatible with `add-event-log-and-status`,
  which makes the claim a structured `{lap, claimedAt}` object read back-compatibly.
- **`move` reuses `store.ComputeInsertOrder`** (head/tail/after with midpoint insertion and
  renumber-on-gap-exhaustion). It operates on todo laps only; an unknown or already-done id
  errors; an `after` target that is done falls back to head with a stderr notice, mirroring
  `add after`. The lap id is never regenerated, and a successful move advances `updatedAt`.
- **`edit` requires ≥1 field flag** — a no-op edit is rejected. Set fields are updated and
  `updatedAt` advances. `edit` may target todo or done laps; editing a done lap succeeds with a
  stderr warning and does not reopen the lap or change `completedAt`.
- **`assign`** is sugar over `edit --assignee`; it follows the same done-lap warning rule.
- **`--json-output`** is honored by `move`, `edit`, and `assign`, each emitting the affected
  task as a `task` object, matching the existing command convention (the repo has JSON-output
  tests for all commands).
- **Command registration and hooks.** `ls`, `move`, `edit`, and `assign` must be registered as
  built-in command names before the hook-only intercept in `internal/cmd/hooks.go:isKnownCommand`
  runs. The mutating commands (`move`, `edit`, `assign`) run the existing before/after hook
  lifecycle with the affected task, `$file`, `$args`, `$output`, and `$exit_code` populated the
  same way as other mutating built-ins.

## Open Product Calls

- Exact active-marker token and done styling: pick the marker string and whether two-line done
  rendering strikes only the title or both rendered lines.
- Edit/output semantics: confirm whether `--description ""` and `--assignee ""` clear fields,
  whether `assign` rejects blank-after-trim roles, and what non-JSON success text should be for
  `move`/`edit`/`assign`.

## Risks

- The default `list` output format changes (one line → two). This is the only behavior
  change; `--oneline` restores the prior shape for any consumer that needs it.
