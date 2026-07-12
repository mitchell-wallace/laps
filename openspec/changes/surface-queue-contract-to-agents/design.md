# Design: Surface the queue contract to agents

## Current shape

The injected `<laps-instructions>` block still teaches the pre-claim workflow:
read the head, work it, and complete the head. Its exact opening marker is also
used as the replacement sentinel, so adding a version attribute without
changing detection would duplicate stale blocks. Separately, the Cobra help
for `get`, `claim`, and `status` omits the queue-state exit protocol even though
head operations implement it as a pinned consumer contract.

## Target shape

- Inject a concise v2 block opened by `<laps-instructions v="2">`.
- Teach the claim → work → done loop, including that bare `done` completes the
  claim rather than whichever lap is currently at head.
- Include the `0`/`10`/`11`/`12` head-operation table and the agent action for
  each state. Held means finish an existing claim but start no new work.
- State that stint descent is transparent and that agents should prefer
  `--active`, `--root`, and `--stint` over raw `-f/--file` targeting.
- Detect instruction blocks by a valid `<laps-instructions...>` opening tag so
  both the legacy and versioned forms are refreshed or removed in place.
- Put the same exit-code table in `get`, `claim`, and `status` long help; add a
  short root-help pointer.

Queue-state exit codes become named constants in a small shared internal
package. Command behavior and documentation-invariant tests both consume those
constants, preventing the injected contract from drifting from implementation.

## Alternatives considered

- **Leave the marker unversioned and add a version line inside the block**:
  simpler detection, but operators cannot identify stale content from the
  opening marker. Rejected because the approved scope explicitly calls for a
  version stamp.
- **Add `laps capabilities --json`**: machine-readable and extensible, but much
  larger than the documentation-only discoverability fix. Deferred as proposed.
- **Generate all help and instruction prose from one template**: eliminates
  duplication but makes terse agent instructions and detailed terminal help
  awkwardly coupled. Shared constants plus invariant tests provide the useful
  consistency without forcing identical prose.

## Verification strategy

- Instruction tests verify fresh v2 injection, idempotence, legacy-block
  refresh, and removal of both legacy and versioned blocks.
- Content invariants require the workflow commands, held etiquette, stint/file
  guidance, and all implemented queue-state exit codes.
- Command tests verify `get --help`, `claim --help`, and `status --help` contain
  an Exit codes section with `0`, `10`, `11`, and `12`, and root help points to
  those commands.
- Run the full Go suite and lint gate.
- Manual smoke in a throwaway repo: seed a legacy block, run `laps on`, inspect
  the v2 replacement, run `laps off`, and inspect all four help surfaces.

## Dependencies and ordering

Independent of the two trapdoor fixes, though those fixes make the documented
workflow safer. No data migration and no CLI behavior change.
