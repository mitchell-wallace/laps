## 1. Event-log infrastructure

- [ ] 1.1 Add a best-effort append-only JSONL writer for `.laps/log.jsonl` (append failure warns on stderr, never changes exit code)
- [ ] 1.2 Define the line schema `{ts, event, cmd, lap?, title?, assignee?, scope, detail{}, session}`; default `scope` to `root`
- [ ] 1.3 Read `LAPS_SESSION` and stamp `session` (empty when unset)
- [ ] 1.4 `init` appends `.laps/log.jsonl` to `.gitignore` (idempotent), alongside `.laps/claim`

## 2. Wire events into commands

- [ ] 2.1 Emit `created` (add, incl. batch `--json`), `completed` (done), `reopened` (done undo), `claimed`/`unclaimed` (claim / claim undo), `deleted` (delete), `pruned` (prune)
- [ ] 2.2 Emit `moved`/`edited` for `move`/`edit`/`assign` with `detail` payloads (pos, from/to)
- [ ] 2.3 Do not log read commands (`get`, `list`, `status`)

## 3. Log reader

- [ ] 3.1 Add `laps log` printing recent events (human-readable), newest last
- [ ] 3.2 Support `-n <count>`, `--lap <id>`, `--session <id>`, `--since <time>`, `--json-output`
- [ ] 3.3 `laps log --lap <id>` shows that lap's full lifecycle
- [ ] 3.4 Tests: event appended per command; best-effort (write failure doesn't fail command); reads don't log; `--lap` filter; `--json-output`

## 4. Status

- [ ] 4.1 Add `laps status` reporting todo/done counts, active (claimed) lap, head lap, assignee breakdown, and state `active|empty|complete`; always exit 0
- [ ] 4.2 Honor `--json-output` with a stable shape Rally can consume
- [ ] 4.3 Tests: counts/head, empty, complete, json shape

## 5. Structured claim

- [ ] 5.1 Change the claim file to `{lap, claimedAt}`; read legacy bare-id files back-compatibly as `{lap, claimedAt: null}`
- [ ] 5.2 `status` surfaces the active claim's age and flags a stale claim
- [ ] 5.3 Tests: claim records `claimedAt`; legacy bare-id still read; status age/stale signal

## 6. Docs & release

- [ ] 6.1 Document the event log, `laps log`, `laps status`, `LAPS_SESSION`, and the structured claim in `README.md`
- [ ] 6.2 Bump `VERSION` per the release process
