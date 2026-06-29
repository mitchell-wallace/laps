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

#### Scenario: Completing a claimed lap logs completion then unclaim
- **WHEN** `laps done` completes a lap that is currently claimed
- **THEN** a `completed` event SHALL be appended, immediately followed by an `unclaimed` event with `detail.reason` of `completed`

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
SHALL show the full lifecycle of a single lap. The reader SHALL apply all filters first and then
truncate to `-n` (filter-then-limit), so the limit bounds matching events shown, not lines
scanned. Output SHALL be ordered newest-last (chronological). The default limit SHALL be `20`.
`--since` SHALL take an RFC3339 timestamp and SHALL be inclusive of the exact timestamp.
Malformed JSONL lines SHALL be skipped with a one-line stderr note per offending line and SHALL
NOT abort the read. `--json-output` SHALL emit a single object `{ "events": [ ... ] }`.

#### Scenario: Reading recent events
- **WHEN** `laps log` runs
- **THEN** it SHALL print recent logged events

#### Scenario: Default limit is twenty
- **WHEN** `laps log` runs with more than twenty events logged and no explicit `-n`
- **THEN** it SHALL print the twenty most recent matching events, ordered newest-last

#### Scenario: Filters apply before the limit
- **WHEN** `laps log --lap <id> -n 5` runs and that lap has more than five events
- **THEN** it SHALL print the five most recent events matching the lap filter, ordered newest-last

#### Scenario: Since filter is RFC3339 and inclusive
- **WHEN** `laps log --since 2026-01-01T00:00:00Z` runs and an event's timestamp equals that value
- **THEN** that event SHALL be included in the output

#### Scenario: Malformed JSONL is skipped not fatal
- **WHEN** `laps log` reads a file containing a line that is not valid JSON
- **THEN** it SHALL emit a one-line note on stderr for that line and SHALL continue reading the remaining lines rather than aborting

#### Scenario: JSON output wraps events
- **WHEN** `laps log --json-output` runs
- **THEN** it SHALL emit a single JSON object whose `events` field is the array of matching events

#### Scenario: Filtering to one lap's lifecycle
- **WHEN** `laps log --lap <id>` runs
- **THEN** it SHALL print only events for that lap, in order

#### Scenario: Missing log is empty
- **WHEN** `laps log` runs before `.laps/log.jsonl` exists
- **THEN** it SHALL behave as an empty log rather than failing
