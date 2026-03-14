#!/bin/bash

# Usage: ./sync_estimates.sh [--dry-run]
DRY_RUN=false
if [[ "$1" == "--dry-run" ]]; then
  DRY_RUN=true
fi

OWNER="fleetdm"
SOURCE_PROJECT_NUMBER=67
TARGET_PROJECT_NUMBER=58
ESTIMATE_FIELD_NAME="estimate"  # case-insensitive

TMP_FILE=$(mktemp)

# Get the project node ID
PROJECT_ID=$(gh project list --owner "$OWNER" --format json | jq -r ".projects[] | select(.number == $TARGET_PROJECT_NUMBER) | .id")

if [[ -z "$PROJECT_ID" ]]; then
  echo "❌ Could not find project ID for project number $TARGET_PROJECT_NUMBER"
  exit 1
fi

# Get the field ID for 'Estimate'
FIELD_ID=$(gh project field-list --owner "$OWNER" --format json "$TARGET_PROJECT_NUMBER" | \
  jq -r '.fields[] | select(.name | ascii_downcase == "estimate") | .id')

if [[ -z "$FIELD_ID" ]]; then
  echo "❌ Could not find field ID for estimate"
  exit 1
fi

echo "✅ Found project ID: $PROJECT_ID"
echo "✅ Found field ID for estimate: $FIELD_ID"

echo "📥 Fetching estimates from source project $SOURCE_PROJECT_NUMBER..."
gh project item-list --limit 500 --format json --owner "$OWNER" "$SOURCE_PROJECT_NUMBER" | \
  jq -r '.items[] | select(.content != null) | [.content.number, (.estimate // empty)] | @tsv' > "$TMP_FILE"

echo "🔁 Syncing estimates to target project $TARGET_PROJECT_NUMBER..."
gh project item-list --limit 500 --format json --owner "$OWNER" "$TARGET_PROJECT_NUMBER" | \
jq -c '.items[] | select(.content != null) | {id: .id, number: .content.number}' | \
while read -r item; do
  ITEM_ID=$(echo "$item" | jq -r .id)
  NUMBER=$(echo "$item" | jq -r .number)
  ESTIMATE=""

  # Match by issue number from the temp file
  while IFS=$'\t' read -r SRC_NUMBER SRC_ESTIMATE; do
    if [[ "$SRC_NUMBER" == "$NUMBER" ]]; then
      ESTIMATE="$SRC_ESTIMATE"
      break
    fi
  done < "$TMP_FILE"

  if [[ -n "$ESTIMATE" ]]; then
    if $DRY_RUN; then
      echo "[Dry-run] Would update issue #$NUMBER (item ID $ITEM_ID) to estimate: $ESTIMATE"
    else
      echo "✅ Updating issue #$NUMBER (item ID $ITEM_ID) to estimate: $ESTIMATE"
      gh project item-edit --id "$ITEM_ID" --project-id "$PROJECT_ID" --field-id "$FIELD_ID" --number "$ESTIMATE"
    fi
  fi
done

rm -f "$TMP_FILE"

