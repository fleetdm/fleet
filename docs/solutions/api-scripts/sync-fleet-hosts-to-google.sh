#!/bin/bash
# Script template used in this guide: https://fleetdm.com/guides/google-conditional-access-integration
# Tells Google which iPhones and iPads are managed by Fleet, by setting a client state under your
# partner ID that Context-Aware Access checks. Google doesn't record serial numbers for iPhones and
# iPads, so the script matches them to Fleet hosts by end user email and device type. Set
# DRY_RUN=true to print changes without making them. Needs curl and jq.
set -euo pipefail

FLEET_URL="https://fleet.example.com"
FLEET_API_TOKEN="${FLEET_API_TOKEN:?Set FLEET_API_TOKEN}"
GOOGLE_ACCESS_TOKEN="${GOOGLE_ACCESS_TOKEN:?Set GOOGLE_ACCESS_TOKEN}"
GOOGLE_CUSTOMER_ID="<customer-ID>"
PARTNER_ID="$GOOGLE_CUSTOMER_ID-fleet"
DRY_RUN="${DRY_RUN:-false}"

fleet() {
  curl -fsS -H "Authorization: Bearer $FLEET_API_TOKEN" "$FLEET_URL/api/v1/fleet/$1"
}

google() { # method path [body]
  curl -fsS -X "$1" -H "Authorization: Bearer $GOOGLE_ACCESS_TOKEN" -H "Content-Type: application/json" \
    "https://cloudidentity.googleapis.com/v1/$2" ${3:+--data "$3"}
}

google_list() { # path key: prints each item under key, across all pages
  local token="" page
  while :; do
    page=$(google GET "$1&pageSize=100&pageToken=$token")
    jq -c ".$2[]?" <<<"$page"
    token=$(jq -r '.nextPageToken // empty' <<<"$page")
    [ -z "$token" ] && break
  done
}

page=0
hosts="[]"
while :; do
  batch=$(fleet "hosts?per_page=50&page=$page&device_mapping=true" | jq '.hosts')
  [ "$(jq length <<<"$batch")" -eq 0 ] && break
  hosts=$(jq -s 'add' <(echo "$hosts") <(echo "$batch"))
  page=$((page + 1))
done

# Fleet host IDs, keyed by "<end user email>/<iphone|ipad>".
fleet_keys=$(jq '
  [.[] | select((.platform == "ios" or .platform == "ipados") and ((.mdm.enrollment_status // "") | startswith("On")))
   | (if .platform == "ipados" then "ipad" else "iphone" end) as $kind | .id as $id
   | (.device_mapping // [])[] | {key: ((.email | ascii_downcase) + "/" + $kind), id: $id}]
  | unique | group_by(.key) | map({key: .[0].key, value: map(.id)}) | from_entries' <<<"$hosts")

devices=$(google_list "devices?customer=customers/my_customer&view=USER_ASSIGNED_DEVICES" devices | jq -s '
  map(select(.deviceType == "IOS") | {key: .name, value: (if ((.model // "") | test("^ipad"; "i")) then "ipad" else "iphone" end)})
  | from_entries')
users=$(google_list "devices/-/deviceUsers?customer=customers/my_customer" deviceUsers | jq -s --argjson devices "$devices" '
  map((.name | split("/deviceUsers/")[0]) as $d | select($devices | has($d))
    | {name, key: (((.userEmail // "") | ascii_downcase) + "/" + $devices[$d])})')
google_counts=$(jq 'group_by(.key) | map({key: .[0].key, value: length}) | from_entries' <<<"$users")

jq -c '.[]' <<<"$users" | while read -r user; do
  name=$(jq -r .name <<<"$user")
  key=$(jq -r .key <<<"$user")
  fleet_count=$(jq --arg k "$key" '.[$k] // [] | length' <<<"$fleet_keys")
  google_count=$(jq --arg k "$key" '.[$k]' <<<"$google_counts")

  host_id=""
  if [ "$fleet_count" -eq 1 ] && [ "$google_count" -eq 1 ]; then
    host_id=$(jq -r --arg k "$key" '.[$k][0]' <<<"$fleet_keys")
  elif [ "$fleet_count" -gt 0 ]; then
    # Don't guess. Google blocks the device until the match is unambiguous.
    echo "Review: $key has $fleet_count Fleet hosts and $google_count Google devices. Not marking $name as managed." >&2
  fi

  current=$(google GET "$name/clientStates/$PARTNER_ID?customer=customers/$GOOGLE_CUSTOMER_ID" 2>/dev/null | jq -r '.managed // empty' || true)
  if [ -n "$host_id" ]; then want=MANAGED compliance=COMPLIANT; else want=UNMANAGED compliance=NON_COMPLIANT; fi
  # Context-Aware Access already blocks devices that Fleet never marked as managed.
  if [ -z "$current" ] && [ "$want" = UNMANAGED ]; then continue; fi
  if [ "$current" = "$want" ]; then continue; fi

  echo "$name ($key): ${current:-none} -> $want"
  if [ "$DRY_RUN" = true ]; then continue; fi
  google PATCH "$name/clientStates/$PARTNER_ID?customer=customers/$GOOGLE_CUSTOMER_ID&updateMask=managed,complianceState,customId" \
    "$(jq -n --arg m "$want" --arg c "$compliance" --arg id "$host_id" '{managed: $m, complianceState: $c, customId: $id}')" >/dev/null
done
