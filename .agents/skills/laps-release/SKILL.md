---
name: laps-release
description: Release the laps project. Use only in the laps repository when preparing, validating, or troubleshooting a laps release.
---

# Laps Release

This skill documents the release process for the `laps` project.

## Branch pipeline

Laps ships through **feature → dev → staging → main** (same pipeline as
rally; promotions are `git merge --ff-only`):

- **feature → dev** — local checks + CI green (CI runs on dev pushes).
- **dev → staging** — requires a test-driving-rally pass covering the
  rally+laps pair at the dev SHA being promoted (rally and laps version
  together — a test drive exercises both).
- **staging → main** — CI green on the staging SHA; the main push (with a
  VERSION bump) fires auto-tag and the release workflow below.

## Workflow

Run the release from `main` after fast-forwarding it to `staging`:

1. **Run local checks:**
   ```
   just audit
   just release-dry
   ```

2. **Bump version** (patch by default):
   ```
   echo "0.X.Y" > VERSION
   ```

3. **Commit and push:**
   ```
   git add VERSION
   git commit -m "chore: bump version to X.Y.Z"
   git push
   ```

4. **Monitor CI:** `auto-tag` creates `vX.Y.Z`, dispatches `release.yml`. Fix any failures and re-push.

## Notes

- Only bump `VERSION`; `.goreleaser.yaml` version is derived from the Git tag.
- `just build` injects version via `git describe`; GoReleaser uses `{{.Version}}` from the tag.
- To redo a release: delete tag locally and on origin, increment VERSION again, re-push.
