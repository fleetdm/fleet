# Committing Changes
- [External Contributors](#external-contributors)
- [Pull Requests](#pull-requests)
  - [Merging Pull Requests](#merging-pull-requests)
  - [Commit Messages](#commit-messages)

## External Contributors

Fleet does not require a CLA for external contributions. External contributors are encouraged to submit Pull Requests (PRs) following the instructions presented in this document.

For significant changes, it is a good idea to discuss the proposal with the Fleet team in an Issue or in #fleet on [osquery Slack](https://join.slack.com/t/osquery/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw) before commencing development. This helps ensure that your PR will be merged.

Please keep in mind that any code merged to the Fleet repository becomes the responsibility of the Fleet team to maintain. Because of this, we are careful to ensure any contributions fit Fleet's vision, are well-tested, and high quality. We will work with contributors to ensure the appropriate standards are met.

## Pull Requests

Each developer (internal or external) creates a fork of the Fleet repository, committing changes to a branch within their fork. Changes are submitted by PR to be merged into Fleet.

GitHub Actions automatically runs testers and linters on each PR. Please ensure that these checks pass. Checks can be run locally as described in [2-Testing.md](./2-Testing.md).

For features that are still in-progress, the Pull Request can be marked as a "Draft". This helps make it clear which PRs are ready for review and merge.

Internal contributors and reviewers are asked to apply the appropriate Labels for PRs. This helps with project management.

PRs that address Issues should include a message indicating that they fix or close the Issue (eg. `Fixes #42`). GitHub uses this to automatically close the associated Issue when the PR is merged.

### Merging Pull Requests

In general, PRs should pass all CI checks and have at least one approving review before merge.

Failing CI checks can be allowed if the failure is clearly unrelated to the changes in the PR. Please leave a comment indicating this before merging.

For simple changes in which the internal author is confident, it can be appropriate to merge without an approving review.

In general, we try to allow internal contributors to merge their own PRs after approval. This gives the opportunity for the author to make any final modifications and edit their own commit message.

For external contributors, the merge must be performed by a teammate with merge permissions. Typically this would be the internal reviewer that approves the PR.

### Commit Messages

GitHub is configured to only allow "Squash Merges", meaning each PR (potentially containing multiple commits) becomes a single commit for merge. Occasionally it may be appropriate to "Rebase Merge" a complex PR that is best left as multiple commits. Please discuss within the PR if this seems appropriate.

GitHub will automatically generate a commit title and description based on the commits within the PR. This is often messy and it is good practice to clean up the generated text. Typically, using the PR title and description is a good way to approach this.

Keep in mind that the commit title and description are what developers see when running `git log` locally. Try to make this information helpful!

Keeping to around 80 character line lengths helps with rendering when folks have narrow, tiled terminal windows.
