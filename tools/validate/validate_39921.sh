#!/bin/bash
#
# Validation script for issue #39921
# Scopes declaration processing in BulkSetPendingMDMHostProfiles to affected hosts only
#
set -euo pipefail

FLEET_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$FLEET_ROOT"

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS_COUNT=0
FAIL_COUNT=0
RESULTS=()

run_test() {
    local name="$1"
    local pattern="$2"
    local timeout="$3"
    local logfile
    logfile=$(mktemp)

    echo -e "${YELLOW}Running: ${name}${NC}"
    if MYSQL_TEST=1 FLEET_MYSQL_ADDRESS=127.0.0.1:3307 FLEET_MYSQL_DATABASE=fleet \
        go test ./server/datastore/mysql/ -run "$pattern" -v -count=1 -timeout "${timeout}s" \
        > "$logfile" 2>&1; then
        echo -e "${GREEN}PASS${NC}: ${name}"
        PASS_COUNT=$((PASS_COUNT + 1))
        RESULTS+=("PASS|${name}|$(tail -1 "$logfile")")
    else
        echo -e "${RED}FAIL${NC}: ${name}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        RESULTS+=("FAIL|${name}|$(tail -5 "$logfile")")
    fi
    rm -f "$logfile"
}

echo "============================================"
echo " Validation for Issue #39921"
echo " Scope declaration processing to affected hosts"
echo "============================================"
echo ""

# Run all three test suites
run_test \
    "BulkSetPendingMDMHostProfiles (all variants)" \
    "TestMDMShared/TestBulkSetPendingMDMHostProfiles" \
    600

run_test \
    "MDMAppleBatchSetHostDeclarationState (DDM)" \
    "TestMDMDDMApple" \
    300

run_test \
    "MDMAppleDDMDeclarationsToken" \
    "TestMDMApple/MDMAppleDDMDeclarationsToken" \
    300

echo ""
echo "============================================"
echo " Results: ${PASS_COUNT} passed, ${FAIL_COUNT} failed"
echo "============================================"
echo ""

# Output markdown report
cat <<'MARKDOWN_EOF'
# Validation Report: Issue #39921

## Problem

When `BulkSetPendingMDMHostProfiles` was called (e.g., after applying a team
spec via `POST /api/latest/fleet/spec/teams`), the declaration processing step
(`mdmAppleBatchSetHostDeclarationStateDB`) computed the desired declaration
state for **ALL enrolled hosts**, regardless of which hosts were actually
affected by the change.

**Before**: `mdmAppleBatchSetHostDeclarationStateDB` always computed desired
state for ALL hosts. With 70k hosts and 30 declarations, this generated a
massive 4-way UNION query joining every host against every declaration, causing
severe performance degradation (50+ second API responses).

## Fix

**After**: When called from `BulkSetPendingMDMHostProfiles`, declarations are
scoped to only the affected hosts (those belonging to the modified team). The
cron job (`MDMAppleBatchSetHostDeclarationState`) continues to process all hosts.

### Changes

1. **`server/datastore/mysql/mdm.go`**:
   - Collect declaration UUIDs from profile UUIDs passed to the function.
   - Add a new `case hasAppleDecls:` block that looks up host UUIDs associated
     with the changed declarations (via team membership and existing assignments).
   - Remove the `!hasAppleDecls` guard so the host UUID lookup runs for
     declarations just like it does for profiles.
   - Pass the scoped `appleHosts` list into `mdmAppleBatchSetHostDeclarationStateDB`.

2. **`server/datastore/mysql/apple_mdm.go`**:
   - Add `hostUUIDs []string` parameter to `mdmAppleBatchSetHostDeclarationStateDB`.
   - Add `hostUUIDs []string` parameter to `mdmAppleGetHostsWithChangedDeclarationsDB`.
   - When `hostUUIDs` is non-empty, use batched host-filtered queries
     (`generateEntitiesToInstallQueryWithDesiredState` and
     `generateEntitiesToRemoveQueryWithDesiredState`) that include
     `h.uuid IN (?)` conditions, avoiding a full table scan.
   - When `hostUUIDs` is empty (cron path), fall through to the original
     `mdmAppleGetAllHostsWithChangedDeclarationsDB` that processes all hosts.

MARKDOWN_EOF

echo "## Test Results"
echo ""
echo "| Status | Test Suite | Details |"
echo "|--------|-----------|---------|"
for r in "${RESULTS[@]}"; do
    status=$(echo "$r" | cut -d'|' -f1)
    name=$(echo "$r" | cut -d'|' -f2)
    details=$(echo "$r" | cut -d'|' -f3-)
    if [ "$status" = "PASS" ]; then
        echo "| PASS | ${name} | ${details} |"
    else
        echo "| FAIL | ${name} | ${details} |"
    fi
done

echo ""
if [ "$FAIL_COUNT" -eq 0 ]; then
    echo "**All ${PASS_COUNT} test suites passed.**"
else
    echo "**${FAIL_COUNT} test suite(s) FAILED.** Please investigate."
    exit 1
fi
