# Committing changes
- [External contributors](#external-contributors)
- [Fleet Device Management team members](#fleet-device-management-team-members)
- [Pull requests](#pull-requests)
  - [Merging pull requests](#merging-pull-requests)
  - [Commit messages](#commit-messages)

## External contributors

Fleet does not require a CLA for external contributions. External contributors are encouraged to submit Pull Requests (PRs) following the instructions presented in this document.

For significant changes, it is good to discuss the proposal with the Fleet team in an Issue or in #fleet on [osquery Slack](https://join.slack.com/t/osquery/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw) before commencing development. This helps make sure that your PR will be merged.

Please keep in mind that any code merged to the Fleet repository becomes the responsibility of the Fleet team to maintain. Because of this, we are careful to make sure any contributions fit Fleet's vision, are well-tested, and are of high quality. We will work with contributors to make sure we meet the appropriate standards.

## Fleet Device Management team members
Fleet Device Management team members may not copy queries from external sources except when that content has an explicit license allowing such use or permission has been granted by the creator.

## Pull requests

Each developer (internal or external) creates a fork of the Fleet repository, committing changes to a branch within their fork. Developers submit changes by PR to be merged into Fleet.

GitHub Actions automatically runs testers and linters on each PR. Please make sure that these checks pass. Checks can be run locally as described in [Testing.md](https://fleetdm.com/docs/contributing/testing-and-local-development).

Mark the Pull Request as a "Draft" for the features still in progress. This helps make it clear which PRs are ready for review and merge.

We ask internal contributors and reviewers to apply the appropriate Labels for PRs. This helps with project management.

PRs that address Issues should include a message indicating that they fix or close the Issue (e.g., `Fixes #42`). GitHub uses this to close the associated Issue automatically when the PR is merged.

### Changes files

#### Goal

As projects move forward and bug fixes and features are added, we want to make sure to track changes in a readable and easy-to-find, way (besides git). For that, we've got the CHANGELOG.md file.

There are two ways to write CHANGELOG files:

1. Having an individual responsible for writing the changes as part of the release process
2. Writing the changelog collaboratively and whoever creates the release just collects the text and organizes the information rather than generating it

Fleet is doing 2, using the concept of changes files.

#### What is it?

A changes file is a file that contains one or more CHANGELOG entries and corresponds roughly to one PR.

The easiest way to see how this works is with an example: This PR https://github.com/fleetdm/fleet/pull/1305 addresses the following issue: https://github.com/fleetdm/fleet/issues/1009

As such, it has one changes file: https://github.com/fleetdm/fleet/pull/1305/files#diff-4f5bba9549628a2b7f0460511a26776e4eaff69f0ddd0c6ee9fa18ee35cc685e

The naming of the file is only important mostly for the uniqueness of the file (to prevent merge conflicts) but also to quickly be able to see what's unreleased at any given time.

This PR also happens to be the one adding the changes directory for the first time, so it contains this file: https://github.com/fleetdm/fleet/pull/1305/files#diff-4eb30cabf796178e0a335a797b0d90bac3d393523eebdbe8f1be37ded949039f, which should be ignored and left there to prevent needing to create the directory after every release.

As part of the release process, whoever is cutting the release will fold in the different changes into the CHANGELOG and then remove them.

#### How to write a changes file

As shown in the example above, the exact contents of the file should follow as much as possible the format that the entry will have in the CHANGELOG file. So the job of the person tagging the release is just copy and paste.

All grammar checks and corrections should happen as part of the PR review.

#### What does not need a changes file?

Not everything needs a changes file, though. The easiest way to differentiate is to ask yourself "Will the CHANGELOG need to reflect the work I'm doing?"

Usually, if it's a bug fix or a new feature, it needs a changes file, but there are exceptions. Here's the incomplete list of them:

- The PR fixes a bug in a previously unreleased change (so there's already a changes file).
- It's an update to the documentation or other supporting material (such as the PR that's adding this text).
- A feature or bug fix was worked on by two separate people (e.g., there's a backend and a frontend component to it); the first person merging a PR will add the changes file in this case. The second won't.

#### When do I add more than one entry to the changes file?

This is very dependent on the case. It'll be very unlikely, but sometimes a PR has a "side effect" that needs to be reflected. For instance, maybe as part of adding a new feature you found and fixed a bug that is tightly coupled to the feature. Arguably, you should've created a separate PR, but life is not that simple sometimes.

#### Why not just add it directly to the CHANGELOG as unreleased?

The reason we are adding one file per change, roughly, is to prevent merging conflicts. If everybody working on Fleet would edit the CHANGELOG file, every PR will have a conflict as soon as one is merged, and collaboration will be very complicated.

### Merging pull requests

In general, PRs should pass all CI checks and have at least one approving review before merge.

Failing CI checks can be allowed if the failure is clearly unrelated to the changes in the PR. Please leave a comment indicating this before merging.

For simple changes in which the internal author is confident, it can be appropriate to merge without an approving review.

In general, we try to allow internal contributors to merge their own PRs after approval. This allows for the author to make any final modifications and edit their own commit message.

For external contributors, the merge must be performed by a teammate with merge permissions. Typically this would be the internal reviewer that approves the PR.

### Commit messages

GitHub is configured only to allow "Squash Merges." meaning each PR (potentially containing multiple commits) becomes a single commit for merge. Occasionally it may be appropriate to "Rebase Merge," a complex PR that is best left as multiple commits. Please discuss within the PR if this seems appropriate.

GitHub will automatically generate a commit title and description based on the commits within the PR. This is often messy and it is good practice to clean up the generated text. Typically, using the PR title and description is a good way to approach this.

Keep in mind that the commit title and description are what developers see when running `git log` locally. Try to make this information helpful!

Keeping to around 80 character line lengths helps with rendering when folks have narrow, tiled terminal windows.

<meta name="pageOrderInSection" value="400">
<meta name="description" value="A guide to contributing to the Fleet GitHub repository.">
