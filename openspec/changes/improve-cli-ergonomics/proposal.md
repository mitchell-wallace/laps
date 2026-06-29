## Why

The operator does most queue work **before and between** runs, so laps' editing and
readability ergonomics matter. Two gaps stand out:

- `list` is information-thin (one terse line per lap) and can't show which lap is
  active, so an operator scanning the queue can't read current state at a glance.
- There is **no way to reorder an existing lap**. `add` only inserts at a position;
  changing a lap's place means delete + re-add, which mints a new id and breaks the
  lap's claim/hook/log identity. `move` is a genuinely missing primitive.

This change adds the between-run editing primitives and a richer default list, all on
today's flat queue with no schema change.

## What Changes

- **`list` default → two lines per lap**: first line shows position number, title, and
  an active-lap marker; second line shows id, assignee, and state. Descriptions are not
  shown. A `--oneline` flag preserves the prior single-line form. `--all`/`--done`
  continue to work, with done laps struck through in both layouts.
- **`ls`** as an alias for `list`.
- **Active-lap marker** in `list`, reading the claimed lap through the central claim-reader
  contract rather than parsing `.laps/claim` directly.
- **`move <id> head|tail|after <id>`** — reorder an existing todo lap, preserving its id,
  using the same ordering rules as `add`.
- **`edit <id> [--title] [--description] [--assignee]`** — in-place field edits; at least
  one field flag required; bumps `updatedAt`.
- **`assign <id> <role>`** — shortcut for `edit <id> --assignee <role>`.

All new commands honor the existing `--json-output` mode.

## Capabilities

### Added Capabilities
- `task-listing`: how laps renders the queue — the two-line default, `--oneline`, the
  `ls` alias, and the active-lap marker.
- `task-editing`: in-place mutation of existing laps — reordering (`move`), field edits
  (`edit`), and the `assign` shortcut.

## Impact

- **Code**: `internal/cmd/list.go` (two-line formatter, `--oneline`, marker, `ls` alias);
  new `internal/cmd/move.go`, `edit.go`, `assign.go`; reuse `store.ComputeInsertOrder`
  for `move`; use `store.ReadClaim` for the marker; update hook-only command recognition
  so `ls`, `move`, `edit`, and `assign` dispatch as built-ins rather than hook-only names.
- **Behavior**: the default `list` output format changes from one line to two; consumers
  that need the old shape pass `--oneline`. Reordering no longer re-creates laps.
- **Out of scope**: color/TTY theming; scope flags (`--active`/`--root`/`--stint`, added
  by `add-stints`); the TUI.
