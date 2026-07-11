# Tasks

## 1. Characterize current behavior

- [ ] 1.1 Add characterization tests: `list -f missing` today exits 0 and creates `.laps/missing.json`; `list -f stints/auth` creates `.laps/stints/auth.json` beside a real `auth.laps.json`. Note intended post-change expectations inline.
- [ ] 1.2 Enumerate all `loadFile` callers in `internal/cmd/*.go` and classify each command as create-on-missing (`add`, `init`) or fail-closed (everything else). Stop and re-scope if a command doesn't fit the two buckets (e.g. verify `stints` subcommands use their own loaders).

## 2. Introduce the new shape

- [ ] 2.1 In `internal/store/store.go`, add `ErrFileNotFound` distinct from `ErrEmptyFile`; make `Load` return not-found only when the file is absent. Update `CheckDefaultStore` (`store.go:687-694`) to accept both for the default store.
- [ ] 2.2 Thread a creation policy into `loadFile` (`internal/cmd/root.go:196-241`): `createMissing` for `add`/`init`; fail-closed exit `3` otherwise for non-default targets. Preserve the empty-but-existing stint prefix-initialization path.
- [ ] 2.3 Add `suggestTarget(beadsDir, fileFlag)` producing the `--stint <name>` hint for `stints/`-prefixed values and names matching existing active stints; unit-test both shapes plus no-suggestion fallback.
- [ ] 2.4 Ensure the JSON error object (`root.go:58-70`) carries the same message and `exitCode: 3` under `--json-output`.

## 3. Migrate usage safely

- [ ] 3.1 Update `README.md`: `--file` section documents fail-closed reads and verb-gated creation; Scope-flags section cross-references the suggestion behavior.
- [ ] 3.2 Release-notes bullet: read commands no longer create missing `-f` targets (exit `3` + hint); `add`/`init` unchanged.
- [ ] 3.3 Cross-check the rally adapter (sibling repo, `internal/laps/adapter.go`) for read-path `-f` usage; record the finding in the change (expected: none). Stop if rally reads a `-f` target it never creates.

## 4. Verify and retire old paths

- [ ] 4.1 Run `just test` and `just lint`; expect green.
- [ ] 4.2 Manual smoke in a throwaway dir (never this repo's `.laps/`): reproduce the typo transcript; assert exit `3`, the hint, and that `find .laps -type f` shows no new files.
- [ ] 4.3 Confirm no remaining code path calls `store.Save` from a read-only command (grep `Save(` usages under `internal/cmd/` and justify each).
