#!/usr/bin/env bash
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
DAYS_TO_KEEP=3 # how much git history to get for initial clone
# Fallback depth used when --shallow-since returns no commits (upstream was quiet).
# The sync only needs the tip commit to populate the working tree.
# We grab a few extra commits so the warning log shows recent upstream activity for debugging.
FALLBACK_DEPTH=3

echo "=== OSV Repository Sync ==="
echo ""

if [ -d "$REPO_DIR/.git" ]; then
    echo "Repository exists, fetching new commits..."
    cd "$REPO_DIR"

    git config core.sparseCheckout true
    echo "osv/" > .git/info/sparse-checkout

    OLD_SHA=$(git rev-parse HEAD)
    OLD_COUNT=$(git log --oneline | wc -l | xargs)

    git fetch origin main

    NEW_SHA=$(git rev-parse origin/main)
    git reset --hard origin/main

    echo ""
    if [ "$OLD_SHA" = "$NEW_SHA" ]; then
        echo "No new commits (already at $NEW_SHA)"
    else
        echo "Updating: $OLD_SHA -> $NEW_SHA"
    fi

    NEW_COUNT=$(git log --oneline | wc -l | xargs)
    echo "History: $OLD_COUNT commits -> $NEW_COUNT commits"

    cd ..
else
    echo "Cloning repository (shallow since ${DAYS_TO_KEEP} days ago)..."

    mkdir -p "$REPO_DIR"
    cd "$REPO_DIR"
    git init --initial-branch=main
    git remote add origin "$REPO_URL"

    git config core.sparseCheckout true
    echo "osv/" > .git/info/sparse-checkout

    if ! git fetch --shallow-since="${DAYS_TO_KEEP} days ago" origin main; then
        echo ""
        echo "WARNING: --shallow-since=${DAYS_TO_KEEP}d returned no commits."
        echo "Upstream has been quiet for >${DAYS_TO_KEEP} days. Falling back to --depth=${FALLBACK_DEPTH}."
        echo ""
        git fetch --depth="${FALLBACK_DEPTH}" origin main
        echo "Recent history:"
        git log --pretty=format:'%h %ci %s' origin/main
        echo ""
        echo ""
    fi
    git checkout -b main --track origin/main

    COMMIT_SHA=$(git rev-parse HEAD)
    COMMIT_COUNT=$(git log --oneline | wc -l | xargs)
    cd ..

    echo ""
    echo "Cloned at: $COMMIT_SHA"
fi

cd "$REPO_DIR"

TODAY_UTC=$(date -u +%Y-%m-%d)
YESTERDAY_UTC=$(date -u -v-1d +%Y-%m-%d 2>/dev/null || date -u -d "yesterday" +%Y-%m-%d)

# Get files changed today (since midnight UTC today)
git log --since="${TODAY_UTC}T00:00:00Z" --name-only --pretty="" -- osv/cve \
    | sed '/^$/d' | sort -u > "../changed_files_today.txt"

# Get files changed yesterday (from midnight yesterday to midnight today UTC)
git log --since="${YESTERDAY_UTC}T00:00:00Z" --until="${TODAY_UTC}T00:00:00Z" --name-only --pretty="" -- osv/cve \
    | sed '/^$/d' | sort -u > "../changed_files_yesterday.txt"

TODAY_COUNT=$(wc -l < "../changed_files_today.txt" | xargs)
YESTERDAY_COUNT=$(wc -l < "../changed_files_yesterday.txt" | xargs)
cd ..

echo "Today: $TODAY_COUNT CVE files changed"
echo "Yesterday: $YESTERDAY_COUNT CVE files changed"

echo ""
echo "Sync Complete"
cd "$REPO_DIR"
FINAL_SHA=$(git rev-parse HEAD)
FINAL_COUNT=$(git log --oneline | wc -l | xargs)
cd ..

du -sh "$REPO_DIR" | awk '{print "Size: " $1}'
echo "REPO_SHA=$FINAL_SHA"
echo "REPO_COMMITS=$FINAL_COUNT"
echo "OSV_DIR=$REPO_DIR/osv/cve"
echo "CHANGED_FILES_TODAY=changed_files_today.txt"
echo "CHANGED_FILES_YESTERDAY=changed_files_yesterday.txt"
echo "TODAY_COUNT=$TODAY_COUNT"
echo "YESTERDAY_COUNT=$YESTERDAY_COUNT"

exit 0
