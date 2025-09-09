#!/usr/bin/env bash

$FLEET_PATH/tools/mdm/make_cfg_profiles.sh \
--file $FLEET_PATH/it-and-security/lib/macos/configuration-profiles/1password-managed-settings.mobileconfig --name "1Password Managed Settings" \
--labels-type include_all --label "test label 2" --next \
--file $FLEET_PATH/it-and-security/lib/windows/configuration-profiles/Enable\ Firewall.xml --name "Windows Enable Firewall" \
--labels-type include_any --label "test label 1" --next \
| curl -X POST "$FLEET_SERVER_URL/api/latest/fleet/configuration_profiles/batch" \
-H "Content-Type: application/json" \
-H "Authorization: Bearer $FLEET_AUTH_TOKEN" \
-d @-
