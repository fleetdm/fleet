#!/bin/bash

# Script to detect changed/new maintained apps in a PR
# This script compares the PR branch with the base branch to find:
# 1. New apps added to apps.json
# 2. Apps with changed manifest files

# Use set -e but allow commands to fail gracefully with || true
set -uo pipefail

# Get repository root
REPO_ROOT="${GITHUB_WORKSPACE:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
APPS_JSON="${REPO_ROOT}/ee/maintained-apps/outputs/apps.json"
OUTPUTS_DIR="${REPO_ROOT}/ee/maintained-apps/outputs"

# Base branch (usually main or the PR's base branch)
# In GitHub Actions, GITHUB_BASE_REF is set for pull_request events
BASE_BRANCH="${GITHUB_BASE_REF:-main}"
# Use origin/ prefix for remote branch reference
BASE_BRANCH_REF="origin/${BASE_BRANCH}"

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed" >&2
    echo "CHANGED_APPS=[]" >> "$GITHUB_OUTPUT"
    echo "HAS_CHANGES=false" >> "$GITHUB_OUTPUT"
    exit 1
fi

# Function to extract app slugs from apps.json
extract_slugs() {
    local apps_file="$1"
    if [ ! -f "$apps_file" ]; then
        echo ""
        return 0
    fi
    jq -r '.apps[].slug' "$apps_file" 2>/dev/null | sort || echo ""
}

# Function to extract app slugs from changed manifest files
extract_slugs_from_changed_manifests() {
    local changed_files="$1"
    local slugs=()
    
    if [ -z "$changed_files" ]; then
        echo ""
        return 0
    fi
    
    while IFS= read -r file; do
        # Skip empty lines
        [ -z "$file" ] && continue
        
        # Extract slug from path like: outputs/app-name/darwin.json or outputs/app-name/windows.json
        if [[ "$file" =~ outputs/([^/]+)/(darwin|windows)\.json$ ]]; then
            app_name="${BASH_REMATCH[1]}"
            platform="${BASH_REMATCH[2]}"
            slug="${app_name}/${platform}"
            slugs+=("$slug")
        fi
    done <<< "$changed_files"
    
    # Remove duplicates and sort
    if [ ${#slugs[@]} -eq 0 ]; then
        echo ""
    else
        printf '%s\n' "${slugs[@]}" | sort -u
    fi
}

# Get changed files in outputs directory
echo "Detecting changed files in outputs directory..."
echo "Comparing HEAD with ${BASE_BRANCH_REF}..."

# Use merge-base to find the common ancestor for comparison
# If merge-base fails, try using the base branch ref directly
MERGE_BASE=""
if git merge-base "${BASE_BRANCH_REF}" HEAD &>/dev/null; then
    MERGE_BASE=$(git merge-base "${BASE_BRANCH_REF}" HEAD 2>/dev/null || echo "")
fi

# If merge-base still failed, try the base branch ref directly
if [ -z "$MERGE_BASE" ]; then
    echo "Warning: Could not find merge-base, using ${BASE_BRANCH_REF} directly"
    MERGE_BASE="${BASE_BRANCH_REF}"
fi

# Get changed files, handling errors gracefully
CHANGED_FILES=""
if git diff --name-only "$MERGE_BASE" HEAD -- "ee/maintained-apps/outputs/" &>/dev/null; then
    CHANGED_FILES=$(git diff --name-only "$MERGE_BASE" HEAD -- "ee/maintained-apps/outputs/" 2>/dev/null || echo "")
else
    echo "Warning: Could not get changed files, assuming no changes"
    CHANGED_FILES=""
fi

# Extract slugs from changed manifest files
CHANGED_MANIFEST_SLUGS=$(extract_slugs_from_changed_manifests "$CHANGED_FILES")

# Get current apps.json slugs
CURRENT_SLUGS=$(extract_slugs "$APPS_JSON")

# Get base branch apps.json slugs
echo "Fetching base branch apps.json from ${MERGE_BASE}..."
BASE_APPS_JSON=""
BASE_SLUGS=""
if git show "${MERGE_BASE}:ee/maintained-apps/outputs/apps.json" &>/dev/null; then
    BASE_APPS_JSON=$(git show "${MERGE_BASE}:ee/maintained-apps/outputs/apps.json" 2>/dev/null || echo "")
    if [ -n "$BASE_APPS_JSON" ]; then
        BASE_SLUGS=$(echo "$BASE_APPS_JSON" | jq -r '.apps[].slug' 2>/dev/null | sort || echo "")
    fi
fi

if [ -z "$BASE_SLUGS" ]; then
    echo "Warning: Could not find apps.json in base branch, treating all current apps as new"
    # If we can't get base slugs, only use manifest changes
    NEW_SLUGS=""
else
    # Find new slugs in apps.json
    NEW_SLUGS=$(comm -13 <(echo "$BASE_SLUGS" || echo "") <(echo "$CURRENT_SLUGS" || echo "") 2>/dev/null || echo "")
fi

# Combine all changed slugs (from manifest changes and new apps)
ALL_CHANGED_SLUGS=$(printf '%s\n' "$CHANGED_MANIFEST_SLUGS" "$NEW_SLUGS" | grep -v '^$' | sort -u || echo "")

# Output results
if [ -z "$ALL_CHANGED_SLUGS" ]; then
    echo "No changed apps detected."
    echo "CHANGED_APPS=[]" >> "$GITHUB_OUTPUT"
    echo "HAS_CHANGES=false" >> "$GITHUB_OUTPUT"
    exit 0
fi

echo "Detected changed apps:"
echo "$ALL_CHANGED_SLUGS" | while read -r slug; do
    [ -n "$slug" ] && echo "  - $slug"
done

# Output as JSON array for GitHub Actions
CHANGED_APPS_JSON=$(echo "$ALL_CHANGED_SLUGS" | jq -R -s -c 'split("\n") | map(select(length > 0))' 2>/dev/null || echo "[]")

echo "CHANGED_APPS=$CHANGED_APPS_JSON" >> "$GITHUB_OUTPUT"
echo "HAS_CHANGES=true" >> "$GITHUB_OUTPUT"


