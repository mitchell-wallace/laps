# Verify Change

Verify that implementation matches OpenSpec change artifacts.

## Input

Optionally specify a change name. If omitted or ambiguous, prompt for a change with implementation tasks.

## Steps

1. Select the change.
2. Run `openspec status --change "<name>" --json` to identify schema and artifacts.
3. Run `openspec instructions apply --change "<name>" --json` and read all available `contextFiles`.
4. Verify completeness:
   - Check task checkboxes.
   - Check delta requirements for likely implementation evidence.
5. Verify correctness:
   - Map requirements and scenarios to implementation evidence.
   - Note divergences or missing coverage.
6. Verify coherence:
   - Check implementation against design decisions when `design.md` exists.
   - Check consistency with project patterns.
7. Produce a report grouped by CRITICAL, WARNING, and SUGGESTION.

## Severity

- CRITICAL: incomplete tasks or missing required implementation.
- WARNING: likely spec/design divergence or missing scenario coverage.
- SUGGESTION: pattern consistency or low-risk improvements.

## Guardrails

- Every issue must be specific and actionable.
- Prefer lower severity when uncertain.
- Note which checks were skipped and why.
- Use file/line references when applicable.
