#!/bin/sh

# This script can be linked or copied to .git/hooks/post-checkout
# It will automatically dump and restore the db based on the branch

PREV_HEAD=$1
CURR_HEAD=$2
BRANCH_CHEKCOUT=$3

DUMP_DIR="${HOME}/.fleet/snapshots/checkouts/"
BACKUP_TOOL="./tools/backup_db/backup.sh"
RESTORE_TOOL="./tools/backup_db/restore.sh"

if [ $BRANCH_CHEKCOUT != 1 ]; then
    echo "Not a branch checkout, exiting backup and restore"
    exit 0
fi

DIFF_MIGRATION_FILES="$(git diff-tree --name-only --no-commit-id -r ${PREV_HEAD} ${CURR_HEAD} server/datastore/mysql/migrations)"

if [ -z "${DIFF_MIGRATION_FILES}" ]; then
    echo "No datastore migrations between branches, leaving db alone"
    exit 0
fi

mkdir -p "${DUMP_DIR}"

## Get the human-readable branch names and paths for db dumps

PREV_BRANCH="$(git name-rev --name-only --exclude='remotes/*' ${PREV_HEAD})"
if [ "${PREV_BRANCH}" = "HEAD" ]; then
    # If the only refernce we get is HEAD, use the commit hash
    PREV_BRANCH="${PREV_HEAD}"
fi
CURR_BRANCH="$(git name-rev --name-only --exclude='remotes/*' ${CURR_HEAD})"
if [ "${CURR_BRANCH}" = "HEAD" ]; then
    CURR_BRANCH="${CURR_HEAD}"
fi

PREV_BRANCH_SAFE="$(echo $PREV_BRANCH | sed 's|/|_|g')"
CURR_BRANCH_SAFE="$(echo $CURR_BRANCH | sed 's|/|_|g')"

PREV_DUMP_NAME="${PREV_BRANCH_SAFE} $(date "+%Y-%m-%d %H:%M:%S").sql.gz"
PREV_DUMP_PATH="${DUMP_DIR}${PREV_DUMP_NAME}"

CURR_DUMP_NAME="$(ls $DUMP_DIR | grep "^${CURR_BRANCH_SAFE} " | head -n 1)"
if [ -n "${CURR_DUMP_NAME}" ]; then
    CURR_DUMP_PATH="${DUMP_DIR}${CURR_DUMP_NAME}"
fi

## Run the tools

echo "Dumping old branch db to ${PREV_DUMP_PATH}"
$BACKUP_TOOL "${PREV_DUMP_PATH}"

if [ -n "${CURR_DUMP_PATH}" ]; then
    echo "Restoring curremt branch db from ${CURR_DUMP_PATH}"
    $RESTORE_TOOL "${CURR_DUMP_PATH}"
else
    echo "No existing backup, leaving db"
fi
