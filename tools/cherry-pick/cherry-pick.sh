#!/bin/bash

# RC_VERSION
# COMMIT_HASH
# DESCRIPTIVE_BRANCH_NAME

RC_BRANCH=rc-minor-fleet-v${RC_VERSION}
CHERRY_PICK_BRANCH="cherry-pick-$DESCRIPTIVE_BRANCH_NAME-into-$RC_VERSION"

git checkout main
git pull origin main
git checkout "$RC_BRANCH"
git pull origin "$RC_BRANCH"
git checkout -b "$CHERRY_PICK_BRANCH"
git cherry-pick "$COMMIT_HASH"
git push origin "$CHERRY_PICK_BRANCH"