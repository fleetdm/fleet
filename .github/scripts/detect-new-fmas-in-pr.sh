#!/bin/bash

# Script to detect changed/new maintained apps in a PR
# This script compares the PR branch with the base branch to find:
# 1. New apps added to apps.json
# 2. Apps with changed manifest files

set -euo pipefail

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
    exit 1
fi

# Function to extract app slugs from apps.json
extract_slugs() {
    local apps_file="$1"
    if [ ! -f "$apps_file" ]; then
        echo ""
        return
    fi
    jq -r '.apps[].slug' "$apps_file" | sort
}

# Function to extract app slugs from changed manifest files
extract_slugs_from_changed_manifests() {
    local changed_files="$1"
    local slugs=()
    
    while IFS= read -r file; do
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
MERGE_BASE=$(git merge-base "${BASE_BRANCH_REF}" HEAD 2>/dev/null || echo "${BASE_BRANCH_REF}")
CHANGED_FILES=$(git diff --name-only "$MERGE_BASE" HEAD -- "ee/maintained-apps/outputs/" 2>/dev/null || echo "")

# Extract slugs from changed manifest files
CHANGED_MANIFEST_SLUGS=$(extract_slugs_from_changed_manifests "$CHANGED_FILES")

# Get current apps.json slugs
CURRENT_SLUGS=$(extract_slugs "$APPS_JSON")

# Get base branch apps.json slugs
echo "Fetching base branch apps.json from ${MERGE_BASE}..."
BASE_APPS_JSON=$(git show "${MERGE_BASE}:ee/maintained-apps/outputs/apps.json" 2>/dev/null || echo "")
BASE_SLUGS=""
if [ -n "$BASE_APPS_JSON" ]; then
    BASE_SLUGS=$(echo "$BASE_APPS_JSON" | jq -r '.apps[].slug' | sort)
else
    echo "Warning: Could not find apps.json in base branch, treating all current apps as new"
fi

# Find new slugs in apps.json
NEW_SLUGS=$(comm -13 <(echo "$BASE_SLUGS" || echo "") <(echo "$CURRENT_SLUGS" || echo "") || echo "")

# Combine all changed slugs (from manifest changes and new apps)
ALL_CHANGED_SLUGS=$(printf '%s\n' "$CHANGED_MANIFEST_SLUGS" "$NEW_SLUGS" | grep -v '^$' | sort -u)

# Output results
if [ -z "$ALL_CHANGED_SLUGS" ]; then
    echo "No changed apps detected."
    echo "CHANGED_APPS=" >> "$GITHUB_OUTPUT"
    echo "HAS_CHANGES=false" >> "$GITHUB_OUTPUT"
    exit 0
fi

echo "Detected changed apps:"
echo "$ALL_CHANGED_SLUGS" | while read -r slug; do
    echo "  - $slug"
done

# Output as JSON array for GitHub Actions
CHANGED_APPS_JSON=$(echo "$ALL_CHANGED_SLUGS" | jq -R -s -c 'split("\n") | map(select(length > 0))')

echo "CHANGED_APPS=$CHANGED_APPS_JSON" >> "$GITHUB_OUTPUT"
echo "HAS_CHANGES=true" >> "$GITHUB_OUTPUT"


