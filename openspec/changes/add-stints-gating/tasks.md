## 1. Hold & release

- [ ] 1.1 Add a held flag to stints; `laps stints hold <name>` sets it, `laps stints release <name>` clears it
- [ ] 1.2 Append `stint.held` / `stint.released` events to the event log
- [ ] 1.3 `laps stints ls` shows the held state
- [ ] 1.4 Tests: hold sets / release clears; events logged; held shown in `stints ls`

## 2. Gated flow ops & exit codes

- [ ] 2.1 When the resolved head is a held stint, `get`/`claim` return no lap and SHALL NOT descend
- [ ] 2.2 Map `get`/`claim` exit codes: `0` lap, `10` held, `11` empty, `12` complete
- [ ] 2.3 Tests: held head → no lap + exit 10; empty → 11; complete → 12; lap present → 0

## 3. Finish-under-hold

- [ ] 3.1 Ensure `done` for the claimed lap succeeds while its stint (or the head stint) is held
- [ ] 3.2 Tests: claimed lap completes under hold; next `get`/`claim` still returns exit 10

## 4. Status

- [ ] 4.1 `laps status` reports a `held` state, the held stint, and the gate message; always exits 0
- [ ] 4.2 Reflect `held` in `--json-output`
- [ ] 4.3 Tests: status held state in text and JSON

## 5. Docs, release & Rally coordination

- [ ] 5.1 Document hold/release, the gate exit codes, and the held status in `README.md`
- [ ] 5.2 Note the `get`/`claim` exit-code contract change for Rally; coordinate the relay-loop update
- [ ] 5.3 Bump `VERSION` per the release process
