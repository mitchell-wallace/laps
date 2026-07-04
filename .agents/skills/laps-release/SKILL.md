---
name: laps-release
description: Release the laps project. Use only in the laps repository when preparing, validating, or troubleshooting a laps release.
---

# Laps Release

This skill documents the release process for the `laps` project.

## Workflow

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
