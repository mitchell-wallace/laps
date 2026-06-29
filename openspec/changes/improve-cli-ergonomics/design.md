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

- **Two-line layout.** Line 1: `<n>. [marker]<title>` where the active marker is `> ` (title
  struck through when done).
  Line 2: `<indent><id> · <assignee|—> · <todo|done>`. Assignee placeholder is an em dash when
  unset. In the two-line layout, done styling strikes through only the title; `--oneline`
  preserves the prior whole-line done strike. The line-2 `<indent>` SHALL be padded to the
  width of the line-1 position prefix (`"<n>. "`) so the id column stays aligned once the queue
  exceeds 9 laps (a fixed 3-space indent misaligns at position 10+).
- **`--oneline` preserves** the existing single-line form by reusing today's
  `formatListTask` (`internal/cmd/list.go:109`) verbatim: `<n>. <id> — <title>` with the
  ` (assignee: <a>)` clause appended **only when the assignee is non-empty** (the literal
  `(assignee: <a>)` in the spec is shorthand, not an unconditional clause — matching
  `TestListOutputUnchangedWithoutAssignee`). The two-line form is the new default per the
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
  - **Exit codes mirror `add`:** an `after` target that does not exist returns exit `3` (via
    `store.ErrTaskNotFound`, as `add after` does at `add.go:157`); a moved id that is unknown or
    already-done, and usage errors, return exit `1`. `move <id> after <id>` (self-reference)
    SHALL error rather than silently no-op.
  - **The after-done stderr notice is the command's responsibility.** `store.ComputeInsertOrder`
    only returns `fallbackHead=true`; it does not print. `move.go` SHALL emit the notice itself
    when `fallbackHead` is true, copying the `fmt.Fprintf(os.Stderr, …)` pattern from
    `add.go:164`.
- **`edit` requires ≥1 field flag** — a no-op edit is rejected. Each field SHALL be gated on
  `cmd.Flags().Changed("<name>")` (mirroring `add.go:73`), NOT on a non-empty value, so that an
  explicitly-passed empty flag (`--description ""`) is distinguishable from an unset flag; the
  ≥1-flag check is `Changed("title") || Changed("description") || Changed("assignee")`. Set
  fields are updated and `updatedAt` advances. `--title` must be nonblank after trimming;
  `--description ""` clears the description; `--assignee ""` clears the assignee; non-empty
  assignees are trimmed; escaped `\n` in descriptions follows `add` behavior. `edit` may target
  todo or done laps; editing a done lap succeeds with a stderr warning and does not reopen the
  lap or change `completedAt`.
- **`assign`** is sugar over `edit --assignee`; it accepts a blank role to clear the assignee and
  follows the same done-lap warning rule.
- **Text success output.** Non-JSON `move`, `edit`, and `assign` print only the affected lap id on
  stdout; warnings and notices go to stderr.
- **`--json-output`** is honored by `move`, `edit`, and `assign`, each emitting the affected
  task as a `task` object, matching the existing command convention (the repo has JSON-output
  tests for all commands).
- **Command registration and hooks.** `ls`, `move`, `edit`, and `assign` must be registered as
  built-in command names before the hook-only intercept in `internal/cmd/hooks.go:isKnownCommand`
  runs. The mutating commands (`move`, `edit`, `assign`) run the existing before/after hook
  lifecycle with the affected task, `$file`, `$args`, `$output`, and `$exit_code` populated the
  same way as other mutating built-ins.

## Risks

- The default `list` output format changes (one line → two). This is the only behavior
  change; `--oneline` restores the prior shape for any consumer that needs it.
