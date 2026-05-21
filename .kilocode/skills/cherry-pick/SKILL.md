---
name: cherry-pick
description: Cherry-pick a merged PR onto a release candidate branch and open a new PR. Use when asked to cherry-pick, backport, or port a PR to an rc-minor or rc-patch branch.
---

# Cherry-pick – kilocode skill

## Important: single session only

**Use only a single agent session for the entire cherry-pick.** Multiple sessions for the same cherry-pick have caused duplicate PRs in the past.

## Arguments

This skill expects two arguments:

1. **Target branch** — the release candidate branch (e.g., `rc-minor-fleet-v4.83.0`, `rc-patch-fleet-v4.82.1`)
2. **Source PR** — a GitHub PR URL or number from the `fleetdm/fleet` repo

## Steps

1. **Fetch the latest remote state**

   ```bash
   git fetch origin
   ```

2. **Identify the merge commit** — find the merge commit SHA for the source PR on `main`.

   ```bash
   gh pr view <PR> --json mergeCommit --jq '.mergeCommit.oid'
   ```

3. **Create a working branch** from the target release branch:

   ```bash
   git checkout -b cherry-pick-<PR_NUMBER>-to-<target-branch> origin/<target-branch>
   ```

4. **Cherry-pick the merge commit** using `-m 1` (mainline parent):

   ```bash
   git cherry-pick -m 1 <merge-commit-sha>
   ```

   - If there are conflicts, resolve them and continue the cherry-pick.
   - If the PR was a squash-merge (single commit, no merge commit), omit `-m 1`.

5. **Push and open a PR** against the target branch:

   ```bash
   git push -u origin HEAD
   gh pr create \
     --base <target-branch> \
     --title "Cherry-pick #<PR_NUMBER> onto <target-branch>" \
     --body "Cherry-pick of https://github.com/fleetdm/fleet/pull/<PR_NUMBER> onto the <target-branch> release branch."
   ```

## Commit message format

Follow the established pattern:

```
Cherry-pick #<PR_NUMBER> onto <target-branch>

Cherry-pick of https://github.com/fleetdm/fleet/pull/<PR_NUMBER> onto the
<target-branch> release branch.
```

If the original commit has a `Co-authored-by` trailer, preserve it.

## Branch naming

```
cherry-pick-<PR_NUMBER>-to-<target-branch>
```

Example: `cherry-pick-41914-to-rc-minor-fleet-v4.83.0`

**Never create a branch whose name matches a protected pattern.** See
[`.kilocode/rules/protected-branches.md`](../../rules/protected-branches.md).
The cherry-pick branch name above is safe because it starts with `cherry-pick-`,
which does not match any protected pattern. Do **not** drop the `cherry-pick-`
prefix or rename the working branch to something starting with `feature-`,
`patch-`, `minor-`, `fleet-v`, `rc-patch-`, or `rc-minor-`.

## Common issues

- **Duplicate PRs** — never run multiple agent sessions for the same cherry-pick.
- **Conflict on cherry-pick** — resolve conflicts manually, then `git cherry-pick --continue`.
- **Migration timestamp ordering** — if the cherry-picked PR includes migrations, verify timestamps are in chronological order on the target branch.

## References

- Release process: https://github.com/fleetdm/fleet/blob/main/docs/Contributing/workflows/releasing-fleet.md
- Backport checker: `tools/release/backport-check.sh`

---

*This file will grow as new patterns and constraints are established.*
