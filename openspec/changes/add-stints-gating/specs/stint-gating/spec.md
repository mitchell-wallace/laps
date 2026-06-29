## ADDED Requirements

### Requirement: Hold and release a stint
The system SHALL provide `laps stints hold <name>` and `laps stints release <name>` to mark a
stint as held or clear the hold. The held flag SHALL travel with the stint reference and SHALL
take effect only when that reference is encountered at the current context head during flow
resolution. Holding SHALL append a `stint.held`
event and releasing SHALL append a `stint.released` event to the event log.

#### Scenario: Hold then release
- **WHEN** `laps stints hold <name>` runs and then `laps stints release <name>` runs
- **THEN** the stint SHALL be marked held and then cleared, with `stint.held` and `stint.released` events appended

#### Scenario: Hold deep in the pipeline has no immediate effect
- **WHEN** a held stint is not at the resolved head
- **THEN** flow operations SHALL be unaffected until that stint reaches the head

### Requirement: Gated flow operations
The system SHALL make `laps get` and `laps claim` return no lap — a clean stop rather than an
error — when `get`/`claim` flow-start resolution encounters a held stint at the current context
head, and SHALL NOT descend into the held stint.

#### Scenario: Get on a held head
- **WHEN** the resolved head is a held stint and `laps get` runs
- **THEN** it SHALL return no lap

#### Scenario: Claim on a held head
- **WHEN** the resolved head is a held stint and `laps claim` runs
- **THEN** it SHALL return no lap

#### Scenario: Nested held stint stops descent
- **WHEN** the root head descends into a parent stint whose head is a held child stint
- **THEN** `laps get` SHALL stop at the held child and SHALL NOT open the child stint file

### Requirement: Queue-state exit codes
`laps get` and `laps claim` SHALL signal queue state via exit code: `0` when a lap is returned,
`10` when gated by a held stint, `11` when the queue is empty, and `12` when all laps are
complete. These codes SHALL apply only to head/flow operations that are not targeting an
explicit id; explicit-id not-found errors SHALL continue to use exit `3`, store/io failures
exit `2`, and hook failures exit `4`.

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
`laps status` SHALL report a `held` state, the held stint, and the gate message when the
resolved head is held and no active-claim precedence rule overrides it, in both text and
`--json-output`, and SHALL exit `0` for valid snapshots.

#### Scenario: Status reports held
- **WHEN** the resolved head is a held stint, there is no active-claim precedence conflict, and `laps status` runs
- **THEN** it SHALL report state `held` with the held stint, and exit `0`
