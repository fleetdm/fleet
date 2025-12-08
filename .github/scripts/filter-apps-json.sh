#!/bin/bash

# Script to filter apps.json to only include specified app slugs
# Usage: filter-apps-json.sh <slugs_json_array> <output_file>

set -euo pipefail

# Get repository root
REPO_ROOT="${GITHUB_WORKSPACE:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
APPS_JSON="${REPO_ROOT}/ee/maintained-apps/outputs/apps.json"

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed" >&2
    exit 1
fi

# Parse arguments
SLUGS_JSON="$1"
OUTPUT_FILE="$2"

if [ -z "$SLUGS_JSON" ] || [ "$SLUGS_JSON" == "[]" ] || [ "$SLUGS_JSON" == "null" ]; then
    echo "No slugs provided, creating empty apps.json"
    echo '{"version": 2, "apps": []}' > "$OUTPUT_FILE"
    exit 0
fi

# Read the original apps.json
if [ ! -f "$APPS_JSON" ]; then
    echo "Error: apps.json not found at $APPS_JSON" >&2
    exit 1
fi

# Filter apps.json to only include the specified slugs
jq --argjson slugs "$SLUGS_JSON" '.apps = (.apps | map(select(.slug as $slug | $slugs | index($slug) != null)))' "$APPS_JSON" > "$OUTPUT_FILE"

echo "Filtered apps.json created with $(jq '.apps | length' "$OUTPUT_FILE") app(s)"

