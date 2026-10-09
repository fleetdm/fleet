#!/bin/bash
# Script template used in this guide: https://fleetdm.com/guides/google-conditional-access-integration
# Tells Google which iPhones and iPads are managed by Fleet, by setting a client state under your
# partner ID that Context-Aware Access checks. Google doesn't record serial numbers for iPhones and
# iPads, so the script matches them to Fleet hosts by the end user's email and device type. It also
# stores the Fleet host ID and serial number on the client state (Google Admin console > the device >
# Third-party services). Set DRY_RUN=true to print changes without making them. Needs curl, jq, and openssl.
set -euo pipefail

FLEET_URL="${FLEET_URL:-https://fleet.example.com}"
FLEET_API_TOKEN="${FLEET_API_TOKEN:?Set FLEET_API_TOKEN}"
# The service account's JSON key, and the admin account it acts as (domain-wide delegation). Or set GOOGLE_ACCESS_TOKEN instead.
GOOGLE_CREDENTIALS="${GOOGLE_CREDENTIALS:-}"
GOOGLE_ADMIN_EMAIL="${GOOGLE_ADMIN_EMAIL:-admin@example.com}"
GOOGLE_CUSTOMER_ID="${GOOGLE_CUSTOMER_ID:-<customer-ID>}" # Google Admin console > Account settings > Customer ID (starts with C)
PARTNER_ID="${GOOGLE_CUSTOMER_ID#C}-fleet" # Google's partner ID uses the customer ID without the leading C
# Which Fleet emails count, comma-separated. Default: the IdP email from enrollment (end user authentication, or an IdP
# username set by an admin). Add "custom" to also use emails set through the API or the UI.
EMAIL_SOURCES="${EMAIL_SOURCES:-mdm_idp_accounts}"
# Which MDM statuses count as managed, comma-separated. Default: Apple Business (automatic) and profile (manual) enrollments.
# Add "On (personal)" to also let personally owned (BYOD) iPhones and iPads in.
ENROLLMENT_STATUSES="${ENROLLMENT_STATUSES:-On (automatic),On (manual)}"
DRY_RUN="${DRY_RUN:-false}"

fleet() {
  curl -fsS -H "Authorization: Bearer $FLEET_API_TOKEN" "$FLEET_URL/api/v1/fleet/$1"
}

b64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }

# Get a Google access token as the admin account: a JWT signed with the service account's key. Needs openssl.
if [ -z "${GOOGLE_ACCESS_TOKEN:-}" ]; then
  [ -n "$GOOGLE_CREDENTIALS" ] || { echo "Set GOOGLE_CREDENTIALS (the service account's JSON key) or GOOGLE_ACCESS_TOKEN." >&2; exit 1; }
  now=$(date +%s)
  jwt="$(printf '{"alg":"RS256","typ":"JWT"}' | b64url).$(jq -nc --arg iss "$(jq -r .client_email <<<"$GOOGLE_CREDENTIALS")" \
    --arg sub "$GOOGLE_ADMIN_EMAIL" --argjson now "$now" '{iss: $iss, sub: $sub, iat: $now, exp: ($now + 600),
    scope: "https://www.googleapis.com/auth/cloud-identity.devices", aud: "https://oauth2.googleapis.com/token"}' | b64url)"
  jwt="$jwt.$(printf '%s' "$jwt" | openssl dgst -sha256 -sign <(jq -r .private_key <<<"$GOOGLE_CREDENTIALS") | b64url)"
  token=$(curl -sS https://oauth2.googleapis.com/token --data-urlencode "assertion=$jwt" \
    --data-urlencode grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer)
  GOOGLE_ACCESS_TOKEN=$(jq -r '.access_token // empty' <<<"$token")
  # unauthorized_client usually means the domain-wide delegation is missing or not active yet (it can take a few minutes).
  [ -n "$GOOGLE_ACCESS_TOKEN" ] || { echo "Google didn't issue a token: $(jq -r '"\(.error): \(.error_description)"' <<<"$token")" >&2; exit 1; }
fi

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

# iPhones and iPads with MDM on in Fleet (ENROLLMENT_STATUSES only).
managed=$(jq --arg statuses "$ENROLLMENT_STATUSES" '($statuses | split(",")) as $ok
  | [.[] | select((.platform == "ios" or .platform == "ipados") and ((.mdm.enrollment_status // "") as $s | $ok | index($s)))]' <<<"$hosts")
# Their Fleet host IDs, keyed by "<end user email>/<iphone|ipad>" (emails from EMAIL_SOURCES only), and each one's serial number.
fleet_keys=$(jq --arg sources "$EMAIL_SOURCES" '
  ($sources | split(",")) as $ok
  | [.[] | (if .platform == "ipados" then "ipad" else "iphone" end) as $kind | .id as $id
   | (.device_mapping // [])[] | select(.source as $s | $ok | index($s)) | {key: ((.email | ascii_downcase) + "/" + $kind), id: $id}]
  | unique | group_by(.key) | map({key: .[0].key, value: map(.id)}) | from_entries' <<<"$managed")
serials=$(jq 'map({key: (.id | tostring), value: (.hardware_serial // "")}) | from_entries' <<<"$managed")

# Flag managed iPhones and iPads that have no usable email: the script can't match them to a Google user.
jq -r --argjson keys "$fleet_keys" --arg sources "$EMAIL_SOURCES" '([$keys[][]] | unique) as $matched
  | .[] | select(.id as $id | $matched | index($id) | not)
  | "No email: Fleet host \(.id) has MDM on but no email from \($sources), so it can'"'"'t be matched to a Google user."' <<<"$managed" >&2

# Refuse to run when no managed iPhone or iPad has a usable email: an outage, a wrong token scope or missing emails would
# otherwise mark every Google device unmanaged at once. The run fails, so it shows in the scheduler (for example GitHub Actions).
if [ "$(jq 'length' <<<"$fleet_keys")" -eq 0 ] && [ "${ALLOW_EMPTY:-false}" != true ]; then
  echo "Fleet returned no managed iPhones or iPads with an email from $EMAIL_SOURCES. Not changing anything. Set ALLOW_EMPTY=true to override." >&2
  exit 1
fi

devices=$(google_list "devices?customer=customers/my_customer&view=USER_ASSIGNED_DEVICES" devices | jq -s '
  map(select(.deviceType == "IOS") | {key: .name, value: (if ((.model // "") | test("^ipad"; "i")) then "ipad" else "iphone" end)})
  | from_entries')
users=$(google_list "devices/-/deviceUsers?customer=customers/my_customer" deviceUsers | jq -s --argjson devices "$devices" '
  map((.name | split("/deviceUsers/")[0]) as $d | select($devices | has($d))
    | {name, key: (((.userEmail // "") | ascii_downcase) + "/" + $devices[$d])})')
google_counts=$(jq 'group_by(.key) | map({key: .[0].key, value: length}) | from_entries' <<<"$users")

changes=0
while read -r user; do
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

  # Use customers/my_customer: with domain-wide delegation, customers/<customer-ID> returns HTTP 400.
  state="$name/clientStates/$PARTNER_ID?customer=customers/my_customer"
  current=$(google GET "$state" 2>/dev/null | jq -c '{managed: (.managed // ""), assetTags: (.assetTags // [])}' ||
    jq -nc '{managed: "", assetTags: []}')
  if [ -n "$host_id" ]; then
    want=$(jq -nc --arg serial "$(jq -r --arg id "$host_id" '.[$id]' <<<"$serials")" '{managed: "MANAGED", assetTags: ([$serial] - [""])}')
  else
    want=$(jq -nc '{managed: "UNMANAGED", assetTags: []}')
  fi
  was=$(jq -r .managed <<<"$current")
  now=$(jq -r .managed <<<"$want")
  # Context-Aware Access already blocks devices that Fleet never marked as managed.
  if [ -z "$was" ] && [ "$now" = UNMANAGED ]; then continue; fi
  if [ "$current" = "$want" ]; then continue; fi

  echo "$name ($key): ${was:-none} -> $now$([ "$was" = "$now" ] && echo " (serial number updated)" || true)"
  changes=$((changes + 1))
  if [ "$DRY_RUN" = true ]; then continue; fi
  # Send the whole state without updateMask: with one, Google adds to assetTags instead of replacing them.
  google PATCH "$state" "$(jq -c --arg id "$host_id" \
    '{managed, complianceState: (if .managed == "MANAGED" then "COMPLIANT" else "NON_COMPLIANT" end), customId: $id, assetTags}' <<<"$want")" >/dev/null
done < <(jq -c '.[]' <<<"$users")

echo "Fleet: $(jq length <<<"$managed") managed iPhones and iPads, $(jq '[.[][]] | unique | length' <<<"$fleet_keys") with an email from $EMAIL_SOURCES. Google: $(jq length <<<"$users") iPhone and iPad users. Changes: $changes$([ "$DRY_RUN" = true ] && echo " (dry run)" || true)."
