#!/bin/bash

# Variables
REPO_OWNER="fleetdm"
REPO_NAME="fleet"
FILE_PATH="it-and-security/lib/macos/policies/update-safari.yml"
BRANCH="main"

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

# Fetch the latest Safari version from SOFA feed
echo "Fetching latest Safari version from SOFA feed..."
safari_feed_response=$(curl -s "https://sofafeed.macadmins.io/v2/safari_data_feed.json" 2>/dev/null)

if [ -z "$safari_feed_response" ] || [[ "$safari_feed_response" == *"404"* ]] || [[ "$safari_feed_response" == *"Not Found"* ]]; then
    echo "Error: Failed to fetch Safari feed from SOFA."
    exit 1
fi

# Parse Safari feed - get the latest version from all AppVersions
# The feed structure has AppVersions array, each with a Latest.ProductVersion
# We prioritize Safari 18 (most common for macOS 15 Sequoia), then get the highest version overall
# First try to get Safari 18 latest version
safari_18_version=$(echo "$safari_feed_response" | jq -r '.AppVersions[] | select(.AppVersion == "Safari 18") | .Latest.ProductVersion' 2>/dev/null | head -n 1)

# If Safari 18 exists, use it; otherwise get the highest version overall
if [ -n "$safari_18_version" ] && [ "$safari_18_version" != "null" ]; then
    latest_safari_version="$safari_18_version"
else
    # Fallback: get the highest version across all AppVersions
    latest_safari_version=$(echo "$safari_feed_response" | jq -r '.AppVersions[] | .Latest.ProductVersion' 2>/dev/null | sort -Vr | head -n 1)
fi

if [ -z "$latest_safari_version" ] || [ "$latest_safari_version" == "null" ]; then
    echo "Error: Failed to parse Safari version from SOFA feed."
    exit 1
fi

# Clean up version string (remove any extra whitespace or characters)
latest_safari_version=$(echo "$latest_safari_version" | xargs)

if [ -z "$latest_safari_version" ]; then
    echo "Error: Failed to fetch the latest Safari version."
    exit 1
fi

echo "Latest Safari version: $latest_safari_version"

# Compare versions and update the file if needed
if [ "$policy_version_number" != "$latest_safari_version" ]; then
    echo "Updating query line with the new version..."

    # Prepare the new query line
    new_query_line="query: SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.apple.Safari') OR EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.apple.Safari' AND version_compare(bundle_short_version, '$latest_safari_version') >= 0);"

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
    # Generate branch name with timestamp right before use
    NEW_BRANCH="update-safari-version-$(date +%s)"
    git checkout -b "$NEW_BRANCH"
    cp "$temp_file" "$FILE_PATH"
    git add "$FILE_PATH"
    git commit -m "Update Safari version number to $latest_safari_version"
    git push origin "$NEW_BRANCH"

    # Create a pull request
    pr_data=$(jq -n --arg title "Update Safari version number to $latest_safari_version" \
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
        --arg r2 "tux234" \
        '{reviewers: [$r1, $r2]}')

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

