## ADDED Requirements

### Requirement: Two-line list output
The `list` command SHALL render each lap on two lines by default. The first line SHALL
show the position number, the title, and a marker when the lap is the active (claimed)
lap. The second line SHALL show the lap id, the assignee (or a placeholder when unset),
and the lap state. Lap descriptions SHALL NOT be shown. The command SHALL accept a
`--oneline` flag that renders each lap on a single line in the prior
`<n>. <id> — <title> (assignee: <a>)` form. Done laps SHALL be struck through under
`--all` and `--done` in both layouts.

#### Scenario: Default two-line rendering
- **WHEN** `laps list` runs with todo laps present
- **THEN** each lap SHALL be rendered on two lines — the first with number, title, and (if active) marker; the second with id, assignee, and state — and no description

#### Scenario: One-line rendering preserved
- **WHEN** `laps list --oneline` runs
- **THEN** each lap SHALL be rendered on a single line in the prior format

#### Scenario: Done laps struck through
- **WHEN** `laps list --all` or `laps list --done` runs
- **THEN** completed laps SHALL be rendered struck through

### Requirement: Active-lap marker
The `list` command SHALL mark the active lap — the lap returned by the central claim-reader
contract — distinctly from other laps. The formatter SHALL NOT parse `.laps/claim` directly.
When no lap is claimed, or the claimed id is not present in the rendered result, no lap SHALL
display the marker. Claim-read failures SHALL fail the command using the normal store/io error
path.

#### Scenario: Claimed lap marked
- **WHEN** a lap is claimed and `laps list` runs
- **THEN** that lap SHALL display the active marker and no other lap SHALL

#### Scenario: No claim, no marker
- **WHEN** no lap is claimed and `laps list` runs
- **THEN** no lap SHALL display the active marker

#### Scenario: Nonmatching claim, no marker
- **WHEN** a claim exists for a lap outside the rendered result and `laps list` runs
- **THEN** no lap SHALL display the active marker

### Requirement: List alias
The system SHALL provide `laps ls` as an alias for `laps list`, accepting the same flags
and producing identical output.

#### Scenario: ls mirrors list
- **WHEN** `laps ls` runs with the same flags as `laps list`
- **THEN** it SHALL produce the same output as `laps list`
