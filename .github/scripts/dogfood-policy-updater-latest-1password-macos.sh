#!/bin/bash

# Variables
REPO_OWNER="fleetdm"
REPO_NAME="fleet"
FILE_PATH="it-and-security/lib/macos/policies/update-1password.yml"
BRANCH="main"
NEW_BRANCH="update-1password-macos-version-$(date +%s)"

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

    # Create a temporary file for the update
    temp_file=$(mktemp)
    echo "$updated_response" > "$temp_file"

    # Configure Git
    git config --global user.name "$DOGFOOD_AUTOMATION_USER_NAME"
    git config --global user.email "$DOGFOOD_AUTOMATION_USER_EMAIL"

    # Clone the repository and create a new branch
    git clone "https://$DOGFOOD_AUTOMATION_TOKEN@github.com/$REPO_OWNER/$REPO_NAME.git" repo || {
        echo "Error: Failed to clone repository."
        exit 1
    }
    cd repo || exit
    git checkout -b "$NEW_BRANCH"
    cp "$temp_file" "$FILE_PATH"
    git add "$FILE_PATH"
    git commit -m "Update 1Password macOS version number to $latest_1password_macos_version"
    git push origin "$NEW_BRANCH"

    # Create a pull request
    pr_data=$(jq -n --arg title "Update 1Password macOS version number to $latest_1password_macos_version" \
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

    # Extract the pull request number from the response
    pr_number=$(echo "$pr_response" | jq -r '.number')
    if [ -z "$pr_number" ] || [ "$pr_number" == "null" ]; then
        echo "Error: Failed to retrieve pull request number."
        exit 1
    fi

    echo "Adding reviewers to PR #$pr_number..."

    # Prepare the reviewers data payload
    reviewers_data=$(jq -n \
        --arg r1 "harrisonravazzolo" \
        --arg r2 "noahtalerman" \
        --arg r3 "lukeheath" \
        --arg r4 "nonpunctual" \
        --arg r5 "ddribeiro" \
        '{reviewers: [$r1, $r2, $r3, $r4, $r5]}')

    # Request reviewers for the pull request
    review_response=$(curl -s -X POST \
        -H "Authorization: token $DOGFOOD_AUTOMATION_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -d "$reviewers_data" \
        "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/pulls/$pr_number/requested_reviewers")

    if echo "$review_response" | grep -q "errors"; then
        echo "Error: Failed to add reviewers. Response: $review_response"
        exit 1
    fi
    echo "Reviewers added successfully."
else
    echo "No updates needed; the version is the same."
fi
