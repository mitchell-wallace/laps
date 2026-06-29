## ADDED Requirements

### Requirement: Hold and release a stint
The system SHALL provide `laps stints hold <name>` and `laps stints release <name>` to mark a
stint as held or clear the hold. The held flag SHALL travel with the stint and SHALL take
effect only when the stint reaches the resolved head. Holding SHALL append a `stint.held`
event and releasing SHALL append a `stint.released` event to the event log.

#### Scenario: Hold then release
- **WHEN** `laps stints hold <name>` runs and then `laps stints release <name>` runs
- **THEN** the stint SHALL be marked held and then cleared, with `stint.held` and `stint.released` events appended

#### Scenario: Hold deep in the pipeline has no immediate effect
- **WHEN** a held stint is not at the resolved head
- **THEN** flow operations SHALL be unaffected until that stint reaches the head

### Requirement: Gated flow operations
When the resolved head is a held stint, `laps get` and `laps claim` SHALL return no lap — a
clean stop rather than an error — and SHALL NOT descend into the held stint.

#### Scenario: Get on a held head
- **WHEN** the resolved head is a held stint and `laps get` runs
- **THEN** it SHALL return no lap

#### Scenario: Claim on a held head
- **WHEN** the resolved head is a held stint and `laps claim` runs
- **THEN** it SHALL return no lap

### Requirement: Queue-state exit codes
`laps get` and `laps claim` SHALL signal queue state via exit code: `0` when a lap is returned,
`10` when gated by a held stint, `11` when the queue is empty, and `12` when all laps are
complete.

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

### Requirement: Hold blocks starting, not finishing
A hold SHALL block starting the next lap (`get`/`claim`) but SHALL NOT block completing the
claimed lap. `laps done` for the claimed lap SHALL succeed while its stint or the head stint is
held.

#### Scenario: Finish under hold
- **WHEN** a lap is claimed, its stint is held, and a bare `laps done` runs
- **THEN** the claimed lap SHALL be completed and the next `laps get` SHALL exit `10`

### Requirement: Held state in status
`laps status` SHALL report a `held` state, the held stint, and the gate message when the
resolved head is held, in both text and `--json-output`, and SHALL always exit `0`.

#### Scenario: Status reports held
- **WHEN** the resolved head is a held stint and `laps status` runs
- **THEN** it SHALL report state `held` with the held stint, and exit `0`
