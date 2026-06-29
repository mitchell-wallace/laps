## ADDED Requirements

### Requirement: Heterogeneous queue with stint references
A queue entry SHALL be either a lap or a stint reference. The on-disk schema SHALL be
version 3, adding a `kind` discriminator with values `lap` and `stint`; an entry without
`kind` SHALL be treated as a lap. A version-2 file SHALL migrate to version 3 by stamping
`kind:"lap"` on every entry and bumping the version. A stint reference SHALL name its stint
by `ref` and SHALL carry its own order key, completion state, and timestamps like a lap.

#### Scenario: Version 2 migrates to version 3
- **WHEN** a version-2 file is loaded
- **THEN** every entry SHALL be stamped `kind:"lap"` and the file version SHALL become 3

#### Scenario: Missing kind is a lap
- **WHEN** a queue entry has no `kind` field
- **THEN** it SHALL be treated as a lap

#### Scenario: Mixed queue
- **WHEN** `laps.json` contains both laps and stint references
- **THEN** both SHALL load and order by their order keys

### Requirement: Stint files
A stint SHALL be stored at `.laps/stints/<name>.laps.json` using the same file schema as the
main queue. Drained stints SHALL be stored under `.laps/stints/archive/`.

#### Scenario: Creating a stint
- **WHEN** `laps stints new <name>` runs
- **THEN** an empty stint file SHALL be created at `.laps/stints/<name>.laps.json`

### Requirement: Stint commands
The system SHALL provide `laps stints` with subcommands `ls`, `new <name>`,
`enqueue <name> [head|tail|after <id>]`, `show <name>`, and `rm <name>`, plus `st` as an
alias for `stints`. `laps stints ls` SHALL show each stint's state and its todo/done counts.

#### Scenario: Listing stints
- **WHEN** `laps stints ls` runs
- **THEN** each stint SHALL be listed with its state (queued/active/done) and todo/done counts

#### Scenario: Alias
- **WHEN** `laps st ls` runs
- **THEN** it SHALL behave identically to `laps stints ls`

### Requirement: Enqueue a stint
`laps stints enqueue <name>` SHALL insert a stint reference into the root queue using the same
ordering rules as `laps add`, defaulting to `tail`. Enqueuing at `head` SHALL preempt the
active stint non-destructively: the active stint's partial progress SHALL be preserved in its
file and resume when the preempting stint drains.

#### Scenario: Enqueue defaults to tail
- **WHEN** `laps stints enqueue <name>` runs with no position
- **THEN** the stint reference SHALL be appended at the tail of the root queue

#### Scenario: Head enqueue preempts non-destructively
- **WHEN** `laps stints enqueue <name> head` runs while another stint is active
- **THEN** the new stint SHALL become active and the preempted stint SHALL retain its progress and resume when the new stint drains

### Requirement: Stint drain and auto-archive
When a stint has no remaining todo laps it SHALL be considered drained. The operation that
drains it SHALL mark its stint reference done (setting `completedAt`) and SHALL move the stint
file to `.laps/stints/archive/`. Draining SHALL be content-based and independent of the stint's
position in the queue.

#### Scenario: Completing the last lap drains and archives
- **WHEN** `laps done` completes the final todo lap of a stint
- **THEN** the stint reference SHALL be marked done and the stint file SHALL move to `.laps/stints/archive/`

#### Scenario: A non-head stint still drains
- **WHEN** the final todo lap of a stint that is not at the head is completed
- **THEN** that stint SHALL drain and archive regardless of its position

### Requirement: Stint reporting and events
Stint operations SHALL append `stint.enqueued`, `stint.completed`, and `stint.archived`
events to the event log, and `laps status` SHALL report the active stint and per-stint
progress.

#### Scenario: Enqueue is logged
- **WHEN** `laps stints enqueue <name>` runs
- **THEN** a `stint.enqueued` event SHALL be appended

#### Scenario: Status reports the active stint
- **WHEN** `laps status` runs while a stint is active
- **THEN** it SHALL report the active stint and its progress
