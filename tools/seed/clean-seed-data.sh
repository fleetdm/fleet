#!/bin/bash

# Clean up seeded data for local development
#
# Deletes all policies, reports, and non-admin users (preserves user ID 1).
# Useful before re-running the seed scripts on a fresh slate without
# doing a full database reset.
#
# Prerequisites: same env file as seed-users-and-fleets.sh
#
# Usage:
#   export FLEET_ENV_PATH=tools/seed/DO_NOT_COMMIT_ENV_FILE
#   bash tools/seed/clean-seed-data.sh

set -euo pipefail

source "$FLEET_ENV_PATH"

# Validate required env vars
missing=""
[ -z "${TOKEN:-}" ] && missing="$missing TOKEN"
[ -z "${SERVER_URL:-}" ] && missing="$missing SERVER_URL"
if [ -n "$missing" ]; then
  echo "ERROR: Missing required env vars:$missing. Check your env file." >&2
  exit 1
fi

API="$SERVER_URL/api/latest/fleet"
AUTH="Authorization: Bearer $TOKEN"

# Helper: extract IDs from a JSON array
extract_ids() {
  python3 -c "
import sys, json
data = json.load(sys.stdin)
key = sys.argv[1]
exclude_id = int(sys.argv[2]) if len(sys.argv) > 2 else 0
items = data.get(key, [])
ids = [item['id'] for item in items if item['id'] != exclude_id]
print(json.dumps(ids))
" "$@"
}

echo "==> Deleting all policies..."

# Delete global policies
global_policy_ids=$(curl $CURL_FLAGS -H "$AUTH" "$API/policies" --insecure | extract_ids "policies")
if [ "$global_policy_ids" != "[]" ]; then
  curl -X POST $CURL_FLAGS -H "$AUTH" "$API/policies/delete" --insecure \
    -d "{\"ids\": $global_policy_ids}" > /dev/null
  echo "[+] deleted global policies"
else
  echo "[+] no global policies to delete"
fi

# Delete fleet policies
fleet_ids=$(curl $CURL_FLAGS -H "$AUTH" "$API/teams" --insecure \
  | python3 -c "import sys,json; [print(t['id']) for t in json.load(sys.stdin).get('teams',[])]" 2>/dev/null)
for fid in $fleet_ids; do
  fleet_policy_ids=$(curl $CURL_FLAGS -H "$AUTH" "$API/teams/$fid/policies" --insecure | extract_ids "policies")
  if [ "$fleet_policy_ids" != "[]" ]; then
    curl -X POST $CURL_FLAGS -H "$AUTH" "$API/teams/$fid/policies/delete" --insecure \
      -d "{\"ids\": $fleet_policy_ids}" > /dev/null
    echo "[+] deleted policies for fleet $fid"
  fi
done

echo ""
echo "==> Deleting all reports..."

# Delete global reports
report_ids=$(curl $CURL_FLAGS -H "$AUTH" "$API/reports" --insecure | extract_ids "queries")
if [ "$report_ids" != "[]" ]; then
  curl -X POST $CURL_FLAGS -H "$AUTH" "$API/reports/delete" --insecure \
    -d "{\"ids\": $report_ids}" > /dev/null
  echo "[+] deleted global reports"
else
  echo "[+] no global reports to delete"
fi

# Delete fleet-scoped reports
for fid in $fleet_ids; do
  fleet_report_ids=$(curl $CURL_FLAGS -H "$AUTH" "$API/reports?team_id=$fid" --insecure | extract_ids "queries")
  if [ "$fleet_report_ids" != "[]" ]; then
    curl -X POST $CURL_FLAGS -H "$AUTH" "$API/reports/delete" --insecure \
      -d "{\"ids\": $fleet_report_ids}" > /dev/null
    echo "[+] deleted reports for fleet $fid"
  fi
done

echo ""
echo "==> Deleting all users (except ID 1)..."

user_ids=$(curl $CURL_FLAGS -H "$AUTH" "$API/users" --insecure | extract_ids "users" 1)
for user_id in $(echo "$user_ids" | python3 -c "import sys,json; [print(i) for i in json.load(sys.stdin)]"); do
  curl -X DELETE $CURL_FLAGS -H "$AUTH" "$API/users/$user_id" --insecure > /dev/null
  echo "[+] deleted user $user_id"
done

if [ "$user_ids" = "[]" ]; then
  echo "[+] no users to delete"
fi

echo ""
echo "==> Done! All seed data cleaned up (admin user preserved)."
