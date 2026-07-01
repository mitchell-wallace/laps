## 1. Schema v3 & stint files

- [x] 1.1 Add `kind` to queue entries (`lap` default, `stint`); treat a missing `kind` as `lap`
- [x] 1.2 Add the stint-ref shape (`kind:"stint"`, `ref`, order key, state, timestamps)
- [x] 1.3 Decode/check the top-level version before strict entry-field decoding so newer files still fail with the clear version-gate message
- [x] 1.4 Migrate v2→v3 after v1→v2 ordering migration: stamp `kind:"lap"` on every entry, bump version; keep the version-gate rejection for newer files
- [x] 1.5 Stint files at `.laps/stints/<name>.laps.json` (same `File` schema); archive dir `.laps/stints/archive/`
- [x] 1.6 Enforce stint-name safety and archive no-overwrite behavior
- [x] 1.7 Tests: v2 migrates; missing `kind` ⇒ lap; mixed queue round-trips; newer files with new fields get a version-gate error; unsafe/colliding names rejected
- [x] 1.8 Globally-unique ids: record an allocated 4-char `prefix` in stint file metadata; make `store.GenerateID` (store.go:413) take the containing scope's prefix — repo prefix (`normalizePrefix(repoRoot)`) for root laps, the stint's prefix for stint laps — so a lap id is unique across all files and its prefix identifies its owning queue
- [x] 1.9 Allocate a stint prefix at `stints new`: first 4 lowercase alphanumerics of the stint name, made unique against the repo prefix and all existing stint prefixes by trying other permutations/substrings of the name chars, then incrementing the last char through `0-9a-z`; widening to 6 chars is a future option. Tests: stint laps carry the stint prefix; root vs stint ids never collide; a name colliding with the repo/another stint prefix gets a distinct prefix

## 2. Read-through resolution

- [x] 2.1 Implement deepest-active resolution: from the root head, descend through active stint refs to the first lap (recursive)
- [x] 2.2 Route `get`/`claim`/`done`/`list` through resolution; keep agent output identical (title/description only)
- [x] 2.3 Classify resolver failures: missing child file, malformed ref, malformed child file, and cycles. Key the cycle-detection visited set on **physical stint identity** (`ref` name / resolved child-file path), NOT the slash-path scope string — a path string is unique per descent and would never detect an `A→B→A` cycle
- [x] 2.4 Tests: `get` descends into the active stint; recursion across a nested stint; failure classes do not loop or silently skip

## 3. Scope flags

- [x] 3.1 Add shared local, mutually-exclusive scope flags `--active`/`-c` (default), `--root`/`-r`, `--stint <name>`/`-s` on queue-targeting commands; error when two are combined
- [x] 3.2 Wire them through only the queue-targeting commands in the design command matrix; non-queue/admin commands reject or omit these flags
- [x] 3.3 Reject invocations that combine raw `--file` with any scope flag. Mechanism: per queue-targeting command, `cmd.MarkFlagsMutuallyExclusive("file", "root", "stint", "active")` (the persistent `file` flag is visible in each command's merged flag set). Because `--active` is the default, it must be implemented as a flag detected via `.Changed` (not a default-`true` value), or cobra's mutual-exclusion check will false-positive on every invocation
- [ ] 3.4 Tests: default active, `--root`, `--stint`, scope-flag mutual-exclusion error, non-queue command behavior, `--file` plus scope flag rejection
- [x] 3.5 Extend the `runMB`/`runMBExecute` reset harness in `cmd_test.go`: the new shared scope-flag package vars and per-command local flag sets (plus the `stints` subcommand flag sets) are not in today's hardcoded reset list (`rootCmd.PersistentFlags`, `addCmd`, `listCmd`, `updateCmd`, `doneUndoCmd`), so a `--root`/`--stint` set in one test leaks `Changed=true` into the next. Add the new flag sets (or replace the hardcoded list with a recursive walk of all command flag sets)

## 4. Scoped structure ops & id resolution

- [x] 4.1 Default `add`/`move`/`edit`/`assign`/`delete` to the active scope (`assign` is an `edit` shortcut, so it follows the same default-active rule — it was missing from the structure-op list)
- [x] 4.2 Resolve every id-taking queue command within the selected scope first (`get`, `claim`, `done`, `add after`, `move`, `edit`, `assign`, `delete`); when an id exists in another stint, fail with a message naming that stint
- [x] 4.3 Add `delete --force`; default `delete` refuses a claimed lap with a stderr warning, while forced delete removes it and clears the matching claim
- [ ] 4.4 Tests: `add head` lands in the active stint; `add --root` lands in root; out-of-scope id error names the stint for each id-taking command group; claimed delete refuses; forced claimed delete clears claim

## 5. Claim scope (preemption-safety)

- [x] 5.1 Add `scope` to the claim record (`{lap, file, scope, claimedAt}`); keep back-compat reads
- [x] 5.2 Bare `done` resolves the claimed lap within its recorded scope, regardless of the current head
- [x] 5.3 Rely on the single-actor structural guarantee: a claimed (todo) lap blocks its stint from draining because drain = "no todo laps", so bare `done` always still finds its recorded scope file. Concurrency is a Non-Goal — an explicit `done <id>` (or a second session) completing the claimed lap is allowed and may drain+archive the stint, leaving a stale claim (accepted, no guard). The only guard is `delete` of a claimed lap (task 4.3)
- [x] 5.4 Ensure claim JSON parsing tolerates future fields added by later changes
- [x] 5.5 Tests: claim records scope; `done` completes the claimed lap after an `enqueue head` preemption; future claim fields are ignored

## 6. Enqueue & preemption

- [x] 6.1 `laps stints enqueue <name> [head|tail|after <id>]` via `ComputeInsertOrder` at root, default tail
- [x] 6.2 `head` preempts the active stint non-destructively; the paused stint resumes from its file when the interloper drains
- [x] 6.3 `after <id>` resolves only in the root queue; if the id exists only inside a stint, fail naming that stint
- [x] 6.4 Treat empty stints as ordinary stint files: allow enqueue and let normal no-todo drain/resolution behavior handle them without a special empty-stint state
- [x] 6.5 Tests: enqueue tail default; enqueue head preempts and later resumes with progress intact; root-only `after` resolution; empty stint can be enqueued and follows normal drain/resolution behavior

## 7. Drain & auto-archive

- [x] 7.1 Detect drain (no todo laps left) in the operation that empties a stint; flip its ref to done and set `completedAt`
- [x] 7.2 Move the drained stint file to `.laps/stints/archive/`. Order the drain to fail safe (see design "Ordering and partial-failure"): check the archive-collision/no-overwrite and target writability **before** any mutation, do the `os.Rename` **last**, and flip the root ref to done only after the rename succeeds, so a partial failure never leaves a `done` root ref over a still-present non-archived stint file; the next flow op idempotently re-drains the empty stint
- [x] 7.3 Keep draining content-based/position-independent (a preempted, non-head stint still drains)
- [x] 7.4 `done undo` scans all queue files (root, active stints, `.laps/stints/archive/`) and reopens the lap with the greatest `CompletedAt` across them (replacing today's single-file `done.go:114-123` max). When that lap is in an archived stint, restore the stint file to `.laps/stints/`, reopen the root stint ref (clear its `completedAt`), then reopen the lap under the existing undo rules (5-minute age gate still applies)
- [x] 7.5 Tests: completing the last lap drains→archives; a non-head stint drains correctly; undo after archive unarchives and reopens the ref/lap

## 8. Commands & rendering

- [x] 8.1 `laps stints ls|new <name>|enqueue <name> [pos]|show <name>|rm <name>`; `st` alias for `stints`
- [x] 8.2 `stints ls` lists stint files, shows lap counts for each, and shows whether each stint is queued
- [x] 8.3 `list --tree` renders the full recursive overview
- [x] 8.4 Add `stints rm --force`; default removal allows unqueued non-archived stints and archived stints (including archived stints with done refs), and refuses non-archived queued/active/claimed stints unless forced
- [x] 8.5 Treat unqueued stints as ordinary listed stint files with `queued=false`; no draft/unqueued lifecycle state
- [x] 8.6 Update `internal/cmd/hooks.go:isKnownCommand` so `stints` and `st` are treated as built-ins
- [x] 8.7 Tests: stints lifecycle commands; `st` alias through `runMBExecute`; `--tree` rendering; `rm` safety behavior including force, archived with done ref, and claim clearing; unqueued display/state behavior

## 9. Log & status integration

- [ ] 9.1 Populate the event-log `scope` with the resolved context; add `stint.enqueued`/`completed`/`archived`
- [ ] 9.2 `status` reports the active stint and per-stint progress
- [ ] 9.3 Hook variables under scoped operations: `$file` is the resolved physical task file and `$scope` is the canonical logical scope (`buildHookVars` at `hooks.go:53` has no `scope` today). Also decide the hook-only (unknown-command) path, which builds its vars map inline at `root.go:96-105` and bypasses `buildHookVars`: add `$scope` there defaulting to `root` (or document that hook-only commands do not receive `$scope`)
- [ ] 9.4 Use canonical scope strings everywhere: `root`, root-level stint names, and slash paths for nesting such as `auth/search`
- [ ] 9.5 Tests: scope reflects the stint; stint events logged; status shows active stint; hook variables under scoped `done`; nested scope encoding

## 10. Cross-change dependency

- [x] 10.1 Confirm `improve-cli-ergonomics` has landed before implementing scoped `move`/`edit`/`assign`, or route those command hooks to a follow-up

## 11. Docs & release

- [ ] 11.1 Document stints, scope flags, `laps stints`, `list --tree`, the claim `scope` field, and the per-stint id prefix scheme (lap ids are globally unique and encode their owning stint) in `README.md`
- [ ] 11.2 Do not bump `VERSION` in this change; `add-stints-gating` owns the final 0.9.0 bump after all four changes land
