#!/usr/bin/env bash

#--------------------------------------------------------------
# This script helps with testing batch setting of configuration
# profiles via the Fleet API. Change this file as needed
# to generate different test cases.
#--------------------------------------------------------------

if [[ -z "$FLEET_PATH" ]]; then
  echo "Error: FLEET_PATH environment variable is not set. This is the path to the Fleet project." >&2
  exit 1
fi

if [[ -z "$FLEET_SERVER_URL" ]]; then
  echo "Error: FLEET_SERVER_URL environment variable is not set. This is the URL of the Fleet server." >&2
  exit 1
fi

if [[ -z "$FLEET_AUTH_TOKEN" ]]; then
  echo "Error: FLEET_AUTH_TOKEN environment variable is not set. This is the authentication token used for Fleet API requests." >&2
  exit 1
fi

# generate request payload
payload="$(
$FLEET_PATH/tools/mdm/make_cfg_profiles.sh \
--file $FLEET_PATH/it-and-security/lib/macos/configuration-profiles/1password-managed-settings.mobileconfig --name "1Password Managed Settings" \
--labels-type include_all --label "test label 2" --next \
--file $FLEET_PATH/it-and-security/lib/windows/configuration-profiles/Enable\ Firewall.xml --name "Windows Enable Firewall" \
--labels-type include_any --label "test label 1" --next \
)"

# make request to Fleet API
curl -X POST "$FLEET_SERVER_URL/api/latest/fleet/configuration_profiles/batch" \
-H "Content-Type: application/json" \
-H "Authorization: Bearer $FLEET_AUTH_TOKEN" \
-d "$payload"
