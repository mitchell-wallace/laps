# Repository Guidelines

## Project Purpose
Laps is a lightweight task tracker for AI coding agents. The project is intentionally small and single-binary, but it is built to adapt to different workflows through features such as command hooks, support for multiple task data files, and instruction injection into agent-facing docs. Changes should preserve that balance: keep the core workflow simple, and make extension points predictable.

# IMPORTANT WARNING
`.laps/laps.json` is the LIVE task tracker for development of `laps` itself, and it MUST stay readable by the INSTALLED system binary (the released version, currently schema v2). Never run a development build of `laps` against this repo's own `.laps/` data.

Why this matters: a dev build carries in-progress schema changes (for example the v2→v3 migration in the `add-stints` change). The dev binary auto-migrates and re-stamps the data file on the first mutation — a ONE-WAY change the older system binary then rejects (`unknown field "kind"`). A single `laps done` / `laps add` from a dev binary in the repo root is enough to corrupt the queue.

If `laps list` run with the system binary errors, treat `.laps/laps.json` as corrupted and repair it textually: drop the dev-only schema fields (e.g. `kind`) and restore the prior `version`. Do NOT drop otherwise-valid laps — lap IDs are derived from the checkout directory name, so IDs with a different prefix (e.g. `work-` when the checkout is under `/workspace`) are legitimate, not corruption, and followup laps added by a VERIFY role are in scope. Never "fix" the file by running the dev binary. Use `git log -- .laps/laps.json` to recover the last known-good state.

Testing the dev binary must happen in a throwaway directory with its OWN `.laps/`, never against this repo's real task file, e.g.:
```
just build                                                              # produces bin/laps
d="$(mktemp -d)" && (cd "$d" && /workspace/bin/laps init && /workspace/bin/laps add "smoke" && /workspace/bin/laps list)
```

## Project Structure & Module Organization
`cmd/laps` contains the CLI entrypoint that builds the `laps` binary. Core application code lives under `internal/`: `internal/cmd` defines Cobra commands, `internal/store` manages task-file persistence, `internal/hooks` handles hook execution, and `internal/instructions` manages injected agent instructions. Example configuration lives in [`examples/`](./examples), and top-level docs such as `README.md`, `SPEC.md`, and `TASKS.md` describe behavior and roadmap.

## Build, Test, and Development Commands
- `just build` builds `bin/laps` with version metadata from Git tags.
- `just test` runs the full Go test suite with `go test ./...`.
- `just lint` runs `golangci-lint run ./...` for static checks.
- `just clean` removes the local build output in `bin/`.
- `just run` builds and runs the dev binary (passes through additional args).

For quick iteration, build once with `just build`, then exercise `bin/laps` inside a throwaway directory (see the IMPORTANT WARNING) — never run the dev binary (`go run ./cmd/laps`, `just run`, or `bin/laps`) from the repo root, since it would mutate this repo's real `.laps/laps.json`.

## Coding Style & Naming Conventions
Use standard Go formatting and imports via `gofmt`. Keep indentation tab-based as Go expects. Package names stay short and lowercase (`store`, `hooks`, `cmd`), exported identifiers use `CamelCase`, and unexported helpers use `camelCase`. Follow the existing CLI pattern: each subcommand gets its own file in `internal/cmd` such as `add.go`, `done.go`, or `prune.go`.

## Testing Guidelines
Tests use Go’s built-in `testing` package. Place tests alongside the code they cover with `_test.go` filenames, following the existing pattern such as `internal/store/store_test.go`. Prefer focused unit tests for parser, store, and command behavior; add regression coverage for any bug fix. Run `just test` before opening a PR.

When you need to smoke-test the binary by hand (not covered by `*_test.go`), run it against an isolated copy of the data in a temporary directory — never against this repo's real `.laps/laps.json` (see the IMPORTANT WARNING). The Go test suite is strongly preferred: it already isolates store/command behavior to temp dirs, so prefer adding a `*_test.go` case over a manual smoke run.

## Commit & Pull Request Guidelines
Recent history uses short, imperative commit subjects, often with Conventional Commit prefixes such as `feat:` and `fix:`. Keep commits scoped to one change, for example: `fix: initialize missing laps.json file automatically`. Pull requests should explain the user-visible behavior change, note any data-file or hook implications, link the relevant issue when applicable, and include terminal output or screenshots when CLI output changes materially.

## Releasing

Laps follows the same branch pipeline as rally (see rally's AGENTS.md):
**feature → dev** (CI green) **→ staging** (cleared by a test-driving-rally
pass of the rally+laps pair) **→ main** (releases). Promotions are
fast-forward only; never merge `dev` straight to `main`.

Releases are automated via GitHub Actions. The process is:

1. Update `VERSION` (repo root) to the new semver (e.g. `0.5.0`).
2. Commit with a conventional commit message (e.g. `chore: bump version to 0.5.0`).
3. Push to `main`. The `auto-tag` workflow detects the VERSION change, creates `vX.Y.Z`, pushes it, and dispatches `release.yml`.
4. The `release` workflow checks if a GitHub Release already exists for that tag (idempotent), then runs GoReleaser if not.

Important notes:
- `main.version` is injected via ldflags in `.goreleaser.yaml` (`-X main.version={{.Version}}`). The source variable `cmd.version` stays `""` — GoReleaser and `just build` (via `git describe`) supply the value.
- `auto-tag` validates VERSION format with `^[0-9]+\.[0-9]+\.[0-9]+$` and skips if unchanged.
- The `release` workflow uses goreleaser-action@v6 with `--clean`. Install scripts are uploaded as release assets via `.goreleaser.yaml` `release.extra_files`.
- If a release needs to be redone, delete the existing tag locally and on origin first, then re-push.

## Configuration & Safety Notes
Laps writes task data under `.laps/` in the repository root. Use `examples/hooks.json` as the reference when adding hooks, and avoid destructive hook commands unless they are clearly documented and tested.
