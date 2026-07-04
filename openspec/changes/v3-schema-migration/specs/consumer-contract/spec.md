## ADDED Requirements

### Requirement: Stable machine-readable list output
The default `laps list` output is operator-facing and MAY evolve. Laps SHALL provide a
stable single-line machine form via `laps list --oneline`, rendering each lap as
`<n>. <id> — <title>` with the ` (assignee: <a>)` clause appended only when an assignee is
set. A change that alters the `--oneline` shape SHALL bump `VERSION`, update the consumer
contract, and note the consumer impact.

#### Scenario: Consumer counts queue size
- **WHEN** a consumer runs `laps list --oneline`
- **THEN** exactly one non-blank line SHALL be emitted per lap
- **AND** the line SHALL match `<n>. <id> — <title>[ (assignee: <a>)]`

#### Scenario: Default list may change freely
- **WHEN** laps changes the default (non-`--oneline`) list rendering
- **THEN** the `--oneline` and `--json-output` forms SHALL remain stable without a consumer migration

### Requirement: Structured claim file
`.laps/claim` SHALL be a JSON object with fields `lap` (the claimed lap id), `file`
(the `.laps`-relative task file), `scope` (canonical logical scope), and `claimedAt`
(nullable RFC3339 timestamp). Laps SHALL read a legacy bare-id claim file back-compatibly.
A change that alters this JSON shape SHALL bump `VERSION` and update the consumer contract.

#### Scenario: Consumer reads the claimed lap id
- **WHEN** a consumer reads `.laps/claim` after `laps claim`
- **THEN** it SHALL parse JSON and read the `lap` field as the claimed lap id

#### Scenario: Legacy bare-id claim tolerated on read
- **WHEN** laps reads a non-JSON bare-id `.laps/claim`
- **THEN** it SHALL treat the trimmed contents as the `lap` id without error

### Requirement: Stable get/claim task-detail format
`laps get` / `laps claim` stdout (non-JSON) SHALL keep the detail format: line 0 is the
title, an optional `Assignee: <name>` line follows, then a blank line, then the
description. Resolution through stints SHALL NOT alter this format (agents see a lap's
title/description only). A change that alters this format SHALL bump `VERSION` and update
the consumer contract.

#### Scenario: Consumer parses a claimed lap
- **WHEN** a consumer parses `laps claim` stdout
- **THEN** line 0 SHALL be the title, an optional `Assignee: <name>` line MAY follow, and the description SHALL follow a blank line

#### Scenario: Stint resolution is transparent
- **WHEN** a lap is served from an active stint
- **THEN** the `get`/`claim` detail output SHALL be identical to a root-queue lap

### Requirement: Queue-state exit codes for get/claim
`laps get` / `laps claim` SHALL exit `0` when a lap is returned, `10` when the resolved
head is held, `11` when the queue is empty, and `12` when all entries are complete. Store/io
failures remain `2`; explicit-id-not-found remains `3`; hook failures remain `4`. A change
that alters these codes SHALL bump `VERSION` and update the consumer contract.

#### Scenario: Empty queue
- **WHEN** `laps claim` finds no lap because the queue is empty
- **THEN** it SHALL exit `11`

#### Scenario: Complete queue
- **WHEN** `laps claim` finds no lap because every entry is done
- **THEN** it SHALL exit `12`

#### Scenario: Held stint at the head
- **WHEN** `laps claim` resolves to a held stint
- **THEN** it SHALL exit `10`

### Requirement: Consumer surface is version-gated
Laps SHALL treat the `list --oneline` shape, the claim JSON shape, the `get`/`claim`
detail format, and the `get`/`claim` queue-state exit codes as the consumer contract. The
default `list`, `laps status`, and `laps log` outputs are operator-facing and are NOT part
of this contract. A change that alters a contract surface SHALL bump `VERSION`, update the
documented consumer contract, and record the consumer impact in its proposal.

#### Scenario: Consumer surface change is signposted
- **WHEN** a change alters a pinned consumer surface
- **THEN** the change SHALL bump `VERSION`, update the consumer-contract documentation, and note the consumer impact

#### Scenario: Operator-facing output is free to evolve
- **WHEN** a change alters the default `list`, `status`, or `log` rendering
- **THEN** it SHALL NOT require a consumer migration, because consumers use the pinned machine forms
