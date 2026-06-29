## 1. Event-log infrastructure

- [ ] 1.1 Add a best-effort append-only JSONL writer for `.laps/log.jsonl` (append failure warns on stderr, never changes exit code)
- [ ] 1.2 Define the line schema `{ts, event, cmd, file, lap?, title?, assignee?, scope, detail{}, session}`; default `scope` to `root`; `file` is the resolved `.laps`-relative task file
- [ ] 1.3 Read `LAPS_SESSION` and stamp `session` (empty when unset)
- [ ] 1.4 `init` appends `.laps/log.jsonl` to `.gitignore` (idempotent), alongside `.laps/claim`, while preserving the full existing file
- [ ] 1.5 Include required `file` identity in event log entries, status JSON, and structured claims
- [ ] 1.6 Update `internal/cmd/hooks.go:isKnownCommand` so `log` and `status` are treated as built-ins

## 2. Wire events into commands

- [ ] 2.1 Emit one event per applied state change: `created` (add, incl. one line per batch `--json` lap), `completed` (done), `reopened` (done undo), `claimed`/`unclaimed` (claim / claim undo), `deleted` (delete), `pruned` (one line per pruned lap)
- [ ] 2.2 Resolve claim-clear replay semantics for `done` clearing a claim (`unclaimed` with reason vs `completed.detail`)
- [ ] 2.3 Preserve `claimedAt` and do not duplicate events for same-lap reclaims; for different-lap replacement emit `unclaimed` with `detail.reason:"replaced"` followed by `claimed`
- [ ] 2.4 For claim-only mutations, append events only after `WriteClaim`/`RemoveClaim` succeeds and do not emit events on failed claim writes/removes
- [ ] 2.5 Emit `moved`/`edited` for `move`/`edit`/`assign` with `detail` payloads (pos, from/to) after `improve-cli-ergonomics` lands, or route them to that change
- [ ] 2.6 Do not log read/admin commands (`get`, `list`, `count`, `status`, `log`, `version`, `help`, hook-only commands, `init`, `on`, `off`, `update`)

## 3. Log reader

- [ ] 3.1 Add `laps log` printing recent events (human-readable), newest last
- [ ] 3.2 Support `-n <count>`, `--lap <id>`, `--session <id>`, `--since <time>`, `--json-output`
- [ ] 3.3 `laps log --lap <id>` shows that lap's full lifecycle
- [ ] 3.4 Resolve log reader filter/output semantics before implementation
- [ ] 3.5 Tests: event appended per affected lap/transition; best-effort (write failure doesn't fail command); reads don't log; `--lap` filter; `--json-output`; missing log behavior; malformed JSONL behavior

## 4. Status

- [ ] 4.1 Add `laps status` reporting todo/done counts, active (claimed) lap, head lap, assignee breakdown, selected file, and state `active|ready|empty|complete`
- [ ] 4.2 Honor `--json-output` with a stable shape Rally can consume; resolve exact field names/nullability before implementation
- [ ] 4.3 Classify status failure modes: valid snapshots exit 0; corrupt/unreadable store, malformed claim JSON, and JSON serialization failures use the normal error path
- [ ] 4.4 Report dangling/non-todo/wrong-file claims as valid snapshots with `claim.valid=false`; do not auto-clear claims silently
- [ ] 4.5 Tests: counts/head, empty, complete, ready, json shape with file identity, corrupt store, malformed claim, dangling/non-todo/wrong-file claim behavior

## 5. Structured claim

- [ ] 5.1 Change the claim file to `{lap, file, claimedAt}`; read legacy bare-id files back-compatibly as `{lap, file: <selected file>, claimedAt: null}` and ignore unknown JSON fields for forward compatibility
- [ ] 5.2 Treat only non-JSON bare tokens as legacy claims; structured-looking invalid JSON or invalid field types are malformed claim errors
- [ ] 5.3 `status` surfaces the active claim's age; add a stale flag only after the stale-policy product call is resolved
- [ ] 5.4 Tests: claim records `claimedAt`; legacy bare-id still read; malformed structured JSON errors; status age and resolved stale behavior

## 6. Docs & release

- [ ] 6.1 Document the event log, `laps log`, `laps status`, `LAPS_SESSION`, and the structured claim in `README.md`
- [ ] 6.2 Do not bump `VERSION` in this change; `add-stints-gating` owns the final 0.9.0 bump after all four changes land
