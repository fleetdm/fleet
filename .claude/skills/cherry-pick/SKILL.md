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

If an RC branch was provided as the second argument, use it. Otherwise, run `git branch -a | grep 'rc-minor-fleet-v'` and pick the one with the highest version number.

## Step 3: Get the merge commit

Use `gh pr view <PR_NUMBER> --json mergeCommit,title,number` to get the merge commit SHA and PR title.

If the PR is not yet merged, tell the user and stop.

## Step 4: Cherry-pick onto a new branch

1. `git fetch origin`
2. Create a new branch off the RC branch:
   ```
   git checkout -b nulmete/<short-description>-cp origin/<rc-branch>
   ```
   Derive `<short-description>` from the PR title (lowercase, hyphens, keep it short — 3-5 words max).
3. Run `git cherry-pick -m 1 <merge-commit-SHA>` (use `-m 1` since these are merge commits).
4. If there are conflicts, stop and tell the user which files conflict. Do NOT attempt to resolve them automatically.

## Step 5: Push and open PR

1. Push the branch: `git push -u origin nulmete/<branch-name>`
2. Open a PR targeting the RC branch (NOT main):
   ```
   gh pr create --base <rc-branch> --title "Cherry-pick #<PR_NUMBER>: <original-title>" --body "$(cat <<'EOF'
   Cherry-pick of #<PR_NUMBER> into the RC branch.
   EOF
   )"
   ```
3. Report the PR URL to the user.
