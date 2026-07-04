# Apply Change

Implement tasks from an OpenSpec change.

## Input

Optionally specify a change name. If omitted, infer from conversation only when clear. If vague or ambiguous, prompt for available changes.

## Steps

1. Select the change.
2. Run `openspec status --change "<name>" --json` to identify the schema and task artifact.
3. Run `openspec instructions apply --change "<name>" --json`.
4. Handle instruction states:
   - `blocked`: explain missing artifacts and suggest continuing the change.
   - `all_done`: congratulate and suggest archive.
   - otherwise: proceed.
5. Read every file listed in `contextFiles`.
6. Show schema, progress, remaining task overview, and dynamic instruction.
7. Implement pending tasks one at a time.
8. After each completed task, update its checkbox from `- [ ]` to `- [x]`.
9. Stop only when all tasks are done, the user interrupts, or a blocker/ambiguity appears.

## Guardrails

- Always read context files before implementation.
- Keep changes minimal and scoped to each task.
- Ask before guessing on ambiguous tasks.
- If implementation reveals a design issue, pause and suggest updating artifacts.
- Use `contextFiles` from the CLI output; do not assume fixed artifact names.
