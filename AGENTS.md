# Repository Guidelines

## Project Purpose
Microbeads is a lightweight task tracker for AI coding agents. The project is intentionally small and single-binary, but it is built to adapt to different workflows through features such as command hooks, support for multiple task data files, and instruction injection into agent-facing docs. Changes should preserve that balance: keep the core workflow simple, and make extension points predictable.

## Project Structure & Module Organization
`cmd/mb` contains the CLI entrypoint that builds the `mb` binary. Core application code lives under `internal/`: `internal/cmd` defines Cobra commands, `internal/store` manages task-file persistence, `internal/hooks` handles hook execution, and `internal/instructions` manages injected agent instructions. Example configuration lives in [`examples/`](./examples), and top-level docs such as `README.md`, `SPEC.md`, and `TASKS.md` describe behavior and roadmap.

## Build, Test, and Development Commands
- `make build` builds `bin/mb` with version metadata from Git tags.
- `make test` runs the full Go test suite with `go test ./...`.
- `make lint` runs `go vet ./...` for static checks.
- `make clean` removes the local build output in `bin/`.

For quick iteration, `go run ./cmd/mb list` is useful when testing CLI behavior without rebuilding.

## Coding Style & Naming Conventions
Use standard Go formatting and imports via `gofmt`. Keep indentation tab-based as Go expects. Package names stay short and lowercase (`store`, `hooks`, `cmd`), exported identifiers use `CamelCase`, and unexported helpers use `camelCase`. Follow the existing CLI pattern: each subcommand gets its own file in `internal/cmd` such as `add.go`, `done.go`, or `prune.go`.

## Testing Guidelines
Tests use Go’s built-in `testing` package. Place tests alongside the code they cover with `_test.go` filenames, following the existing pattern such as `internal/store/store_test.go`. Prefer focused unit tests for parser, store, and command behavior; add regression coverage for any bug fix. Run `make test` before opening a PR.

## Commit & Pull Request Guidelines
Recent history uses short, imperative commit subjects, often with Conventional Commit prefixes such as `feat:` and `fix:`. Keep commits scoped to one change, for example: `fix: initialize missing mb.json file automatically`. Pull requests should explain the user-visible behavior change, note any data-file or hook implications, link the relevant issue when applicable, and include terminal output or screenshots when CLI output changes materially.

## Configuration & Safety Notes
Microbeads writes task data under `.beads/` in the repository root. Use `examples/mb-hooks.json` as the reference when adding hooks, and avoid destructive hook commands unless they are clearly documented and tested.
