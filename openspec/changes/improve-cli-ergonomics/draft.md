# Draft — improve-cli-ergonomics (1)

> Pre-formalisation notes. Status: draft. Change #1 of four. Lowest-risk; ships first.

## Why

Between-run editing and readability quality-of-life on today's flat queue. The operator does most
work **before/between** runs, so the editing primitives matter; `move` in particular is a genuinely
**missing primitive** — today you can only insert at a position, so reordering an existing lap means
delete + re-add, which changes its id (breaking claim/hook/log trails).

No schema change here (the claim-file format change is owned by change 2).

## In scope

- **`list` default → ~2 lines per lap**, no descriptions:
  - line 1: number + title + active marker
  - line 2: `id · assignee · state`
  - `--oneline` keeps the current terse single-line form.
  - `--all` / `--done` keep working (done items dimmed / struck).
- **`ls` alias** for `list`.
- **Active-lap marker** in `list` (the claimed lap). Coordinate with change 2 (which makes the
  claim the "active" pointer); if 1 ships first, the marker reads `.laps/claim` directly.
- **`move <id> head|tail|after <id>`** — reorder an existing todo lap; reuses `ComputeInsertOrder`.
  The missing reorder primitive; preserves the lap's id (and thus its claim/hook/log identity).
- **`edit <id> [--title ...] [--description ...] [--assignee ...]`** — in-place field edits;
  bumps `updatedAt`.
- **`assign <id> <role>`** — shortcut for `edit <id> --assignee <role>`.

## Out / deferred

- Color / TTY theming (separate, optional).
- Scope flags (`--active`/`--root`/`--stint`) — added in 3a; `move`/`edit` gain them then.
- TUI — downgraded: the operator watches Rally summaries, not the laps queue directly.

## Dependencies

- None. Establishes `move`/`edit` so change 2's event log covers them from the start.

## Open questions (for formalisation)

- Exact 2-line layout; how `--all`/`--done` render in 2-line mode.
- `edit` with no flags → open `$EDITOR`, or require ≥1 flag? (Lean: flags-only in v1, `$EDITOR` later.)
- Does the active-lap marker belong here or in change 2?
