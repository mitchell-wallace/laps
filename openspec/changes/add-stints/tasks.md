## 1. Schema v3 & stint files

- [ ] 1.1 Add `kind` to queue entries (`lap` default, `stint`); treat a missing `kind` as `lap`
- [ ] 1.2 Add the stint-ref shape (`kind:"stint"`, `ref`, order key, state, timestamps)
- [ ] 1.3 Migrate v2→v3: stamp `kind:"lap"` on every entry, bump version; keep the version-gate rejection for newer files
- [ ] 1.4 Stint files at `.laps/stints/<name>.laps.json` (same `File` schema); archive dir `.laps/stints/archive/`
- [ ] 1.5 Tests: v2 migrates; missing `kind` ⇒ lap; mixed queue round-trips

## 2. Read-through resolution

- [ ] 2.1 Implement deepest-active resolution: from the root head, descend through active stint refs to the first lap (recursive)
- [ ] 2.2 Route `get`/`claim`/`done`/`list` through resolution; keep agent output identical (title/description only)
- [ ] 2.3 Tests: `get` descends into the active stint; recursion across a nested stint

## 3. Scope flags

- [ ] 3.1 Add persistent, mutually-exclusive scope flags `--active`/`-c` (default), `--root`/`-r`, `--stint <name>`/`-s`; error when two are combined
- [ ] 3.2 Wire them through resolution for every verb (sugar over `-f` for `--stint`/`--root`; descent semantics for `--active`)
- [ ] 3.3 Tests: default active, `--root`, `--stint`, mutual-exclusion error

## 4. Scoped structure ops & id resolution

- [ ] 4.1 Default `add`/`move`/`edit`/`delete` to the active scope
- [ ] 4.2 Resolve an explicit id within the selected scope; when it exists in another stint, fail with a message naming that stint
- [ ] 4.3 Tests: `add head` lands in the active stint; `add --root` lands in root; out-of-scope id error names the stint

## 5. Claim scope (preemption-safety)

- [ ] 5.1 Add `scope` to the claim record (`{lap, scope, claimedAt}`); keep back-compat reads
- [ ] 5.2 Bare `done` resolves the claimed lap within its recorded scope, regardless of the current head
- [ ] 5.3 Enforce the invariant: a claimed, undone lap keeps its stint from draining
- [ ] 5.4 Tests: claim records scope; `done` completes the claimed lap after an `enqueue head` preemption

## 6. Enqueue & preemption

- [ ] 6.1 `laps stints enqueue <name> [head|tail|after <id>]` via `ComputeInsertOrder` at root, default tail
- [ ] 6.2 `head` preempts the active stint non-destructively; the paused stint resumes from its file when the interloper drains
- [ ] 6.3 Tests: enqueue tail default; enqueue head preempts and later resumes with progress intact

## 7. Drain & auto-archive

- [ ] 7.1 Detect drain (no todo laps left) in the operation that empties a stint; flip its ref to done and set `completedAt`
- [ ] 7.2 Move the drained stint file to `.laps/stints/archive/`
- [ ] 7.3 Keep draining content-based/position-independent (a preempted, non-head stint still drains)
- [ ] 7.4 Tests: completing the last lap drains→archives; a non-head stint drains correctly

## 8. Commands & rendering

- [ ] 8.1 `laps stints ls|new <name>|enqueue <name> [pos]|show <name>|rm <name>`; `st` alias for `stints`
- [ ] 8.2 `stints ls` shows each stint's state (queued/active/done) and todo/done counts
- [ ] 8.3 `list --tree` renders the full recursive overview
- [ ] 8.4 Tests: stints lifecycle commands; `st` alias; `--tree` rendering

## 9. Log & status integration

- [ ] 9.1 Populate the event-log `scope` with the resolved context; add `stint.enqueued`/`completed`/`archived`
- [ ] 9.2 `status` reports the active stint and per-stint progress
- [ ] 9.3 Tests: scope reflects the stint; stint events logged; status shows active stint

## 10. Docs & release

- [ ] 10.1 Document stints, scope flags, `laps stints`, `list --tree`, and the claim `scope` field in `README.md`
- [ ] 10.2 Bump `VERSION` per the release process
