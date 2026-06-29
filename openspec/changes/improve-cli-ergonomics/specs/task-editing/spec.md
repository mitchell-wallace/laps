## ADDED Requirements

### Requirement: Reorder an existing lap
The system SHALL provide `laps move <id> <head|tail|after <id>>` to move an existing todo
lap to a new queue position, preserving the lap's id and using the same ordering rules as
`laps add`. The command SHALL operate only on todo laps; it SHALL fail when the id is
unknown or already done. When the `after` target is a done lap, the moved lap SHALL be
placed at the head, with a notice on stderr, mirroring `laps add after`. A successful move
SHALL advance the moved lap's `updatedAt` timestamp.

#### Scenario: Move to head
- **WHEN** `laps move <id> head` runs for a todo lap
- **THEN** the lap SHALL become the head of the todo queue and retain its id

#### Scenario: Move after another lap
- **WHEN** `laps move <id> after <other>` runs and `<other>` is a todo lap
- **THEN** the lap SHALL be positioned immediately after `<other>`

#### Scenario: Move rejects done or unknown ids
- **WHEN** `laps move <id> ...` names a lap that is done or does not exist
- **THEN** the command SHALL fail without modifying the queue

#### Scenario: Move after a done target falls back to head
- **WHEN** `laps move <id> after <done-id>` names a done `<done-id>`
- **THEN** the lap SHALL be placed at the head and a notice SHALL be written to stderr

#### Scenario: Move after an unknown target fails
- **WHEN** `laps move <id> after <missing-id>` names no existing lap as the target
- **THEN** the command SHALL fail without modifying the queue

### Requirement: Edit lap fields
The system SHALL provide `laps edit <id>` with `--title`, `--description`, and
`--assignee` flags that update the named fields of an existing lap in place and update its
`updatedAt` timestamp. At least one field flag SHALL be required; an invocation with no
field flags SHALL fail and leave the lap unchanged. The command SHALL allow editing done laps;
when it edits a done lap, it SHALL warn on stderr while preserving `isDone` and `completedAt`.

#### Scenario: Edit a field
- **WHEN** `laps edit <id> --assignee SENIOR` runs
- **THEN** the lap's assignee SHALL become `SENIOR` and its `updatedAt` SHALL advance

#### Scenario: Edit with no fields fails
- **WHEN** `laps edit <id>` runs with no field flags
- **THEN** the command SHALL fail and the lap SHALL be unchanged

#### Scenario: Edit a done lap warns but preserves completion
- **WHEN** `laps edit <done-id> --title Updated` runs for a done lap
- **THEN** the command SHALL succeed, warn on stderr, update the title and `updatedAt`, and preserve `isDone` and `completedAt`

### Requirement: Assign shortcut
The system SHALL provide `laps assign <id> <role>` as a shortcut equivalent to
`laps edit <id> --assignee <role>`. Assigning a done lap SHALL follow the same warn-but-allow
behavior as `edit`.

#### Scenario: Assign sets the assignee
- **WHEN** `laps assign <id> VERIFY` runs
- **THEN** the lap's assignee SHALL become `VERIFY`

### Requirement: Structured output for edits
The `move`, `edit`, and `assign` commands SHALL honor the global `--json-output` flag,
emitting the affected lap as a `task` object.

#### Scenario: JSON output for an edit
- **WHEN** `laps move <id> head --json-output` runs
- **THEN** the affected lap SHALL be emitted as a JSON `task` object

### Requirement: Edit commands participate in hook dispatch
The `move`, `edit`, and `assign` commands SHALL be registered as built-ins before hook-only
command interception and SHALL run the existing before/after hook lifecycle for mutating
commands. Hook variables SHALL include the affected task, selected task file, positional args
as existing built-in hooks receive them, success output, and exit code.

#### Scenario: Move is not intercepted as hook-only
- **WHEN** `laps move <id> head` runs in a repo with hooks configured for unknown command names
- **THEN** the built-in `move` command SHALL execute rather than dispatching as a hook-only command

#### Scenario: Edit hook context
- **WHEN** `laps edit <id> --title Updated` succeeds
- **THEN** before/after hooks for `edit` SHALL receive the affected lap id, title, file, output, and exit code
