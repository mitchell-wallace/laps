# Tasks

## 1. Characterize current behavior

- [x] 1.1 Add failing-intent characterization tests in `internal/cmd/cmd_test.go`: `laps stinst ls` currently exits 0 with no output; an undeclared hook-only name exits 0; a declared hook-only name fires hooks. Mark the first two with the intended post-change expectations (non-zero + `unknown command`).
- [x] 1.2 Inventory every name in `isKnownCommand` (`internal/cmd/hooks.go:114-120`) against `rootCmd.Commands()` names/aliases and the pseudo tokens (`help`, `--version`, `completion`); record any name in the switch with no Cobra command backing it. Stop and re-scope if one exists (false assumption check).

Inventory result: every former switch entry was backed by a registered Cobra
name/alias or the intercepted `--version` pseudo token; no orphaned name was
found.

## 2. Introduce the new shape

- [x] 2.1 Add `builtinNames()` in `internal/cmd/root.go` (or a new `registry.go`) walking `rootCmd.Commands()` for `Name()` + `Aliases`, plus `help`/`completion`; unit-test it.
- [x] 2.2 Add `hooksDeclare(beadsDir, name string) bool` using `hooks.Load`; malformed hooks.json keeps exit `2`; missing file declares nothing.
- [x] 2.3 Rewire `Execute` (`internal/cmd/root.go:129-170`): known → Cobra; unknown+declared → existing hook-only dispatch; unknown+undeclared → fall through to `rootCmd.Execute()`.
- [x] 2.4 Set `rootCmd.SuggestionsMinimumDistance = 2` and verify Cobra emits suggestions for near-miss typos.
- [x] 2.5 Delete `isKnownCommand` and replace the per-name guard tests (`cmd_test.go:5262-5283`) with a single parity test over `builtinNames()`.

## 3. Migrate usage safely

- [x] 3.1 Confirm `splitArgs` `-f` extraction and `$args`/`$1..$n` hook vars are unchanged on the declared hook-only path (extend existing hook tests rather than rewriting them).
- [x] 3.2 Update `README.md` "Hook-only commands" (declaration-gated invocation; undeclared names error) and retitle "Reserved command names" as advisory.
- [x] 3.3 Add a release-notes bullet: undeclared unknown commands change from silent exit 0 to a non-zero `unknown command` error; not a pinned consumer-contract surface.

## 4. Verify and retire old paths

- [x] 4.1 Run `just test` and `just lint`; expect green.
- [x] 4.2 Manual smoke in a throwaway dir (never this repo's `.laps/`, per CLAUDE.md): `laps stinst ls` errors with suggestion; `laps worktree` with the example hooks.json still fires hooks; `laps worktree` without hooks errors.
- [x] 4.3 Grep for any remaining reference to `isKnownCommand`; expect none.
