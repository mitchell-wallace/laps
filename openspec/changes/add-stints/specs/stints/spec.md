## ADDED Requirements

### Requirement: Heterogeneous queue with stint references
A queue entry SHALL be either a lap or a stint reference. The on-disk schema SHALL be
version 3, adding a `kind` discriminator with values `lap` and `stint`; an entry without
`kind` SHALL be treated as a lap. A version-2 file SHALL migrate to version 3 by stamping
`kind:"lap"` on every entry and bumping the version. A stint reference SHALL name its stint
by `ref` and SHALL carry its own id, title/display field, order key, completion state, and
timestamps like a lap. Stint-reference ids SHALL be unique within their containing queue file;
the `ref` SHALL be the stable lookup key for the child stint file.

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
main queue. Drained stints SHALL be stored under `.laps/stints/archive/`. Stint names SHALL be
file-safe names rather than paths: blank names, path separators, `.`/`..`, and names colliding
with an existing active or archived stint SHALL be rejected. Archive moves SHALL NOT overwrite
an existing file.

#### Scenario: Creating a stint
- **WHEN** `laps stints new <name>` runs
- **THEN** an empty stint file SHALL be created at `.laps/stints/<name>.laps.json`

#### Scenario: Unsafe stint name rejected
- **WHEN** `laps stints new ../auth` runs
- **THEN** the command SHALL fail without creating or overwriting files outside `.laps/stints/`

### Requirement: Globally-unique lap ids via stint prefixes
Lap ids SHALL be globally unique across the root queue and all stint files, so a lap id also
identifies its owning queue. Each stint SHALL be allocated a 4-character prefix at creation,
recorded in the stint file metadata; laps created inside a stint SHALL use that prefix, while
root laps SHALL keep the repository prefix. A stint prefix SHALL be derived from the stint name
(first 4 lowercase alphanumerics) and SHALL be made unique against the repository prefix and all
existing stint prefixes by trying alternative permutations/substrings of the stint-name
characters and then incrementing the final character through `0-9a-z`.

#### Scenario: Stint laps use the stint prefix
- **WHEN** a lap is created inside stint `auth`
- **THEN** its id SHALL begin with `auth`'s allocated prefix and SHALL NOT collide with any root or other-stint lap id

#### Scenario: Prefix collision is resolved
- **WHEN** a new stint's first-4-character prefix collides with the repository prefix or an existing stint prefix
- **THEN** allocation SHALL pick a different unique 4-character prefix rather than reuse the colliding one

### Requirement: Stint commands
The system SHALL provide `laps stints` with subcommands `ls`, `new <name>`,
`enqueue <name> [head|tail|after <id>]`, `show <name>`, and `rm <name> [--force]`, plus `st` as an
alias for `stints`. `laps stints` and `laps st` SHALL be registered as built-ins before
hook-only command interception. `laps stints ls` SHALL list stint files with lap counts and a
queued indicator; unqueued and empty stints SHALL be ordinary listed stint files rather than a
separate lifecycle state. `stints rm` SHALL remove unqueued non-archived stint files and archived
stints, including archived stints that still have a done root ref. It SHALL refuse non-archived
queued, active, or claimed stints unless `--force` is supplied; forced removal SHALL remove
matching root refs and clear matching claims.

#### Scenario: Listing stints
- **WHEN** `laps stints ls` runs
- **THEN** each stint SHALL be listed with lap counts and whether it is queued

#### Scenario: Alias
- **WHEN** `laps st ls` runs
- **THEN** it SHALL behave identically to `laps stints ls`

#### Scenario: Remove queued stint requires force
- **WHEN** `laps stints rm auth` names a non-archived queued stint
- **THEN** it SHALL fail without removing the stint file or root ref

#### Scenario: Forced remove queued stint
- **WHEN** `laps stints rm auth --force` names a non-archived queued stint
- **THEN** it SHALL remove the stint file and matching root ref, and clear a matching claim if present

#### Scenario: Remove archived stint with done ref
- **WHEN** `laps stints rm auth` names an archived stint that still has a done root ref
- **THEN** it SHALL remove the archived file and the done root ref

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
position in the queue. `laps done undo` SHALL reopen the globally most-recent completion by
scanning all queue files (root, active stints, and `.laps/stints/archive/`) for the greatest
`completedAt`; when that lap is in an archived stint it SHALL restore the stint file from archive,
reopen the stint reference, and reopen the lap under the existing undo rules.

#### Scenario: Completing the last lap drains and archives
- **WHEN** `laps done` completes the final todo lap of a stint
- **THEN** the stint reference SHALL be marked done and the stint file SHALL move to `.laps/stints/archive/`

#### Scenario: Archive collision is refused
- **WHEN** a drained stint would archive over an existing archived file with the same name
- **THEN** the command SHALL fail without overwriting the archived file

#### Scenario: A non-head stint still drains
- **WHEN** the final todo lap of a stint that is not at the head is completed
- **THEN** that stint SHALL drain and archive regardless of its position

#### Scenario: Undo unarchives a drained stint
- **WHEN** `laps done undo` reopens the globally latest completed lap and that lap's stint file is archived
- **THEN** the stint file SHALL move back to `.laps/stints/`, the stint reference SHALL reopen, and the lap SHALL reopen

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
