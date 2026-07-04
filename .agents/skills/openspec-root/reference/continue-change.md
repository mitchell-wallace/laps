# Continue Change

Create the next ready artifact for an OpenSpec change.

## Input

Optionally specify a change name. If omitted, prompt for selection from recent active changes.

## Steps

1. If needed, run `openspec list --json` and ask the user to choose from the most recent active changes.
2. Run `openspec status --change "<name>" --json`.
3. If `isComplete` is true, report completion and suggest apply or archive.
4. Find the first artifact with `status: "ready"`.
5. Run `openspec instructions <artifact-id> --change "<name>" --json`.
6. Read completed dependency files from the instruction output.
7. Create the artifact at `outputPath` using `template`, `instruction`, `context`, and `rules` as guidance.
8. Run `openspec status --change "<name>"` and summarize progress.

## Guardrails

- Create exactly one artifact per invocation.
- Never skip artifacts or create them out of order.
- Do not copy `context`, `rules`, or project-context blocks into artifact files.
- Ask if the required artifact content is unclear.
- Verify the artifact file exists before reporting success.
