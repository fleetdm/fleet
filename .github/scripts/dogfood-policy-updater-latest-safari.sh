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

# Extract the query section (may be multi-line)
# Handle indented query: (starts with spaces followed by "query:")
# The query is a YAML multiline string that continues until the next key at the same indentation level (2 spaces)
query_section=$(echo "$response" | awk '/^[[:space:]]*query:/{flag=1} flag && /^  [a-zA-Z_-]+:/ && !/^[[:space:]]*query:/{flag=0} flag')

if [ -z "$query_section" ]; then
    echo "Error: Could not find the query section in the file."
    exit 1
fi

# Extract Safari 18 and Safari 26 version numbers from the query
# Safari 18 is for macOS 15.x, Safari 26 is for macOS 26.x
# The query uses "version LIKE '15.%'" and "version LIKE '26.%'"
policy_safari_18_version=$(echo "$query_section" | grep -A 5 "version LIKE '15\.%" | grep "version_compare" | grep -oE "'[0-9]+\.[0-9]+(\.[0-9]+)?'" | sed "s/'//g" | head -n 1)
policy_safari_26_version=$(echo "$query_section" | grep -A 5 "version LIKE '26\.%" | grep "version_compare" | grep -oE "'[0-9]+\.[0-9]+(\.[0-9]+)?'" | sed "s/'//g" | head -n 1)

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
    echo "Updating policy query with new Safari versions..."

    # Replace the query section in the response
    # Build the new query inline in awk to avoid multi-line variable issues
    # Handle indented query: (starts with spaces followed by "query:")
    # The query is a YAML multiline string that continues until the next key at the same indentation level
    updated_response=$(echo "$response" | awk -v safari_26="$safari_26_version" -v safari_18="$safari_18_version" '
        BEGIN {
            in_query = 0
            query_started = 0
        }
        /^[[:space:]]*query:/ {
            query_started = 1
            in_query = 1
            # Print the new query section with updated versions
            print "  query: |"
            print "    SELECT 1 WHERE "
            print "      NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '\''com.apple.Safari'\'')"
            print "      OR ("
            print "        EXISTS (SELECT 1 FROM os_version WHERE version LIKE '\''26.%'\'')"
            print "        AND EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '\''com.apple.Safari'\'' AND version_compare(bundle_short_version, '\''" safari_26 "'\'' ) >= 0)"
            print "      )"
            print "      OR ("
            print "        EXISTS (SELECT 1 FROM os_version WHERE version LIKE '\''15.%'\'')"
            print "        AND EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '\''com.apple.Safari'\'' AND version_compare(bundle_short_version, '\''" safari_18 "'\'' ) >= 0)"
            print "      );"
            next
        }
        # After query started, skip lines until we find the next key at the same indentation level (2 spaces)
        query_started && /^  [a-zA-Z_-]+:/ {
            in_query = 0
            query_started = 0
            print
            next
        }
        # Skip lines that are part of the query block (indented content)
        query_started {
            next
        }
        # Print all other lines
        {
            print
        }
    ')
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
    commit_message="Update Safari versions: Safari 18 (macOS 15) to $safari_18_version, Safari 26 (macOS 26) to $safari_26_version"
    if [ "$policy_safari_18_version" != "$safari_18_version" ]; then
        commit_message="$commit_message

- Updated Safari 18 version from $policy_safari_18_version to $safari_18_version"
    fi
    if [ "$policy_safari_26_version" != "$safari_26_version" ]; then
        commit_message="$commit_message
- Updated Safari 26 version from $policy_safari_26_version to $safari_26_version"
    fi
    
    git commit -m "$commit_message"
    git push origin "$NEW_BRANCH"

    # Create a pull request
    pr_title="Update Safari versions: Safari 18 to $safari_18_version, Safari 26 to $safari_26_version"
    pr_data=$(jq -n --arg title "$pr_title" \
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
    echo "No updates needed; Safari versions are current."
fi

