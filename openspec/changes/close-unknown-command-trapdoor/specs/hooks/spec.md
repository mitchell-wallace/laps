# hooks (delta)

## ADDED Requirements

### Requirement: Unknown commands SHALL fail loudly

When the first positional token of a `laps` invocation is neither a built-in
command (a registered command name or alias) nor a hook-only command declared
in `.laps/hooks.json`, laps SHALL exit non-zero and print an
`unknown command` error to stderr, including near-miss suggestions when a
registered command is within edit distance.

#### Scenario: Typo of a built-in

- **GIVEN** an initialized repo with no `.laps/hooks.json`
- **WHEN** the user runs `laps stinst ls`
- **THEN** the exit code is non-zero
- **AND** stderr contains `unknown command` and suggests `stints`
- **AND** no task file or hook is touched

#### Scenario: Undeclared hook-only name

- **GIVEN** a `.laps/hooks.json` whose hooks declare only `command: "worktree"`
- **WHEN** the user runs `laps deploy`
- **THEN** the exit code is non-zero and stderr reports an unknown command

### Requirement: Hook-only dispatch SHALL be declaration-gated

A non-built-in command name SHALL be dispatched as a hook-only command if and
only if at least one hook entry in `.laps/hooks.json` declares that name in
its `command` field. Declared hook-only dispatch SHALL preserve existing
semantics: `before` and `after` hooks fire with the documented variables
(`$command`, `$file`, `$scope`, `$args`, `$1..$n`), passback stdout is
printed, and `-f`/`--file` is honored for `$file` resolution.

#### Scenario: Declared hook-only command still works

- **GIVEN** a `.laps/hooks.json` with a `before` hook for `command: "worktree"`
- **WHEN** the user runs `laps worktree feature-x`
- **THEN** the hook runs with `$1` = `feature-x` and laps exits `0`

### Requirement: Built-in command registration SHALL be single-source

The set of built-in command names used by dispatch SHALL be derived from the
registered Cobra command tree (names plus aliases). Adding a new subcommand
SHALL require no second registration for it to be dispatched.

#### Scenario: Newly registered subcommand is dispatched

- **GIVEN** a build in which a new subcommand has been added via `rootCmd.AddCommand`
- **WHEN** the user invokes that subcommand
- **THEN** the Cobra command runs (it is not intercepted as a hook-only command)
