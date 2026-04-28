# Microbeads — Tasks

Phased checklist. Each phase ends with a working, testable slice. Check items off as they land.

## Phase 0 — Bootstrap

- [ ] `go mod init github.com/mitchell-wallace/microbeads`
- [ ] Pick CLI library (cobra) and add dependency
- [ ] Set up repo layout: `cmd/mb/main.go`, `internal/store/`, `internal/cmd/`, `internal/hooks/`, `internal/instructions/`
- [ ] Add `mb --version` wired to ldflags-injected `version` var
- [ ] Add a basic Makefile or `justfile` with `build`, `test`, `lint`
- [ ] CI: GitHub Actions running `go test ./...` and `go vet`

## Phase 1 — Core data model & storage

- [ ] Define `Task` struct matching SPEC fields, with JSON tags
- [ ] Define file envelope `{version, tasks}` and load/save functions
- [ ] Implement repo-root discovery: walk up looking for `.beads/`, stopping at `.git`, creating `.beads/` next to `.git` if needed
- [ ] Implement target-file resolution: `mb.json` default; `-f name` accepts with or without `.json`
- [ ] Implement empty/missing default + candidate hint behavior
- [ ] Implement ID generation (folder prefix + sha256 slice, with collision extension)
- [ ] Unit tests for: discovery, file resolution, id generation, collision extension

## Phase 2 — Core commands

- [ ] `mb add <head|tail|after> [id] (--title|--json)` with mutual-exclusion check and missing-position hint
- [ ] `mb get [head|<id>]` printing title + blank line + description
- [ ] `mb list` (todo only by default; `--all`, `--done`)
- [ ] `mb done` (head only, no id form; sets timestamps; prints id)
- [ ] `mb delete <id>`
- [ ] Exit codes per SPEC for empty-state and usage errors
- [ ] Integration tests covering each command end-to-end against a temp `.beads/` dir

## Phase 3 — Prune

- [ ] `mb prune [N]` — default N=20, retain N most recent done by completedAt
- [ ] `mb prune 0` deletes all done
- [ ] Tests for boundary cases: fewer than N done, exactly N, more than N, 0 with empty file

## Phase 4 — `mb on` / `mb off`

- [ ] Implement instructions block writer (idempotent replace, not append)
- [ ] `mb on`: write to `AGENTS.md` (create if missing); also `CLAUDE.md`, `GEMINI.md` if they exist
- [ ] `mb off`: remove block from any of those three files
- [ ] Tests: fresh repo, repo with existing AGENTS.md, repo with all three, re-running `on` after edits

## Phase 5 — Hooks

- [ ] Define hooks file schema and loader (`.beads/mb-hooks.json`)
- [ ] Hook dispatcher: filter by `(command, when)`, run in array order
- [ ] Variable substitution for `$id`, `$title`, `$description`, `$command`, `$exit_code`, `$output`, `$file` (and `${var}` form)
- [ ] Shell execution (`/bin/sh -c` on unix; `cmd /C` on windows)
- [ ] `before` hooks abort mb command on non-zero exit (exit code 4)
- [ ] `after` hooks always run; receive mb's stdout as `$output` and its exit code
- [ ] `passback: true` appends hook stdout to mb's stdout
- [ ] Hook-only commands: when user-defined hook command name doesn't match a built-in, `mb <name>` runs hooks and nothing else
- [ ] Tests: before-abort, after-on-failure, passback on/off, hook-only invocation, ordering, missing/malformed hooks file

## Phase 6 — Polish & docs

- [ ] Helpful `mb help` and `mb <cmd> --help` text for every command
- [ ] README with quickstart, command reference, hooks examples
- [ ] Example `mb-hooks.json` file in `examples/`
- [ ] Runbook for the AGENTS.md instruction string — confirm wording with a real agent session

## Phase 7 — Distribution

- [ ] `.goreleaser.yaml` covering linux/darwin/windows × amd64/arm64
- [ ] Tag-driven release workflow in GitHub Actions
- [ ] `install.sh`: detect OS/arch, download latest release, install to `~/.local/bin`, ensure on PATH (bash/zsh/fish), idempotent
- [ ] `install.ps1`: install to `%LOCALAPPDATA%\Programs\mb\`, update user PATH
- [ ] Smoke-test install scripts on fresh linux + macos + windows VMs/containers
- [ ] Document install one-liner in README

## Phase 8 — Rally bridge (separate, lives in rally repo)

These tasks belong in [rally](https://github.com/mitchell-wallace/rally), not microbeads. Listed here for reference so we don't forget the integration story.

- [ ] Rally installer writes a default `.beads/mb-hooks.json` with rally-aware entries (e.g. `mb done` → `rally robots commit`, `mb worktree` → `rally robots worktree $title`)
- [ ] Rally's agent-instruction surface mentions `mb` alongside its own commands so agents know to consult head
- [ ] Decide whether rally bundles the `mb` binary or invokes the user's `mb` from PATH (recommend: PATH only — rally checks for `mb` and prompts to install if missing)
