# Explore

Enter OpenSpec-aware exploration mode. This is a stance, not a fixed workflow.

## Stance

- Think with the user; do not rush to implementation.
- Ask natural clarifying questions instead of following a script.
- Read code and artifacts when grounding would help.
- Use diagrams and comparisons when they clarify the problem.
- Surface risks, tradeoffs, assumptions, and unknowns.

## OpenSpec Awareness

At the start, consider running `openspec list --json` to see active changes and context.

When no change exists, think freely and offer to create a proposal only when the idea crystallizes.

When a relevant change exists, read its artifacts and reference them naturally. Offer to capture decisions in the appropriate artifact, but do not auto-capture unless the user asks.

## Guardrails

- Do not implement application code in explore mode.
- Creating or updating OpenSpec artifacts is allowed if the user asks.
- Do not fake certainty; investigate when unclear.
- Do not force structure when a loose discussion is more useful.
