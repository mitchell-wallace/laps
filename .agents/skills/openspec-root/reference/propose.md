# Propose

Create a new change and generate all artifacts needed for implementation in one step.

## Input

The request should include a kebab-case change name or a description. If unclear, ask what the user wants to build or fix.

## Steps

1. Derive or confirm a kebab-case change name.
2. Run `openspec new change "<name>"`.
3. Run `openspec status --change "<name>" --json`.
4. Read `applyRequires` and artifact dependency/status data.
5. Track artifact progress with the todo tool if available.
6. Create ready artifacts in dependency order:
   - Run `openspec instructions <artifact-id> --change "<name>" --json`.
   - Read dependency artifacts.
   - Write the artifact to `outputPath` using the template and instruction.
   - Re-run status after each artifact.
7. Stop once all `applyRequires` artifacts are complete.
8. Show `openspec status --change "<name>"` and summarize what is ready.

## Guardrails

- Create all artifacts required before implementation.
- Ask if critical context is unclear; otherwise keep momentum.
- Do not copy `context`, `rules`, or project-context blocks into artifacts.
- If the change exists, ask whether to continue it or create a new one.
