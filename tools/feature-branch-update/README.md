
# Feature update tool

This script is for automating keeping your long lived feature branch that can't use GH 'update branch' button because of merge conflicts.

The usage is `./tools/feature-branch-update/update.sh <PR Number>` This will checkout the branch tied to the PR. Create a temporary branch named $USER-$PRNUMBER-mu (main update) and rebase main. At this point you should have merge conflicts to resolve. Resolve and commit the resolutions then run the script with the PR number again. It will see you are already on the update branch and conclude by opening a PR with the changes targeted back to the branch of the feature.
