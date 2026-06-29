## 1. Hold & release

- [ ] 1.1 Add a held flag to non-archived stint file metadata; missing `held` defaults to `false` and folds into schema v3 before the first v3/0.9.0 binary ships
- [ ] 1.2 `laps stints hold <name>` / `release <name>` target any non-archived stint, including not-yet-enqueued stints; archived stints are refused
- [ ] 1.3 Make hold/release idempotent: already-held/already-released operations do not append duplicate events
- [ ] 1.4 Append `stint.held` / `stint.released` events only when state changes
- [ ] 1.5 `laps stints ls` shows held as resolved by the remaining rendering product call
- [ ] 1.6 Tests: hold sets / release clears before and after enqueue; default missing held is false; archived target refused; idempotent operations do not double-log; held shown in `stints ls`

## 2. Gated flow ops & exit codes

- [ ] 2.1 Implement held detection for `get`/`claim` flow-start resolution and status gate probing: at each context head, a held `kind:"stint"` ref returns held and SHALL NOT descend into the child file
- [ ] 2.2 Map head/flow `get`/`claim` exit codes: `0` lap, `10` held, `11` empty, `12` complete, while preserving existing explicit-id/store/hook failure codes
- [ ] 2.3 For clean state exits, do not use the generic error helper: text mode emits no stdout for `10`/`11`/`12`, held warnings go to stderr, and JSON mode emits a small state object on stdout
- [ ] 2.4 Allow `get <id>` to inspect a held stint with a warning; block `claim <id>` into a held stint with exit `10`, no claim mutation, and the same warning
- [ ] 2.5 Tests: held head → no lap + exit 10 + warning; nested held encountered during descent → exit 10; empty → 11; complete → 12; lap present → 0; explicit id not found remains 3; `get <id>` held inspection warns; `claim <id>` held target exits 10 and does not claim; hook sees exit code

## 3. Finish-under-hold

- [ ] 3.1 Ensure `done` for the claimed lap succeeds while its stint (or the head stint) is held
- [ ] 3.2 Ensure final-lap completion under hold still runs drain/archive instead of leaving a drained held stint stuck
- [ ] 3.3 Tests: non-final claimed lap completes under hold and next `get`/`claim` returns exit 10; final claimed lap completes under hold and next flow state is next lap/empty/complete rather than held

## 4. Status

- [ ] 4.1 Valid active claims keep `status.state=active`; include gate metadata separately when the next head is held
- [ ] 4.2 `laps status` reports a primary `held` state, the held stint, and the gate message for valid snapshots only when no valid active claim takes precedence; exits 0 for valid snapshots
- [ ] 4.3 Reflect `held` in `--json-output` with the resolved clean state shape
- [ ] 4.4 Tests: status held state in text and JSON; active-claim precedence behavior

## 5. Docs, release & Rally coordination

- [ ] 5.1 Document hold/release, the gate exit codes, and the held status in `README.md`
- [ ] 5.2 Note the `get`/`claim` exit-code contract change for Rally; coordinate the relay-loop update
- [ ] 5.3 After all four changes land, bump `VERSION` to `0.9.0` per the release process
