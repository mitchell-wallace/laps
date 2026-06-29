## ADDED Requirements

### Requirement: Read-through resolution for flow operations
The flow operations `get`, `claim`, `done`, and `list` SHALL resolve to the deepest active
context: starting at the root head and descending through active stint references until the
head is a lap. Resolution SHALL be recursive, supporting nested stints, and SHALL be invisible
to agents — `get` SHALL return a lap's title and description regardless of nesting.

#### Scenario: Get descends into the active stint
- **WHEN** the root head is a stint reference and `laps get` runs
- **THEN** it SHALL return the head lap of that stint, with no indication of nesting

#### Scenario: Resolution is recursive
- **WHEN** the active stint's head is itself a stint reference
- **THEN** resolution SHALL continue descending until it reaches a lap

### Requirement: Scope flags
Every command SHALL accept mutually-exclusive scope flags that select the resolution target:
`--active`/`-c` (the deepest active context, the default), `--root`/`-r` (the root queue, no
descent), and `--stint <name>`/`-s` (a named stint, no descent). Combining two scope flags
SHALL be an error.

#### Scenario: Default is active
- **WHEN** a command runs with no scope flag
- **THEN** it SHALL resolve to the deepest active context

#### Scenario: Root scope
- **WHEN** a command runs with `--root`
- **THEN** it SHALL resolve to the root queue without descending

#### Scenario: Named stint scope
- **WHEN** a command runs with `--stint <name>`
- **THEN** it SHALL resolve to that stint without descending

#### Scenario: Combined scope flags error
- **WHEN** a command is given two scope flags
- **THEN** it SHALL fail with an error

### Requirement: Scoped structure operations
The structure operations `add`, `move`, `edit`, and `delete` SHALL default to the active scope.
An explicit id SHALL resolve within the selected scope; when the id is not in scope but exists
in another stint, the command SHALL fail with a message naming that stint.

#### Scenario: Add defaults to the active stint
- **WHEN** `laps add head ...` runs while a stint is active
- **THEN** the new lap SHALL be added at the head of the active stint

#### Scenario: Root-scoped add
- **WHEN** `laps add tail --root ...` runs
- **THEN** the new lap SHALL be added to the root queue

#### Scenario: Out-of-scope id names the stint
- **WHEN** an explicit id is not in the selected scope but exists in another stint
- **THEN** the command SHALL fail with a message naming that stint

### Requirement: Claim records scope
A claim SHALL record the scope in which the lap was claimed in addition to the lap id. A bare
`laps done` SHALL complete the claimed lap within its recorded scope, regardless of the current
head, so preemption or a concurrent enqueue cannot redirect it. A claimed, undone lap SHALL
keep its stint from draining, so the recorded scope SHALL always remain resolvable.

#### Scenario: Claim stores scope
- **WHEN** `laps claim` records a claim inside a stint
- **THEN** the claim record SHALL include that stint as its scope

#### Scenario: Done survives preemption
- **WHEN** a lap is claimed, another stint is then enqueued at the head, and a bare `laps done` runs
- **THEN** `done` SHALL complete the originally claimed lap within its recorded scope

### Requirement: Logged scope reflects resolution
When a command resolves into a stint, the `scope` field of any event it logs SHALL be that
stint's path rather than `root`.

#### Scenario: Event records the resolved scope
- **WHEN** a lap inside stint `auth` is completed
- **THEN** the logged `completed` event SHALL have `scope` of `auth`
