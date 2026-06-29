## 1. Schema v3 & stint files

- [ ] 1.1 Add `kind` to queue entries (`lap` default, `stint`); treat a missing `kind` as `lap`
- [ ] 1.2 Add the stint-ref shape (`kind:"stint"`, `ref`, order key, state, timestamps)
- [ ] 1.3 Decode/check the top-level version before strict entry-field decoding so newer files still fail with the clear version-gate message
- [ ] 1.4 Migrate v2→v3 after v1→v2 ordering migration: stamp `kind:"lap"` on every entry, bump version; keep the version-gate rejection for newer files
- [ ] 1.5 Stint files at `.laps/stints/<name>.laps.json` (same `File` schema); archive dir `.laps/stints/archive/`
- [ ] 1.6 Enforce stint-name safety and archive no-overwrite behavior
- [ ] 1.7 Tests: v2 migrates; missing `kind` ⇒ lap; mixed queue round-trips; newer files with new fields get a version-gate error; unsafe/colliding names rejected

## 2. Read-through resolution

- [ ] 2.1 Implement deepest-active resolution: from the root head, descend through active stint refs to the first lap (recursive)
- [ ] 2.2 Route `get`/`claim`/`done`/`list` through resolution; keep agent output identical (title/description only)
- [ ] 2.3 Classify resolver failures: missing child file, malformed ref, malformed child file, and cycles
- [ ] 2.4 Tests: `get` descends into the active stint; recursion across a nested stint; failure classes do not loop or silently skip

## 3. Scope flags

- [ ] 3.1 Add shared local, mutually-exclusive scope flags `--active`/`-c` (default), `--root`/`-r`, `--stint <name>`/`-s` on queue-targeting commands; error when two are combined
- [ ] 3.2 Wire them through only the queue-targeting commands in the design command matrix; non-queue/admin commands reject or omit these flags
- [ ] 3.3 Reject invocations that combine raw `--file` with any scope flag
- [ ] 3.4 Tests: default active, `--root`, `--stint`, scope-flag mutual-exclusion error, non-queue command behavior, `--file` plus scope flag rejection

## 4. Scoped structure ops & id resolution

- [ ] 4.1 Default `add`/`move`/`edit`/`delete` to the active scope
- [ ] 4.2 Resolve every id-taking queue command within the selected scope first (`get`, `claim`, `done`, `add after`, `move`, `edit`, `assign`, `delete`); when an id exists in another stint, fail with a message naming that stint
- [ ] 4.3 Resolve the delete-claimed-lap product call before implementation
- [ ] 4.4 Tests: `add head` lands in the active stint; `add --root` lands in root; out-of-scope id error names the stint for each id-taking command group

## 5. Claim scope (preemption-safety)

- [ ] 5.1 Add `scope` to the claim record (`{lap, scope, claimedAt}`); keep back-compat reads
- [ ] 5.2 Bare `done` resolves the claimed lap within its recorded scope, regardless of the current head
- [ ] 5.3 Enforce the invariant: a claimed, undone lap keeps its stint from draining
- [ ] 5.4 Ensure claim JSON parsing tolerates future fields added by later changes
- [ ] 5.5 Tests: claim records scope; `done` completes the claimed lap after an `enqueue head` preemption; future claim fields are ignored

## 6. Enqueue & preemption

- [ ] 6.1 `laps stints enqueue <name> [head|tail|after <id>]` via `ComputeInsertOrder` at root, default tail
- [ ] 6.2 `head` preempts the active stint non-destructively; the paused stint resumes from its file when the interloper drains
- [ ] 6.3 `after <id>` resolves only in the root queue; if the id exists only inside a stint, fail naming that stint
- [ ] 6.4 Treat empty stints as ordinary stint files: allow enqueue and let normal no-todo drain/resolution behavior handle them without a special empty-stint state
- [ ] 6.5 Tests: enqueue tail default; enqueue head preempts and later resumes with progress intact; root-only `after` resolution; empty stint can be enqueued and follows normal drain/resolution behavior

## 7. Drain & auto-archive

- [ ] 7.1 Detect drain (no todo laps left) in the operation that empties a stint; flip its ref to done and set `completedAt`
- [ ] 7.2 Move the drained stint file to `.laps/stints/archive/`
- [ ] 7.3 Keep draining content-based/position-independent (a preempted, non-head stint still drains)
- [ ] 7.4 `done undo` restores the archived stint file and reopens the stint ref when undoing a completion from an archived stint
- [ ] 7.5 Tests: completing the last lap drains→archives; a non-head stint drains correctly; undo after archive unarchives and reopens the ref/lap

## 8. Commands & rendering

- [ ] 8.1 `laps stints ls|new <name>|enqueue <name> [pos]|show <name>|rm <name>`; `st` alias for `stints`
- [ ] 8.2 `stints ls` lists stint files, shows lap counts for each, and shows whether each stint is queued
- [ ] 8.3 `list --tree` renders the full recursive overview
- [ ] 8.4 Resolve `stints rm` safety semantics before implementation
- [ ] 8.5 Treat unqueued stints as ordinary listed stint files with `queued=false`; no draft/unqueued lifecycle state
- [ ] 8.6 Update `internal/cmd/hooks.go:isKnownCommand` so `stints` and `st` are treated as built-ins
- [ ] 8.7 Tests: stints lifecycle commands; `st` alias through `runMBExecute`; `--tree` rendering; `rm` safety behavior; unqueued display/state behavior

## 9. Log & status integration

- [ ] 9.1 Populate the event-log `scope` with the resolved context; add `stint.enqueued`/`completed`/`archived`
- [ ] 9.2 `status` reports the active stint and per-stint progress
- [ ] 9.3 Resolve hook `$file`/`$scope` behavior under resolution before implementation
- [ ] 9.4 Resolve canonical scope string encoding before implementation, including nested stints
- [ ] 9.5 Tests: scope reflects the stint; stint events logged; status shows active stint; hook variables under scoped `done`; nested scope encoding

## 10. Cross-change dependency

- [ ] 10.1 Confirm `improve-cli-ergonomics` has landed before implementing scoped `move`/`edit`/`assign`, or route those command hooks to a follow-up

## 11. Docs & release

- [ ] 11.1 Document stints, scope flags, `laps stints`, `list --tree`, and the claim `scope` field in `README.md`
- [ ] 11.2 Do not bump `VERSION` in this change; `add-stints-gating` owns the final 0.9.0 bump after all four changes land
