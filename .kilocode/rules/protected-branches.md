# Protected branches — never create these

Kilo Code MUST NOT create, push, or check out a new branch whose name matches any
of the protected patterns below. These patterns are enforced by GitHub branch
protection rules on `fleetdm/fleet`. Pushing a new branch matching one of them
will be rejected by the remote and may also accidentally apply protection rules
to an unintended branch.

## Protected patterns

The following names (and any name matching the glob) are protected:

- `main`
- `patch[_-]*`     (e.g. `patch_foo`, `patch-foo`)
- `feature[_-]*`   (e.g. `feature_foo`, `feature-foo`)
- `fleet-v*`       (e.g. `fleet-v4.83.0`)
- `minor[_-]*`     (e.g. `minor_foo`, `minor-foo`)
- `rc-patch[_-]*`  (e.g. `rc-patch-fleet-v4.82.1`)
- `rc-minor[_-]*`  (e.g. `rc-minor-fleet-v4.83.0`)

> Note: `[_-]` matches a literal underscore or hyphen. Anything starting with
> `feature-`, `feature_`, `patch-`, `patch_`, `minor-`, `minor_`, `rc-patch-`,
> `rc-patch_`, `rc-minor-`, `rc-minor_`, or `fleet-v` is protected.

## Rules

The goal is to prevent **accidental** creation of branches that fall under
branch protection. Once a branch matches a protected pattern, you can't push
to it directly — every change has to go through a PR — which is almost never
what you want for a routine working branch (e.g. a GitOps update, a fix, a
cherry-pick).

1. Do not derive a branch name directly from a topic, file, or PR title when
   that name would match a protected pattern. For example, a GitOps change
   for `patch-adobe-acrobat-reader.yml` must NOT become a branch called
   `patch-adobe-acrobat-reader-all-devices`, because that matches `patch[_-]*`.
2. **Always** prefix new working branches with a non-protected identifier
   (typically the user's GitHub login, e.g. `<github-username>/<short-desc>`,
   or a task-specific prefix like `cherry-pick-<PR>-to-<target>`).
3. Before running `git checkout -b`, `git branch`, or `git push -u origin`,
   validate the proposed branch name against the patterns above. If it
   matches, pick a safe name instead.
4. If the user **explicitly** asks for a branch name that matches a protected
   pattern (e.g. cutting an actual release branch), proceed — but first warn
   that the branch will be protected and direct pushes will be rejected, then
   confirm.

## Safe naming examples

- `<github-username>/<short-description>`
- `<github-username>/<short-description>-cp`           (cherry-pick)
- `cherry-pick-<PR_NUMBER>-to-<target-branch>`
- `fix-<short-description>`
- `chore-<short-description>`

## Unsafe naming examples (do not create)

- `feature-add-search`            ← matches `feature[_-]*`
- `feature_add_search`            ← matches `feature[_-]*`
- `patch-fix-bug`                 ← matches `patch[_-]*`
- `minor-update-deps`             ← matches `minor[_-]*`
- `fleet-v4.83.0`                 ← matches `fleet-v*`
- `rc-minor-fleet-v4.84.0`        ← matches `rc-minor[_-]*`
- `rc-patch-fleet-v4.82.2`        ← matches `rc-patch[_-]*`
