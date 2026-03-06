---
name: release
description: Trigger the GitHub Actions Release workflow and watch it complete
disable-model-invocation: true
allowed-tools: Bash, Read
---

# Release

Trigger the full release pipeline via GitHub Actions. This runs semantic-release (which determines the next semver version), GoReleaser, container image staging, and kustomize base push.

## Steps

1. Ensure the working tree is clean and the local branch is up to date with origin:
   ```
   git status
   git pull --rebase origin main
   ```
   If there are uncommitted changes, stop and ask the user to commit first.

2. Trigger the Release workflow and watch it:
   ```
   gh workflow run Release
   sleep 3
   RUN_ID=$(gh run list --workflow=Release --limit=1 --json databaseId --jq '.[0].databaseId')
   gh run watch "${RUN_ID}"
   ```

3. After the workflow completes, check the result:
   - If successful: show the new release version, URL, and artifacts using `gh release list --limit 1`
   - If the "No release summary" step ran (meaning semantic-release found no releasable commits): inform the user that no new version was created and explain that commits need conventional commit prefixes like `feat:` or `fix:` to trigger a release
   - If failed: show the failing step and suggest `gh run view <RUN_ID> --log-failed` to investigate

4. Report a concise summary to the user with the release URL and version.
