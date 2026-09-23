#!/bin/bash
# Sets Okta Verify's OrgUrl in /etc/okta/adminconfig.json, keeping any other
# admin-managed keys already in the file, then restarts the Okta services.
# https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/managed-app-configs-linux.htm

set -euo pipefail

ORG_URL="https://fleetdm.okta.com"
CONFIG_DIR="/etc/okta"
CONFIG_FILE="$CONFIG_DIR/adminconfig.json"

if [ "$(id -u)" -ne 0 ]; then
  echo "[configure-okta-verify] This script must run as root." >&2
  exit 1
fi

mkdir -p "$CONFIG_DIR"
if getent group okta-ff >/dev/null 2>&1; then
  chown root:okta-ff "$CONFIG_DIR"
fi
chmod 1775 "$CONFIG_DIR"

TMP="$(mktemp "$CONFIG_DIR/.adminconfig.XXXXXX")"
trap 'rm -f "$TMP"' EXIT

if [ -s "$CONFIG_FILE" ] && command -v python3 >/dev/null 2>&1; then
  python3 - "$CONFIG_FILE" "$TMP" "$ORG_URL" <<'EOF'
import json, sys
src, dst, org_url = sys.argv[1:4]
try:
    with open(src) as f:
        data = json.load(f)
    if not isinstance(data, dict) or not isinstance(data.get("configuration"), dict):
        raise ValueError
except ValueError:
    data = {"configuration": {}}
data["configuration"]["OrgUrl"] = org_url
with open(dst, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
EOF
else
  cat >"$TMP" <<EOF
{
  "configuration": {
    "OrgUrl": "$ORG_URL"
  }
}
EOF
fi

chown root:root "$TMP"
chmod 0444 "$TMP"
mv -f "$TMP" "$CONFIG_FILE"
trap - EXIT

# Restart failures shouldn't fail the script: the desktop app picks up the file
# on relaunch or reboot either way.
for svc in okta-authenticator.service okta-feature-flag.service; do
  if systemctl cat "$svc" >/dev/null 2>&1; then
    systemctl restart "$svc" || echo "[configure-okta-verify] Couldn't restart $svc." >&2
  fi
done

echo "[configure-okta-verify] Set OrgUrl to $ORG_URL in $CONFIG_FILE."
