# Tasks

## 1. Version and refresh the injected artifact

- [x] 1.1 Rewrite `blockContent` as a concise v2 claim → work → done contract with the required exit actions, held etiquette, transparent stint resolution, and scope-flag guidance.
- [x] 1.2 Generalize block detection/replacement/removal to recognize both legacy and versioned opening tags while preserving surrounding content.
- [x] 1.3 Add instruction tests for fresh v2 injection, idempotence, legacy refresh, and disabling both marker forms.

## 2. Keep the contract aligned with behavior

- [x] 2.1 Introduce named shared constants for queue-state exit codes and use them in `exitForQueueState` / state-name mapping.
- [x] 2.2 Add content-invariant tests proving the block names every queue-state code and required action/guidance.
- [x] 2.3 Keep the v2 block within ±50% of the legacy block's word count; record the before/after counts.

Word counts (`strings.Fields`, including markers): legacy 105, v2 122
(+16%, within the approved ±50% budget).

## 3. Surface the contract in CLI help

- [x] 3.1 Add an `Exit codes` section to the `Long` help for `get`, `claim`, and `status`, documenting `0`, `10`, `11`, and `12` for head `get`/`claim`.
- [x] 3.2 Add a one-line root-help pointer to the detailed command help.
- [x] 3.3 Add command tests covering all four help surfaces and all four codes.
- [x] 3.4 Update README/release notes to mention the v2 refresh and newly self-contained help.

## 4. Verify

- [x] 4.1 Run the full Go test suite and lint gate.
- [x] 4.2 Manual smoke in a throwaway repo: refresh a legacy block, remove the v2 block, and inspect root/get/claim/status help.
