#!/bin/bash

# Check if the correct number of arguments are provided
if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <commit-sha> <target-branch>"
  exit 1
fi

# assign input arguments to variables
COMMIT_SHA=$1
TARGET_BRANCH=$2
NEW_BRANCH="backport-${COMMIT_SHA}"
REPOSITORY="https://api.github.com/repos/fleetdm/fleet"

# get_github_token tries to get a token using the gh CLI
get_github_token_from_cli() {
  if ! command -v gh &> /dev/null
  then
    echo "gh CLI could not be found, please install it or provide a GitHub token."
    exit 1
  fi

  GH_TOKEN=$(gh auth token)
  if [ -z "$GH_TOKEN" ]; then
    echo "Failed to retrieve GitHub token using gh CLI."
    exit 1
  fi
  echo $GH_TOKEN
}

# check if the GITHUB_TOKEN environment variable is set
if [ -z "$GITHUB_TOKEN" ]; then
  echo "GitHub token not provided, attempting to retrieve it using gh CLI..."
  GITHUB_TOKEN=$(get_github_token_from_cli)
fi

# check if the GitHub token is still empty
if [ -z "$GITHUB_TOKEN" ]; then
  echo "Error: GitHub token is still empty. Exiting."
  exit 1
fi

# grab the SHA of the RC branch
BASE_SHA=$(curl -s -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" $REPOSITORY/git/refs/heads/$TARGET_BRANCH | jq -r .object.sha)

# create a new branch using the RC as the base branch
curl -s -X POST -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" $REPOSITORY/git/refs \
  -d "{\"ref\": \"refs/heads/${NEW_BRANCH}\", \"sha\": \"${BASE_SHA}\"}" > /dev/null 2>&1

# create a new commit with the contents of the original commit
COMMIT_INFO=$(curl -s -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" $REPOSITORY/commits/${COMMIT_SHA})
TREE_SHA=$(echo $COMMIT_INFO | jq -r .commit.tree.sha)
COMMIT_TITLE=$(echo $COMMIT_INFO | jq -r .commit.message)
NEW_COMMIT_SHA=$(curl -s -X POST -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" $REPOSITORY/git/commits \
  -d "{\"message\": \"Backport: ${COMMIT_TITLE}\", \"tree\": \"${TREE_SHA}\", \"parents\": [\"${BASE_SHA}\"]}" | jq -r .sha)

# associate that commit with the branch
curl -s -X PATCH -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" $REPOSITORY/git/refs/heads/${NEW_BRANCH} \
  -d "{\"sha\": \"${NEW_COMMIT_SHA}\", \"force\": true}" > /dev/null 2>&1

#  create a PR
PR_TITLE="backport: ${COMMIT_TITLE}"
PR_BODY="This is a backport of commit ${COMMIT_SHA} to ${TARGET_BRANCH}."

PR_URL=$(curl -s -X POST -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" $REPOSITORY/pulls \
  -d "{\"title\":\"${PR_TITLE}\", \"body\":\"${PR_BODY}\", \"head\":\"${NEW_BRANCH}\", \"base\":\"${TARGET_BRANCH}\"}" | jq .html_url)

echo "Successfully created backport branch and pull request: $PR_URL"

exit 0
