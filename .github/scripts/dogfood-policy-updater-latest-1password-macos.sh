#!/bin/bash

# Variables
REPO_OWNER="fleetdm"
REPO_NAME="fleet"
FILE_PATH="it-and-security/lib/macos/policies/update-1password.yml"
BRANCH="main"

# Ensure required environment variables are set
if [ -z "$DOGFOOD_AUTOMATION_TOKEN" ]; then
    echo "Error: Missing required environment variable DOGFOOD_AUTOMATION_TOKEN."
    exit 1
fi

# GitHub API URL
FILE_URL="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/contents/$FILE_PATH?ref=$BRANCH"

# Fetch the file contents from GitHub
response=$(curl -s -H "Authorization: token $DOGFOOD_AUTOMATION_TOKEN" -H "Accept: application/vnd.github.v3.raw" "$FILE_URL")

if [ -z "$response" ] || [[ "$response" == *"Not Found"* ]]; then
    echo "Error: Failed to fetch file or file does not exist in the repository."
    exit 1
fi

# Extract the query line
query_line=$(echo "$response" | grep 'query:')
if [ -z "$query_line" ]; then
    echo "Error: Could not find the query line in the file."
    exit 1
fi

# Extract the version number from the query line
policy_version_number=$(echo "$query_line" | grep -oE "'[0-9]+\.[0-9]+(\.[0-9]+)?'" | sed "s/'//g")
if [ -z "$policy_version_number" ]; then
    echo "Error: Failed to extract the policy version number."
    exit 1
fi

echo "Policy version number: $policy_version_number"

# Fetch the latest 1Password macOS version
latest_1password_macos_version=$(curl -s https://releases.1password.com/mac/index.xml | grep "<title>" | grep -v "beta\|preview\|test" | grep -o "[0-9]\+\.[0-9]\+\.[0-9]\+\(\.[0-9]\+\)*" | sort -t. -k1,1nr -k2,2nr -k3,3nr -k4,4nr | head -1)

if [ -z "$latest_1password_macos_version" ]; then
    echo "Error: Failed to fetch the latest macOS version."
    exit 1
fi

echo "Latest 1Password macOS version: $latest_1password_macos_version"

# Compare versions and update the file if needed
if [ "$policy_version_number" != "$latest_1password_macos_version" ]; then
    echo "Updating query line with the new version..."

    # Prepare the new query line
    new_query_line="query: SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE name = '1Password.app') OR EXISTS (SELECT 1 FROM apps WHERE name = '1Password.app' AND version_compare(bundle_short_version, '$latest_1password_macos_version') >= 0);"

    # Update the response
    updated_response=$(echo "$response" | sed "s/query: .*/$new_query_line/")
    if [ -z "$updated_response" ]; then
        echo "Error: Failed to update the query line."
        exit 1
    fi

    # Write the updated content to the file
    echo "$updated_response" > "$FILE_PATH"
    echo "1Password policy file updated: $FILE_PATH"
    echo "Files updated successfully. PR will be created by GitHub Actions workflow."
else
    echo "No updates needed; the version is the same."
fi
