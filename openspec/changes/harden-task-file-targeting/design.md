# Design: Harden task-file targeting

## Current shape

Targeting has two models:

1. **Scope flags** (`internal/cmd/scope.go:21-66`): `--stint <name>` resolves
   via `store.ResolveStintFile` (canonical `.laps/stints/<name>.laps.json`,
   name validated), stats the path, and exits `3` when absent. `--root`
   resolves the default file. Mutually exclusive with `-f`.
2. **Raw `-f`** (`store.ResolveFile`, `internal/store/store.go:119-127`):
   appends `.json` if missing and joins under `.laps/`. No validation, no
   existence check.

Both funnel into `loadFile` (`internal/cmd/root.go:196-241`), where
`store.Load` returns `ErrEmptyFile` for **both** missing and zero-length files
(`store.go:24-25, 350-365`), and `loadFile` reacts by creating and saving a
fresh `{version, tasks: []}` file — on every command, including pure reads.
So `laps list -f stints/auth` (missing the `.laps` suffix) creates
`.laps/stints/auth.json`, a file that is not a stint (misses
`ActiveStintNameForPath`'s `.laps.json` suffix check, `store.go:172-185`),
is invisible to `stints ls`, and answers "empty queue" forever after.

## Target shape

- `store` distinguishes `ErrFileNotFound` (stat says absent) from
  `ErrEmptyFile` (exists, zero bytes). `CheckDefaultStore` treats both as OK
  for the default store (unchanged on-demand init of `laps.json`).
- `loadFile` gains an explicit policy parameter (or a thin
  `loadFileForWriteableTarget` wrapper): `createMissing` true only for `add`
  and `init`. All other commands map `ErrFileNotFound` on a non-default target
  to exit `3`:

  ```
  laps: task file stints/auth.json not found (did you mean --stint auth?)
  ```

- `suggestTarget`: if the `-f` value (pre-`.json`-append) starts with
  `stints/`, or its base name matches an existing active stint, suggest
  `--stint <name>`; otherwise plain not-found. Reuses `store.StintsDir` and
  the existing stint enumeration used by `stints ls`.
- Empty-but-existing files keep today's initialize-in-place behavior (that
  path is how `stints new` bootstraps prefixes, `root.go:199-213`).

## Alternatives considered

- **Reject `-f` into `.laps/stints/` entirely** (force `--stint`): tighter,
  but breaks legitimate archived-stint inspection and the documented raw
  escape hatch; the fail-closed check plus suggestion achieves the safety
  without removing capability. Rejected.
- **Add a global `--create` flag instead of verb-gating**: explicit, but every
  agent transcript would need the flag for the common `add -f` staging flow,
  and forgetting it converts a valid workflow into an error. Verb-gating
  matches intent (mutating "bring into existence" verbs create). Rejected.
- **Deprecate `-f` post-1.0**: largest simplification (one targeting model),
  but multi-file support is documented and used by `add-stints` tooling
  history; too aggressive for the first post-1.0 minor. Recorded for future
  consideration, not chosen.

## Migration and rollout

Single PR, post-1.0 minor. No data migration. Release notes: read commands no
longer create missing `-f` targets; exit `3` + suggestion instead. Verify
rally makes no read-path `-f` calls (its adapter uses default-store commands;
cross-check `internal/laps/adapter.go` in the sibling repo during rollout, per
`v3-schema-migration` tasks 3.x).

Optional follow-up (not this change): a diagnostics pass that flags existing
stray files — see `introduce-laps-doctor`.

## Verification strategy

- `just test`: new cases in `internal/cmd/cmd_test.go` and
  `internal/store/store_test.go`:
  - `list -f nope` exits `3`, creates nothing (assert directory contents).
  - `list -f stints/auth` with existing stint `auth` exits `3` and suggests
    `--stint auth`; no `.laps/stints/auth.json` appears.
  - `add -f newfile --title t` still creates `.laps/newfile.json`.
  - `init` still initializes; default-store on-demand init unchanged.
  - Empty-but-existing stint file still gets prefix-initialized.
- `just lint`.
- Manual smoke in a throwaway dir reproducing the original session transcript
  (`stints new auth`, then the typo'd list) — expect exit `3` + hint.

## Dependencies and ordering

Independent of, but thematically paired with, `close-unknown-command-trapdoor`
(the two silent-success trapdoors). `introduce-laps-doctor` builds on this by
flagging stray files created before the fix.
