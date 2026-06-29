## ADDED Requirements

### Requirement: Read-through resolution for flow operations
The flow operations `get`, `claim`, `done`, and `list` SHALL resolve to the deepest active
context: starting at the root head and descending through active stint references until the
head is a lap. Resolution SHALL be recursive, supporting nested stints, and SHALL be invisible
to agents — `get` SHALL return a lap's title and description regardless of nesting. Resolution
SHALL detect missing child files, malformed stint references, malformed child files, and cycles
with a visited set; these failures SHALL error rather than loop or silently skip a stint.

#### Scenario: Get descends into the active stint
- **WHEN** the root head is a stint reference and `laps get` runs
- **THEN** it SHALL return the head lap of that stint, with no indication of nesting

#### Scenario: Resolution is recursive
- **WHEN** the active stint's head is itself a stint reference
- **THEN** resolution SHALL continue descending until it reaches a lap

#### Scenario: Cyclic resolution fails
- **WHEN** recursive resolution encounters a stint ref already visited in the current descent
- **THEN** the command SHALL fail with a cycle error and SHALL NOT mutate any file

### Requirement: Scope flags
Queue-targeting commands SHALL accept shared local, mutually-exclusive scope flags that select the resolution
target: `--active`/`-c` (the deepest active context, the default), `--root`/`-r` (the root queue,
no descent), and `--stint <name>`/`-s` (a named stint, no descent). Combining two scope flags
SHALL be an error. Commands without a queue target (`init`, `on`, `off`, `update`, `version`,
`help`, hook-only commands, `log`, and `status`) SHALL NOT inherit these flags unless their own
specification adds scope-specific filtering. Raw `--file` SHALL be mutually exclusive with all
scope flags.

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

#### Scenario: File flag with scope flag errors
- **WHEN** a command is given `--file other --stint auth`
- **THEN** it SHALL fail with an error rather than choosing one target model

### Requirement: Scoped structure operations
The structure operations `add`, `move`, `edit`, `assign`, and `delete` SHALL default to the
active scope (`assign` follows `edit`, of which it is a shortcut).
Every id-taking queue operation SHALL resolve explicit ids within the selected scope first,
including `get <id>`, `claim <id>`, `done <id>`, `add after <id>`, `move`, `edit`, `assign`,
and `delete`. When the id is not in scope but exists in another stint, the command SHALL fail
with a message naming that stint and SHALL NOT mutate any file. `stints enqueue after <id>`
SHALL resolve the `after` id only in the root queue; if the id exists only inside a stint, it
SHALL fail with a message naming that stint.
Deleting a claimed lap SHALL refuse by default with a stderr warning; `delete --force` SHALL
remove the lap and clear the matching claim.

#### Scenario: Add defaults to the active stint
- **WHEN** `laps add head ...` runs while a stint is active
- **THEN** the new lap SHALL be added at the head of the active stint

#### Scenario: Root-scoped add
- **WHEN** `laps add tail --root ...` runs
- **THEN** the new lap SHALL be added to the root queue

#### Scenario: Out-of-scope id names the stint
- **WHEN** an explicit id is not in the selected scope but exists in another stint
- **THEN** the command SHALL fail with a message naming that stint

#### Scenario: Claimed delete requires force
- **WHEN** `laps delete <claimed-id>` runs for a claimed lap
- **THEN** the command SHALL fail with a warning and SHALL leave the claim and lap unchanged

#### Scenario: Forced claimed delete clears claim
- **WHEN** `laps delete --force <claimed-id>` runs for a claimed lap
- **THEN** the command SHALL remove the lap and clear the matching claim

#### Scenario: Stint enqueue after is root-only
- **WHEN** `laps stints enqueue auth after <id>` names an id that exists only inside another stint
- **THEN** the command SHALL fail with a message naming that stint and SHALL NOT enqueue `auth`

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
stint's canonical scope string rather than `root`. Canonical scopes SHALL be `root`, the
root-level stint name, or slash paths for nested stints such as `auth/search`. Hooks for scoped
operations SHALL receive `$file` as the resolved physical task file and `$scope` as the same
canonical logical scope string.

#### Scenario: Event records the resolved scope
- **WHEN** a lap inside stint `auth` is completed
- **THEN** the logged `completed` event SHALL have `scope` of `auth`

#### Scenario: Nested scope is slash encoded
- **WHEN** a lap inside nested stint `search` under `auth` is completed
- **THEN** the logged event and hook `$scope` SHALL use `auth/search`
