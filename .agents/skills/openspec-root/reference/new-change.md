# New Change

Start a new OpenSpec change and stop after showing first-artifact instructions.

## Input

The request should include a kebab-case change name or a description. If unclear, ask what the user wants to build or fix.

## Steps

1. Derive or confirm a kebab-case change name.
2. Use the default schema unless the user explicitly names another schema or asks to see workflows.
3. Run `openspec new change "<name>"`, adding `--schema <name>` only when explicitly requested.
4. Run `openspec status --change "<name>"`.
5. Identify the first ready artifact.
6. Run `openspec instructions <first-artifact-id> --change "<name>"`.
7. Stop and ask whether to create the first artifact.

## Guardrails

- Do not create artifacts in this workflow.
- Do not proceed without understanding what the user wants to build.
- If the name is invalid, ask for a valid kebab-case name.
- If the change already exists, suggest continuing it instead.
