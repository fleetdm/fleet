#!/bin/bash

# Script to filter apps.json to only include specified app slugs
# Usage: filter-apps-json.sh <slugs_json_array | slugs_json_file> <output_file>

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
SLUGS_INPUT="$1"
OUTPUT_FILE="$2"

# Accept the slugs as either a literal JSON array string or a path to a file containing
# the JSON array. The file form avoids cross-shell quoting problems: Windows PowerShell
# mangles embedded quotes when forwarding a JSON string as a native-command argument to
# bash, corrupting the value before jq sees it. Callers passing a literal JSON string
# (e.g. on macOS) are unaffected since that value is not a path to an existing file.
if [ -n "$SLUGS_INPUT" ] && [ -f "$SLUGS_INPUT" ]; then
    SLUGS_JSON="$(cat "$SLUGS_INPUT")"
else
    SLUGS_JSON="$SLUGS_INPUT"
fi

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

