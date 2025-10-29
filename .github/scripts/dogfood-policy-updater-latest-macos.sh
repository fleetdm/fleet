#!/bin/bash

# Variables
REPO_OWNER="fleetdm"
REPO_NAME="fleet"
POLICY_FILE_PATH="it-and-security/lib/macos/policies/latest-macos.yml"
WORKSTATIONS_FILE="it-and-security/teams/workstations.yml"
WORKSTATIONS_CANARY_FILE="it-and-security/teams/workstations-canary.yml"
BRANCH="main"
NEW_BRANCH="update-macos-version-$(date +%s)"

# Ensure required environment variables are set
if [ -z "$DOGFOOD_AUTOMATION_TOKEN" ] || [ -z "$DOGFOOD_AUTOMATION_USER_NAME" ] || [ -z "$DOGFOOD_AUTOMATION_USER_EMAIL" ]; then
    echo "Error: Missing required environment variables."
    exit 1
fi

# Function to calculate 4 Sundays from today
# COMMENTED OUT: Deadline calculation logic temporarily disabled
# calculate_deadline() {
#     # Get current date
#     current_date=$(date +%Y-%m-%d)
#     
#     # Calculate days until next Sunday (0 = Sunday, 1 = Monday, ..., 6 = Saturday)
#     current_day=$(date +%u)  # 1-7 (Monday=1, Sunday=7)
#     days_to_next_sunday=$((7 - current_day))
#     if [ $days_to_next_sunday -eq 0 ]; then
#         days_to_next_sunday=7
#     fi
#     
#     # Calculate 4 Sundays from today
#     days_to_deadline=$((days_to_next_sunday + 21))  # 3 more weeks (21 days)
#     
#     # Calculate the deadline date
#     deadline_date=$(date -d "$current_date + $days_to_deadline days" +%Y-%m-%d)
#     echo "$deadline_date"
# }

# Function to fetch file content from GitHub
fetch_file_content() {
    local file_path="$1"
    local file_url="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/contents/$file_path?ref=$BRANCH"
    
    local response=$(curl -s -H "Authorization: token $DOGFOOD_AUTOMATION_TOKEN" -H "Accept: application/vnd.github.v3.raw" "$file_url")
    
    if [ -z "$response" ] || [[ "$response" == *"Not Found"* ]]; then
        echo "Error: Failed to fetch file $file_path or file does not exist in the repository."
        return 1
    fi
    
    echo "$response"
}

# Function to extract current minimum version from team file content
extract_minimum_version() {
    local content="$1"
    local minimum_version=$(echo "$content" | grep -A 5 "macos_updates:" | grep "minimum_version:" | sed 's/.*minimum_version: *"\([^"]*\)".*/\1/')
    echo "$minimum_version"
}

# Function to extract current deadline from team file content
# COMMENTED OUT: Deadline extraction logic temporarily disabled
# extract_deadline() {
#     local content="$1"
#     local deadline=$(echo "$content" | grep -A 5 "macos_updates:" | grep "deadline:" | sed 's/.*deadline: *"\([^"]*\)".*/\1/')
#     echo "$deadline"
# }

# Function to update team file content with new version and deadline
# COMMENTED OUT: Team file update logic temporarily disabled
# update_team_file_content() {
#     local content="$1"
#     local new_version="$2"
#     local new_deadline="$3"
#     
#     # Update minimum_version
#     content=$(echo "$content" | sed "s/minimum_version: \"[^\"]*\"/minimum_version: \"$new_version\"/")
#     
#     # Update deadline
#     content=$(echo "$content" | sed "s/deadline: \"[^\"]*\"/deadline: \"$new_deadline\"/")
#     
#     echo "$content"
# }

# Fetch the latest macOS version and previous major version
echo "Fetching latest macOS versions..."
macos_versions=$(curl -s "https://sofafeed.macadmins.io/v2/macos_data_feed.json" | \
jq -r '.. | objects | select(has("ProductVersion")) | .ProductVersion' | sort -Vr)

if [ -z "$macos_versions" ]; then
    echo "Error: Failed to fetch macOS versions."
    exit 1
fi

# Get the latest version (first in sorted list)
latest_macos_version=$(echo "$macos_versions" | head -n 1)

# Extract major version number from latest version (e.g., "15.7" -> "15")
latest_major_version=$(echo "$latest_macos_version" | cut -d. -f1)

# Find all unique major versions in the data
all_major_versions=$(echo "$macos_versions" | cut -d. -f1 | sort -Vr | uniq)

# Find the previous major version (second highest major version)
previous_major_version=$(echo "$all_major_versions" | head -n 2 | tail -n 1)

# Find the latest version of the previous major version
previous_major_latest_version=$(echo "$macos_versions" | grep "^$previous_major_version\." | head -n 1)

if [ -z "$latest_macos_version" ]; then
    echo "Error: Failed to fetch the latest macOS version."
    exit 1
fi

echo "Latest macOS version: $latest_macos_version"
if [ -n "$previous_major_latest_version" ]; then
    echo "Latest previous major version (v$previous_major_version): $previous_major_latest_version"
else
    echo "Warning: No previous major version found for v$previous_major_version"
fi

# Initialize update flags
policy_update_needed=false
# COMMENTED OUT: Team updates flag temporarily disabled
# team_updates_needed=false
updates_needed=false

# Check policy file
echo "Checking policy file..."
POLICY_FILE_URL="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/contents/$POLICY_FILE_PATH?ref=$BRANCH"
policy_response=$(curl -s -H "Authorization: token $DOGFOOD_AUTOMATION_TOKEN" -H "Accept: application/vnd.github.v3.raw" "$POLICY_FILE_URL")

if [ -z "$policy_response" ] || [[ "$policy_response" == *"Not Found"* ]]; then
    echo "Error: Failed to fetch policy file or file does not exist in the repository."
    exit 1
fi

# Extract the query line from policy
query_line=$(echo "$policy_response" | grep 'query:')
if [ -z "$query_line" ]; then
    echo "Error: Could not find the query line in the policy file."
    exit 1
fi

# Extract version numbers from the query line (handle both single version and OR conditions)
policy_versions=$(echo "$query_line" | grep -oE "'[0-9]+\.[0-9]+(\.[0-9]+)?'" | sed "s/'//g" | sort -Vr)
if [ -z "$policy_versions" ]; then
    echo "Error: Failed to extract policy version numbers."
    exit 1
fi

# Get the highest version from policy (should be the latest)
policy_latest_version=$(echo "$policy_versions" | head -n 1)
echo "Policy latest version: $policy_latest_version"

# Check if policy needs updating
policy_needs_update=false
if [ "$policy_latest_version" != "$latest_macos_version" ]; then
    policy_needs_update=true
    echo "Policy needs update: latest version changed from $policy_latest_version to $latest_macos_version"
fi

# If we have a previous major version, check if it's included in the policy
if [ -n "$previous_major_latest_version" ]; then
    if ! echo "$policy_versions" | grep -q "^$previous_major_latest_version$"; then
        policy_needs_update=true
        echo "Policy needs update: previous major version $previous_major_latest_version not found in policy"
    fi
fi

if [ "$policy_needs_update" = true ]; then
    policy_update_needed=true
    updates_needed=true
fi

# COMMENTED OUT: Team files check logic temporarily disabled
# Check team files
# echo "Checking team files..."
# workstations_content=$(fetch_file_content "$WORKSTATIONS_FILE")
# if [ $? -ne 0 ]; then
#     echo "Warning: Could not fetch workstations file, skipping team updates."
# else
#     workstations_canary_content=$(fetch_file_content "$WORKSTATIONS_CANARY_FILE")
#     if [ $? -ne 0 ]; then
#         echo "Warning: Could not fetch workstations-canary file, skipping team updates."
#     else
#         # Extract current versions and deadlines
#         current_workstations_version=$(extract_minimum_version "$workstations_content")
#         current_workstations_deadline=$(extract_deadline "$workstations_content")
#         current_workstations_canary_version=$(extract_minimum_version "$workstations_canary_content")
#         current_workstations_canary_deadline=$(extract_deadline "$workstations_canary_content")
# 
#         echo "Current Workstations minimum_version: $current_workstations_version"
#         echo "Current Workstations deadline: $current_workstations_deadline"
#         echo "Current Workstations (canary) minimum_version: $current_workstations_canary_version"
#         echo "Current Workstations (canary) deadline: $current_workstations_canary_deadline"
# 
#         # Calculate new deadline
#         new_deadline=$(calculate_deadline)
#         echo "New deadline (4 Sundays from today): $new_deadline"
# 
#         # Check if team updates are needed
#         # Only update deadline if there's a new macOS version
#         if [ "$current_workstations_version" != "$latest_macos_version" ]; then
#             team_updates_needed=true
#             updates_needed=true
#         elif [ "$current_workstations_deadline" != "$new_deadline" ] && [ "$policy_update_needed" = true ]; then
#             # Only update deadline if policy was updated (meaning there's a new version)
#             team_updates_needed=true
#             updates_needed=true
#         fi
# 
#         if [ "$current_workstations_canary_version" != "$latest_macos_version" ]; then
#             team_updates_needed=true
#             updates_needed=true
#         elif [ "$current_workstations_canary_deadline" != "$new_deadline" ] && [ "$policy_update_needed" = true ]; then
#             # Only update deadline if policy was updated (meaning there's a new version)
#             team_updates_needed=true
#             updates_needed=true
#         fi
#     fi
# fi

# Create updates if needed
if [ "$updates_needed" = true ]; then
    echo "Updates needed. Creating pull request..."

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

    # Update policy file if needed
    if [ "$policy_update_needed" = true ]; then
        echo "Updating policy file..."
        
        # Build the query with both versions
        if [ -n "$previous_major_latest_version" ]; then
            new_query_line="query: SELECT 1 FROM os_version WHERE version >= '$latest_macos_version' OR version >= '$previous_major_latest_version';"
        else
            new_query_line="query: SELECT 1 FROM os_version WHERE version >= '$latest_macos_version';"
        fi
        
        updated_policy_response=$(echo "$policy_response" | sed "s/query: .*/$new_query_line/")
        
        if [ -z "$updated_policy_response" ]; then
            echo "Error: Failed to update the policy query line."
            exit 1
        fi
        
        echo "$updated_policy_response" > "$POLICY_FILE_PATH"
        git add "$POLICY_FILE_PATH"
    fi

    # COMMENTED OUT: Team files update logic temporarily disabled
    # Update team files if needed
    # if [ "$team_updates_needed" = true ]; then
    #     echo "Updating team files..."
    #     updated_workstations_content=$(update_team_file_content "$workstations_content" "$latest_macos_version" "$new_deadline")
    #     updated_canary_content=$(update_team_file_content "$workstations_canary_content" "$latest_macos_version" "$new_deadline")
    #     
    #     echo "$updated_workstations_content" > "$WORKSTATIONS_FILE"
    #     echo "$updated_canary_content" > "$WORKSTATIONS_CANARY_FILE"
    #     
    #     git add "$WORKSTATIONS_FILE" "$WORKSTATIONS_CANARY_FILE"
    # fi

    # Create commit message
    commit_message="Update macOS version to $latest_macos_version"
    if [ -n "$previous_major_latest_version" ]; then
        commit_message="$commit_message (includes previous major v$previous_major_version: $previous_major_latest_version)"
    fi
    
    if [ "$policy_update_needed" = true ]; then
        commit_message="$commit_message

- Updated policy to include latest version: $latest_macos_version"
        if [ -n "$previous_major_latest_version" ]; then
            commit_message="$commit_message
- Updated policy to include previous major version: $previous_major_latest_version"
        fi
        if [ -n "$policy_latest_version" ]; then
            commit_message="$commit_message
- Previous policy version: $policy_latest_version"
        fi
    fi
    # COMMENTED OUT: Team updates commit message logic temporarily disabled
    # if [ "$team_updates_needed" = true ]; then
    #     commit_message="$commit_message
    # - Updated team minimum_version from $current_workstations_version to $latest_macos_version
    # - Updated team deadline from $current_workstations_deadline to $new_deadline (4 Sundays from today)
    # - Applied to both workstations and workstations-canary teams"
    # fi

    git commit -m "$commit_message"
    git push origin "$NEW_BRANCH"

    # Create a pull request
    if [ -n "$previous_major_latest_version" ]; then
        pr_title="Update latest macOS versions to $latest_macos_version and $previous_major_latest_version"
    else
        pr_title="Update latest macOS version to $latest_macos_version"
    fi
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
    echo "No updates needed; all versions and deadlines are current."
fi
