---
name: push-reference-docs
description: Move reference doc updates from one release docs branch to another (e.g., 4.89 → 4.90) when a feature is pushed to a later release. Handles three PR states — open (retarget), closed-without-merge (apply-only), merged (revert + apply).
allowed-tools: Bash(git *), Bash(gh pr *), Bash(gh api *), Read, Grep, Glob
effort: medium
---

Move reference doc changes from one release docs branch to another. Use when a feature was documented for release X but is being pushed to release Y — the doc changes need to be reverted from X's docs branch and applied to Y's docs branch.

Arguments: $ARGUMENTS

Usage: `/push-reference-docs <PR_NUMBER> <TARGET_DOCS_BRANCH>`

- `PR_NUMBER` (required): The docs PR number that was (or will be) merged into the source docs branch.
- `TARGET_DOCS_BRANCH` (required): The docs branch for the release the feature is moving to (e.g., `docs-v4.90.0`).

The source docs branch is auto-detected from the PR's base branch.

## Step 1: Fetch and get PR details

1. Fetch upstream. The upstream remote is often SSH (`git@github.com:...`), which can fail silently and leave a stale cached ref — a stale ref causes branches to be based on an old snapshot, producing extra files in the PR diff. Always verify the fetch succeeded:
   ```
   git fetch upstream 2>&1
   ```
   If it fails (e.g. "Permission denied (publickey)"), switch to HTTPS and retry:
   ```
   git remote set-url upstream https://github.com/<UPSTREAM_REPO>.git
   git fetch upstream <TARGET_DOCS_BRANCH>
   ```

2. Get PR details (include `state` to detect closed-without-merge):
   ```
   gh pr view <PR_NUMBER> --json title,baseRefName,headRefName,mergeCommit,commits,url,state
   ```
3. Extract:
   - `SOURCE_DOCS_BRANCH` = the PR's `baseRefName` (e.g., `docs-v4.89.0`)
   - `PR_STATE` = `state` — one of `OPEN`, `MERGED`, `CLOSED`
   - `MERGE_COMMIT` = `mergeCommit.oid` — null if the PR is not yet merged or was closed without merging
   - `ALL_COMMITS` = all commit SHAs in `commits[].oid`, in order (oldest first)
   - `HEAD_COMMIT` = the last commit SHA in `commits[].oid`
   - `PR_TITLE` = the PR title
   - `UPSTREAM_REPO` = the org/repo from the PR URL (e.g., `fleetdm/fleet`):
     ```
     gh pr view <PR_NUMBER> --json url --jq '.url | split("/")[3:5] | join("/")'
     ```
4. Get your GitHub username: `gh api user --jq .login`

## Step 2: Branch based on PR state

There are three cases. Check `PR_STATE` first, then `MERGE_COMMIT`.

### If the PR is OPEN and not yet merged (`PR_STATE == "OPEN"`) — retarget path

The simplest and correct approach: retarget the original PR from the source docs branch to the target docs branch. This avoids creating a revert branch with an empty diff (a git revert against a branch that doesn't have the changes yet is always a no-op).

1. Retarget the original PR. Always use the REST API — `gh pr edit --base` fails on fleetdm/fleet with a GraphQL "Projects (classic)" deprecation error:
   ```
   gh api repos/<UPSTREAM_REPO>/pulls/<PR_NUMBER> --method PATCH --field base=<TARGET_DOCS_BRANCH> --jq '.base.ref'
   ```
2. Report to the user: "PR #N has been retargeted from `<SOURCE_DOCS_BRANCH>` to `<TARGET_DOCS_BRANCH>`. No separate revert or apply PR is needed — the existing PR now targets the correct branch."
3. **Stop here.** Steps 3 and 4 are not needed for the open/unmerged case.

### If the PR is CLOSED without merging (`PR_STATE == "CLOSED"` and `MERGE_COMMIT` is null) — apply-only path

The changes were never applied to the source branch, so no revert is needed. Only create the apply PR.

**Check for an existing apply PR first:** search for any open PR against `<TARGET_DOCS_BRANCH>` that references `<PR_NUMBER>`:
```
gh pr list --repo <UPSTREAM_REPO> --state open --base <TARGET_DOCS_BRANCH> --search "<PR_NUMBER>" --json number,title,url
```
If one exists, **verify its diff before reusing it**:
```
gh pr diff <EXISTING_PR_NUMBER> --stat
```
Compare the file count and line count to the original PR's diff stat (`gh pr diff <PR_NUMBER> --stat`). If they match, retarget or use as-is. If the existing PR has significantly more files or lines, its branch was based on a stale upstream ref — discard it (let the user close it) and create a fresh branch below.

- `WORKING_COMMITS = ALL_COMMITS` (cherry-pick all commits in order, not just HEAD_COMMIT — the first commit usually contains the bulk of the changes)
- Skip Step 3 entirely.
- Proceed to Step 4, cherry-picking all commits in `ALL_COMMITS` order.

Note: You cannot retarget a closed PR via the API — GitHub returns a 422 error. A new PR must be created.

### If the PR IS merged (`PR_STATE == "MERGED"` / `MERGE_COMMIT` is non-null) — revert + apply path

- `WORKING_COMMIT = MERGE_COMMIT`
- Proceed to Steps 3 and 4.

## Step 3: Create the revert PR (from source docs branch)

This PR removes the doc changes from the source release's docs branch.

1. Create the revert branch from the tip of the source docs branch (which already contains the merge commit in its history):
   ```
   git checkout -b <username>/revert-pr<N>-from-<SOURCE_DOCS_BRANCH> upstream/<SOURCE_DOCS_BRANCH>
   ```
2. Revert the merge commit. Check if it has multiple parents:
   ```
   git rev-list --parents -n 1 <WORKING_COMMIT>
   ```
   - Multiple parents → `git revert -m 1 --no-edit <WORKING_COMMIT>`
   - Single parent → `git revert --no-edit <WORKING_COMMIT>`
3. If there are conflicts, stop and tell the user which files conflict.
4. Push: `git push -u origin HEAD`
5. Open the PR:
   ```
   gh pr create --repo <UPSTREAM_REPO> --base <SOURCE_DOCS_BRANCH> \
     --title "Revert \"<PR_TITLE>\" from <SOURCE_DOCS_BRANCH>" \
     --body "$(cat <<'EOF'
   Reverts #<PR_NUMBER> from `<SOURCE_DOCS_BRANCH>`. Feature is moving to `<TARGET_DOCS_BRANCH>`.

   **Related:** #<PR_NUMBER>
   EOF
   )"
   ```

## Step 4: Create the apply PR (to target docs branch)

This PR adds the doc changes to the new release's docs branch.

1. Create a branch from the target docs branch:
   ```
   git checkout -b <username>/pr<N>-docs-to-<TARGET_DOCS_BRANCH> upstream/<TARGET_DOCS_BRANCH>
   ```
2. Cherry-pick commits:
   - **CLOSED path (multiple commits):** cherry-pick all commits in `ALL_COMMITS` order:
     ```
     git cherry-pick <sha1> <sha2> <sha3> ...
     ```
   - **MERGED path (merge commit):** check parent count first:
     - Multiple parents → `git cherry-pick -m 1 <WORKING_COMMIT>`
     - Single parent → `git cherry-pick <WORKING_COMMIT>`
3. If there are conflicts, resolve them manually — the target branch may have received commits since the cherry-picked commit was authored. Keep all content: the new additions from the cherry-pick plus any new sections added by later commits on the target branch. After resolving: `git add <file> && git cherry-pick --continue --no-edit`.
4. **Verify the diff before pushing.** Run `git diff upstream/<TARGET_DOCS_BRANCH>...HEAD --stat` and confirm the file count and line count match the original PR's diff stat. If they don't, something went wrong with the cherry-pick or the upstream ref is stale.
5. Push: `git push -u origin HEAD`
   - If you previously pushed this branch with a different base (e.g., after correcting a stale upstream ref), force-push: `git push --force origin HEAD`
6. Open the PR:
   ```
   gh pr create --repo <UPSTREAM_REPO> --base <TARGET_DOCS_BRANCH> \
     --title "<PR_TITLE>" \
     --body "$(cat <<'EOF'
   Moves reference doc changes from #<PR_NUMBER> to `<TARGET_DOCS_BRANCH>`.

   Originally documented for `<SOURCE_DOCS_BRANCH>` — feature pushed to this release.

   **Related:** #<PR_NUMBER>
   EOF
   )"
   ```

## Step 5: Report to user

- **Open/unmerged path**: report that the original PR was retargeted and include its URL.
- **Closed-without-merge path**: report the apply PR URL. Note that no revert was needed since the changes were never merged.
- **Merged path**: report the revert PR URL and the apply PR URL.
