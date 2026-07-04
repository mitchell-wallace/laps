# Archive Change

Archive a completed OpenSpec change.

## Input

Optionally specify a change name. If omitted, prompt for selection from active changes.

## Steps

1. Run `openspec list --json` if the change is not explicit, then ask the user to select.
2. Run `openspec status --change "<name>" --json` and check artifact completion.
3. If artifacts are incomplete, list them and confirm before proceeding.
4. Read the tasks file if present; if incomplete tasks exist, warn and confirm.
5. Check for delta specs under `openspec/changes/<name>/specs/`.
6. If delta specs exist, compare them to `openspec/specs/<capability>/spec.md`, summarize sync needs, and ask whether to sync first.
7. If sync is requested, follow `sync-specs.md` before archiving.
8. Create `openspec/changes/archive` if needed.
9. Move the change to `openspec/changes/archive/YYYY-MM-DD-<name>`.
10. Summarize the archive location, schema, sync status, and warnings.

## Guardrails

- Do not auto-select a change when the name is omitted.
- Do not block archive on warnings if the user confirms.
- Preserve `.openspec.yaml` by moving the whole change directory.
- If the target archive directory already exists, stop and report the conflict.
