#!/bin/bash
# Script template used in this guide: https://fleetdm.com/guides/duo-conditional-access-integration
# Writes macos.csv, windows.csv, and linux.csv for Duo's device_cache_sync.py, with the
# device IDs of Fleet hosts. Set REQUIRE_PASSING_CRITICAL_POLICIES=true to only include hosts
# that are passing all critical policies. Needs curl and jq.
set -euo pipefail

FLEET_URL="https://fleet.example.com"
FLEET_API_TOKEN="${FLEET_API_TOKEN:?Set FLEET_API_TOKEN}"
WINDOWS_REPORT_ID="<Windows-MachineGuid-report-ID>"
REQUIRE_PASSING_CRITICAL_POLICIES="${REQUIRE_PASSING_CRITICAL_POLICIES:-false}"

api() {
  curl -fsS -H "Authorization: Bearer $FLEET_API_TOKEN" "$FLEET_URL/api/v1/fleet/$1"
}

page=0
hosts="[]"
while :; do
  batch=$(api "hosts?per_page=500&page=$page&populate_policies=$REQUIRE_PASSING_CRITICAL_POLICIES" | jq '.hosts')
  [ "$(jq length <<<"$batch")" -eq 0 ] && break
  hosts=$(jq -s 'add' <(echo "$hosts") <(echo "$batch"))
  page=$((page + 1))
done
trusted=$(jq --argjson require "$REQUIRE_PASSING_CRITICAL_POLICIES" '[.[] | select(($require | not) or ([.policies[]? | select(.critical and .response == "fail")] | length == 0))]' <<<"$hosts")

# Duo identifies macOS and Linux hosts by hardware UUID, the same as Fleet's host UUID.
jq -r '"device_id", (.[] | select(.platform == "darwin") | .uuid)' <<<"$trusted" >macos.csv
jq -r '"device_id", (.[] | select(.platform as $p | ["darwin", "windows", "ios", "ipados", "android", "chrome"] | index($p) | not) | .uuid)' <<<"$trusted" >linux.csv

# Duo identifies Windows hosts by MachineGuid, collected by the Fleet report.
api "reports/$WINDOWS_REPORT_ID/report" |
  jq -r --argjson trusted "$trusted" '
    ($trusted | map(select(.platform == "windows") | .id)) as $ids |
    "device_id", ([.results[] | select(.host_id as $h | $ids | index($h)) | .columns.machine_guid] | unique[])
  ' >windows.csv

wc -l macos.csv windows.csv linux.csv
