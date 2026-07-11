# Design: Close the unknown-command trapdoor

## Current shape

`cmd.Execute` (`internal/cmd/root.go:98-173`) runs three phases before Cobra:
`--json-output` detection, `--version` interception, then the hook-only
intercept. The intercept calls `splitArgs` to find the first non-flag token
and, when `isKnownCommand(token)` is false, dispatches `before`/`after` hooks
for that name and returns nil — **exit 0 even when zero hooks match**.

`isKnownCommand` (`internal/cmd/hooks.go:114-120`) is a hand-maintained switch
listing 23 names. The Cobra tree registered via `rootCmd.AddCommand` in each
command file's `init()` is the second, authoritative registry. The two have
already drifted at least twice in this repo's history (guarded after the fact
by name-by-name tests in `cmd_test.go:5262-5283`).

Consequences:

1. Typos exit 0 silently (`laps stinst ls`).
2. New built-ins are swallowed until someone remembers the switch.
3. Cobra's suggestion engine is dead code for top-level commands.

## Target shape

```
Execute(args)
  ├─ jsonOutput / --version handling (unchanged)
  ├─ token := first positional (splitArgs, unchanged)
  ├─ if token == "" or builtinNames().contains(token): rootCmd.Execute()
  ├─ else if hooksDeclare(token): hook-only dispatch (unchanged semantics)
  └─ else: rootCmd.Execute()   // Cobra prints "unknown command %q for laps"
                               // + suggestions, exits non-zero
```

- `builtinNames()` walks `rootCmd.Commands()` collecting `cmd.Name()` and
  `cmd.Aliases`, plus `help` and `completion` (Cobra built-ins). Computed once
  after all `init()` registration has run — a `sync.Once` or direct call in
  `Execute` both work since `init()` ordering guarantees registration first.
- `hooksDeclare(token)` loads `.laps/hooks.json` via `hooks.Load` and reports
  whether any hook entry has `command == token`. Load errors keep today's
  exit-`2` path. A missing/empty hooks file declares nothing.
- The scope-flag rejection for hook-only commands
  (`scopeFlagInArgs`, `internal/cmd/scope.go:75-91`) stays on the hook-only
  branch only.
- Set `rootCmd.SuggestionsMinimumDistance = 2` so `stinst` suggests `stints`.

## Alternatives considered

- **Keep the switch, add a parity test**: rejected — fixes drift but not the
  silent exit-0 typo path, and keeps two registries.
- **Require `laps run <hook-name>` for hook-only commands**: cleanest
  namespace separation and zero collision risk, but breaks the documented
  `laps worktree` invocation style and every existing hooks.json consumer.
  Rejected for now; could be added later as an alias without removing
  declaration-gated direct invocation.
- **Auto-register hook-only commands as real Cobra commands at startup**:
  rejected — hooks.json is per-repo runtime data; mutating the command tree
  from it complicates help output and testing for little gain.

## Migration and rollout

Single PR, post-1.0 minor. No data migration. Release notes and README both
state: hook-only commands must be declared in `.laps/hooks.json` to be
invokable (they always were in practice — an undeclared invocation did
nothing); undeclared names now error. The README reserved-name list is
retitled as advisory guidance for forward compatibility.

## Verification strategy

- `just test` — new cases in `internal/cmd/cmd_test.go`:
  - typo of a built-in (`stinst`) exits non-zero, stderr contains
    `unknown command` and the suggestion `stints`.
  - undeclared hook-only name exits non-zero.
  - declared hook-only name still fires before/after hooks, passback, `$args`
    vars, and `-f` extraction (extend the existing hook-only tests).
  - parity: for every `rootCmd.Commands()` name/alias, dispatch reaches the
    Cobra command (generalizes the existing per-name guards).
  - `--json-output` unknown-command error shape.
- `just lint`.
- Manual smoke in a throwaway dir per CLAUDE.md warning: typo, declared hook,
  undeclared hook.

## Dependencies and ordering

Independent. Complements `surface-queue-contract-to-agents` (loud errors are
the other half of discoverability). Should land before any new subcommand is
added post-1.0 so the new subcommand cannot be swallowed.
