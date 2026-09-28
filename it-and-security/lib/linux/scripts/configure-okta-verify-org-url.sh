#!/bin/bash
# Points Okta Verify at Fleet's Okta org via /etc/okta/adminconfig.json.
# https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/managed-app-configs-linux.htm

set -euo pipefail

CONFIG_DIR="/etc/okta"
CONFIG_FILE="$CONFIG_DIR/adminconfig.json"

mkdir -p "$CONFIG_DIR"
if getent group okta-ff >/dev/null; then
  chown root:okta-ff "$CONFIG_DIR"
fi
chmod 1775 "$CONFIG_DIR"

# Write to a temp file and rename so the app never reads a partial file.
TMP="$(mktemp "$CONFIG_DIR/.adminconfig.XXXXXX")"
trap 'rm -f "$TMP"' EXIT
cat >"$TMP" <<'EOF'
{
  "configuration": {
    "OrgUrl": "https://fleetdm.okta.com"
  }
}
EOF
chmod 0444 "$TMP"
mv -f "$TMP" "$CONFIG_FILE"

systemctl try-restart okta-authenticator.service okta-feature-flag.service || true

echo "Set Okta Verify OrgUrl in $CONFIG_FILE."
