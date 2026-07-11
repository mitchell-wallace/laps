# Proposal: Harden task-file targeting (fail-closed reads, explicit creation)

## Intent

Any laps command aimed at a nonexistent task file via `-f/--file` silently
creates that file and reports an empty queue. For an AI agent, a spelling slip
like `-f stints/auth.laps` vs `-f stints/auth` does not error — it mints a new
unrelated queue file and answers "empty", which reads as "no work to do".
Live sessions this week hit exactly this, and the repo's own `.laps/` contains
a stray `test2.json` from the same footgun. Post-1.0, read paths must be
fail-closed: a missing target is exit `3` with a suggestion, and file creation
happens only on explicit intent.

## Evidence

- `internal/store/store.go:119-127` — `ResolveFile` appends `.json` to any
  `-f` value with no existence or canonical-name check; `store.go:24-25` —
  `ErrEmptyFile` conflates "does not exist" with "empty".
- `internal/cmd/root.go:196-213` — `loadFile` responds to `ErrEmptyFile` by
  **creating and saving** the file, for every command including read-only
  `list`, `get`, `count`, and `log`.
- `internal/cmd/scope.go:48-66` — the sanctioned `--stint <name>` path stats
  the file and exits `3` (`stint auth not found`) when missing; raw `-f` gets
  no such check. Two targeting models with opposite safety properties.
- Verified on a scratch repo: `laps list -f stints/auth` exits `0`, prints
  nothing, and creates `.laps/stints/auth.json` alongside the canonical
  `.laps/stints/auth.laps.json` (which `stints new auth` had created). The
  stray file is invisible to `stints ls` but shadows future `-f` calls.
- `/workspace/laps/.laps/test2.json` — a stray file in this repo's live data
  directory, real-world residue of the same behavior.
- `README.md` (Scope flags) — `--file` is documented as the raw escape hatch,
  mutually exclusive with scope flags; nothing warns that it auto-creates.

## Scope

In scope:
- Read/flow commands (`get`, `claim`, `list`, `count`, `status`, `log`,
  `done`, `move`, `edit`, `assign`, `delete`, `prune`) exit `3` with
  `task file <name> not found` when the `-f` target does not exist.
- Near-miss guidance: when the `-f` value points under `stints/` or matches an
  existing stint name, the error suggests the `--stint <name>` form.
- Explicit creation intent: `laps add` and `laps init` may create a missing
  `-f` target (creation is the point of those verbs); everything else is
  fail-closed. Split `ErrEmptyFile` into not-found vs empty so the policy is
  expressible.
- Default-store behavior unchanged: a missing `.laps/laps.json` is still
  initialized on demand (`store.CheckDefaultStore`, `store.go:687-694`).

Out of scope:
- Removing or deprecating `-f` (documented multi-file support stays).
- Changing `--stint` / `--root` / `--active` semantics.
- The four pinned consumer-contract surfaces (exit `3` for not-found is
  already the contract's meaning for exit `3`).

## Proposed path

Split `store.Load`'s not-found signal from its empty signal
(`ErrFileNotFound` vs `ErrEmptyFile`). Give `loadFile` a creation policy
derived from the command (create-on-missing for `add`/`init`; fail-closed
otherwise), threaded the same way scope flags already are. Add a
`suggestTarget(beadsDir, fileFlag)` helper that recognizes the two observed
typo shapes: a path under `stints/` (suggest `--stint <base>`) and a bare name
matching an existing stint (suggest `--stint <name>`).

## Expected payoff

- The dominant silent-wrong-answer path for agents ("queue is empty" when the
  queue was misspelled) becomes a deterministic exit `3` with a fix-it hint.
- `.laps/` stops accumulating stray queue files that shadow real ones.
- `-f` and `--stint` converge on one safety model; the consumer contract's
  exit-code taxonomy (`3` = not found) gains a consistent meaning for files.

## Risks and unknowns

- **Behavior change**: scripts that relied on `laps add -f newfile` implicitly
  creating a file keep working (add creates), but scripts that relied on,
  e.g., `laps list -f scratch` creating a file now fail. Judged acceptable:
  read-verbs-create was never documented. Release-notes callout required.
- Rally must be checked for any read-path `-f` use against files it has not
  created (`v3-schema-migration` tasks 3.x list rally's touchpoints; none use
  `-f` today, verify during implementation).
- Stint-file prefix allocation on first write (`root.go:199-227`) must keep
  working for the `add`/`init` creation path.

## Spec impact

Contract delta in the scope-resolution domain: file targeting becomes
fail-closed on read, creation becomes verb-gated. See
`specs/scope-resolution/spec.md`.
