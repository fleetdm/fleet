#!/bin/bash
# Sync Canonical OSV repository using shallow clone with rolling window
# Usage: ./sync-and-detect-changes.sh
#
# Outputs:
#   - Creates/updates ubuntu-security-notices directory (shallow clone)
#   - changed_files_today.txt and changed_files_yesterday.txt
#
# Exit codes:
#   0: Success
#   1: Error occurred

set -euo pipefail

# Configuration
REPO_URL="https://github.com/canonical/ubuntu-security-notices.git"
REPO_DIR="ubuntu-security-notices"
DAYS_TO_KEEP=3 # how much git history to keep

echo "=== OSV Repository Sync ==="
echo ""

if [ -d "$REPO_DIR/.git" ]; then
    echo "Repository exists, updating with rolling window..."
    cd "$REPO_DIR"

    OLD_SHA=$(git rev-parse HEAD)
    OLD_COUNT=$(git log --oneline | wc -l | xargs)

    git fetch --update-shallow --shallow-since="${DAYS_TO_KEEP} days ago" origin main --quiet

    NEW_SHA=$(git rev-parse origin/main)

    if [ "$OLD_SHA" = "$NEW_SHA" ]; then
        echo "  No new commits (already at $NEW_SHA)"
    else
        echo "  Updating: $OLD_SHA -> $NEW_SHA"
        git reset --hard origin/main --quiet
    fi

    NEW_COUNT=$(git log --oneline | wc -l | xargs)
    echo "  History: $OLD_COUNT commits -> $NEW_COUNT commits"

    cd ..
else
    echo "Cloning repository (shallow since ${DAYS_TO_KEEP} days ago)..."
    git clone --shallow-since="${DAYS_TO_KEEP} days ago" --quiet "$REPO_URL" "$REPO_DIR"

    cd "$REPO_DIR"
    COMMIT_SHA=$(git rev-parse HEAD)
    COMMIT_COUNT=$(git log --oneline | wc -l | xargs)
    cd ..

    echo "  Cloned at: $COMMIT_SHA"
    echo "  History: $COMMIT_COUNT commits"
    du -sh "$REPO_DIR" | awk '{print "  Size: " $1}'
fi

cd "$REPO_DIR"

# Get files changed today (since midnight UTC today)
TODAY_UTC=$(date -u +%Y-%m-%d)
git log --since="${TODAY_UTC}T00:00:00Z" --name-only --pretty="" -- osv/cve \
    | sort -u > "../changed_files_today.txt"

# Get files changed yesterday (from midnight yesterday to midnight today UTC)
YESTERDAY_UTC=$(date -u -v-1d +%Y-%m-%d 2>/dev/null || date -u -d "yesterday" +%Y-%m-%d)
git log --since="${YESTERDAY_UTC}T00:00:00Z" --until="${TODAY_UTC}T00:00:00Z" --name-only --pretty="" -- osv/cve \
    | sort -u > "../changed_files_yesterday.txt"

TODAY_COUNT=$(wc -l < "../changed_files_today.txt" | xargs)
YESTERDAY_COUNT=$(wc -l < "../changed_files_yesterday.txt" | xargs)
cd ..

echo "  Today: $TODAY_COUNT CVE files changed"
echo "  Yesterday: $YESTERDAY_COUNT CVE files changed"

echo ""
echo "=== Sync Complete ==="
cd "$REPO_DIR"
FINAL_SHA=$(git rev-parse HEAD)
FINAL_COUNT=$(git log --oneline | wc -l | xargs)
cd ..

echo "REPO_SHA=$FINAL_SHA"
echo "REPO_COMMITS=$FINAL_COUNT"
echo "OSV_DIR=$REPO_DIR/osv/cve"
echo "CHANGED_FILES_TODAY=changed_files_today.txt"
echo "CHANGED_FILES_YESTERDAY=changed_files_yesterday.txt"
echo "TODAY_COUNT=$TODAY_COUNT"
echo "YESTERDAY_COUNT=$YESTERDAY_COUNT"

exit 0
