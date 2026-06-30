## 1. Event-log infrastructure

- [x] 1.1 Add a best-effort append-only JSONL writer for `.laps/log.jsonl` (append failure warns on stderr, never changes exit code)
- [ ] 1.2 Define the line schema `{ts, event, cmd, file, lap?, title?, assignee?, scope, detail{}, session}`; default `scope` to `root`; `file` is the resolved `.laps`-relative task file
- [x] 1.3 Read `LAPS_SESSION` and stamp `session` (empty when unset)
- [x] 1.4 `init` appends `.laps/log.jsonl` to `.gitignore` (idempotent), alongside `.laps/claim`, while preserving the full existing file. Replace today's early-`break` scan (`init.go:46-48` stops at the first `.laps/claim` match and then never appends a second entry, so `log.jsonl` is missed when `claim` is already present): scan the complete file, collect which of the two entries are missing, and append only those. Update the success message (`init.go:75` currently hard-codes "Added .laps/claim to .gitignore") to reflect whichever entries were actually added
- [ ] 1.5 Include required `file` identity in event log entries, status JSON, and structured claims
- [x] 1.6 Update `internal/cmd/hooks.go:isKnownCommand` so `log` and `status` are treated as built-ins

## 2. Wire events into commands

- [x] 2.1 Emit one event per applied state change: `created` (add, incl. one line per batch `--json` lap), `completed` (done), `reopened` (done undo), `claimed`/`unclaimed` (claim / claim undo), `deleted` (delete), `pruned` (one line per pruned lap)
- [x] 2.2 `done` clearing a claim SHALL emit `unclaimed` with `detail.reason:"completed"` immediately after the `completed` event (DECIDED: separate `unclaimed` event for log uniformity with the `replaced` reason)
- [x] 2.3 Preserve `claimedAt` and do not duplicate events for same-lap reclaims; for different-lap replacement emit `unclaimed` with `detail.reason:"replaced"` followed by `claimed`
- [x] 2.4 For claim-only mutations, append events only after `WriteClaim`/`RemoveClaim` succeeds and do not emit events on failed claim writes/removes
- [x] 2.5 Emit `moved` for `move` and `edited` for `edit`/`assign` (there is no `assigned` event — `assign` is an `edit` shortcut, so it logs `edited`) with `detail` payloads (pos, from/to) after `improve-cli-ergonomics` lands, or route them to that change
- [x] 2.6 Do not log read/admin commands (`get`, `list`, `count`, `status`, `log`, `version`, `help`, hook-only commands, `init`, `on`, `off`, `update`)

## 3. Log reader

- [ ] 3.1 Add `laps log` printing recent events (human-readable), newest last
- [ ] 3.2 Support `-n <count>`, `--lap <id>`, `--session <id>`, `--since <time>`, `--json-output`
- [ ] 3.3 `laps log --lap <id>` shows that lap's full lifecycle
- [ ] 3.4 `laps log` semantics (DECIDED): filter first (`--lap`/`--session`/`--since`), then limit to `-n` (default `20`); print newest-last (chronological); `--since` takes an RFC3339 timestamp and is inclusive of the exact timestamp; skip malformed JSONL lines with a one-line stderr note per line (never abort the read); `--json-output` emits a single `{ "events": [ ... ] }` object
- [ ] 3.5 Tests: event appended per affected lap/transition; best-effort (write failure doesn't fail command); reads don't log; `--lap` filter; `--json-output`; missing log behavior; malformed JSONL behavior

## 4. Status

- [x] 4.1 Add `laps status` reporting todo/done counts, active (claimed) lap, head lap, assignee breakdown, selected file, and state `active|ready|empty|complete`
- [x] 4.2 Honor `--json-output` with a stable shape Rally can consume: expose `claimedAt` (nullable RFC3339 UTC timestamp, `null` when no lap is claimed) and `ageSeconds` (integer seconds since `claimedAt`, `null` when `claimedAt` is `null`); do NOT add a `stale` boolean in this change (DECIDED: defer the stale flag until a threshold/policy is chosen)
- [x] 4.3 Classify status failure modes: valid snapshots exit 0; corrupt/unreadable store, malformed claim JSON, and JSON serialization failures use the normal error path
- [x] 4.4 Report dangling/non-todo/wrong-file claims as valid snapshots with `claim.valid=false`; do not auto-clear claims silently
- [x] 4.5 Tests: counts/head, empty, complete, ready, json shape with file identity, corrupt store, malformed claim, dangling/non-todo/wrong-file claim behavior

## 5. Structured claim

- [x] 5.1 Change the claim file to `{lap, file, claimedAt}`; read legacy bare-id files back-compatibly as `{lap, file: <selected file>, claimedAt: null}` and ignore unknown JSON fields for forward compatibility
- [x] 5.1a Define the Go API change this requires: introduce a structured claim type (e.g. `store.Claim{Lap, File string; ClaimedAt *time.Time}`) and change `store.ReadClaim`/`WriteClaim` signatures off the bare `string` (`internal/store/claim.go:13,25`). Decide the no-claim sentinel (today callers test `claimedID == ""` at `done.go:51` and `claim.go:83`; with a struct, use a zero-value/`ok bool` or a nil pointer) and update every caller: `claim.go:57` (write) / `:79` (read in undo), `done.go:46` (read) / `:92` (re-read before clear), the new `status.go`, and the `list` active-marker `ReadClaim` call added by `improve-cli-ergonomics` (which lands first, so its marker call site breaks compilation on the signature change). The `file` for a legacy bare-id is the **caller's resolved selected file**, which the store layer does not know — so `ReadClaim` must take the selected file (or callers stamp it after read); pin which
- [x] 5.2 Treat only non-JSON bare tokens as legacy claims; structured-looking invalid JSON or invalid field types are malformed claim errors. A **missing or empty/whitespace** claim file SHALL remain "no claim" (not a malformed-claim error), preserving today's `done.go:51` / `claim.go:83` "no claim" branches
- [ ] 5.3 `status` surfaces `claimedAt` (nullable RFC3339 UTC) and `ageSeconds` (integer, null when no claim) now; a stale flag is deferred until the stale-policy product call is resolved — do not add a `stale` boolean in this change
- [ ] 5.4 Tests: claim records `claimedAt`; legacy bare-id still read; malformed structured JSON errors; missing/empty claim file is "no claim"; status age and resolved stale behavior
- [x] 5.5 Migrate existing claim tests to the structured API: `internal/store/store_test.go` `TestClaimCRUD` (asserts `WriteClaim(beadsDir, "test-id-123")` and a `string` round-trip) and the `cmd_test.go` `done`/claim assertions that read `store.ReadClaim` and compare against `""` (≈ lines 2118, 2143). A signature/return-type change breaks compilation and these assertions if not migrated alongside

## 6. Docs & release

- [ ] 6.1 Document the event log, `laps log`, `laps status`, `LAPS_SESSION`, and the structured claim in `README.md`
- [ ] 6.2 Do not bump `VERSION` in this change; `add-stints-gating` owns the final 0.9.0 bump after all four changes land
