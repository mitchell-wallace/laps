## Context

The v3 line is implemented across four changes (`add-event-log-and-status`,
`improve-cli-ergonomics`, `add-stints`, `add-stints-gating`); `VERSION` is `0.9.0` and all
tasks are checked off. The released/installed binary many consumers run is still the
pre-v3 (schema-v2) line, so 0.9.0 is the cutover, not a fait accompli.

Rally is the sole real consumer and it drives laps three ways: (1) subprocess calls whose
**stdout** it text-parses, (2) direct **file reads** of `.laps/claim` and
`.laps/laps.json`, and (3) **agent-facing commands** (`done`/`handoff`/`wrapup`) bridged
through hooks. Reviewing the four changes against that surface found four breaks beyond
the schema, listed below. The lesson is that laps never documented which surfaces are
load-bearing for consumers, so the breaks were silent. This change pins that contract and
coordinates the cutover, then opens the `prepare-laps` stint-workflow redesign.

## Goals / Non-Goals

**Goals:**
- Ship laps 0.9.0 as the v3 cutover.
- Pin the laps consumer contract (the machine-readable surface) as a versioned guarantee.
- Coordinate rally's phase-1 adoption so the two can cut over together.
- Frame the `prepare-laps` stint-workflow redesign without prematurely closing it.

**Non-Goals:**
- New laps behavior — the contract already exists in v3; this change documents and pins it.
- Deciding the final `prepare-laps` modality/naming/trigger phrases — deliberately open.
- Multi-consumer support; rally is the only consumer and the two version together.
- Breaking the agent-facing contract (`done`/`handoff`/`wrapup` stdout, `$id`/`$args`
  hook vars) — confirmed unchanged by the four changes and out of scope to alter.

## Decisions

### Pin the consumer contract (the core decision)
Laps SHALL document, in `README.md`, the machine-readable surface consumers depend on,
and SHALL treat it as version-gated: a future change that alters any of these surfaces
requires a version bump and a consumer-facing note. The surfaces, as implemented in v3:

| Surface | Stable form | Where it lives in v3 |
|---|---|---|
| List, single-line | `laps list --oneline` → `<n>. <id> — <title>[ (assignee: <a>)]` | `internal/cmd/list.go` `--oneline` |
| Claim file | `.laps/claim` JSON `{lap, file, scope, claimedAt}` | `internal/store/claim.go` `Claim` |
| Task detail (`get`/`claim`) | line 0 title, optional `Assignee: <a>`, blank line, description | `get.go` / `claim.go` |
| Queue-state exit codes (`get`/`claim`) | `0` lap, `10` held, `11` empty, `12` complete | `add-stints-gating` |

Note the default `list` output and the `laps status`/`laps log` readers are **not** part of
the stable contract — they are operator-facing and may evolve; consumers use `--oneline`
(or `--json-output`) for machine parsing.

### The four breaking changes (the report)

These are the consumer-visible breaks beyond the schema, each confirmed against rally source.

1. **`laps list` default is now two lines per lap** (`improve-cli-ergonomics`).
   Rally `QueueSize` (`internal/laps/adapter.go:91`) counts non-blank lines, so the
   reported queue size **doubles**, corrupting the progress UI
   (`LapsStarted`/`LapsTotal`, `task.LapsRemaining`). `--oneline` already exists and
   restores the prior shape. **Fix (rally): call `laps list --oneline`.**

2. **`.laps/claim` is now structured JSON** (`add-event-log-and-status`).
   The claim is `{lap, file, scope, claimedAt}` (`internal/store/claim.go`). Rally
   `ReadClaim` (`internal/laps/adapter.go:79`) reads the file and `TrimSpace`s it as a
   bare id, so `lap.ID` becomes the whole JSON object — silently breaking the relay's lap
   identity (wrong-lap attribution). The "legacy bare-id read back-compat" in laps is a
   read-side tolerance for *old* files, not a continued write of bare ids. **Fix (rally):
   parse the claim JSON and read the `lap` field (tolerate a bare-id fallback for
   transitional files).**

3. **`get`/`claim` exit codes `3` → `10/11/12`** (`add-stints-gating`).
   Rally treats *any* non-zero exit as NoLap (`adapter.go:39-42`, `61-64`), so `11`
   (empty) and `12` (complete) still end the relay correctly — but `10` (held stint) also
   ends the relay rather than pausing it. **Phase-1 benign** (no stints enqueued ⇒ only
   `11`/`12` can occur, both meaning "stop"); **phase-2** must distinguish `10` so a held
   stint pauses the relay instead of ending it.

4. **Bare `laps list` descends into the active stint** (`add-stints`).
   Once a stint is active, bare `list` shows the stint's laps, not root, so `QueueSize`
   counts a different population. **Phase-1 benign** (no stints ⇒ bare `list` ≡ root);
   **phase-2** must decide what `QueueSize` should count across a pipeline (root-only via
   `--root --oneline`, the active scope, or `--tree`).

**Confirmed non-breaking (coupling points, kept stable):**
- Direct `.laps/laps.json` read (`internal/relay/runner.go:3490`) uses Go's default
  `json.Unmarshal`, which ignores unknown fields, so the new `kind`/`ref` entry fields are
  harmlessly skipped and the `id`/`isDone`/`completedAt` reads still work.
- `laps add head` stdout is the new lap id (trimmed) — unchanged; `parseLapOutput` and the
  e2e expectations still hold.
- `laps done` first stdout line is the completed lap's title — unchanged; event logging
  writes to `.laps/log.jsonl`, not stdout.
- Agent-facing `done`/`handoff`/`wrapup` commands and `$id`/`$args` hook vars — unchanged;
  `add-stints` only *adds* `$scope`, it does not alter existing vars.
- `laps version` advisory check — unchanged; still a warning, never a hard fail.

### Phased rollout

**Phase 1 — compatibility over existing behaviour.** Goal: rally runs against 0.9.0 with
identical observable behaviour when no stints are used. Scope: fixes #1 and #2 (hard
breaks present even without stints), the exit-code adoption for #3 (no behaviour change
yet, just stop relying on `3`), and the `MinLapsVersion` bump + docs. Stints are not used
in phase 1, so #4 and the held-vs-empty distinction are inert.

**Phase 2 — the stint-based workflow.** Goal: stints become the default way multi-change
work is prepared and pipelined. Scope: the `prepare-laps` skill redesign (open, below),
rally distinguishing exit `10` to pause vs end, and deciding what `QueueSize` counts
across a stint pipeline. Phase 2 starts after phase 1 is live and stints are actually
exercised.

### Release + MinLapsVersion coordination
Laps tags/releases 0.9.0; rally bumps `MinLapsVersion` to `0.9.0` in the same window using
its `updating-laps-version` skill (the six sync locations). Because rally's installer
fetches the matching companion, the two cut over together. The advisory-only version
check stays advisory (warn, never hard-fail).

### Exit-code adoption in rally (phase-1 mechanics)
Rally's "non-zero ⇒ NoLap" relay-source contract already does the right thing for `11`/`12`
in phase 1, so phase-1 needs no relay-loop change — only the removal of any assumption that
the specific code is `3`, plus a comment that `10` is reserved for phase-2 hold handling.

## Implementation Contracts

- **Contract is documentation + pinning, not new code.** The four surfaces already behave
  as specified in v3; laps adds a `README.md` "Consumer contract" section and this change's
  `consumer-contract` spec. No laps command changes behaviour in this change.
- **Rally tasks are cross-repo.** The rally-side edits live in the rally repo but are
  tracked as this change's phase-1 tasks because laps/rally version together. They are
  marked `[rally]` in `tasks.md`.
- **Phase-2 items are exploratory.** They are written as open tasks, not committed
  requirements, until the skill modality decision lands (see below).
- **Version gating.** A future change that alters a pinned consumer surface SHALL bump
  `VERSION`, update the contract section, and note the consumer impact in its proposal.

## Open: skill modalities (phase 2, deliberately undecided)

Laps now supports several preparation modalities, and `prepare-laps` (which today produces
one flat queue) must decide how to present them:

- **Flat laps** — today's behaviour: one `.laps/laps.json` queue per change.
- **Single stint** — one prepared queue per change at `.laps/stints/<name>.laps.json`,
  enqueued into the root as a stint ref.
- **Nested stints** — stints within stints (engine-supported; creation tooling is flat today).
- **Mixed** — plain laps and stint refs together in the root queue.

The operator's stated lean: **one shared `SKILL.md` as the common entry point**, with
shorter reference documents (or sections) per modality, so the planning logic is not
duplicated and does not drift — while preserving the easy user-facing trigger
`prepare-laps for <change-name>`. An alternate `prepare-stints for <change-name>` trigger
that maps to the same skill's stint mode is attractive for callers who already think in
stints, but it risks implying a separate skill.

**Open questions this change records but does not answer:**
- Is the skill one file with a "choose your modality" section, or one `SKILL.md` plus
  referenced per-mode docs (`flat.md`, `stints.md`, …)?
- What are the canonical trigger phrases, and does `prepare-stints` exist as an alias to
  the same skill or not at all?
- When is a stint the default? (Leading candidate: default to a stint for OpenSpec-change
  input, flat laps for ad-hoc/non-OpenSpec input — but this is a judgement call to revisit.)
- How does the skill teach enqueue order / hold-release given auto-advancing pipelines?

These should be settled in an explore/propose cycle on the rally side, then promoted into a
spec'd `stint-workflow-skill` capability (and its own change) rather than frozen here.

## Resolved Product Calls

- **Where the consumer contract lives — DECIDED: laps.** Laps owns the surface consumers
  parse; pinning it in laps (not rally) is what lets the next laps change see the break
  before it ships. Rally's role is to adopt, not to define.
- **`list --oneline` is the stable machine form — DECIDED.** The two-line default is
  operator-facing and free to evolve; `--oneline` (and `--json-output`) are the pinned
  consumer forms.
- **Phase 1 keeps relay behaviour identical — DECIDED.** No stints in phase 1 ⇒ the
  held/empty/complete distinction and stint-descent are inert; rally only needs the two
  hard fixes (#1, #2) plus the version bump. Exit-code adoption is mechanical, not
  behavioural, this phase.
- **Skill modality/naming — DEFERRED.** Recorded as open above; settled in a follow-up
  explore cycle, not in this change.

## Risks

- **The two hard breaks (#1, #2) are silent.** Neither errors — a doubled count and a
  wrong lap id both look like normal operation until integrity bugs surface. Mitigation:
  phase 1 lands before any stint use, and the contract section makes the surfaces
  discoverable for the next reviewer.
- **Stint-descent (#4) changes `QueueSize`'s meaning once stints land.** Mitigation: phase
  2 explicitly decides the count semantics; phase 1 is unaffected.
- **Cross-repo coordination drift.** Rally edits live in another repo. Mitigation: tasks
  are tracked here with `[rally]` tags and gated on the coordinated 0.9.0 release window.
