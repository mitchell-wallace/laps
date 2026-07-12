# Changelog

## Unreleased

- Unknown and undeclared hook-only commands now fail with a non-zero
  `unknown command` error instead of silently succeeding. Declared hook-only
  commands continue to work as before.
- Read and flow commands no longer create missing `-f/--file` targets. They
  exit `3` with a correction hint for stint-like targets; `add` and default
  `init` creation are unchanged.
- `laps on` now refreshes agent docs to a versioned v2 queue contract, and
  root/get/claim/status help surfaces the `0`/`10`/`11`/`12` protocol.
