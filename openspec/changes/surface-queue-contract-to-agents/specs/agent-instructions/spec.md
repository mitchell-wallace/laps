# agent-instructions (delta)

## ADDED Requirements

### Requirement: Injected instructions SHALL teach the current queue protocol

The injected laps instructions SHALL teach agents to claim a lap, work the
claim, and complete it with bare `laps done`. They SHALL state that stint
resolution is transparent and recommend scope flags over raw file targeting.

#### Scenario: Agent enables laps instructions

- **WHEN** `laps on` writes an instruction block
- **THEN** the block describes the claim → work → done loop
- **AND** it says bare `done` completes the claimed lap
- **AND** it recommends `--active`, `--root`, or `--stint` over `-f/--file`

### Requirement: Queue-state actions SHALL be discoverable

The instruction block and the long help for `get`, `claim`, and `status` SHALL
document head-operation exit codes `0`, `10`, `11`, and `12`. The instruction
block SHALL map them respectively to run, stop-held, idle, and finished actions.
Held guidance SHALL allow finishing an existing claim but forbid starting new
work.

#### Scenario: Agent encounters a held queue

- **WHEN** a head `get` or `claim` exits `10`
- **THEN** the injected instructions tell the agent to finish any existing
  claimed lap, start nothing new, and stop

### Requirement: Instruction blocks SHALL be versioned and refreshable

New blocks SHALL use `<laps-instructions v="2">`. Enabling or disabling laps
SHALL recognize both the legacy `<laps-instructions>` opener and versioned
openers, replacing or removing exactly one complete block without disturbing
surrounding content.

#### Scenario: Legacy block is refreshed

- **GIVEN** an agent document containing a legacy instruction block
- **WHEN** `laps on` runs
- **THEN** the legacy block is replaced in place by one v2 block
- **AND** surrounding user content remains unchanged
