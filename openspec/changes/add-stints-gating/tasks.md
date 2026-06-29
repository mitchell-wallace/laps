## 1. Hold & release

- [ ] 1.1 Add a held flag to stint references; `laps stints hold <name>` sets it, `laps stints release <name>` clears it
- [ ] 1.2 Resolve held schema/version ownership before implementation (fold into v3 vs bump to v4; default missing `held:false`)
- [ ] 1.3 Resolve hold/release target semantics before implementation (pre-enqueue, archived, duplicate refs, idempotency)
- [ ] 1.4 Append `stint.held` / `stint.released` events to the event log
- [ ] 1.5 `laps stints ls` shows held as resolved by the product call
- [ ] 1.6 Tests: hold sets / release clears; default missing held is false; older-version rejection; target semantics; events logged; held shown in `stints ls`

## 2. Gated flow ops & exit codes

- [ ] 2.1 Implement held detection for `get`/`claim` flow-start resolution and status gate probing: at each context head, a held `kind:"stint"` ref returns held and SHALL NOT descend into the child file
- [ ] 2.2 Map head/flow `get`/`claim` exit codes: `0` lap, `10` held, `11` empty, `12` complete, while preserving existing explicit-id/store/hook failure codes
- [ ] 2.3 Resolve clean state output shape before implementation; do not use the generic error helper for `10`/`11`/`12`
- [ ] 2.4 Resolve non-starting scoped command and explicit-id semantics under hold before implementation
- [ ] 2.5 Tests: held head → no lap + exit 10; nested held encountered during descent → exit 10; empty → 11; complete → 12; lap present → 0; explicit id not found remains 3; resolved explicit-id held behavior; hook sees exit code

## 3. Finish-under-hold

- [ ] 3.1 Ensure `done` for the claimed lap succeeds while its stint (or the head stint) is held
- [ ] 3.2 Ensure final-lap completion under hold still runs drain/archive instead of leaving a drained held stint stuck
- [ ] 3.3 Tests: non-final claimed lap completes under hold and next `get`/`claim` returns exit 10; final claimed lap completes under hold and next flow state is next lap/empty/complete rather than held

## 4. Status

- [ ] 4.1 Resolve status precedence with active claims before implementation
- [ ] 4.2 `laps status` reports a `held` state, the held stint, and the gate message for valid snapshots with no active-claim precedence conflict; exits 0 for valid snapshots
- [ ] 4.3 Reflect `held` in `--json-output` with the resolved clean state shape
- [ ] 4.4 Tests: status held state in text and JSON; active-claim precedence behavior

## 5. Docs, release & Rally coordination

- [ ] 5.1 Document hold/release, the gate exit codes, and the held status in `README.md`
- [ ] 5.2 Note the `get`/`claim` exit-code contract change for Rally; coordinate the relay-loop update
- [ ] 5.3 After all four changes land, bump `VERSION` to `0.9.0` per the release process
