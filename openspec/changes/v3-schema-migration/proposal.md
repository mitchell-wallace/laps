## Why

The v3 line landed across four changes — `add-event-log-and-status`,
`improve-cli-ergonomics`, `add-stints`, `add-stints-gating` — each implemented and
checked off, with `VERSION` staged at `0.9.0`. The schema bump is the obvious headline,
but reviewing the four changes against the one real consumer (the sibling `rally` repo)
surfaced **four consumer-visible breaking changes that are not the schema**, plus the
uncharted question of how the stint-based workflow changes `prepare-laps`.

Laps and rally "ship and version together" and rally bundles laps as a first-class
companion, so laps owns the consumer contract. Today that contract is **implicit** —
rally parses `laps` stdout and reads `.laps/` files directly, and there is no document
that says which surfaces are load-bearing for consumers or that they are version-gated.
That is exactly why these four breaks were silent: nothing marked `list`'s line shape, the
claim file's format, or `get`/`claim` exit codes as a contract. This change ships v3,
formalizes the consumer contract, coordinates rally's adoption, and opens the
`prepare-laps` stint-workflow redesign.

## What Changes

- **Release laps 0.9.0** — tag + release the four changes as the v3 line. The released
  binary many consumers currently run is still the pre-v3 (schema-v2) line; 0.9.0 is the
  cutover.
- **Formalize the laps consumer contract (laps-owned).** Pin the machine-readable surface
  consumers depend on so future changes version-gate it instead of breaking it silently:
  `list --oneline` (stable single-line form), the structured `.laps/claim` JSON shape,
  the `get`/`claim` task-detail stdout format, and the `get`/`claim` queue-state exit codes
  (`0`/`10`/`11`/`12`). Document these in `README.md` as the consumer contract.
- **Rally phase-1 compatibility (cross-repo, tracked here as the adoption dependency).**
  Rally must adopt the new surface before it can run against 0.9.0:
  - `ReadClaim` (`internal/laps/adapter.go:79`) reads `.laps/claim` and trims it as a bare
    id; the claim is now `{lap,file,scope,claimedAt}` JSON, so the relay's lap identity
    breaks. Parse the `lap` field.
  - `QueueSize` (`internal/laps/adapter.go:91`) counts non-blank lines of `laps list`;
    the default is now two lines per lap, so the count doubles. Use `laps list --oneline`.
  - `get`/`claim` exit codes move from `3` to `10/11/12`; rally treats any non-zero as
    "no lap", which stays correct for `11`/`12` but turns a held stint (`10`) into a relay
    stop instead of a pause. Phase-1 benign (no stints in use); phase-2 must distinguish them.
  - Bump `MinLapsVersion` (`internal/release/release.go:29`) to `0.9.0` across the
    locations listed in rally's `updating-laps-version` skill, and refresh the
    `prepare-laps` version floor and agent-prompt/docs references.
- **`prepare-laps` phase-2 — stint-based workflow (open).** Evolve the skill for the new
  modalities laps now supports: flat laps, single stint, nested stints, and mixed
  laps+stints. The lean is one shared `SKILL.md` entry point with per-mode references,
  keeping the easy `prepare-laps for <change>` trigger. **The skill naming, trigger
  phrases, and modality split are open questions this change frames but does not close.**

## Capabilities

### Added Capabilities
- `consumer-contract`: the laps-owned machine-readable surface consumers depend on — the
  stable `list --oneline` form, the structured claim JSON, the `get`/`claim` task-detail
  format, and the queue-state exit codes — and the rule that these are version-gated.

### Explored (not spec'd yet)
- `stint-workflow-skill`: the `prepare-laps` evolution for flat/stint/nested/mixed
  preparation. Deliberately exploratory in this change (see design "Open: skill
  modalities"); it becomes its own spec'd capability once the modality and naming
  decisions land.

## Impact

- **Code (laps)**: `README.md` gains a "Consumer contract" section pinning the
  machine-readable surface; release tooling ships 0.9.0. No laps behavior change — the
  contract already exists in the v3 implementation, this change documents and pins it.
- **Code (rally, cross-repo)**: `internal/laps/adapter.go` (claim JSON read, `--oneline`
  list, exit-code handling), `internal/release/release.go` (`MinLapsVersion`), CI pin,
  `AGENTS.md`/`README.md`/`tooling-distribution` spec, agent prompts, and the
  `prepare-laps` skill. Tracked here because the two version together.
- **Behavior**: consumers that parsed the old one-line `list` or the bare-id claim file
  break against 0.9.0 until they adopt; documented as the contract cutover.
- **Coordination**: depends on all four landed changes. Rally's `MinLapsVersion` bump and
  laps' 0.9.0 release are a coordinated cutover (rally's installer fetches the matching
  companion).
- **Out of scope**: new laps features; auto-hold heuristics; the TUI; color/TTY theming;
  deciding the final `prepare-laps` modality/naming (framed, not closed, here).
