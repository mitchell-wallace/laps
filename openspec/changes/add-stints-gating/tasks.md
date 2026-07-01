## 0. Nested drain prerequisite

- [x] 0.1 Extend active resolution/drain preparation to return the physical parent chain for nested stint refs: parent path, parent file, ref index/pointer, child path, child stint name, and canonical scope for each descent step. Keep cycle detection keyed on physical child identity, not canonical slash scope.
- [x] 0.2 Rework drain finalization to update the immediate parent ref for the drained child stint, archive that child file, and cascade upward while parent stints have no todo laps. Root-level drain behavior must remain unchanged.
- [x] 0.3 Preserve partial-failure safety at every cascade step: preflight archive targets and save targets before mutation; if archive rename succeeds but saving the parent/root ref fails, restore the archived file back to active before returning. Never leave a done parent ref over a present active child file, or a todo parent ref pointing at a missing active child file when the archived file exists.
- [x] 0.4 Update cross-stint `done undo` so an archived nested-stint lap restores the child file, reopens the immediate parent ref, reopens any ancestor refs/files needed to make the lap reachable, and then reopens the lap under the existing latest-completion and 5-minute-age rules.
- [x] 0.5 Tests: completing the final lap in `root -> auth -> search` marks `auth`'s `search` ref done and archives `search`; draining `search` cascades to archive `auth` when `auth` has no remaining todo laps; archive collision/forced save failure at a nested level leaves no dangling parent ref; `done undo` for an archived nested lap restores child + ancestors + lap.

## 1. Hold & release

- [ ] 1.1 Add a held flag to non-archived stint file metadata; missing `held` defaults to `false` and folds into schema v3 before the first v3/0.9.0 binary ships
- [ ] 1.2 `laps stints hold <name>` / `release <name>` target any non-archived stint, including not-yet-enqueued stints; archived stints are refused
- [ ] 1.3 Make hold/release idempotent: already-held/already-released operations do not append duplicate events
- [ ] 1.4 Append `stint.held` / `stint.released` events only when state changes
- [ ] 1.5 `laps stints ls` renders `held` as a SEPARATE badge/marker alongside the queued/active/done lifecycle state, not as a replacement (DECIDED: hold is orthogonal to lifecycle; the lifecycle column keeps its value and a held marker flags held metadata regardless of whether the hold has taken effect at the head)
- [ ] 1.6 Tests: hold sets / release clears before and after enqueue; default missing held is false; archived target refused; idempotent operations do not double-log; held shown in `stints ls`

## 2. Gated flow ops & exit codes

- [ ] 2.1 Implement held detection for `get`/`claim` flow-start resolution and status gate probing: at each context head, a held `kind:"stint"` ref returns held and SHALL NOT descend into the child file
- [ ] 2.2 Map head/flow `get`/`claim` exit codes: `0` lap, `10` held, `11` empty, `12` complete, while preserving existing explicit-id/store/hook failure codes. Use the decided empty-vs-complete mapping (see Resolved Product Calls): `12` = everything resolvable from root head is done and nothing enqueueable remains; `11` = root resolves to zero todo because nothing was enqueued (incl. only-unqueued stint files / empty active stint file)
- [ ] 2.3 For clean state exits, do not use the generic `exit()` helper (it always writes a `laps:` stderr line / error-JSON to stderr). Add a dedicated `exitState(code)` that sets the captured `exitCode`, in JSON mode prints the small queue-state object to **stdout** via `printJSON`, then panics `*exitError{code}` with no stderr error line; held cases also warn on stderr. It MUST go through the existing panic→recover→`os.Exit` path so `runAfterHooksDeferred` still observes the code (a bare `os.Exit`/`return` would skip the after-hook)
- [ ] 2.6 Replace the empty `exit(3,"no head task")` at the post-defer `task == nil` site in `get.go`/`claim.go:49-52` (already after the after-hook defer at `get.go:46`) with `exitState(11)` empty vs `exitState(12)` complete; leave `checkDefault`'s `exit(2)`-on-corrupt and explicit-id-not-found `exit(3)` unchanged (note `checkDefault`'s `ErrEmptyState→exit(3)` branch is dead — nothing returns `ErrEmptyState` — so it is not the empty path). Have flow-start resolution return a single typed queue-state (`lap|held|empty|complete`) consumed by both `get`/`claim` and the `status` gate probe
- [ ] 2.4 Allow `get <id>` to inspect a held stint with a warning; block `claim <id>` into a held stint with exit `10`, no claim mutation, and the same warning
- [ ] 2.4a A hold gates ONLY flow-start (`get`/`claim`): `list`, `count`, `add`, `edit`, `assign`, and `delete` operate normally inside/under a held stint (DECIDED). Verify these commands are unaffected by a held stint (no exit-code change, no warning, no mutation block) and add tests covering at least `list`/`add`/`edit`/`delete` under a held head
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
