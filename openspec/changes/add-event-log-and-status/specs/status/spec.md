## ADDED Requirements

### Requirement: Queue status snapshot
The system SHALL provide `laps status` reporting the todo and done counts, the active
(claimed) lap, the head lap, and the assignee breakdown of todo laps, together with a queue
state of `active`, `empty`, or `complete`. `laps status` SHALL always exit 0 and SHALL honor
`--json-output` with a stable shape.

#### Scenario: Status reports the queue
- **WHEN** `laps status` runs with laps present
- **THEN** it SHALL report todo/done counts, the head lap, the active lap (if any), and the assignee breakdown

#### Scenario: Empty queue
- **WHEN** `laps status` runs with no laps
- **THEN** it SHALL report state `empty`

#### Scenario: All laps complete
- **WHEN** `laps status` runs and every lap is done
- **THEN** it SHALL report state `complete`

#### Scenario: JSON output
- **WHEN** `laps status --json-output` runs
- **THEN** it SHALL emit the snapshot as a JSON object

### Requirement: Active-lap timestamp
The claim record SHALL include a `claimedAt` timestamp. The claim file SHALL be read
back-compatibly: a legacy bare-id file SHALL be interpreted as a claim with a null
timestamp. `laps status` SHALL surface the age of the active claim so stale claims left by
crashed sessions are visible.

#### Scenario: Claim records the time
- **WHEN** `laps claim` records a claim
- **THEN** the claim record SHALL include a `claimedAt` timestamp

#### Scenario: Legacy bare-id claim still read
- **WHEN** the claim file contains a legacy bare id with no JSON structure
- **THEN** it SHALL be read as a claim for that lap with a null `claimedAt`

#### Scenario: Status surfaces claim age
- **WHEN** `laps status` runs with an active claim
- **THEN** it SHALL report how long that lap has been claimed
