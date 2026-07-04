# Sync Specs

Apply delta specs from an active change to main specs.

## Input

Optionally specify a change name. If omitted or ambiguous, prompt for a change that has delta specs.

## Steps

1. If needed, run `openspec list --json` and ask the user to select a change with delta specs.
2. Find delta specs under `openspec/changes/<name>/specs/*/spec.md`.
3. For each capability, read the delta spec and the main spec at `openspec/specs/<capability>/spec.md` if it exists.
4. Apply changes intelligently:
   - `ADDED Requirements`: add missing requirements or update existing ones to match intent.
   - `MODIFIED Requirements`: merge partial updates while preserving unmentioned content.
   - `REMOVED Requirements`: remove the requirement block.
   - `RENAMED Requirements`: rename the requirement from `FROM` to `TO`.
5. Create a new main spec if the capability does not exist.
6. Summarize updated capabilities and changed requirements.

## Guardrails

- Read both delta and main specs before editing.
- Preserve existing content not mentioned in the delta.
- Make the operation idempotent.
- Ask when the intended merge is unclear.
