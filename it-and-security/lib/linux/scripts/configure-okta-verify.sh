#!/bin/bash
# Post-install: pre-populate the org URL so end users aren't prompted to type
# it during Okta Verify enrollment. Okta Verify for Linux reads a single JSON
# policy file at /etc/okta/adminconfig.json; see:
# https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/managed-app-configs-linux.htm
#
# TODO(it-team): replace OKTA_ORG_URL below with this org's real Okta domain
# (e.g. https://your-org.okta.com) before this PR is merged.

set -euo pipefail

OKTA_ORG_URL="https://REPLACE-ME.okta.com"
CONFIG_DIR="/etc/okta"
CONFIG_FILE="$CONFIG_DIR/adminconfig.json"

if [ "$(id -u)" -ne 0 ]; then
  echo "[configure-okta-verify] This script must run as root." >&2
  exit 1
fi

mkdir -p "$CONFIG_DIR"

cat > "$CONFIG_FILE" <<JSON
{
  "OrgUrl": "$OKTA_ORG_URL"
}
JSON

# The policy file must be world-readable (Okta Verify, the okta-ff service,
# and root all read it) but writable only by root. The package's own postinst
# owns /etc/okta's directory permissions; this script only sets the file's.
chown root:root "$CONFIG_FILE"
chmod 0444 "$CONFIG_FILE"

echo "[configure-okta-verify] Wrote $CONFIG_FILE."
