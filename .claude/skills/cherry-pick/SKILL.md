---
name: cherry-pick
description: Cherry-pick a merged PR into the current RC branch. Use when asked to "cherry-pick", "cp into RC", or after merging a PR that needs to go into the current release.
allowed-tools: Bash(git *), Bash(gh pr *), Bash(gh api *), Read, Grep, Glob
effort: low
---

Cherry-pick a merged PR into the current RC branch. Arguments: $ARGUMENTS

Usage: `/cherry-pick <PR_NUMBER> [RC_BRANCH]`

- `PR_NUMBER` (required): The PR number to cherry-pick (e.g. `43078`). If not provided, ask the user.
- `RC_BRANCH` (optional): The target RC branch name (e.g. `rc-minor-fleet-v4.83.0`). If not provided, auto-detect the most recent one.

## Step 1: Ensure main is up to date

1. `git fetch origin`
2. `git checkout main`
3. `git pull origin main`

## Step 2: Identify the RC branch

If an RC branch was provided as the second argument, use it (but still confirm with the user before proceeding).

Otherwise, auto-detect by listing both minor and patch RC branches:

```
git for-each-ref 'refs/remotes/origin/rc-minor-fleet-v*' 'refs/remotes/origin/rc-patch-fleet-v*' --format='%(refname:strip=3)' | grep -E '^rc-(minor|patch)-fleet-v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V
```

From the results, suggest the most recent `rc-minor-fleet-v*` branch as the default. If patch branches also exist, mention them as alternatives. **Always ask the user to confirm the target RC branch before proceeding.**

## Step 3: Get the merge commit and GitHub username

1. Get the PR title:
   ```
   gh pr view <PR_NUMBER> --json title --jq .title
   ```
2. Get the merge commit SHA:
   ```
   gh pr view <PR_NUMBER> --json mergeCommit --jq .mergeCommit.oid
   ```
   If this returns `null` or an empty value, the PR is not yet merged. Tell the user and stop.
3. Get the GitHub username: `gh api user --jq .login`

## Step 4: Cherry-pick onto a new branch

1. Create a new branch off the RC branch:
   ```
   git checkout -b <github-username>/<short-description>-cp origin/<rc-branch>
   ```
   Derive `<short-description>` from the PR title (lowercase, hyphens, keep it short — 3-5 words max).
2. Check whether the commit is a merge commit by inspecting its parents:
   ```
   git rev-list --parents -n 1 <merge-commit-SHA>
   ```
   If the commit has multiple parents, run:
   ```
   git cherry-pick -m 1 <merge-commit-SHA>
   ```
   Otherwise (squash-merged or rebased), run:
   ```
   git cherry-pick <merge-commit-SHA>
   ```
3. If there are conflicts, stop and tell the user which files conflict. Do NOT attempt to resolve them automatically.

## Step 5: Push and open PR

1. Push the branch: `git push -u origin HEAD`
2. Open a PR targeting the RC branch (NOT main):
   ```
   gh pr create --base <rc-branch> --title "Cherry-pick #<PR_NUMBER>: <original-title>" --body "$(cat <<'EOF'
   Cherry-pick of #<PR_NUMBER> into the RC branch.
   EOF
   )"
   ```
3. Report the PR URL to the user.
