## ADDED Requirements

### Requirement: Append-only event log
The system SHALL maintain an append-only event log at `.laps/log.jsonl`, writing one JSON
object per applied state change. Log writing SHALL be native to the commands (not a
user-configurable hook) and best-effort: when an append fails, the command SHALL still
succeed and SHALL report the failure on stderr. The system SHALL NOT rewrite or rotate the
log.

#### Scenario: Mutating command appends an event
- **WHEN** a mutating command applies a state change
- **THEN** the system SHALL append one JSON line describing that change to `.laps/log.jsonl`

#### Scenario: Log failure does not fail the command
- **WHEN** appending to the log fails
- **THEN** the command SHALL still succeed with its normal exit code and SHALL report the failure on stderr

### Requirement: Logged events
The system SHALL log these state transitions: `created`, `completed`, `reopened`,
`claimed`, `unclaimed`, `moved`, `edited`, `deleted`, `pruned`. The system SHALL NOT log
read-only or admin commands (`get`, `list`, `count`, `status`, `log`, `version`, `help`,
hook-only commands, `init`, `on`, `off`, `update`). Commands that affect multiple laps SHALL
write one event per affected lap/transition after the store save succeeds.

#### Scenario: Completing a lap logs it
- **WHEN** `laps done` completes a lap
- **THEN** a `completed` event for that lap SHALL be appended

#### Scenario: Claim write failure does not log
- **WHEN** `laps claim` fails to write the claim file
- **THEN** no `claimed` event SHALL be appended

#### Scenario: Same-lap reclaim does not duplicate events
- **WHEN** `laps claim <id>` runs for the same lap that is already claimed
- **THEN** the existing `claimedAt` SHALL be preserved and no duplicate `claimed` event SHALL be appended

#### Scenario: Replacing a claim logs replacement
- **WHEN** `laps claim <new-id>` replaces a different claimed lap
- **THEN** an `unclaimed` event with `detail.reason` of `replaced` SHALL be appended before the new `claimed` event

#### Scenario: Reads are not logged
- **WHEN** `laps get` or `laps list` runs
- **THEN** no event SHALL be appended

#### Scenario: Batch add logs each created lap
- **WHEN** `laps add tail --json '[{"title":"A"},{"title":"B"}]'` succeeds
- **THEN** two `created` events SHALL be appended, one for each new lap

### Requirement: Event schema and attribution
Each log line SHALL be a JSON object containing `ts` (UTC timestamp), `event`, `cmd`, `file`,
and `scope` (defaulting to `root`), and SHALL include `lap`, `title`, `assignee`, and an
event-specific `detail` object where applicable. `file` SHALL be the resolved `.laps`-relative
task file for the mutation. Each line SHALL carry a `session` field
populated from the `LAPS_SESSION` environment variable, empty when the variable is unset.

#### Scenario: Session stamped from environment
- **WHEN** a mutating command runs with `LAPS_SESSION` set
- **THEN** the appended line's `session` SHALL equal that value

#### Scenario: Scope defaults to root
- **WHEN** an event is logged with no stint context
- **THEN** its `scope` SHALL be `root`

#### Scenario: File identity stamped
- **WHEN** a mutating command runs against `--file auth`
- **THEN** the appended line's `file` SHALL identify `auth.json`

### Requirement: Log gitignored on init
`laps init` SHALL ensure `.laps/log.jsonl` is gitignored, appending it to `.gitignore`
when absent, alongside `.laps/claim`. It SHALL preserve all existing `.gitignore` lines and
append only entries that are missing.

#### Scenario: Init ignores the log
- **WHEN** `laps init` runs and `.gitignore` does not list `.laps/log.jsonl`
- **THEN** it SHALL append `.laps/log.jsonl` to `.gitignore`

### Requirement: Event-log reader
The system SHALL provide `laps log` to read the event log, supporting `-n <count>`,
`--lap <id>`, `--session <id>`, `--since <time>`, and `--json-output`. `laps log --lap <id>`
SHALL show the full lifecycle of a single lap.

#### Scenario: Reading recent events
- **WHEN** `laps log` runs
- **THEN** it SHALL print recent logged events

#### Scenario: Filtering to one lap's lifecycle
- **WHEN** `laps log --lap <id>` runs
- **THEN** it SHALL print only events for that lap, in order

#### Scenario: Missing log is empty
- **WHEN** `laps log` runs before `.laps/log.jsonl` exists
- **THEN** it SHALL behave as an empty log rather than failing
