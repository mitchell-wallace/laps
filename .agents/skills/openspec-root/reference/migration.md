# Migration

Use this when consolidating older canonical OpenSpec skills and opsx commands into `openspec-root`.

## Goal

Keep one advertised OpenSpec skill while preserving detailed workflow guidance as on-demand references.

## Remove Canonical Files

Drop canonical skill folders named:

- `openspec-new-change`
- `openspec-continue-change`
- `openspec-propose`
- `openspec-ff-change`
- `openspec-apply-change`
- `openspec-verify-change`
- `openspec-sync-specs`
- `openspec-archive-change`
- `openspec-explore`

Drop canonical command/workflow files named like:

- `opsx-new`
- `opsx-continue`
- `opsx-propose`
- `opsx-ff`
- `opsx-apply`
- `opsx-verify`
- `opsx-sync`
- `opsx-archive`
- `opsx-explore`

## Keep Custom Files

Do not delete custom OpenSpec-adjacent skills unless explicitly requested. Examples to keep:

- `openspec-plan-review`
- `openspec-risk-audit`
- project-specific OpenSpec policy or review skills

## Placement

Put `openspec-root` under the shared skill directory, typically `.agents/skills/openspec-root/` or a central reusable skills repo. Keep project-specific skills in the project repo.
