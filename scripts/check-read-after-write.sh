#!/bin/bash
#
# check-read-after-write.sh
#
# Detects read-after-write anti-patterns in frontend code. In environments with
# database read replica lag, refetching data immediately after a write can return
# stale values. Instead, use the write response to update the local cache
# directly via queryClient.setQueryData().
#
# Usage: ./scripts/check-read-after-write.sh
#
# Exit code: 1 if violations found, 0 otherwise.

set -euo pipefail

VIOLATIONS=0
FRONTEND_DIR="frontend"

echo "Checking for read-after-write patterns in ${FRONTEND_DIR}/..."
echo ""

# Helper: count violations from a grep, filtering out test files, comments,
# type declarations, prop interfaces, and prop-passing (JSX attributes).
check_pattern() {
  local pattern="$1"
  local message="$2"

  while IFS= read -r match; do
    if [[ -n "$match" ]]; then
      echo "VIOLATION: $match"
      echo "  -> $message"
      echo ""
      VIOLATIONS=$((VIOLATIONS + 1))
    fi
  done < <(
    grep -rn "$pattern" "$FRONTEND_DIR" --include='*.tsx' --include='*.ts' 2>/dev/null \
      | grep -v '\.tests\.\|\.test\.\|__tests__' \
      | grep -v '^\s*//' \
      | grep -v ':\s*//' \
      | grep -v ':\s*/\*' \
      | grep -v ': () => void' \
      | grep -v 'refetch:' \
      | grep -v '={.*}' \
      | grep -v 'interface \|type ' \
      || true
  )
}

# Pattern 1: invalidateQueries(["config"]) -- should use setQueryData
check_pattern \
  'invalidateQueries(\["config"\])' \
  'Use queryClient.setQueryData(["config"], response) with the write response instead.'

# Pattern 2: Actual calls to refetchConfig/refetchAppConfig/etc. (not
# declarations, not prop-passing, not test files).
check_pattern \
  'refetchConfig()' \
  'Use queryClient.setQueryData(["config"], response) with the write response instead of refetching.'

check_pattern \
  'refetchAppConfig()' \
  'Use queryClient.setQueryData(["config"], response) with the write response instead of refetching.'

check_pattern \
  'refetchTeamConfig()' \
  'Use queryClient.setQueryData(["teamConfig", teamId], response) instead of refetching.'

check_pattern \
  'refetchGlobalConfig()' \
  'Use queryClient.setQueryData(["config"], response) instead of refetching.'

check_pattern \
  'refetchIntegrations()' \
  'Use queryClient.setQueryData(["config"], response) with the write response instead of refetching.'

echo "---"
if [[ $VIOLATIONS -gt 0 ]]; then
  echo "Found $VIOLATIONS read-after-write violation(s)."
  echo ""
  echo "These patterns cause stale data in environments with database read replica lag."
  echo "Instead of refetching after a write, use the write response to update the cache:"
  echo ""
  echo "  // Bad: refetches from a potentially stale replica"
  echo "  await configAPI.update(data);"
  echo "  refetchConfig();"
  echo ""
  echo "  // Good: uses the write response directly"
  echo "  const updatedConfig = await configAPI.update(data);"
  echo "  queryClient.setQueryData([\"config\"], updatedConfig);"
  echo "  setConfig(updatedConfig);"
  exit 1
else
  echo "No read-after-write violations found."
  exit 0
fi
