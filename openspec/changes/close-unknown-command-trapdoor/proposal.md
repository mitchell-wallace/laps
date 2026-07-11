# Proposal: Close the unknown-command trapdoor

## Intent

`laps <anything-not-in-a-hand-maintained-list>` currently exits `0` and prints
nothing. This is the single worst failure mode laps offers an AI agent: a
typo'd or newly added subcommand is a *silent success*. Two of ~20 live agent
sessions this week lost time to it (`stinst ls`-style typos and a new
subcommand that was registered in Cobra but not in `isKnownCommand`, so the
hook intercept swallowed it). Post-1.0, the dispatch path should be fail-closed
and single-source.

## Evidence

- `internal/cmd/root.go:129-170` — the pre-Cobra intercept: any first token not
  in `isKnownCommand` is treated as a hook-only command. With no matching hooks
  in `.laps/hooks.json`, `hooks.Dispatch` returns nothing, and the process
  exits `0` with no output.
- `internal/cmd/hooks.go:114-120` — `isKnownCommand` is a hand-maintained
  string switch that must be updated in lockstep with every
  `rootCmd.AddCommand` call. Nothing enforces parity.
- `internal/cmd/cmd_test.go:5269-5282, 6527, 7131` — regression tests exist
  precisely because `log`, `status`, `move`, `edit`, and `assign` were each at
  risk of (or bitten by) being swallowed; the tests guard names one by one
  instead of fixing the dual registration.
- Verified on a scratch repo with the dev binary: `laps stinst ls` → exit `0`,
  zero output. Cobra's built-in "unknown command … did you mean" suggestions
  never fire because the intercept runs before `rootCmd.Execute()`.
- `README.md` "Hook-only commands" and "Reserved command names" sections — the
  intercept exists to support hook-only commands like `laps worktree`, and the
  README already maintains a growing reserved-name list as a manual workaround
  for the namespace collision this design creates.

## Scope

In scope:
- Derive the built-in command set from the Cobra command tree (names +
  aliases) instead of the `isKnownCommand` switch.
- Dispatch hook-only commands **only when at least one hook in
  `.laps/hooks.json` declares that command name**.
- Route every other unknown token through Cobra so it produces a non-zero
  exit, an `unknown command` error on stderr, and Cobra's near-miss
  suggestions.
- Update the README Hooks section (hook-only commands must be declared; the
  reserved-name list becomes advisory rather than load-bearing).
- Regression tests for: typo'd built-in, undeclared hook-only name, declared
  hook-only name, `--json-output` error shape.

Out of scope:
- Any change to hook execution semantics, variables, or passback.
- A namespaced `laps run <hook>` invocation style (recorded as a design
  alternative; could land later as an alias).
- Changes to the four pinned consumer-contract surfaces.

## Proposed path

Replace the membership test in `Execute` with a lookup against
`rootCmd.Commands()` (walking `Name()` plus `Aliases`, and the few pseudo
tokens like `help` and `--version`). Before treating an unknown token as
hook-only, load `.laps/hooks.json` and check whether any hook declares
`command == <token>`; if none does, fall through to `rootCmd.Execute()` and
let Cobra report the unknown command with suggestions and a non-zero exit.
`isKnownCommand` is deleted; a small parity test asserts every registered
command dispatches through Cobra.

## Expected payoff

- A typo'd command becomes a loud, suggestive error instead of a silent no-op
  — directly fixes the most common agent-session failure observed this week.
- New subcommands need exactly one registration point; the "forgot
  isKnownCommand" class of bug is structurally impossible.
- Hook-only commands keep working exactly as documented, with the added
  benefit that a *misspelled* hook-only command now also errors.

## Risks and unknowns

- **Behavior change**: invoking an undeclared name goes from exit `0`/silence
  to a non-zero error. Any script that (unwisely) probed laps with unknown
  commands would newly fail. Mitigation: this is not one of the four pinned
  consumer-contract surfaces; call it out in release notes and the README, and
  land it in the first post-1.0 minor.
- Hook loading now happens on the unknown-command path before deciding to
  error; a malformed `hooks.json` must keep its existing exit-`2` behavior.
- `splitArgs` (`internal/cmd/hooks.go:122-150`) must keep extracting `-f` for
  hook-only dispatch; the refactor must not change declared-hook behavior.

## Spec impact

Contract delta in the hooks domain: hook-only command dispatch becomes
declaration-gated, and unknown commands become errors. See
`specs/hooks/spec.md`.
