## ADDED Requirements

### Requirement: Reorder an existing lap
The system SHALL provide `laps move <id> <head|tail|after <id>>` to move an existing todo
lap to a new queue position, preserving the lap's id and using the same ordering rules as
`laps add`. The command SHALL operate only on todo laps; it SHALL fail when the id is
unknown or already done. When the `after` target is a done lap, the moved lap SHALL be
placed at the head, with a notice on stderr, mirroring `laps add after`.

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

### Requirement: Edit lap fields
The system SHALL provide `laps edit <id>` with `--title`, `--description`, and
`--assignee` flags that update the named fields of an existing lap in place and update its
`updatedAt` timestamp. At least one field flag SHALL be required; an invocation with no
field flags SHALL fail and leave the lap unchanged.

#### Scenario: Edit a field
- **WHEN** `laps edit <id> --assignee SENIOR` runs
- **THEN** the lap's assignee SHALL become `SENIOR` and its `updatedAt` SHALL advance

#### Scenario: Edit with no fields fails
- **WHEN** `laps edit <id>` runs with no field flags
- **THEN** the command SHALL fail and the lap SHALL be unchanged

### Requirement: Assign shortcut
The system SHALL provide `laps assign <id> <role>` as a shortcut equivalent to
`laps edit <id> --assignee <role>`.

#### Scenario: Assign sets the assignee
- **WHEN** `laps assign <id> VERIFY` runs
- **THEN** the lap's assignee SHALL become `VERIFY`

### Requirement: Structured output for edits
The `move`, `edit`, and `assign` commands SHALL honor the global `--json-output` flag,
emitting the affected lap as a `task` object.

#### Scenario: JSON output for an edit
- **WHEN** `laps move <id> head --json-output` runs
- **THEN** the affected lap SHALL be emitted as a JSON `task` object
