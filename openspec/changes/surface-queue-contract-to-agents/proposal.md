# Proposal: Surface the queue contract to agents (instructions block + CLI help)

## Intent

laps' primary consumers are AI agents, and laps' one channel for teaching them
— the `<laps-instructions>` block that `laps on` injects into
AGENTS.md/CLAUDE.md/GEMINI.md — still describes the pre-claim, pre-stint,
pre-exit-code product. Meanwhile the load-bearing queue-state exit codes
(`10`/`11`/`12`) are documented only in the README: `laps get --help` and
`laps claim --help` never mention them. Live sessions this week flagged
exit-code discoverability directly. This change makes the CLI and the injected
instructions teach the v3 contract agents actually run against.

## Evidence

- `internal/instructions/instructions.go:15-24` — the injected block tells
  agents `laps get head` / `laps done` / `laps add`; it does not mention
  `claim`, queue-state exit codes, held-stint etiquette, or that a bare `done`
  completes the *claimed* lap. It predates four shipped changes
  (`add-event-log-and-status` through `add-stints-gating`).
- Verified with the dev binary: `laps get --help` output contains no mention
  of exit codes `10`/`11`/`12` (nor does `claim --help`); the only
  documentation is `README.md` "Queue-state exit codes".
- `README.md` Consumer contract, pinned surface #4 — the exit codes are an
  explicit contract surface for orchestrators; `internal/cmd/root.go:85-96`
  and `internal/cmd/resolution.go:317` implement them. A contract this
  load-bearing that is invisible from `--help` fails the agent that reaches
  for `--help` first (which is what agents do).
- `internal/instructions/instructions.go:85-115` (`replaceBlock`) — the block
  is replaced wholesale on `laps on`, so refreshing content is already safe
  and idempotent; there is just no versioning to tell operators a repo's
  block is stale.
- Field report: ~20 agent sessions this week; low exit-code discoverability
  called out as a recurring friction.

## Scope

In scope:
- Rewrite `blockContent` to the current contract: claim → work → done loop;
  exit-code table for head `get`/`claim` (`0`/`10`/`11`/`12`) with the action
  each should trigger (run / stop-held / idle / finished); held-stint
  etiquette (finish the claimed lap, start nothing new); a warning to prefer
  scope flags over `-f`; note that stint resolution is transparent.
- Version-stamp the block (e.g. `<laps-instructions v="2">`) so `laps on`
  refreshes stale blocks and `laps off`/detection keep working with both
  forms.
- Add an "Exit codes" section to the `--help` (Long text) of `get`, `claim`,
  and `status`, and a one-line pointer in root `laps --help`.
- Keep the block short enough for an agent context (target: the current
  block's size, ±50%).

Out of scope:
- New machine-readable capability descriptors (`laps capabilities --json`) —
  recorded as a design alternative.
- Any behavior change to the exit codes themselves or to `on`/`off` file
  handling beyond block content and version detection.

## Proposed path

Treat the instruction block as a versioned product artifact with spec-backed
required content, not an incidental string constant. Update
`internal/instructions/instructions.go` content + version detection, update
the Cobra `Long` strings in `internal/cmd/get.go`, `claim.go`, `status.go`,
and assert content invariants in tests so the block cannot silently drift from
the README contract again (a test cross-checks that every exit code named in
the block matches `exitForQueueState`).

## Expected payoff

- Every agent session in every laps-enabled repo gets the correct operating
  contract for free at `laps on` time — the highest-leverage per-token
  documentation laps can ship.
- `--help` becomes sufficient to learn the queue-state protocol without
  leaving the terminal; direct fix for the discoverability seed.
- Version-stamped blocks give operators a signal ("re-run `laps on`") when
  future contract changes land.

## Risks and unknowns

- Injected-doc changes touch consumer repos' AGENTS.md on the next `laps on`;
  the block replacement is already idempotent, but the diff noise should be
  called out in release notes.
- Keeping the block terse vs. complete is a judgment call; the spec pins the
  minimum required facts, not the wording.
- No production behavior risk: strings, help text, and block versioning only.

## Spec impact

New capability spec for the injected agent instructions (required content and
refresh semantics). See `specs/agent-instructions/spec.md`.
