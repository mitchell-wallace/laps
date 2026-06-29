## 1. List rendering

- [ ] 1.1 Add a two-line default formatter to `list`: line 1 = number + title + active marker, line 2 = `id · assignee · state`, no descriptions
- [ ] 1.2 Add a `--oneline` flag that renders the current single-line format (`<n>. <id> — <title> (assignee: X)`)
- [ ] 1.3 Render done laps struck through under `--all`/`--done` in both layouts
- [ ] 1.4 Active-lap marker: use the central claim-reader contract and mark the matching lap; no marker when nothing is claimed or the claim is outside the rendered result
- [ ] 1.5 Register `ls` as an alias of `list` (same flags, same output)
- [ ] 1.6 Update `internal/cmd/hooks.go:isKnownCommand` so `ls` is treated as a built-in and is not intercepted as hook-only
- [ ] 1.7 Tests: two-line default, `--oneline`, marker present/absent/nonmatching claim, `ls` alias through `runMBExecute`, `--json-output` unchanged

## 2. Move

- [ ] 2.1 Add `move <id> head|tail|after <id>` reusing `store.ComputeInsertOrder`, preserving the lap id
- [ ] 2.2 Operate on todo laps only; error on unknown or already-done id; `after` a done target falls back to head with a stderr notice (mirror `add after`)
- [ ] 2.3 Honor `--json-output`, returning `{task}`
- [ ] 2.4 Run before/after hooks with the affected task and populated `$output`/`$exit_code`
- [ ] 2.5 Update `internal/cmd/hooks.go:isKnownCommand` so `move` is treated as a built-in
- [ ] 2.6 Advance `updatedAt` on successful `move`
- [ ] 2.7 Tests: move to head/tail/after; id preserved; done/unknown moved id errors; missing `after` target error; after-done fallback; hook context; `runMBExecute` dispatch; `updatedAt` advances

## 3. Edit & assign

- [ ] 3.1 Add `edit <id> [--title] [--description] [--assignee]` updating set fields and bumping `updatedAt`; require ≥1 field flag
- [ ] 3.2 Add `assign <id> <role>` as a shortcut for `edit <id> --assignee <role>`
- [ ] 3.3 Honor `--json-output`, returning `{task}` for both
- [ ] 3.4 Run before/after hooks with the affected task and populated `$output`/`$exit_code`
- [ ] 3.5 Resolve open edit semantics before implementation: blank title, empty description/assignee clearing, assignee trimming, escaped `\\n` handling, and blank `assign` role
- [ ] 3.6 Allow `edit`/`assign` on done laps with a stderr warning; preserve done state and `completedAt`
- [ ] 3.7 Update `internal/cmd/hooks.go:isKnownCommand` so `edit` and `assign` are treated as built-ins
- [ ] 3.8 Tests: edit each field; no-flags error; field validation/normalization from the resolved semantics; done-target warning/preservation; assign sets assignee; `--json-output`; hook context; `runMBExecute` dispatch

## 4. Docs & release

- [ ] 4.1 Update `README.md` command reference (list two-line / `--oneline`, `ls`, `move`, `edit`, `assign`) and remove those names from any reserved hook-only command-name guidance
- [ ] 4.2 Do not bump `VERSION` in this change; `add-stints-gating` owns the final 0.9.0 bump after all four changes land
