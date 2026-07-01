## ADDED Requirements

### Requirement: Nested stint drain is parent-chain aware
Before gated flow semantics are layered on top of drain/archive, the system SHALL correctly drain
nested stints. When completing the final todo lap in a nested stint, drain SHALL update the stint
reference in that stint's immediate parent queue, archive the drained child stint file, and cascade
upward while ancestor stints have no remaining todo laps. Drain SHALL preserve the existing
partial-failure safety guarantees at every level: a failure SHALL NOT leave a done parent ref over a
present active child file, or a todo parent ref pointing at a missing active child file when the
archived child file exists. `done undo` for an archived nested-stint lap SHALL restore the child
file, reopen the immediate parent ref, reopen any ancestor refs/files needed to make the lap
reachable, and then reopen the lap under the existing latest-completion and age-gate rules.

#### Scenario: Nested child drains through parent ref
- **WHEN** the final todo lap in `root -> auth -> search` is completed
- **THEN** the `search` ref inside `auth` SHALL be marked done and `search.laps.json` SHALL move to archive

#### Scenario: Nested drain cascades upward
- **WHEN** draining child stint `search` leaves parent stint `auth` with no remaining todo laps
- **THEN** `auth` SHALL also drain and archive, and the root `auth` ref SHALL be marked done

#### Scenario: Nested drain failure leaves no dangling parent ref
- **WHEN** a nested drain encounters an archive collision or forced save failure at any cascade level
- **THEN** it SHALL fail without leaving a done parent ref over a present active child file, or a todo parent ref pointing at a missing active child file when the archived child exists

#### Scenario: Undo restores archived nested stint
- **WHEN** `laps done undo` reopens the latest completed lap and that lap is inside an archived nested stint
- **THEN** the nested stint file, required ancestor refs/files, and the lap SHALL be reopened

### Requirement: Hold and release a stint
The system SHALL provide `laps stints hold <name>` and `laps stints release <name>` to mark a
non-archived stint as held or clear the hold, including stints that are not yet enqueued. The
held flag SHALL live on the stint file metadata, fold into schema version 3 before the first
v3/0.9.0 binary ships, default to `false` when absent, and take effect only when a reference to
that stint is encountered at the current context head during flow resolution. Holding SHALL
append a `stint.held` event and releasing SHALL append a `stint.released` event only when the
state changes.

#### Scenario: Hold then release
- **WHEN** `laps stints hold <name>` runs and then `laps stints release <name>` runs
- **THEN** the stint SHALL be marked held and then cleared, with `stint.held` and `stint.released` events appended

#### Scenario: Hold deep in the pipeline has no immediate effect
- **WHEN** a held stint is not at the resolved head
- **THEN** flow operations SHALL be unaffected until that stint reaches the head

#### Scenario: Hold before enqueue
- **WHEN** `laps stints hold auth` runs for a non-archived stint that is not queued
- **THEN** the stint SHALL be marked held and the hold SHALL take effect after the stint is enqueued and reaches the head

#### Scenario: Hold is idempotent
- **WHEN** `laps stints hold auth` runs for an already-held stint
- **THEN** it SHALL leave the stint held and SHALL NOT append another `stint.held` event

### Requirement: Gated flow operations
The system SHALL make head `laps get` and head `laps claim` return no lap — a clean stop rather
than an error — when `get`/`claim` flow-start resolution encounters a held stint at the current
context head, and SHALL NOT select a lap from the held stint. Held interactions SHALL warn on
stderr that the stint is held and should not be implemented yet. The set of gated commands is
closed: only `get` and `claim` flow-start are gated by hold. `list`, `count`, `add`, `edit`,
`assign`, and `delete` SHALL operate normally inside or under a held stint.

#### Scenario: Get on a held head
- **WHEN** the resolved head is a held stint and `laps get` runs
- **THEN** it SHALL return no lap, exit with the held state code, and warn on stderr

#### Scenario: Claim on a held head
- **WHEN** the resolved head is a held stint and `laps claim` runs
- **THEN** it SHALL return no lap, exit with the held state code, leave the claim unchanged, and warn on stderr

#### Scenario: Nested held stint stops descent
- **WHEN** the root head descends into a parent stint whose head is a held child stint
- **THEN** `laps get` SHALL stop at the held child and SHALL NOT select a lap from the child stint

#### Scenario: Explicit get may inspect held stint
- **WHEN** `laps get <id>` targets a lap inside a held stint
- **THEN** it SHALL return the lap and warn on stderr that the stint is held and should not be implemented yet

#### Scenario: Explicit claim is blocked by held stint
- **WHEN** `laps claim <id>` targets a lap inside a held stint
- **THEN** it SHALL exit `10`, leave the claim unchanged, and warn on stderr that the stint is held and should not be implemented yet

#### Scenario: Non-flow-start scoped commands are unaffected by hold
- **WHEN** a stint is held and `laps list`, `laps count`, `laps add`, `laps edit`, `laps assign`, or `laps delete` runs targeting that stint
- **THEN** each command SHALL operate normally with no exit-code change, no held warning, and no mutation block

### Requirement: Queue-state exit codes
`laps get` and `laps claim` SHALL signal queue state via exit code: `0` when a lap is returned,
`10` when gated by a held stint, `11` when the queue is empty, and `12` when all laps are
complete. Exit `11` and `12` SHALL apply only to head/flow operations that are not targeting an
explicit id; exit `10` SHALL also apply to explicit `claim <id>` attempts into a held stint.
Explicit-id not-found errors SHALL continue to use exit `3`, store/io failures exit `2`, and
hook failures exit `4`. Text mode SHALL emit no command stdout for `10`/`11`/`12`; hook passback,
if configured, MAY still write its own stdout. JSON mode SHALL emit a small queue-state object on
stdout. Held cases SHALL warn on stderr.

#### Scenario: Lap returned
- **WHEN** `laps get` resolves a lap
- **THEN** it SHALL exit `0`

#### Scenario: Gated
- **WHEN** `laps get` resolves to a held stint at the head
- **THEN** it SHALL exit `10`

#### Scenario: Empty
- **WHEN** `laps get` runs with no laps anywhere
- **THEN** it SHALL exit `11`

#### Scenario: Complete
- **WHEN** `laps get` runs and every lap is done
- **THEN** it SHALL exit `12`

#### Scenario: Explicit id not found is still not-found
- **WHEN** `laps get <missing-id>` runs
- **THEN** it SHALL exit `3` rather than `10`, `11`, or `12`

### Requirement: Hold blocks starting, not finishing
A hold SHALL block starting the next lap (`get`/`claim`) but SHALL NOT block completing the
claimed lap. `laps done` for the claimed lap SHALL succeed while its stint or the head stint is
held.

#### Scenario: Finish under hold
- **WHEN** a non-final lap is claimed, its stint is held, and a bare `laps done` runs
- **THEN** the claimed lap SHALL be completed and the next `laps get` SHALL exit `10`

#### Scenario: Final lap under hold drains
- **WHEN** the final todo lap of a held stint is claimed and a bare `laps done` runs
- **THEN** the claimed lap SHALL be completed, normal drain/archive behavior SHALL run, and the next flow state SHALL NOT be held by that drained stint

### Requirement: Held state in status
`laps status` SHALL report a primary `held` state, the held stint, and the gate message when the
resolved head is held and no valid active claim takes precedence, in both text and
`--json-output`, and SHALL exit `0` for valid snapshots. When a valid active claim exists,
`status.state` SHALL remain `active` and held gate metadata SHALL be included separately. This
`held` state **extends** the `status` capability's state taxonomy (`active`/`ready`/`empty`/
`complete`, added by `add-event-log-and-status`); the merged set of `status.state` values is
`active`/`ready`/`empty`/`complete`/`held`, and consumers SHALL NOT treat the original four as a
closed list.

#### Scenario: Status reports held
- **WHEN** the resolved head is a held stint, there is no active-claim precedence conflict, and `laps status` runs
- **THEN** it SHALL report state `held` with the held stint, and exit `0`

#### Scenario: Active claim takes precedence over held gate
- **WHEN** a valid lap is claimed and the next head is a held stint
- **THEN** `laps status` SHALL report state `active` and include held gate metadata separately
