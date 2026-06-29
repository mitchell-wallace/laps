## ADDED Requirements

### Requirement: Queue status snapshot
The system SHALL provide `laps status` reporting the todo and done counts, the active
(claimed) lap, the head lap, and the assignee breakdown of todo laps, together with a queue
state taxonomy of `active`, `ready`, `empty`, and `complete`, plus the selected file identity.
`ready` means todo laps exist and no valid lap is claimed. `laps status` SHALL exit 0 for valid
repo/store snapshots and SHALL honor `--json-output` with a stable shape. Corrupt or unreadable
task files, malformed claim JSON, and serialization failures SHALL use the normal CLI error path
rather than being reported as healthy status snapshots. Claims pointing at deleted, pruned,
done, or wrong-file laps SHALL produce a valid degraded snapshot with `claim.valid=false`; the
system SHALL NOT auto-clear such claims silently.

#### Scenario: Status reports the queue
- **WHEN** `laps status` runs with laps present
- **THEN** it SHALL report the selected file, todo/done counts, the head lap, the active lap (if any), and the assignee breakdown

#### Scenario: Empty queue
- **WHEN** `laps status` runs with no laps
- **THEN** it SHALL report state `empty`

#### Scenario: All laps complete
- **WHEN** `laps status` runs and every lap is done
- **THEN** it SHALL report state `complete`

#### Scenario: Todo laps with no claim
- **WHEN** todo laps exist and no lap is claimed
- **THEN** `laps status` SHALL report state `ready`, the head lap, and no active lap

#### Scenario: Dangling claim is degraded status
- **WHEN** the claim points at a deleted, done, or wrong-file lap
- **THEN** `laps status` SHALL exit `0`, report `claim.valid=false`, and SHALL NOT clear the claim

#### Scenario: JSON output
- **WHEN** `laps status --json-output` runs
- **THEN** it SHALL emit the snapshot as a JSON object

### Requirement: Active-lap timestamp
The claim record SHALL include `file` and `claimedAt`. The claim file SHALL be read
back-compatibly: a legacy bare-id file SHALL be interpreted as a claim for the selected file
with a null timestamp, and unknown fields in a structured claim SHALL be ignored for forward compatibility.
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
- **WHEN** the claim file contains `{lap, file, claimedAt, scope}`
- **THEN** this change SHALL read `lap`, `file` when present, and `claimedAt` successfully and ignore `scope`

#### Scenario: Malformed structured claim errors
- **WHEN** the claim file starts as JSON but is malformed or has invalid field types
- **THEN** the claim reader SHALL report a malformed claim error rather than treating the bytes as a legacy id

#### Scenario: Status surfaces claim age
- **WHEN** `laps status` runs with an active claim
- **THEN** it SHALL report how long that lap has been claimed
