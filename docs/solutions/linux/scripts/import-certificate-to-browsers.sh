#!/bin/bash
# Script template used in this guide: https://fleetdm.com/guides/pingfederate-conditional-access-integration
# Imports a device certificate into each user's Chrome and Firefox certificate stores (NSS) so browsers can present it.
# Needs openssl and certutil/pk12util (libnss3-tools on Debian/Ubuntu, nss-tools on RHEL). Run as root.
set -euo pipefail

CERT_PATH="/opt/company/certificate.pem"
KEY_PATH="/opt/company/CustomerUserNetworkAccess.key"
KEY_PASSWORD="${PASSWORD:-}" # Set by the Hydrant/EST script. Leave empty if the key isn't encrypted.
NICKNAME="Fleet device certificate"

for home in /home/*; do
  user=$(stat -c %U "$home")
  [ "$user" = "UNKNOWN" ] || [ "$user" = "root" ] && continue

  p12=$(mktemp)
  openssl pkcs12 -export -in "$CERT_PATH" -inkey "$KEY_PATH" -passin "pass:$KEY_PASSWORD" \
    -name "$NICKNAME" -out "$p12" -passout pass:
  chown "$user" "$p12"

  # Chrome/Chromium, Firefox (deb and snap), and snap Chromium keep separate stores.
  dbs=("$home/.pki/nssdb")
  for d in "$home"/.mozilla/firefox/*/ "$home"/snap/firefox/common/.mozilla/firefox/*/ "$home"/snap/chromium/current/.pki/nssdb/; do
    [ -f "$d/cert9.db" ] && dbs+=("${d%/}")
  done

  for db in "${dbs[@]}"; do
    runuser -u "$user" -- mkdir -p "$db"
    [ -f "$db/cert9.db" ] || runuser -u "$user" -- certutil -N -d "sql:$db" --empty-password
    runuser -u "$user" -- certutil -D -d "sql:$db" -n "$NICKNAME" 2>/dev/null || true # Replace on renewal.
    runuser -u "$user" -- pk12util -i "$p12" -d "sql:$db" -W "" -K ""
  done

  rm -f "$p12"
done
