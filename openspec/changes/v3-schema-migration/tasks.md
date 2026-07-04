## 1. Release laps 0.9.0 (the v3 cutover)

- [ ] 1.1 Confirm the four v3 changes are implemented and `VERSION` is `0.9.0`
- [ ] 1.2 Tag `v0.9.0` and run the release workflow (auto-tag → release.yml → GoReleaser); verify the released binary reports `0.9.0` and schema v3
- [ ] 1.3 Smoke-test the released binary in a throwaway dir (never against this repo's own `.laps/laps.json` — see AGENTS.md WARNING): `init`, `add`, `list --oneline`, `claim`, `done`, `stints new|enqueue|ls`, `status`, `log`

## 2. Formalize the consumer contract (laps-owned)

- [ ] 2.1 Add a "Consumer contract" section to `README.md` pinning: `list --oneline` shape, `.laps/claim` JSON shape, `get`/`claim` task-detail stdout format, and `get`/`claim` queue-state exit codes (`0`/`10`/`11`/`12`); state that the default `list`, `status`, and `log` outputs are operator-facing and not stable
- [ ] 2.2 State the version-gating rule: a change that alters a pinned surface requires a `VERSION` bump, a contract-section update, and a consumer-impact note in its proposal
- [ ] 2.3 Land the `consumer-contract` delta spec (this change's `specs/consumer-contract/spec.md`)

## 3. Rally phase-1 compatibility (cross-repo — `[rally]` tasks, gated on the 0.9.0 release)

- [ ] 3.1 [rally] `ReadClaim` (`internal/laps/adapter.go:79`): parse `.laps/claim` as JSON and return the `lap` field; tolerate a bare-id fallback for transitional files
- [ ] 3.2 [rally] `QueueSize` (`internal/laps/adapter.go:91`): switch to `laps list --oneline` so the count is one-per-lap under the v3 two-line default
- [ ] 3.3 [rally] Stop assuming `get`/`claim` use exit `3`; document that `10` is reserved for phase-2 hold handling (no relay-loop behaviour change this phase — `11`/`12` still end the relay correctly)
- [ ] 3.4 [rally] Bump `MinLapsVersion` to `0.9.0` via the `updating-laps-version` skill (all sync locations: `internal/release/release.go`, `.github/workflows/test.yml` CI pin, `AGENTS.md`, `README.md`, `openspec/specs/tooling-distribution/spec.md`, `.claude/skills/prepare-laps/SKILL.md`)
- [ ] 3.5 [rally] Refresh agent-prompt/docs references that assume the bare-id claim file or one-line list; confirm `laps done`/`handoff`/`wrapup` and `$id`/`$args` hook vars are unchanged (no edit expected)
- [ ] 3.6 [rally] Verify the direct `.laps/laps.json` read (`internal/relay/runner.go:3490`) still ignores the new `kind`/`ref` fields (uses default `json.Unmarshal`); add a regression note/comment so a future strict-decode switch does not silently break it
- [ ] 3.7 [rally] Run the real-laps e2e tests against the 0.9.0 companion to confirm identical observable behaviour with no stints

## 4. prepare-laps phase-2 — stint-based workflow (OPEN; do not implement until the modality decision lands)

- [ ] 4.1 Run an explore/propose cycle on the skill modalities (flat / single stint / nested / mixed) and the shared-`SKILL.md`-with-references structure; settle naming and trigger phrases (`prepare-laps for <change>` kept; whether `prepare-stints for <change>` is an alias to the same skill or absent)
- [ ] 4.2 Decide when a stint is the default (leading candidate: stint for OpenSpec-change input, flat laps for ad-hoc input) — revisit
- [ ] 4.3 Decide what `QueueSize` counts across a stint pipeline (`--root --oneline`, active scope, or `--tree`) and whether rally distinguishes exit `10` to pause vs end the relay
- [ ] 4.4 Promote the settled design into a spec'd `stint-workflow-skill` capability in its own change (not frozen here)

## 5. Coordination & release notes

- [ ] 5.1 Write the 0.9.0 release notes naming the consumer-visible breaks (#1 list default, #2 claim JSON, #3 exit codes) and the adoption steps
- [ ] 5.2 Confirm rally's installer fetches the 0.9.0 companion in the same release window
- [ ] 5.3 Note the schema-v2→v3 one-way migration risk for any consumer still running a pre-v3 binary against a v3-mutated `.laps/laps.json` (cross-reference AGENTS.md WARNING)
