## 1. List rendering

- [x] 1.1 Add a two-line default formatter to `list`: line 1 = number + title + active marker `> `, line 2 = `id · assignee · state`, no descriptions
- [x] 1.2 Add a `--oneline` flag that renders the current single-line format (`<n>. <id> — <title> (assignee: X)`)
- [x] 1.3 Render done laps struck through under `--all`/`--done`: title-only strike in two-line mode, prior whole-line strike under `--oneline`
- [x] 1.4 Active-lap marker: use the central claim-reader contract and mark the matching lap; no marker when nothing is claimed or the claim is outside the rendered result
- [x] 1.5 Register `ls` as an alias of `list` (same flags, same output)
- [x] 1.6 Update `internal/cmd/hooks.go:isKnownCommand` so `ls` is treated as a built-in and is not intercepted as hook-only
- [x] 1.7 Tests: two-line default, `--oneline`, marker present/absent/nonmatching claim, `ls` alias through `runMBExecute`, `--json-output` unchanged
- [x] 1.8 Migrate existing `list` assertions in `internal/cmd/cmd_test.go` to the new default: `TestListDefault` (expects N lines for N laps), `TestListOutputUnchangedWithoutAssignee`, `TestListOutputIncludesAssignee`, the `Contains(out, id+" — ...")` check, and the `idxBefore(list, "— A", "— B")` ordering assertions (these rely on the `— <title>` substring that two-line line 1 no longer carries). Re-point the terse-shape assertions at `--oneline` so the legacy one-line format stays under test rather than deleted

## 2. Move

- [ ] 2.1 Add `move <id> head|tail|after <id>` reusing `store.ComputeInsertOrder`, preserving the lap id
- [ ] 2.2 Operate on todo laps only; error on unknown or already-done id (exit `1`); `after` a missing target errors via `store.ErrTaskNotFound` (exit `3`, like `add after`); `after` a done target falls back to head with a stderr notice that `move.go` emits itself (copy the `fmt.Fprintf(os.Stderr, …)` from `add.go:164` — `ComputeInsertOrder` only returns `fallbackHead`, it does not print); `move <id> after <id>` (self-reference) errors
- [ ] 2.3 Honor `--json-output`, returning `{task}`
- [ ] 2.4 Run before/after hooks with the affected task and populated `$output`/`$exit_code`
- [ ] 2.5 Update `internal/cmd/hooks.go:isKnownCommand` so `move` is treated as a built-in
- [ ] 2.6 Advance `updatedAt` on successful `move`
- [ ] 2.7 Tests: move to head/tail/after; id preserved; done/unknown moved id errors; missing `after` target error; after-done fallback; hook context; `runMBExecute` dispatch; `updatedAt` advances

## 3. Edit & assign

- [ ] 3.1 Add `edit <id> [--title] [--description] [--assignee]` updating set fields and bumping `updatedAt`; require ≥1 field flag. Gate each field on `cmd.Flags().Changed("<name>")` (mirror `add.go:73`), not on a non-empty value, so `--description ""` (clear) is distinguishable from an unset flag; the ≥1-flag check is `Changed("title") || Changed("description") || Changed("assignee")`
- [ ] 3.2 Add `assign <id> <role>` as a shortcut for `edit <id> --assignee <role>`
- [ ] 3.3 Honor `--json-output`, returning `{task}` for both
- [ ] 3.4 Run before/after hooks with the affected task and populated `$output`/`$exit_code`
- [ ] 3.5 Implement edit semantics: blank title errors; `--description ""` and `--assignee ""` clear fields; non-empty assignees trim; descriptions handle escaped `\\n` like `add`; blank `assign` role clears assignee
- [ ] 3.6 Allow `edit`/`assign` on done laps with a stderr warning; preserve done state and `completedAt`
- [ ] 3.7 Update `internal/cmd/hooks.go:isKnownCommand` so `edit` and `assign` are treated as built-ins
- [ ] 3.8 Tests: edit each field; no-flags error; field validation/normalization; blank `assign` clears assignee; done-target warning/preservation; assign sets assignee; non-JSON success prints id; `--json-output`; hook context; `runMBExecute` dispatch
- [ ] 3.9 Extend the `runMB`/`runMBExecute` reset harness in `cmd_test.go`: zero the new package-level vars (`listOneline`, `edit*`, `move*`) and register `editCmd.Flags()`, `moveCmd.Flags()`, `assignCmd.Flags()` (and the `list` flagset for `--oneline`) in the `flag.Changed`-reset loop. Because `edit`'s semantics hinge on `Changed`, a leaked `Changed=true` from one test silently corrupts the next (e.g. a prior `--description ""` clears description in a later title-only edit)

## 4. Docs & release

- [ ] 4.1 Update `README.md` command reference (list two-line / `--oneline`, `ls`, `move`, `edit`, `assign`) and remove those names from any reserved hook-only command-name guidance
- [ ] 4.2 Do not bump `VERSION` in this change; `add-stints-gating` owns the final 0.9.0 bump after all four changes land
