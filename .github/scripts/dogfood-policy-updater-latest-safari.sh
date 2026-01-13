#!/bin/bash

# Variables
REPO_OWNER="fleetdm"
REPO_NAME="fleet"
FILE_PATH="it-and-security/lib/macos/policies/update-safari.yml"
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

# Extract Safari 18 and Safari 26 version numbers from the query line
# Safari 18 is for macOS 15.x, Safari 26 is for macOS 26.x
# The query has two version_compare calls - first is Safari 26, second is Safari 18
policy_safari_26_version=$(echo "$query_line" | grep -oE "version_compare\([^,]+,\s*'[0-9]+\.[0-9]+(\.[0-9]+)?'" | head -1 | grep -oE "'[0-9]+\.[0-9]+(\.[0-9]+)?'" | sed "s/'//g")
policy_safari_18_version=$(echo "$query_line" | grep -oE "version_compare\([^,]+,\s*'[0-9]+\.[0-9]+(\.[0-9]+)?'" | tail -1 | grep -oE "'[0-9]+\.[0-9]+(\.[0-9]+)?'" | sed "s/'//g")

if [ -z "$policy_safari_18_version" ] || [ -z "$policy_safari_26_version" ]; then
    echo "Error: Failed to extract Safari version numbers from policy."
    echo "Safari 18 version found: $policy_safari_18_version"
    echo "Safari 26 version found: $policy_safari_26_version"
    exit 1
fi

echo "Policy Safari 18 version: $policy_safari_18_version"
echo "Policy Safari 26 version: $policy_safari_26_version"

# Fetch the latest Safari version from SOFA feed
echo "Fetching latest Safari version from SOFA feed..."
safari_feed_response=$(curl -s "https://sofafeed.macadmins.io/v2/safari_data_feed.json" 2>/dev/null)
curl_exit_code=$?

# Check if it's valid JSON first
if ! echo "$safari_feed_response" | jq empty 2>/dev/null; then
    echo "Error: Failed to fetch Safari feed from SOFA - invalid JSON response."
    exit 1
fi

# Check for HTTP errors in the JSON (if the API returns error JSON)
if echo "$safari_feed_response" | jq -e '.error' >/dev/null 2>&1; then
    echo "Error: SOFA API returned an error"
    echo "$safari_feed_response" | jq '.error'
    exit 1
fi

if [ $curl_exit_code -ne 0 ] || [ -z "$safari_feed_response" ]; then
    echo "Error: Failed to fetch Safari feed from SOFA."
    exit 1
fi

# Parse Safari feed - get latest versions for Safari 18 (macOS 15) and Safari 26 (macOS 26)
# The feed structure has AppVersions array, each with a Latest.ProductVersion
safari_18_version=$(echo "$safari_feed_response" | jq -r '.AppVersions[] | select(.AppVersion == "Safari 18") | .Latest.ProductVersion' 2>/dev/null | head -n 1)
safari_26_version=$(echo "$safari_feed_response" | jq -r '.AppVersions[] | select(.AppVersion == "Safari 26") | .Latest.ProductVersion' 2>/dev/null | head -n 1)

if [ -z "$safari_18_version" ] || [ "$safari_18_version" == "null" ]; then
    echo "Error: Failed to parse Safari 18 version from SOFA feed."
    exit 1
fi

if [ -z "$safari_26_version" ] || [ "$safari_26_version" == "null" ]; then
    echo "Error: Failed to parse Safari 26 version from SOFA feed."
    exit 1
fi

# Clean up version strings (remove any extra whitespace or characters)
safari_18_version=$(echo "$safari_18_version" | xargs)
safari_26_version=$(echo "$safari_26_version" | xargs)

echo "Safari 18 (macOS 15) latest version: $safari_18_version"
echo "Safari 26 (macOS 26) latest version: $safari_26_version"

# Check if updates are needed
update_needed=false
if [ "$policy_safari_18_version" != "$safari_18_version" ]; then
    echo "Safari 18 version update needed: $policy_safari_18_version -> $safari_18_version"
    update_needed=true
fi

if [ "$policy_safari_26_version" != "$safari_26_version" ]; then
    echo "Safari 26 version update needed: $policy_safari_26_version -> $safari_26_version"
    update_needed=true
fi

# Update the file if needed
if [ "$update_needed" = true ]; then
    echo "Updating query line with new Safari versions..."

    # Prepare the new query line
    new_query_line="query: SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.apple.Safari') OR (EXISTS (SELECT 1 FROM os_version WHERE version LIKE '26.%') AND EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.apple.Safari' AND version_compare(bundle_short_version, '$safari_26_version') >= 0)) OR (EXISTS (SELECT 1 FROM os_version WHERE version LIKE '15.%') AND EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.apple.Safari' AND version_compare(bundle_short_version, '$safari_18_version') >= 0));"

    # Update the response
    updated_response=$(echo "$response" | sed "s/query: .*/$new_query_line/")
    if [ -z "$updated_response" ]; then
        echo "Error: Failed to update the query line."
        exit 1
    fi

    # Write the updated content to the file
    echo "$updated_response" > "$FILE_PATH"
    echo "Safari policy file updated: $FILE_PATH"
    echo "Files updated successfully. PR will be created by GitHub Actions workflow."
else
    echo "No updates needed; Safari versions are current."
fi

