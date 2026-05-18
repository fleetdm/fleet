---
name: pr-approvals
description: Analyze a PR to determine which files still need approval and from whom, based on CODEOWNERS and website/config/custom.js. Use when the user asks "who needs to approve", "why is the PR blocked", "pr approvals", or similar.
---

# PR Approvals

Quickly determine which reviewers are still needed for a PR and which files they own.

## Step 1: Get the blocking info from GitHub

Run these in parallel:
- `gh pr view <number> --json mergeStateStatus,mergeable,reviewDecision` to check if it's actually blocked
- `gh api repos/<owner>/<repo>/pulls/<number> --jq '.requested_reviewers[].login'` to get **who GitHub is still waiting on**
- `gh pr view <number> --json reviews --jq '.reviews[] | select(.state == "APPROVED") | .author.login'` to see who already approved
- `gh pr view <number> --json files --jq '.files[].path'` to get changed files

If the PR is not blocked, just report that and stop.

## Step 2: Map requested reviewers to files

**IMPORTANT: Distinguish between blocking and non-blocking reviewers.**

- **CODEOWNERS** (repo root) creates **required/blocking** reviews. GitHub's branch protection enforces these. Last matching pattern wins.
- **website/config/custom.js** `githubRepoDRIByPath` and `githubRepoMaintainersByPath` auto-request reviewers but do **NOT** block merges. These are courtesy requests.

For each still-requested reviewer, check whether they are required by a CODEOWNERS pattern that matches a changed file. If they only appear in custom.js, they are requested but not blocking.

## Step 3: Report concisely

Output two sections:

**Blocking (CODEOWNERS-required):**
- Who is still needed, and which files/patterns require their approval

**Requested but not blocking (custom.js DRI):**
- Who was auto-requested, and for which files -- note these don't block the merge

Also note who has already approved and what they covered.

Do NOT give a full breakdown of every file's ownership. Only show what's relevant to the still-requested reviewers.
