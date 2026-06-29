## Context

Laps drains one flat queue at a time. The operator prepares work per OpenSpec change and
wants several changes staged and pulled in as a pipeline, while agents keep seeing the same
"read head → do it → mark done" contract. Storage already supports multiple files (`-f`), so
the work is a resolution + ergonomics layer, not new storage.

A **stint** is a prepared queue for one change, stored at `.laps/stints/<name>.laps.json`.
`laps.json` stays canonical and may hold laps, **stint references**, or any mix. The active
context is whatever stint ref sits on the path from the resolved root head down to the head
lap — **derived from queue position, never stored** (no active-pointer file), which removes a
class of desync bugs.

## Goals / Non-Goals

**Goals:**
- `laps.json` canonical; a queue entry is a lap or a stint ref (schema v3, back-compat).
- Flow ops descend to the deepest active stint, invisibly to agents.
- One consistent scope vocabulary (`--active`/`--root`/`--stint`) across queue-targeting verbs.
- Preemption-safe completion: claims survive head changes.
- Auto-advancing pipeline with auto-archive of drained stints.

**Non-Goals:**
- Hold / release / gating / exit codes (`add-stints-gating`).
- Cross-layer moves (lap between stints, stint↔root) — within-layer reorder only.
- Nested-stint **creation** UX — the engine recurses, but `enqueue` targets root only.
- Color / TTY theming; the TUI.
- Concurrency / multi-session safety. An explicit `laps done <id>` (or a second session)
  completing a lap that another session has claimed is **allowed** and may drain+archive the
  stint, leaving a stale claim — that is accepted, not guarded. (`delete` of a claimed lap is the
  one guarded case; see "Deleting claimed laps".)

## Decisions

- **Schema v3 / `kind`.** Entries gain `kind` (`lap` default, `stint`); missing `kind` ⇒ lap.
  Migration stamps `kind:"lap"` and bumps the version. A stint ref reuses the entry envelope
  plus `ref`, ordered by the same order key so `enqueue head|tail|after` is just
  `ComputeInsertOrder` at the root layer.
- **Globally-unique ids via stint prefixes (decided).** Today every lap id is
  `<repo-prefix>-<hash>`, where `<repo-prefix>` is the first 4 lowercase alphanumerics of the repo
  directory name (`store.GenerateID` / `normalizePrefix`, `store.go:413-444`), so ids are unique
  only within a file. To make a lap id globally unique *and* self-identify its containing queue,
  **each stint gets its own 4-char prefix**: laps created inside a stint use the stint's prefix,
  while root laps keep the repo prefix. A stint's prefix is allocated once at `stints new` and
  recorded in the stint file metadata (so prefix→stint lookup is O(1) and allocation can check
  existing prefixes). Allocation: take the first 4 lowercase alphanumerics of the stint name; on
  collision with the repo prefix or an existing stint prefix, try other permutations/substrings of
  the stint-name characters, then fall back to incrementing the last character through `0-9a-z`
  until unique. 4 chars is plenty for now and can widen to 6 later without breaking existing ids.
  Consequence: a lap id's prefix maps directly to its owning stint (or root), so scoped
  explicit-id resolution ("`a7` is in stint `search`") and the active-lap marker can identify the
  scope from the id alone — no separate file/scope comparison is needed, which resolves the
  `list`-marker ambiguity that arose with descending `list`.
- **Read-through resolution.** Flow ops start at the root head and descend through active
  stint refs to the first lap; recursive for nesting. Agents see a lap's title/description and
  never know it was nested.
- **Scope flags.** Shared local flags on queue-targeting commands, mutually exclusive, default
  `--active`. `-r`/`-s`/`-c` are the
  only foreclosure-free shorthands (`-t`→title, `-a`→assignee/all, `-d`→description stay free
  for their natural owners). They are **not** plain sugar over `-f`: `-f auth` →
  `.laps/auth.json` via `ResolveFile`, but `--stint auth` → `.laps/stints/auth.laps.json` (a
  different directory and a `.laps.json` infix), so scope-flag path resolution needs a dedicated
  stint-path helper distinct from `ResolveFile`. `--active`/`--root` additionally add the
  descend / no-descend semantics `-f` can't express. Raw `--file` is mutually exclusive with
  scope flags so one invocation has only one target model. Agents use bare verbs (implicit
  `--active`); `prepare-laps` writes the long forms for an explicit structural model.
- **Scoped id resolution.** Explicit-id structure ops resolve within the selected scope; if the
  id lives in another stint, the error names it ("`a7` is in stint `search` — re-run with
  `-s search`"). No silent cross-file action.
- **Canonical scope strings.** Logical scope values use slash paths: `root` for the root queue,
  `auth` for a root-level stint, and `auth/search` for nested stints. Claims, events, status, and
  hook variables SHALL use this encoding. The resolver's **cycle-detection visited set is a
  separate concept** and MUST NOT use the slash-path string: a cumulative path is unique on every
  descent (an `A→B→A` cycle yields `A`, `A/B`, `A/B/A`, …), so a path-keyed visited set never sees
  a repeat and loops forever. The visited set MUST key on **physical stint identity** — the `ref`
  name / resolved child-file path, which is constant per stint regardless of where it sits in the
  path — so a revisited stint is detected.
- **Claim records scope (preemption-safety).** With literal `enqueue head` preemption, a bare
  lap-id claim could complete against the wrong context. So the claim becomes
  `{lap, file, scope, claimedAt}` and bare `done` resolves the claimed lap within its **recorded**
  scope. Invariant (single-actor): a claimed lap is still a todo lap, and drain means "no todo
  laps", so the stint cannot drain while the claim holds — under one actor doing claim→done the
  recorded scope file always still exists when bare `done` looks. This is a **single-actor
  structural guarantee** and that is sufficient: concurrency is a Non-Goal, so an explicit
  `laps done <id>` (or a second session) completing the claimed lap is permitted and may
  drain+archive the stint, leaving a stale claim — accepted. `delete` is the only guarded case
  (refuses a claimed lap unless `--force`, see "Deleting claimed laps"), because delete discards
  in-flight work whereas completing it is legitimate progress.
- **Deleting claimed laps.** `delete` refuses to remove a claimed lap by default, warning on
  stderr. `delete --force <id>` removes it and clears the matching claim; the forced mutation is
  explicit because it can discard in-flight work.
- **Drain → auto-archive.** A stint with no todo laps is drained; the draining operation flips
  its ref to done and moves the file to `.laps/stints/archive/`. Draining is content-based and
  position-independent — a preempted, non-head stint still drains when its last lap completes,
  and the done ref is skipped on later advance.
  - **Ordering and partial-failure.** The drain spans up to three non-transactional writes
    (complete the lap in the stint file, flip+save the root ref, `os.Rename` into `archive/`),
    so order them to fail safe: check the archive-collision (the no-overwrite case) and target
    writability **before** any mutation, then complete the lap, then do the `os.Rename` **last**
    and flip the root ref to done only after the rename succeeds. Define the contract when the
    rename still fails (e.g. crash): the lap completion stands, the ref stays todo, and the next
    flow op re-attempts the drain on the now-empty stint (drain is content-based, so it is
    idempotent) — never leave a `done` root ref pointing at a still-present non-archived stint
    file, which `stints ls`/`rm`'s "archived stint with done ref" matrix would then mishandle.
- **`done undo` unarchives across stints (decided).** `done undo` reopens the globally
  most-recent completion wherever it lives. Today undo (`done.go:114-123`) loads one file and
  takes the max `CompletedAt` within it; because lap ids are now globally unique (see
  "Globally-unique ids"), undo SHALL instead scan **all** queue files — root, active stints, and
  `.laps/stints/archive/` — and select the lap with the greatest `CompletedAt` across them. If
  that lap lives in an archived stint, undo restores the stint file to `.laps/stints/`, reopens
  the root stint ref (clearing its `completedAt`), then reopens the lap under the existing undo
  rules (the 5-minute age gate still applies). Scanning all files is deterministic and reliable —
  it does **not** depend on the best-effort event log; a persisted last-completion pointer is a
  possible later optimisation, not required now.
- **Enqueue / preemption.** Default tail (planned-work order). `head` preempts the active stint;
  because each stint's progress lives in its own file, preemption is non-destructive and the
  paused stint resumes when the interloper drains.
- **Stint listing includes unqueued files.** `stints ls` lists stint files, shows lap counts for
  each, and shows whether each stint is currently queued. Empty and unqueued stints are ordinary
  stint files with no special state beyond their counts and queued flag.
- **`stints rm` safety.** `stints rm <name>` removes unqueued non-archived stint files and
  archived stints, including archived stints that still have a done root ref. It refuses
  non-archived queued, active, or claimed stints unless `--force` is supplied; forced removal also
  removes matching root refs and clears matching claims.
- **Nesting.** The resolution engine and drain cascade are depth-agnostic (honoring "even if
  stints nest"), but creation tooling stays flat in this change — `enqueue` targets root only.
- **Integration.** Events carry the resolved `scope` and add `stint.enqueued`/`completed`/
  `archived` (the event log already defaults `scope` to `root`); `status` reports the active
  stint and per-stint progress; `list --tree` renders the full recursive overview. Hooks under
  scoped operations receive `$file` as the resolved physical task file and a new `$scope` value
  using the canonical scope string.

## Implementation Contracts

- **Schema loading order.** Before adding strict `kind`/`ref` fields, load enough top-level
  envelope data to reject newer schema versions with the existing clear "update laps" message;
  then run v1->v2 ordering migration and v2->v3 `kind:"lap"` stamping in order.
- **Stint names are file names, not paths.** `stints new`/`enqueue`/archive SHALL reject blank
  names, path separators, `.`/`..`, and names that would collide with an existing active or
  archived stint file. Archive moves SHALL be no-overwrite.
- **Resolver failures are classified.** Resolution SHALL keep a visited set for stint refs and
  fail deterministically for cycles, missing child files, malformed stint refs, and malformed
  child files instead of looping or silently skipping a ref. The visited set keys on physical
  stint identity (`ref` name / resolved child-file path), not the slash-path scope string (see
  "Canonical scope strings"), so an `A→B→A` cycle is caught.
- **Command scope matrix.** Queue data commands accept scope flags only when they operate on a
  queue snapshot or queue entry: `add`, `get`, `claim`, `done`, `list`, `count`, `delete`,
  `prune`, `move`, `edit`, and `assign`. Admin/readers without a queue target (`init`, `on`,
  `off`, `update`, `version`, `help`, hook-only commands, `log`, and `status`) do not accept
  these flags unless their own change explicitly adds scope-specific filters. Only the
  **flow ops** `get`/`claim`/`done`/`list` *recursively aggregate/descend* through active stint
  refs (read-through resolution). For the non-flow commands `count`/`prune`: `--active` resolves
  the active stint chain only to *locate* the target file and then operates on **that single file
  only** (no recursive aggregation into nested children), while `--root`/`--stint <name>` name the
  target file directly. So `count --active` counts the deepest active context's own queue and
  `prune --root` prunes root. Spell this out per command so `--active` is not silently equated
  with recursive aggregation everywhere.
- **Explicit id resolution.** Every id-taking queue operation (`get <id>`, `claim <id>`,
  `done <id>`, `add after <id>`, `move`, `edit`, `assign`, and `delete`) resolves ids inside
  the selected scope first; if the id exists in another stint, the error names that stint and
  does not mutate any file. `stints enqueue after <id>` is root-queue structural work: it
  resolves the `after` id only in root, and if the id exists only in a stint it fails naming
  that stint.

## Resolved Product Calls (operator decisions)

- **`done undo` across drained stints — DECIDED: build it now.** Undo scans all queue files
  (root, active stints, archive) for the globally latest `completedAt` and unarchives when that
  lap is in an archived stint (see the "`done undo` unarchives across stints" decision). Reliable
  via the all-files scan; no dependency on the best-effort event log.
- **Completion vs deletion of a claimed lap — DECIDED: guard delete only.** `delete` of a claimed
  lap refuses without `--force`; completing a claimed lap (incl. explicit `done <id>` or a second
  session) is allowed and may drain the stint. Concurrency is a Non-Goal (see Non-Goals and the
  "Claim records scope" decision).
- **`list` rendering once stints land — DECIDED.** Lap ids are made globally unique via stint
  prefixes (see "Globally-unique ids"), so the active-lap marker stays id-based and is
  unambiguous even though `list` descends — no scope/file-aware marker needed. For stint-ref
  entries in a non-`--tree` `list` (e.g. `list --root` over a queue holding refs), render a single
  line `<n>. <name>/ (stint · <N> laps[, held])` so a mixed queue is legible without `--tree`.
- **Bare `laps list` descends — ACCEPTED.** As a flow op, bare `laps list` with a stint active
  shows the *active stint's* laps, not root. Kept for agent consistency; documented, with
  `list --tree` / `--root` as the operator overview.
- **Binary version vs schema version — ACCEPTED.** Schema reaches v3 here while `VERSION` stays
  `0.8.1` until `add-stints-gating` bumps `0.9.0`, so a v3-writing build still reports `0.8.1`.
  Accepted per the single-bump release plan.

## Risks

- The resolution layer touches every flow op; the mitigations are that active context is
  derived (not stored) and that claims pin their scope, so head changes can't misroute `done`.
- Schema migration is one-way (v2→v3). Note the graceful "update laps" version-gate message is
  **not** reachable for already-shipped 0.8.x binaries: `store.Load` decodes with
  `DisallowUnknownFields` (`store.go:128-129`) and the version gate lives later in `loadFile`
  (`root.go:160`), so an old binary hits the unknown `kind`/`ref` fields and fails with the
  generic "exists but is not a valid laps task file" error before it ever checks the version.
  The clean version-gate message only applies to the **new** binary reading a future (v4) file —
  which is exactly why task 1.3 must read the envelope/version *before* strict entry decoding and
  surface the version for future-version files without strict-decoding their entries.
