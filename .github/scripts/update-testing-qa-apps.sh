#!/bin/bash

# Script to update the fleet_maintained_apps list in testing-and-qa.yml
# with any new apps from apps.json.
#
# This script:
# 1. Reads all available apps from apps.json
# 2. Reads currently listed apps from testing-and-qa.yml
# 3. Identifies missing apps
# 4. Adds missing apps to the appropriate section (macOS or Windows)
# 5. Maintains the existing format and comments

set -euo pipefail

# Get repository root
REPO_ROOT="${GITHUB_WORKSPACE:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
APPS_JSON="${REPO_ROOT}/ee/maintained-apps/outputs/apps.json"
YAML_FILE="${REPO_ROOT}/it-and-security/teams/testing-and-qa.yml"
TEMP_FILE=$(mktemp)

# Check if required files exist
if [ ! -f "$APPS_JSON" ]; then
    echo "Error: apps.json not found at $APPS_JSON" >&2
    exit 1
fi

if [ ! -f "$YAML_FILE" ]; then
    echo "Error: testing-and-qa.yml not found at $YAML_FILE" >&2
    exit 1
fi

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed" >&2
    exit 1
fi

echo "Loading apps from apps.json..."
TOTAL_APPS=$(jq '.apps | length' "$APPS_JSON")
echo "Found $TOTAL_APPS total apps in apps.json"

# Extract currently listed app slugs from YAML
echo "Loading current apps from testing-and-qa.yml..."
CURRENT_SLUGS=$(grep -E "^\s+- slug:" "$YAML_FILE" | sed 's/.*slug: \([^# ]*\).*/\1/' | tr -d ' ' | sort -u)
CURRENT_COUNT=$(echo "$CURRENT_SLUGS" | grep -c . || true)
echo "Found $CURRENT_COUNT currently listed apps"

# Find missing apps and separate by platform
DARWIN_APPS=()
WINDOWS_APPS=()

while IFS= read -r app_json; do
    slug=$(echo "$app_json" | jq -r '.slug')
    if ! echo "$CURRENT_SLUGS" | grep -q "^${slug}$"; then
        platform=$(echo "$app_json" | jq -r '.platform')
        name=$(echo "$app_json" | jq -r '.name')
        if [ "$platform" = "darwin" ]; then
            DARWIN_APPS+=("${slug}|${name}")
        elif [ "$platform" = "windows" ]; then
            WINDOWS_APPS+=("${slug}|${name}")
        fi
    fi
done < <(jq -c '.apps[]' "$APPS_JSON")

# Sort apps by slug for consistency
if [ ${#DARWIN_APPS[@]} -gt 0 ]; then
    IFS=$'\n'
    DARWIN_APPS=($(printf '%s\n' "${DARWIN_APPS[@]}" | sort))
    unset IFS
fi

if [ ${#WINDOWS_APPS[@]} -gt 0 ]; then
    IFS=$'\n'
    WINDOWS_APPS=($(printf '%s\n' "${WINDOWS_APPS[@]}" | sort))
    unset IFS
fi

TOTAL_MISSING=$((${#DARWIN_APPS[@]} + ${#WINDOWS_APPS[@]}))

if [ $TOTAL_MISSING -eq 0 ]; then
    echo "No new apps to add. Everything is up to date!"
    exit 0
fi

echo "Found $TOTAL_MISSING new apps to add:"
for app in "${DARWIN_APPS[@]+"${DARWIN_APPS[@]}"}" "${WINDOWS_APPS[@]+"${WINDOWS_APPS[@]}"}"; do
    slug="${app%%|*}"
    name="${app#*|}"
    echo "  - $name ($slug)"
done

# Find the fleet_maintained_apps section boundaries
FLEET_APPS_START=$(grep -n "fleet_maintained_apps:" "$YAML_FILE" | head -1 | cut -d: -f1)
if [ -z "$FLEET_APPS_START" ]; then
    echo "Error: Could not find fleet_maintained_apps section in YAML file" >&2
    exit 1
fi

# Find where the fleet_maintained_apps section ends (next top-level key or end of file)
TOTAL_LINES=$(wc -l < "$YAML_FILE")
FLEET_APPS_END=$TOTAL_LINES
for ((i=$((FLEET_APPS_START + 1)); i<=TOTAL_LINES; i++)); do
    line=$(sed -n "${i}p" "$YAML_FILE")
    # Check if this is a top-level key (starts at column 0, not indented)
    if [[ "$line" =~ ^[a-zA-Z] ]] && [[ ! "$line" =~ ^[[:space:]] ]]; then
        FLEET_APPS_END=$((i - 1))
        break
    fi
done

# Find macOS and Windows comment lines
MACOS_COMMENT_LINE=$(awk -v start="$FLEET_APPS_START" -v end="$FLEET_APPS_END" 'NR >= start && NR <= end && /# macOS apps/ {print NR; exit}' "$YAML_FILE")
WINDOWS_COMMENT_LINE=$(awk -v start="$FLEET_APPS_START" -v end="$FLEET_APPS_END" 'NR >= start && NR <= end && /# Windows apps/ {print NR; exit}' "$YAML_FILE")

# Find last darwin app and first windows app
LAST_DARWIN_LINE=$(awk -v start="$FLEET_APPS_START" -v end="$FLEET_APPS_END" 'NR >= start && NR <= end && /\/darwin/ {last=NR} END {print last+0}' "$YAML_FILE")
FIRST_WINDOWS_LINE=$(awk -v start="$FLEET_APPS_START" -v end="$FLEET_APPS_END" 'NR >= start && NR <= end && /\/windows/ {print NR; exit}' "$YAML_FILE")

# Build the updated file
{
    # Copy everything before fleet_maintained_apps
    head -n $((FLEET_APPS_START - 1)) "$YAML_FILE"
    
    # Copy the fleet_maintained_apps: line
    sed -n "${FLEET_APPS_START}p" "$YAML_FILE"
    
    # Handle macOS section
    # Copy existing macOS apps (from start to last darwin or windows comment)
    MACOS_SECTION_END=$FLEET_APPS_END
    if [ -n "$WINDOWS_COMMENT_LINE" ]; then
        MACOS_SECTION_END=$((WINDOWS_COMMENT_LINE - 1))
    elif [ -n "$FIRST_WINDOWS_LINE" ]; then
        MACOS_SECTION_END=$((FIRST_WINDOWS_LINE - 1))
    fi
    
    if [ "$MACOS_SECTION_END" -gt "$FLEET_APPS_START" ]; then
        # Copy existing macOS section
        sed -n "$((FLEET_APPS_START + 1)),${MACOS_SECTION_END}p" "$YAML_FILE"
    fi
    
    # Add macOS comment if we have new macOS apps and comment doesn't exist
    if [ ${#DARWIN_APPS[@]} -gt 0 ] && [ -z "$MACOS_COMMENT_LINE" ] && [ -z "$LAST_DARWIN_LINE" ]; then
        echo "    # macOS apps"
    fi
    
    # Add new macOS apps
    if [ ${#DARWIN_APPS[@]} -gt 0 ]; then
        for app in "${DARWIN_APPS[@]}"; do
            slug="${app%%|*}"
            name="${app#*|}"
            printf "    - slug: %s # %s for macOS\n" "$slug" "$name"
            echo "      self_service: true"
        done
    fi
    
    # Handle Windows section
    # Add Windows comment if needed
    if [ -n "$WINDOWS_COMMENT_LINE" ]; then
        # Copy Windows comment line
        sed -n "${WINDOWS_COMMENT_LINE}p" "$YAML_FILE"
        # Copy existing Windows apps
        if [ "$WINDOWS_COMMENT_LINE" -lt "$FLEET_APPS_END" ]; then
            sed -n "$((WINDOWS_COMMENT_LINE + 1)),${FLEET_APPS_END}p" "$YAML_FILE"
        fi
    elif [ -n "$FIRST_WINDOWS_LINE" ]; then
        # No Windows comment, but there are Windows apps
        # Copy existing Windows apps
        sed -n "${FIRST_WINDOWS_LINE},${FLEET_APPS_END}p" "$YAML_FILE"
    elif [ ${#WINDOWS_APPS[@]} -gt 0 ]; then
        # No existing Windows apps, add comment
        echo "    # Windows apps"
    fi
    
    # Add new Windows apps
    if [ ${#WINDOWS_APPS[@]} -gt 0 ]; then
        for app in "${WINDOWS_APPS[@]}"; do
            slug="${app%%|*}"
            name="${app#*|}"
            printf "    - slug: %s # %s for Windows\n" "$slug" "$name"
            echo "      self_service: true"
        done
    fi
    
    # Copy everything after fleet_maintained_apps section
    if [ "$FLEET_APPS_END" -lt "$TOTAL_LINES" ]; then
        tail -n +$((FLEET_APPS_END + 1)) "$YAML_FILE"
    fi
    
} > "$TEMP_FILE"

# Replace original file
mv "$TEMP_FILE" "$YAML_FILE"

echo "Successfully added $TOTAL_MISSING new apps to testing-and-qa.yml"
