#!/bin/bash

# Seed users, fleets, policies, and reports for local development.
# See tools/seed/README.md for full documentation.

set -euo pipefail

source "$FLEET_ENV_PATH"

API="$SERVER_URL/api/latest/fleet"
AUTH="Authorization: Bearer $TOKEN"

# Validate required env vars
missing=""
[ -z "${SEED_PASSWORD:-}" ] && missing="$missing SEED_PASSWORD"
[ -z "${TOKEN:-}" ] && missing="$missing TOKEN"
[ -z "${SERVER_URL:-}" ] && missing="$missing SERVER_URL"
if [ -n "$missing" ]; then
  echo "ERROR: Missing required env vars:$missing. Check your env file." >&2
  exit 1
fi

# Configure fleetctl with the same token so apply commands work without a separate login
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FLEETCTL="${FLEETCTL:-./build/fleetctl}"
$FLEETCTL config set --address "$SERVER_URL" --token "$TOKEN" 2>/dev/null

# Helper: get fleet ID and exact name by partial match (to handle emoji prefixes),
# creating the fleet if it doesn't exist. Returns "id|exact_name".
# SECURITY: $name is interpolated into inline Python. Only pass hardcoded strings,
# never user input, to avoid code injection.
get_fleet_id() {
  local name="$1"
  local result
  result=$(curl $CURL_FLAGS -H "$AUTH" "$API/teams" --insecure \
    | python3 -c "
import sys, json
teams = json.load(sys.stdin).get('teams', [])
matches = [(t['id'], t['name']) for t in teams if '$name'.lower() in t['name'].lower()]
if matches:
    print(f'{matches[0][0]}|{matches[0][1]}')
else:
    print('')
" 2>/dev/null)

  if [ -z "$result" ]; then
    echo "Fleet '$name' not found, creating it..." >&2
    result=$(curl -X POST $CURL_FLAGS -H "$AUTH" -H "Content-Type: application/json" \
      "$API/teams" --insecure -d "{\"name\": \"$name\"}" \
      | python3 -c "import sys,json; t=json.load(sys.stdin)['team']; print(f\"{t['id']}|{t['name']}\")" 2>/dev/null)
    if [ -z "$result" ]; then
      echo "ERROR: Failed to create fleet '$name'. Is your dev server running with a premium license?" >&2
      exit 1
    fi
    echo "Created fleet '${result#*|}' (ID: ${result%%|*})" >&2
  else
    echo "Found fleet '${result#*|}' (ID: ${result%%|*})" >&2
  fi
  echo "$result"
}

# Helper: create user via /users/admin (ignores errors if user already exists)
# Sets LAST_USER_CREATED=1 if created, 0 if skipped.
LAST_USER_CREATED=0
create_user() {
  local data="$1"
  local name email
  name=$(echo "$data" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['name'])")
  email=$(echo "$data" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('email',''))")
  local result
  result=$(curl -X POST $CURL_FLAGS -H "$AUTH" -H "Content-Type: application/json" "$API/users/admin" --insecure -d "$data" 2>&1)
  if echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if 'user' in d else 1)" 2>/dev/null; then
    echo "Created user: $name ($email, pw: \$SEED_PASSWORD)"
    LAST_USER_CREATED=1
  else
    echo "Skipped user (may already exist): $name"
    LAST_USER_CREATED=0
  fi
}

# Helper: create API-only user via /users/api_only (email and password are auto-generated,
# API token is returned in the response)
# Sets LAST_USER_CREATED=1 if created, 0 if skipped.
create_api_only_user() {
  local data="$1"
  local name
  name=$(echo "$data" | python3 -c "import sys,json; print(json.load(sys.stdin)['name'])")
  local result
  result=$(curl -X POST $CURL_FLAGS -H "$AUTH" -H "Content-Type: application/json" "$API/users/api_only" --insecure -d "$data" 2>&1)
  local token
  token=$(echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('token',''))" 2>/dev/null)
  if echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if 'user' in d else 1)" 2>/dev/null; then
    echo "Created API-only user: $name"
    LAST_USER_CREATED=1
    if [ -n "$token" ]; then
      echo ""
      echo "  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓"
      echo "  ┃    SAVE THIS TOKEN — IT IS ONLY SHOWN ONCE!    ┃"
      echo "  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛"
      echo "  API token: $token"
      echo ""
    fi
  else
    LAST_USER_CREATED=0
    local errmsg
    errmsg=$(echo "$result" | python3 -c "
import sys, json
d = json.load(sys.stdin)
msg = d.get('message', '')
errors = d.get('errors', [])
details = '; '.join(e.get('name','') + ': ' + e.get('reason','') for e in errors if e.get('reason'))
print(details if details else msg if msg else 'unknown error')
" 2>/dev/null)
    echo "Skipped API-only user: $name — $errmsg"
  fi
}

echo "==> Finding fleets..."
FLEET_1_RESULT=$(get_fleet_id "Workstations")
FLEET_1_ID="${FLEET_1_RESULT%%|*}"
FLEET_1_NAME="${FLEET_1_RESULT#*|}"

FLEET_2_RESULT=$(get_fleet_id "mobile devices")
FLEET_2_ID="${FLEET_2_RESULT%%|*}"
FLEET_2_NAME="${FLEET_2_RESULT#*|}"

echo ""
echo "==> Creating users..."

# Global admin
create_user '{
  "name": "Anna G. Admin",
  "email": "anna@organization.com",
  "password": "'"$SEED_PASSWORD"'",
  "invited_by": 1,
  "global_role": "admin",
  "admin_forced_password_reset": false
}'

# Global maintainer
create_user '{
  "name": "Mary G. Maintainer",
  "email": "mary@organization.com",
  "password": "'"$SEED_PASSWORD"'",
  "invited_by": 1,
  "global_role": "maintainer",
  "admin_forced_password_reset": false
}'

# Global observer
create_user '{
  "name": "Oliver G. Observer",
  "email": "oliver@organization.com",
  "password": "'"$SEED_PASSWORD"'",
  "invited_by": 1,
  "global_role": "observer",
  "admin_forced_password_reset": false
}'

# Global observer+
create_user '{
  "name": "Opal G. Observer+",
  "email": "opal@organization.com",
  "password": "'"$SEED_PASSWORD"'",
  "invited_by": 1,
  "global_role": "observer_plus",
  "admin_forced_password_reset": false
}'

# Global technician
create_user '{
  "name": "Tessa G. Technician",
  "email": "tessa@organization.com",
  "password": "'"$SEED_PASSWORD"'",
  "invited_by": 1,
  "global_role": "technician",
  "admin_forced_password_reset": false
}'

# Mixed roles (observer on Workstations, maintainer on Mobile devices)
create_user "{
  \"name\": \"Marco Mixed Roles\",
  \"email\": \"marco@organization.com\",
  \"password\": \"$SEED_PASSWORD\",
  \"invited_by\": 1,
  \"global_role\": null,
  \"admin_forced_password_reset\": false,
  \"teams\": [
    {\"id\": $FLEET_1_ID, \"role\": \"observer\"},
    {\"id\": $FLEET_2_ID, \"role\": \"maintainer\"}
  ]
}"

# Fleet admin (Workstations)
create_user "{
  \"name\": \"Anita T. Admin\",
  \"email\": \"anita@organization.com\",
  \"password\": \"$SEED_PASSWORD\",
  \"invited_by\": 1,
  \"global_role\": null,
  \"admin_forced_password_reset\": false,
  \"teams\": [{\"id\": $FLEET_1_ID, \"role\": \"admin\"}]
}"

# Fleet maintainer (Workstations)
create_user "{
  \"name\": \"Manny T. Maintainer\",
  \"email\": \"manny@organization.com\",
  \"password\": \"$SEED_PASSWORD\",
  \"invited_by\": 1,
  \"global_role\": null,
  \"admin_forced_password_reset\": false,
  \"teams\": [{\"id\": $FLEET_1_ID, \"role\": \"maintainer\"}]
}"

# Fleet observer (Workstations)
create_user "{
  \"name\": \"Toni T. Observer\",
  \"email\": \"toni@organization.com\",
  \"password\": \"$SEED_PASSWORD\",
  \"invited_by\": 1,
  \"global_role\": null,
  \"admin_forced_password_reset\": false,
  \"teams\": [{\"id\": $FLEET_1_ID, \"role\": \"observer\"}]
}"

# Fleet observer+ (Workstations)
create_user "{
  \"name\": \"Topanga T. Observer+\",
  \"email\": \"topanga@organization.com\",
  \"password\": \"$SEED_PASSWORD\",
  \"invited_by\": 1,
  \"global_role\": null,
  \"admin_forced_password_reset\": false,
  \"teams\": [{\"id\": $FLEET_1_ID, \"role\": \"observer_plus\"}]
}"

# Fleet technician (Workstations)
create_user "{
  \"name\": \"Terry T. Technician\",
  \"email\": \"terry@organization.com\",
  \"password\": \"$SEED_PASSWORD\",
  \"invited_by\": 1,
  \"global_role\": null,
  \"admin_forced_password_reset\": false,
  \"teams\": [{\"id\": $FLEET_1_ID, \"role\": \"technician\"}]
}"

# Global GitOps
create_user '{
  "name": "Gina G. GitOps",
  "email": "gina@organization.com",
  "password": "'"$SEED_PASSWORD"'",
  "invited_by": 1,
  "global_role": "gitops",
  "admin_forced_password_reset": false
}'

# Fleet GitOps (Workstations)
create_user "{
  \"name\": \"Gordon T. GitOps\",
  \"email\": \"gordon@organization.com\",
  \"password\": \"$SEED_PASSWORD\",
  \"invited_by\": 1,
  \"global_role\": null,
  \"admin_forced_password_reset\": false,
  \"teams\": [{\"id\": $FLEET_1_ID, \"role\": \"gitops\"}]
}"

# API-only user (full access, created via /users/admin with api_only flag)
create_user '{
  "name": "Apollo G. API-only (full access)",
  "email": "apollo@organization.com",
  "password": "'"$SEED_PASSWORD"'",
  "invited_by": 1,
  "global_role": "maintainer",
  "api_only": true,
  "admin_forced_password_reset": false
}'
[ "$LAST_USER_CREATED" -eq 1 ] && echo "  ^ API-only user — use fleetctl login or API token auth only, no UI access"

# API-only user with restricted endpoints (created via /users/api_only)
create_api_only_user '{
  "name": "Reggie G. API-only (restricted)",
  "global_role": "admin",
  "api_endpoints": [
    {"method": "GET", "path": "/api/v1/fleet/hosts"},
    {"method": "GET", "path": "/api/v1/fleet/hosts/{id}"}
  ]
}'
[ "$LAST_USER_CREATED" -eq 1 ] && echo "  ^ Restricted API-only user — token auth only, no fleetctl or UI access"

echo ""
echo "==> Applying global policies and reports..."

$FLEETCTL apply -f "$SCRIPT_DIR/standard-policies.yml"
echo "[+] applied global policies"

$FLEETCTL apply -f "$SCRIPT_DIR/standard-reports.yml"
echo "[+] applied global reports"

echo ""
echo "==> Applying fleet-scoped policies..."

$FLEETCTL apply -f "$SCRIPT_DIR/fleet-workstations-policies.yml" --policies-fleet "$FLEET_1_NAME"
echo "[+] applied $FLEET_1_NAME policies"

$FLEETCTL apply -f "$SCRIPT_DIR/fleet-mobile-policies.yml" --policies-fleet "$FLEET_2_NAME"
echo "[+] applied $FLEET_2_NAME policies"

echo ""
echo "==> Applying fleet-scoped reports..."
# Reports need the exact fleet name in the YAML, so we substitute the discovered names.
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

sed "s|fleet: FLEET_NAME|fleet: $FLEET_1_NAME|" "$SCRIPT_DIR/fleet-workstations-reports.yml" > "$tmpdir/workstations-reports.yml"
$FLEETCTL apply -f "$tmpdir/workstations-reports.yml"
echo "[+] applied $FLEET_1_NAME reports"

sed "s|fleet: FLEET_NAME|fleet: $FLEET_2_NAME|" "$SCRIPT_DIR/fleet-mobile-reports.yml" > "$tmpdir/mobile-reports.yml"
$FLEETCTL apply -f "$tmpdir/mobile-reports.yml"
echo "[+] applied $FLEET_2_NAME reports"

echo ""
echo "==> Done! All users, policies, and reports seeded."
