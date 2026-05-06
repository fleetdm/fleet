---
name: bug-hunt
description: Find a locally-reproducible bug from the project board, reproduce it, fix it, test it, and open a draft PR. Use when the user says "bug hunt", "find a bug", or "let's fix a bug".
---

# Bug Hunt

End-to-end workflow: find a bug from the project board, reproduce it locally, fix it, and PR it.

## Step 1: Find reproducible bugs

- Pull bug tickets from the Fleet project board: https://github.com/orgs/fleetdm/projects/67/views/19
- Filter to the **Estimated** column (unless the user says otherwise).
- Read each bug issue and assess: **can this be reproduced locally on this Mac with a local Fleet server?**
- Exclude bugs that need multi-host setups, specific device enrollments (DEP/ADE), manual DB state manipulation, or intermittent timing over hours/days.
- Present the reproducible bugs sorted **oldest first**, with a one-line summary each.

## Step 2: Pick one

- Suggest the **oldest** bug as the default.
- Ask the user which one to pick. Wait for confirmation before proceeding.

## Step 3: Reproduce the bug

- Read the issue thoroughly. Understand the root cause by reading the relevant code.
- Set up whatever is needed locally (GitOps YAML, profiles, config files, etc.).
- **Actually run the reproduction** against a local Fleet server — don't just read the code.
- If the server isn't running, try to start it or ask the user to help.
- Capture the **before** output showing the bug in action.

## Step 4: Fix the bug

- Create a branch: `fix-<issue-number>-short-description`.
- Make the minimal code change needed.
- Build and verify it compiles.
- Re-run the reproduction with the fix applied. Capture the **after** output.

## Step 5: Test

- **Manual testing**: Run the actual command/UI flow end-to-end, both before (unfixed) and after (fixed). Show real terminal output as evidence.
- **Unit tests**: Write focused tests that exercise the fix directly. Verify they **fail on unfixed code** and **pass on the fix** (git stash round-trip).
- Run linters (`make lint-go-incremental`).

## Step 6: Draft PR

**Ask the user before creating the PR.** When they say yes, create a **draft PR** structured as:

### PR body template

```
Closes #<issue>

## Local reproduction
- Describe the setup (what files were created, what config)
- The exact commands run
- The buggy output observed (before fix)

---

## Code changes
- Summary: what changed overall (N files, N lines)
- Per-file walkthrough (exclude test files):
  - What changed and why, with diff snippets

- Before/after comparison of the user-visible output

---

## Testing

### Manual testing
- Numbered steps of what was done
- Before (unfixed) output
- After (fixed) output

### Unit tests added
- New test file(s) and what each test covers
- Confirmation they fail on unfixed code, pass on fixed code
```
