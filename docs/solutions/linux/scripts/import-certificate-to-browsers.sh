#!/bin/bash
# Script template used in this guide: https://fleetdm.com/guides/pingfederate-conditional-access-integration
# Imports a device certificate into each user's Chrome and Firefox certificate stores (NSS) so browsers can present it.
# Needs openssl and certutil/pk12util (libnss3-tools on Debian/Ubuntu, nss-tools on RHEL). Run as root.
set -euo pipefail

CERT_PATH="/opt/company/certificate.pem"
KEY_PATH="/opt/company/CustomerUserNetworkAccess.key"
KEY_PASSWORD="${PASSWORD:-}" # Set by the Hydrant/EST script. Leave empty if the key isn't encrypted.
NICKNAME="Fleet device certificate"

# Imports the certificate into one store. Returns nonzero if the store is locked or the import fails.
import_into() {
  local user=$1 db=$2 dir=$3 backup=""
  runuser -u "$user" -- mkdir -p "$db" || return 1
  if [ ! -f "$db/cert9.db" ]; then
    runuser -u "$user" -- certutil -N -d "sql:$db" --empty-password || return 1
  fi
  if runuser -u "$user" -- certutil -L -d "sql:$db" -n "$NICKNAME" >/dev/null 2>&1; then
    # Back up the current certificate so a failed renewal doesn't leave the browser without one.
    # The export needs the store password, so a Firefox Primary Password stops here, before any change.
    backup="$dir/previous.p12"
    runuser -u "$user" -- pk12util -o "$backup" -n "$NICKNAME" -d "sql:$db" -W "" -K "" >/dev/null || return 1
    # -F also deletes the private key, so old keys don't build up with each renewal.
    runuser -u "$user" -- certutil -F -d "sql:$db" -n "$NICKNAME" || return 1
  fi
  if runuser -u "$user" -- pk12util -i "$dir/new.p12" -d "sql:$db" -W "" -K "" >/dev/null; then
    rm -f "$backup"
    return 0
  fi
  [ -n "$backup" ] && runuser -u "$user" -- pk12util -i "$backup" -d "sql:$db" -W "" -K "" >/dev/null
  rm -f "$backup"
  return 1
}

failed=0
for home in /home/*; do
  user=$(stat -c %U "$home")
  [ "$user" = "UNKNOWN" ] || [ "$user" = "root" ] && continue

  dir=$(mktemp -d)
  openssl pkcs12 -export -in "$CERT_PATH" -inkey "$KEY_PATH" -passin "pass:$KEY_PASSWORD" \
    -name "$NICKNAME" -out "$dir/new.p12" -passout pass:
  chown -R "$user" "$dir"

  # Chrome/Chromium, Firefox (deb, snap, and Mozilla's apt package), and snap Chromium keep separate stores.
  dbs=("$home/.pki/nssdb")
  # Snap Chromium doesn't create its store until it first runs, so create it here.
  [ -d "$home/snap/chromium/current" ] && dbs+=("$home/snap/chromium/current/.pki/nssdb")
  for d in "$home"/.mozilla/firefox/*/ "$home"/.config/mozilla/firefox/*/ "$home"/snap/firefox/common/.mozilla/firefox/*/; do
    [ -f "$d/cert9.db" ] && dbs+=("${d%/}")
  done

  # One locked or failed store shouldn't stop imports for the remaining stores and users.
  for db in "${dbs[@]}"; do
    if ! import_into "$user" "$db" "$dir"; then
      echo "Skipped $db: the store has a Primary Password or the import failed." >&2
      failed=1
    fi
  done

  rm -rf "$dir"
done

# Exit nonzero so Fleet shows the script as failed when any store was skipped.
exit "$failed"
