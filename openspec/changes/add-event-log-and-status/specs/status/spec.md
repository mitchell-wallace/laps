## ADDED Requirements

### Requirement: Queue status snapshot
The system SHALL provide `laps status` reporting the todo and done counts, the active
(claimed) lap, the head lap, and the assignee breakdown of todo laps, together with a queue
state taxonomy that distinguishes at least active claimed work, an empty queue, an all-complete
queue, and todo laps with no active claim. `laps status` SHALL exit 0 for valid repo/store
snapshots and SHALL honor `--json-output` with a stable shape. Corrupt or unreadable task
files, malformed claim JSON, and serialization failures SHALL use the normal CLI error path
rather than being reported as healthy status snapshots.

#### Scenario: Status reports the queue
- **WHEN** `laps status` runs with laps present
- **THEN** it SHALL report todo/done counts, the head lap, the active lap (if any), and the assignee breakdown

#### Scenario: Empty queue
- **WHEN** `laps status` runs with no laps
- **THEN** it SHALL report state `empty`

#### Scenario: All laps complete
- **WHEN** `laps status` runs and every lap is done
- **THEN** it SHALL report state `complete`

#### Scenario: Todo laps with no claim
- **WHEN** todo laps exist and no lap is claimed
- **THEN** `laps status` SHALL report the todo/no-claim state, the head lap, and no active lap

#### Scenario: JSON output
- **WHEN** `laps status --json-output` runs
- **THEN** it SHALL emit the snapshot as a JSON object

### Requirement: Active-lap timestamp
The claim record SHALL include a `claimedAt` timestamp. The claim file SHALL be read
back-compatibly: a legacy bare-id file SHALL be interpreted as a claim with a null
timestamp, and unknown fields in a structured claim SHALL be ignored for forward compatibility.
Only non-JSON bare tokens SHALL be treated as legacy claims; structured-looking invalid JSON or
invalid field types SHALL be malformed claim errors. `laps status` SHALL surface the age of the
active claim so stale claims left by crashed sessions are visible.

#### Scenario: Claim records the time
- **WHEN** `laps claim` records a claim
- **THEN** the claim record SHALL include a `claimedAt` timestamp

#### Scenario: Legacy bare-id claim still read
- **WHEN** the claim file contains a legacy bare id with no JSON structure
- **THEN** it SHALL be read as a claim for that lap with a null `claimedAt`

#### Scenario: Future claim fields ignored
- **WHEN** the claim file contains `{lap, claimedAt, scope}`
- **THEN** this change SHALL read `lap` and `claimedAt` successfully and ignore `scope`

#### Scenario: Malformed structured claim errors
- **WHEN** the claim file starts as JSON but is malformed or has invalid field types
- **THEN** the claim reader SHALL report a malformed claim error rather than treating the bytes as a legacy id

#### Scenario: Status surfaces claim age
- **WHEN** `laps status` runs with an active claim
- **THEN** it SHALL report how long that lap has been claimed
