## 1. List rendering

- [ ] 1.1 Add a two-line default formatter to `list`: line 1 = number + title + active marker, line 2 = `id · assignee · state`, no descriptions
- [ ] 1.2 Add a `--oneline` flag that renders the current single-line format (`<n>. <id> — <title> (assignee: X)`)
- [ ] 1.3 Render done laps struck through under `--all`/`--done` in both layouts
- [ ] 1.4 Active-lap marker: read `.laps/claim` and mark the matching lap; no marker when nothing is claimed
- [ ] 1.5 Register `ls` as an alias of `list` (same flags, same output)
- [ ] 1.6 Tests: two-line default, `--oneline`, marker present/absent, `ls` alias, `--json-output` unchanged

## 2. Move

- [ ] 2.1 Add `move <id> head|tail|after <id>` reusing `store.ComputeInsertOrder`, preserving the lap id
- [ ] 2.2 Operate on todo laps only; error on unknown or already-done id; `after` a done target falls back to head with a stderr notice (mirror `add after`)
- [ ] 2.3 Honor `--json-output`, returning `{task}`
- [ ] 2.4 Tests: move to head/tail/after; id preserved; done/unknown id errors; after-done fallback

## 3. Edit & assign

- [ ] 3.1 Add `edit <id> [--title] [--description] [--assignee]` updating set fields and bumping `updatedAt`; require ≥1 field flag
- [ ] 3.2 Add `assign <id> <role>` as a shortcut for `edit <id> --assignee <role>`
- [ ] 3.3 Honor `--json-output`, returning `{task}` for both
- [ ] 3.4 Tests: edit each field; no-flags error; assign sets assignee; `--json-output`

## 4. Docs & release

- [ ] 4.1 Update `README.md` command reference (list two-line / `--oneline`, `ls`, `move`, `edit`, `assign`)
- [ ] 4.2 Bump `VERSION` per the release process
