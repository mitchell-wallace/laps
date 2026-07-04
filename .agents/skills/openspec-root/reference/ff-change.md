# Fast-Forward Change

Create all artifacts needed for implementation in one pass.

## Input

The request should include a kebab-case change name or a description. If unclear, ask what the user wants to build or fix.

## Steps

1. Derive or confirm a kebab-case change name.
2. Run `openspec new change "<name>"`.
3. Run `openspec status --change "<name>" --json`.
4. Read `applyRequires` and the artifact dependency/status list.
5. Track artifact progress with the todo tool if available.
6. Loop through ready artifacts in dependency order:
   - Run `openspec instructions <artifact-id> --change "<name>" --json`.
   - Read completed dependency files.
   - Create the artifact at `outputPath` using the template and instruction.
   - Re-run status.
7. Stop when every artifact in `applyRequires` is done.
8. Show final `openspec status --change "<name>"` and summarize created artifacts.

## Guardrails

- Create all artifacts needed for implementation, not necessarily every possible artifact.
- Prefer reasonable decisions to maintain momentum, but ask on critical uncertainty.
- Do not copy instruction-only `context` or `rules` into artifacts.
- If the change already exists, suggest continuing it instead.
