#!/bin/bash

# Variables
REPO_OWNER="fleetdm"
REPO_NAME="fleet"
FILE_PATH="it-and-security/lib/macos/policies/latest-macos.yml"
BRANCH="main"
NEW_BRANCH="update-macos-version-$(date +%s)"

# Ensure required environment variables are set
if [ -z "$DOGFOOD_AUTOMATION_TOKEN" ] || [ -z "$DOGFOOD_AUTOMATION_USER_NAME" ] || [ -z "$DOGFOOD_AUTOMATION_USER_EMAIL" ]; then
    echo "Error: Missing required environment variables."
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

# Fetch the latest macOS version
latest_macos_version=$(curl -s "https://sofafeed.macadmins.io/v1/macos_data_feed.json" | \
jq -r '.. | objects | select(has("ProductVersion")) | .ProductVersion' | sort -Vr | head -n 1)

if [ -z "$latest_macos_version" ]; then
    echo "Error: Failed to fetch the latest macOS version."
    exit 1
fi

echo "Latest macOS version: $latest_macos_version"

# Compare versions and update the file if needed
if [ "$policy_version_number" != "$latest_macos_version" ]; then
    echo "Updating query line with the new version..."

    # Prepare the new query line
    new_query_line="query: SELECT 1 FROM os_version WHERE version >= '$latest_macos_version';"

    # Update the response
    updated_response=$(echo "$response" | sed "s/query: .*/$new_query_line/")
    if [ -z "$updated_response" ]; then
        echo "Error: Failed to update the query line."
        exit 1
    fi

    # Create a temporary file for the update
    temp_file=$(mktemp)
    echo "$updated_response" > "$temp_file"

    # Configure Git
    git config --global user.name "$DOGFOOD_GIT_USER_NAME"
    git config --global user.email "$DOGFOOD_GIT_USER_EMAIL"

    # Clone the repository and create a new branch
    git clone "https://$DOGFOOD_AUTOMATION_TOKEN@github.com/$REPO_OWNER/$REPO_NAME.git" repo || {
        echo "Error: Failed to clone repository."
        exit 1
    }
    cd repo || exit
    git checkout -b "$NEW_BRANCH"
    cp "$temp_file" "$FILE_PATH"
    git add "$FILE_PATH"
    git commit -m "Update macOS version number to $latest_macos_version"
    git push origin "$NEW_BRANCH"

    # Create a pull request
    pr_data=$(jq -n --arg title "Update macOS version number to $latest_macos_version" \
                 --arg head "$NEW_BRANCH" \
                 --arg base "$BRANCH" \
                 '{title: $title, head: $head, base: $base}')

    pr_response=$(curl -s -H "Authorization: token $DOGFOOD_AUTOMATION_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -X POST \
        -d "$pr_data" \
        "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/pulls")

    if [[ "$pr_response" == *"Validation Failed"* ]]; then
        echo "Error: Failed to create a pull request. Response: $pr_response"
        exit 1
    fi

    echo "Pull request created successfully."
else
    echo "No updates needed; the version is the same."
fi
