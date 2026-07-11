# scope-resolution (delta)

## ADDED Requirements

### Requirement: Read paths SHALL be fail-closed for missing file targets

A command that does not create queue content (`get`, `claim`, `list`, `count`,
`status`, `log`, `done`, `move`, `edit`, `assign`, `delete`, `prune`) SHALL
exit `3` with a `task file <name> not found` error when its `-f/--file` target
does not exist, and SHALL NOT create any file.

#### Scenario: Misspelled stint path on a read command

- **GIVEN** an initialized repo with stint `auth` (`.laps/stints/auth.laps.json`)
- **WHEN** the user runs `laps list -f stints/auth`
- **THEN** the exit code is `3`
- **AND** stderr suggests `--stint auth`
- **AND** no file named `.laps/stints/auth.json` is created

#### Scenario: Unknown file on a read command

- **GIVEN** an initialized repo
- **WHEN** the user runs `laps count -f scratch`
- **THEN** the exit code is `3` and `.laps/scratch.json` is not created

### Requirement: File creation SHALL be verb-gated

Only `laps add` and `laps init` SHALL create a missing `-f/--file` target.
The default store `.laps/laps.json` SHALL continue to be initialized on
demand regardless of command.

#### Scenario: add creates a new named file

- **GIVEN** an initialized repo without `.laps/staging.json`
- **WHEN** the user runs `laps add tail -f staging --title "t"`
- **THEN** the command succeeds and `.laps/staging.json` exists with the new lap

#### Scenario: Default store still self-initializes

- **GIVEN** a repo with a `.laps/` directory but no `laps.json`
- **WHEN** the user runs `laps list`
- **THEN** the command succeeds against an empty default queue
